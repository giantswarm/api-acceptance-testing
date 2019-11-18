package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/gsclientgen/client/info"
	"github.com/giantswarm/gsclientgen/client/key_pairs"
	"github.com/giantswarm/gsclientgen/client/nodepools"
	"github.com/giantswarm/gsclientgen/models"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/node-pools-acceptance-test/pkg/client"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/cliutil"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/kubeconfig"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/load"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/shell"
)

// TestClient verifies whether the given client can authenticate.
func TestClient(giantSwarmClient *client.Client) error {
	params := info.NewGetInfoParams()
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	infoResponse, err := giantSwarmClient.GSClientGen.Info.GetInfo(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("Client initialized and user authenticated")
	fmt.Printf("Installation name: %s\n", infoResponse.Payload.General.InstallationName)

	return nil
}

// CreateClusterUsingDefaults tests
// - whether we can create a cluster
// - whether defaults are applied as expected.
func CreateClusterUsingDefaults(giantSwarmClient *client.Client) (string, string, error) {
	var creationResult *clusters.AddClusterV5Created
	var err error

	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	org := "giantswarm"
	req := &models.V5AddClusterRequest{
		Owner:          &org,
		ReleaseVersion: "8.6.0", // TODO: temporary hack
	}
	params := clusters.NewAddClusterV5Params().WithBody(req)
	creationResult, err = giantSwarmClient.GSClientGen.Clusters.AddClusterV5(params, authWriter)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	// Verify cluster details
	if creationResult.Payload.Name != "Unnamed cluster" {
		cliutil.Complain(microerror.Newf("Cluster name is not 'Unnamed cluster' but '%s'", creationResult.Payload.Name))
	}
	if creationResult.Payload.APIEndpoint == "" {
		cliutil.Complain(microerror.New("Cluster api_endpoint is empty"))
	}
	if creationResult.Payload.Master == nil {
		cliutil.Complain(microerror.New("Cluster master is empty"))
	} else if creationResult.Payload.Master.AvailabilityZone == "" {
		cliutil.Complain(microerror.New("Cluster master.availability_zone is empty"))
	}
	if creationResult.Payload.ReleaseVersion == "" {
		cliutil.Complain(microerror.New("Cluster release_version is empty"))
	}
	if creationResult.Payload.CreateDate == "" {
		cliutil.Complain(microerror.New("Cluster create_date is empty"))
	}
	if creationResult.Payload.Owner == "" {
		cliutil.Complain(microerror.New("Cluster owner is empty"))
	}

	if creationResult.Payload.ID == "" {
		// we can't continue without this
		return "", "", microerror.New("Cluster ID is empty")
	}

	cliutil.PrintSuccess("Cluster created with ID %s", creationResult.Payload.ID)
	return creationResult.Payload.ID, creationResult.Payload.APIEndpoint, nil
}

// CreateNodePoolUsingDefaults ensures that a node pool can be created with minimal spec and defaults apply.
func CreateNodePoolUsingDefaults(giantSwarmClient *client.Client, clusterID string) (string, error) {
	var err error
	var creationResult *nodepools.AddNodePoolCreated

	req := &models.V5AddNodePoolRequest{}
	params := nodepools.NewAddNodePoolParams().WithClusterID(clusterID).WithBody(req)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", microerror.Mask(err)
	}
	creationResult, err = giantSwarmClient.GSClientGen.Nodepools.AddNodePool(params, authWriter)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// validation creation result
	if creationResult.Payload.Name == "" {
		cliutil.Complain(microerror.New("'name' is missing in node pool creation response"))
	}

	if len(creationResult.Payload.AvailabilityZones) == 0 {
		cliutil.Complain(microerror.New("'availability_zones' in node pool creation response is empty"))
	} else if len(creationResult.Payload.AvailabilityZones) > 1 {
		cliutil.Complain(microerror.Newf("'availability_zones' has % items instead of 1", len(creationResult.Payload.AvailabilityZones)))
	}

	if creationResult.Payload.Scaling == nil {
		cliutil.Complain(microerror.New("'scaling' is missing in node pool creation response"))
	} else {
		if creationResult.Payload.Scaling.Min != 3 {
			cliutil.Complain(microerror.New("'scaling.min' in node pool creation response is != 3"))
		}
		if creationResult.Payload.Scaling.Max != 10 {
			cliutil.Complain(microerror.New("'scaling.max' in node pool creation response is != 10"))
		}
	}

	if creationResult.Payload.Subnet == "" {
		cliutil.Complain(microerror.New("'subnet' is missing in node pool creation response"))
	}

	if creationResult.Payload.NodeSpec == nil {
		cliutil.Complain(microerror.New("'node_spec' is missing in node pool creation response"))
	} else {
		if creationResult.Payload.NodeSpec.Aws == nil {
			cliutil.Complain(microerror.New("'node_spec.aws' is missing in node pool creation response"))
		} else {
			if creationResult.Payload.NodeSpec.Aws.InstanceType == "" {
				cliutil.Complain(microerror.New("'node_spec.aws.instance_type' is missing in node pool creation response"))
			} else {
				cliutil.PrintInfo("'node_spec.aws.instance_type' is %s", creationResult.Payload.NodeSpec.Aws.InstanceType)
			}
		}

		if creationResult.Payload.NodeSpec.VolumeSizesGb == nil {
			cliutil.Complain(microerror.New("'node_spec.volume_sizes_gb' is missing in node pool creation response"))
		} else {
			if creationResult.Payload.NodeSpec.VolumeSizesGb.Docker == 0 {
				cliutil.Complain(microerror.New("'node_spec.volume_sizes_gb.docker' in node pool creation response is zero"))
			}
			if creationResult.Payload.NodeSpec.VolumeSizesGb.Kubelet == 0 {
				cliutil.Complain(microerror.New("'node_spec.volume_sizes_gb.kubelet' in node pool creation response is zero"))
			}
		}
	}

	if creationResult.Payload.ID == "" {
		// we can't continue without this
		return "", microerror.New("Node pool ID is missing in node pool creation response")
	}

	return creationResult.Payload.ID, nil

}

// CreateNodePoolWithCustomParams checks the creation of a node pool with some custom properties.
func CreateNodePoolWithCustomParams(giantSwarmClient *client.Client, clusterID string, instanceType string, availabilityZones []string) (string, error) {
	var err error
	var creationResult *nodepools.AddNodePoolCreated

	// Build request body using the given arguments.
	req := &models.V5AddNodePoolRequest{}
	if instanceType != "" {
		req.NodeSpec = &models.V5AddNodePoolRequestNodeSpec{
			Aws: &models.V5AddNodePoolRequestNodeSpecAws{
				InstanceType: instanceType,
			},
		}
	}
	if len(availabilityZones) != 0 {
		req.AvailabilityZones = &models.V5AddNodePoolRequestAvailabilityZones{
			Zones: availabilityZones,
		}
	}

	params := nodepools.NewAddNodePoolParams().WithClusterID(clusterID).WithBody(req)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", microerror.Mask(err)
	}
	creationResult, err = giantSwarmClient.GSClientGen.Nodepools.AddNodePool(params, authWriter)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// validate response.
	if creationResult.Payload.NodeSpec != nil {
		if creationResult.Payload.NodeSpec.Aws != nil {
			if instanceType != "" && creationResult.Payload.NodeSpec.Aws.InstanceType != instanceType {
				cliutil.Complain(microerror.Newf("'node_spec.aws.instance_type' in node pool creation response is not %s", instanceType))
			}
		} else {
			cliutil.Complain(microerror.New("'node_spec.aws' in node pool creation response is nil"))
		}
	} else {
		cliutil.Complain(microerror.New("'node_spec' in node pool creation response is nil"))
	}

	if len(creationResult.Payload.AvailabilityZones) != 0 {
		if len(availabilityZones) != 0 {
			if !cmp.Equal(creationResult.Payload.AvailabilityZones, availabilityZones) {
				cliutil.Complain(microerror.Newf("\n%s\n", cmp.Diff(availabilityZones, creationResult.Payload.AvailabilityZones)))
			}
		}
	} else {
		cliutil.Complain(microerror.New("'availability_zones' in node pool creation response is empty"))
	}

	return creationResult.Payload.ID, nil
}

// CreateKeyPair tests key pair creation for the new cluster
// and stores away the key pair in a kubectl config file for later use.
func CreateKeyPair(giantSwarmClient *client.Client, clusterID string, clusterAPIEndpoint string) (string, error) {
	description := "test key pair"
	req := &models.V4AddKeyPairRequest{
		TTLHours:                 12,
		Description:              &description,
		CertificateOrganizations: "system:masters",
		CnPrefix:                 "user@giantswarm.io",
	}
	params := key_pairs.NewAddKeyPairParams().WithClusterID(clusterID).WithBody(req)

	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", microerror.Mask(err)
	}

	addKeyPairResponse, err := giantSwarmClient.GSClientGen.KeyPairs.AddKeyPair(params, authWriter)
	if err != nil {
		if myErr, ok := err.(*key_pairs.AddKeyPairDefault); ok {
			return "", microerror.Maskf(err, "Code=%d, Details: %s", myErr.Code(), myErr.Payload.Message)
		}
		return "", microerror.Mask(err)
	}

	if addKeyPairResponse.Payload.ID == "" {
		return "", microerror.New("'id' in key pair creation response is empty")
	}

	// store kubeconfig file
	fs := afero.NewOsFs()
	path := fmt.Sprintf("kubeconfig_uat_%s_%s.yaml", clusterID, cleanupKeyPairID(addKeyPairResponse.Payload.ID))
	cliutil.PrintInfo("Storing the key pair in kubeconfig file %s", path)
	err = kubeconfig.WriteKubeconfigFile(fs, path, clusterAPIEndpoint, addKeyPairResponse.Payload.CertificateAuthorityData, addKeyPairResponse.Payload.ClientCertificateData, addKeyPairResponse.Payload.ClientKeyData)
	if err != nil {
		return "", microerror.Mask(err)
	}

	cliutil.PrintSuccess("Key pair for cluster %s has been created with ID %s", clusterID, addKeyPairResponse.Payload.ID)
	return path, nil
}

// RunKubectlCommandToTestKeyPair used kubectl to get a list of cluster nodes and returns an error if that fails.
func RunKubectlCommandToTestKeyPair(kubeconfigPath string) error {
	out, exitCode, err := shell.RunCommand(context.Background(), "kubectl", []string{}, "--kubeconfig", kubeconfigPath, "get", "nodes")
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("kubectl get nodes exited with code %d and printed:\n\n", exitCode)
	cliutil.PrintInfo(out)
	return nil
}

// DeployTestApp attempts to deploy a helloworld app on the cluster.
// Returns the ingress URL of the app.
func DeployTestApp(kubeconfigPath string, clusterAPIEndpoint string) (string, error) {
	// cluster base domain based on API endpoint
	clusterBaseDomain := strings.Replace(clusterAPIEndpoint, "https://api.", "", 1)

	templatePath := "./testapp-manifest.yaml.template"
	manifestPath := "./testapp-manifest.yaml"
	fs := afero.NewOsFs()
	templateData, err := afero.ReadFile(fs, templatePath)
	if err != nil {
		return "", microerror.Mask(err)
	}

	manifest := strings.Replace(string(templateData), "CLUSTER_BASE_DOMAIN", clusterBaseDomain, -1)
	err = afero.WriteFile(fs, manifestPath, []byte(manifest), 0644)
	if err != nil {
		return "", microerror.Mask(err)
	}

	out, exitCode, err := shell.RunCommand(context.Background(), "kubectl", []string{}, "--kubeconfig", kubeconfigPath, "apply", "-f", manifestPath)
	if err != nil {
		return "", microerror.Mask(err)
	}

	endpoint := "http://test." + clusterBaseDomain + "/delay/1"

	// Wait for the ingress to be reachable.
	start := time.Now()
	operation := func() error {
		resp, err := http.Get(endpoint)
		if err != nil {
			return err
		}

		cliutil.PrintInfo("Got status code %d", resp.StatusCode)
		if resp.StatusCode >= 400 {
			return microerror.Mask(fmt.Errorf("Got bad response from endpoint: status code %d", resp.StatusCode))
		}

		return nil
	}
	err = backoff.Retry(operation, backoff.NewConstantBackOff(10*time.Second))
	if err != nil {
		return "", microerror.Mask(err)
	}

	duration := time.Now().Sub(start)
	cliutil.PrintInfo("Ingress at %s reached after %s", endpoint, duration)

	cliutil.PrintSuccess("kubectl apply exited with code %d and printed:\n\n", exitCode)
	cliutil.PrintInfo(out)
	return endpoint, nil
}

// CreateLoadOnIngress sets a constant load on the given URL.
func CreateLoadOnIngress(ingressEndpoint string) {
	go load.ProduceLoad(ingressEndpoint, 5*time.Hour, 100_000_000_000)
}

// IncreaseTestAppReplicas increases the test app replicas.
func IncreaseTestAppReplicas(kubeconfigPath string) error {
	out, exitCode, err := shell.RunCommand(context.Background(), "kubectl", []string{}, "--kubeconfig", kubeconfigPath, "scale", "--replicas=5", "deployment/e2e-app")
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("kubectl scale exited with code %d and printed:\n\n", exitCode)
	cliutil.PrintInfo(out)
	return nil
}

// DeleteCluster tests whether a cluster gets deleted okay.
func DeleteCluster(giantSwarmClient *client.Client, clusterID string) error {
	deleteClusterOneParams := clusters.NewDeleteClusterParams().WithClusterID(clusterID)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = giantSwarmClient.GSClientGen.Clusters.DeleteCluster(deleteClusterOneParams, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("Cluster %s has been deleted", clusterID)
	return nil
}

// DeleteNodePool tests whether a node pool can be deleted.
func DeleteNodePool(giantSwarmClient *client.Client, clusterID string, nodePoolID string) error {
	params := nodepools.NewDeleteNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = giantSwarmClient.GSClientGen.Nodepools.DeleteNodePool(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("Nodepool %s/%s has been deleted", clusterID, nodePoolID)
	return nil
}

// ScaleNodePool tests whether a node pool can be scaled.
func ScaleNodePool(giantSwarmClient *client.Client, clusterID string, nodePoolID string, min int, max int) error {
	modifyBody := &models.V5ModifyNodePoolRequest{
		Scaling: &models.V5ModifyNodePoolRequestScaling{
			Min: int64(min),
			Max: int64(max),
		},
	}
	params := nodepools.NewModifyNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID).WithBody(modifyBody)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	response, err := giantSwarmClient.GSClientGen.Nodepools.ModifyNodePool(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Payload.Scaling == nil {
		cliutil.Complain(microerror.New("'scaling' is missing in node pool modification response"))
	} else {
		if response.Payload.Scaling.Min != int64(min) {
			cliutil.Complain(microerror.Newf("'scaling.min' in node pool modification response is not %d", min))
		}
		if response.Payload.Scaling.Min != int64(max) {
			cliutil.Complain(microerror.Newf("'scaling.min' in node pool modification response is not %d", max))
		}
	}

	cliutil.PrintSuccess("Nodepool %s/%s has been scaled", clusterID, nodePoolID)
	return nil
}

// RenameNodePool tests whether a node pool can be renamed.
func RenameNodePool(giantSwarmClient *client.Client, clusterID string, nodePoolID string, name string) error {
	modifyBody := &models.V5ModifyNodePoolRequest{
		Name: name,
	}
	params := nodepools.NewModifyNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID).WithBody(modifyBody)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	response, err := giantSwarmClient.GSClientGen.Nodepools.ModifyNodePool(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Payload.Name != name {
		cliutil.Complain(microerror.Newf("'name' in node pool modification response is not %q", name))
	}

	cliutil.PrintSuccess("Nodepool %s/%s has been renamed", clusterID, nodePoolID)
	return nil
}

// GetNodePoolDetails returns details on a node pool
func GetNodePoolDetails(giantSwarmClient *client.Client, clusterID string, nodePoolID string) (*models.V5GetNodePoolResponse, error) {
	params := nodepools.NewGetNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	details, err := giantSwarmClient.GSClientGen.Nodepools.GetNodePool(params, authWriter)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return details.Payload, nil
}
