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
	"github.com/giantswarm/gsclientgen/client/node_pools"
	"github.com/giantswarm/gsclientgen/models"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/api-acceptance-test/pkg/client"
	"github.com/giantswarm/api-acceptance-test/pkg/cliutil"
	"github.com/giantswarm/api-acceptance-test/pkg/kubeconfig"
	"github.com/giantswarm/api-acceptance-test/pkg/load"
	"github.com/giantswarm/api-acceptance-test/pkg/shell"
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
func CreateClusterUsingDefaults(giantSwarmClient *client.Client, ownerOrg string, releaseVersion string) (string, string, error) {
	var creationResult *clusters.AddClusterV5Created
	var err error

	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	clusterName := "api-acceptance-testing "
	if releaseVersion != "" {
		clusterName += "v" + releaseVersion + " "
	}
	clusterName += time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	req := &models.V5AddClusterRequest{
		Name:           clusterName,
		Owner:          &ownerOrg,
		ReleaseVersion: releaseVersion,
	}
	params := clusters.NewAddClusterV5Params().WithBody(req)
	creationResult, err = giantSwarmClient.GSClientGen.Clusters.AddClusterV5(params, authWriter)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	// Verify cluster details
	if creationResult.Payload.Name != clusterName {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster name is not 'Unnamed cluster' but '%s'", creationResult.Payload.Name))
	}
	if creationResult.Payload.APIEndpoint == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster api_endpoint is empty"))
	}
	if creationResult.Payload.Master == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster master is empty"))
	} else if creationResult.Payload.Master.AvailabilityZone == "" {
		cliutil.PrintInfo("'creationResult.Payload.Master.AvailabilityZone' is empty.")
	} else {
		cliutil.PrintInfo("'creationResult.Payload.Master.AvailabilityZone' is %s", creationResult.Payload.Master.AvailabilityZone)
	}
	if creationResult.Payload.MasterNodes == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane master is empty"))
	} else if creationResult.Payload.MasterNodes.AvailabilityZones == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane availability_zones is empty"))
	} else {
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.AvailabilityZones' is %s", creationResult.Payload.MasterNodes.AvailabilityZones)
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.HighAvailability' is %t", creationResult.Payload.MasterNodes.HighAvailability)
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.NumReady' is %d", creationResult.Payload.MasterNodes.NumReady)
	}
	if creationResult.Payload.ReleaseVersion == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster release_version is empty"))
	} else {
		cliutil.PrintInfo("'creationResult.Payload.ReleaseVersion' is %s", creationResult.Payload.ReleaseVersion)
	}
	if creationResult.Payload.CreateDate == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster create_date is empty"))
	}
	if creationResult.Payload.Owner == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster owner is empty"))
	}

	if creationResult.Payload.ID == "" {
		// we can't continue without this
		return "", "", microerror.Maskf(assertionFailedError, "Cluster ID is empty")
	}

	cliutil.PrintSuccess("Cluster created with ID %s", creationResult.Payload.ID)
	return creationResult.Payload.ID, creationResult.Payload.APIEndpoint, nil
}

// CreateClusterUsingDefaults tests
// - whether we can create a cluster
// - whether defaults are applied as expected.
func CreateCluster(giantSwarmClient *client.Client, ownerOrg string, releaseVersion string, style string, AZ string, HighAvailability bool) (string, string, error) {
	var creationResult *clusters.AddClusterV5Created
	var err error

	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	clusterName := "Antonia api-acceptance-testing "
	if releaseVersion != "" {
		clusterName += "v" + releaseVersion + " "
	}
	clusterName += time.Now().UTC().Format("2006-01-02 15:04:05 UTC")

	req := &models.V5AddClusterRequest{
		Name:           clusterName,
		Owner:          &ownerOrg,
		ReleaseVersion: releaseVersion,
	}

	if style == "old" {
		req = &models.V5AddClusterRequest{
			Name:           clusterName,
			Owner:          &ownerOrg,
			ReleaseVersion: releaseVersion,
			Master: &models.V5AddClusterRequestMaster{
				AvailabilityZone: AZ,
			},
		}
	} else if style == "new" {
		req = &models.V5AddClusterRequest{
			Name:           clusterName,
			Owner:          &ownerOrg,
			ReleaseVersion: releaseVersion,
			MasterNodes: &models.V5AddClusterRequestMasterNodes{
				HighAvailability: &HighAvailability,
			},
		}
	}

	params := clusters.NewAddClusterV5Params().WithBody(req)
	creationResult, err = giantSwarmClient.GSClientGen.Clusters.AddClusterV5(params, authWriter)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	// Verify cluster details
	if creationResult.Payload.Name != clusterName {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster name is not 'Unnamed cluster' but '%s'", creationResult.Payload.Name))
	}
	if creationResult.Payload.APIEndpoint == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster api_endpoint is empty"))
	}
	if creationResult.Payload.Master == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster master is empty"))
	} else if creationResult.Payload.Master.AvailabilityZone == "" {
		cliutil.PrintInfo("'creationResult.Payload.Master.AvailabilityZone' is empty.")
	} else {
		cliutil.PrintInfo("'creationResult.Payload.Master.AvailabilityZone' is %s", creationResult.Payload.Master.AvailabilityZone)
	}
	if creationResult.Payload.MasterNodes == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane master is empty"))
	} else if creationResult.Payload.MasterNodes.AvailabilityZones == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane availability_zones is empty"))
	} else {
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.AvailabilityZones' is %s", creationResult.Payload.MasterNodes.AvailabilityZones)
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.HighAvailability' is %t", creationResult.Payload.MasterNodes.HighAvailability)
		cliutil.PrintInfo("'creationResult.Payload.MasterNodes.NumReady' is %d", creationResult.Payload.MasterNodes.NumReady)
	}
	if creationResult.Payload.ReleaseVersion == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster release_version is empty"))
	} else {
		cliutil.PrintInfo("'creationResult.Payload.ReleaseVersion' is %s", creationResult.Payload.ReleaseVersion)
	}
	if creationResult.Payload.CreateDate == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster create_date is empty"))
	}
	if creationResult.Payload.Owner == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster owner is empty"))
	}

	if creationResult.Payload.ID == "" {
		// we can't continue without this
		return "", "", microerror.Maskf(assertionFailedError, "Cluster ID is empty")
	}

	cliutil.PrintSuccess("Cluster created with ID %s", creationResult.Payload.ID)
	return creationResult.Payload.ID, creationResult.Payload.APIEndpoint, nil
}

// CreateNodePoolUsingDefaults ensures that a node pool can be created with minimal spec and defaults apply.
func CreateNodePoolUsingDefaults(giantSwarmClient *client.Client, clusterID string) (string, error) {
	var err error
	var creationResult *node_pools.AddNodePoolCreated

	req := &models.V5AddNodePoolRequest{}
	params := node_pools.NewAddNodePoolParams().WithClusterID(clusterID).WithBody(req)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", microerror.Mask(err)
	}
	creationResult, err = giantSwarmClient.GSClientGen.NodePools.AddNodePool(params, authWriter)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// validation creation result
	if creationResult.Payload.Name == "" {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'name' is missing in node pool creation response"))
	}

	if len(creationResult.Payload.AvailabilityZones) == 0 {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'availability_zones' in node pool creation response is empty"))
	} else if len(creationResult.Payload.AvailabilityZones) > 1 {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'availability_zones' has % items instead of 1", len(creationResult.Payload.AvailabilityZones)))
	}

	if creationResult.Payload.Scaling == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling' is missing in node pool creation response"))
	} else {
		if creationResult.Payload.Scaling.Min != 3 {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling.min' in node pool creation response is != 3"))
		}
		if creationResult.Payload.Scaling.Max != 10 {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling.max' in node pool creation response is != 10"))
		}
	}

	if creationResult.Payload.NodeSpec == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec' is missing in node pool creation response"))
	} else {
		if creationResult.Payload.NodeSpec.Aws == nil {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.aws' is missing in node pool creation response"))
		} else {
			if creationResult.Payload.NodeSpec.Aws.InstanceType == "" {
				cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.aws.instance_type' is missing in node pool creation response"))
			} else {
				cliutil.PrintInfo("'node_spec.aws.instance_type' is %s", creationResult.Payload.NodeSpec.Aws.InstanceType)
			}
		}

		if creationResult.Payload.NodeSpec.VolumeSizesGb == nil {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.volume_sizes_gb' is missing in node pool creation response"))
		} else {
			if creationResult.Payload.NodeSpec.VolumeSizesGb.Docker == 0 {
				cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.volume_sizes_gb.docker' in node pool creation response is zero"))
			}
			if creationResult.Payload.NodeSpec.VolumeSizesGb.Kubelet == 0 {
				cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.volume_sizes_gb.kubelet' in node pool creation response is zero"))
			}
		}
	}

	if creationResult.Payload.ID == "" {
		// we can't continue without this
		return "", microerror.Maskf(assertionFailedError, "Node pool ID is missing in node pool creation response")
	}

	return creationResult.Payload.ID, nil

}

// CreateNodePoolWithCustomParams checks the creation of a node pool with some custom properties.
func CreateNodePoolWithCustomParams(giantSwarmClient *client.Client, clusterID string, instanceType string, availabilityZones []string) (string, error) {
	var err error
	var creationResult *node_pools.AddNodePoolCreated

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

	params := node_pools.NewAddNodePoolParams().WithClusterID(clusterID).WithBody(req)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return "", microerror.Mask(err)
	}
	creationResult, err = giantSwarmClient.GSClientGen.NodePools.AddNodePool(params, authWriter)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// validate response.
	if creationResult.Payload.NodeSpec != nil {
		if creationResult.Payload.NodeSpec.Aws != nil {
			if instanceType != "" && creationResult.Payload.NodeSpec.Aws.InstanceType != instanceType {
				cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.aws.instance_type' in node pool creation response is not %s", instanceType))
			}
		} else {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec.aws' in node pool creation response is nil"))
		}
	} else {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'node_spec' in node pool creation response is nil"))
	}

	if len(creationResult.Payload.AvailabilityZones) != 0 {
		if len(availabilityZones) != 0 {
			if !cmp.Equal(creationResult.Payload.AvailabilityZones, availabilityZones) {
				cliutil.Complain(microerror.Maskf(assertionFailedError, "\n%s\n", cmp.Diff(availabilityZones, creationResult.Payload.AvailabilityZones)))
			}
		}
	} else {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'availability_zones' in node pool creation response is empty"))
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
		if _, ok := err.(*key_pairs.AddKeyPairServiceUnavailable); ok {
			return "", microerror.Mask(notYetAvailableError)
		}
		return "", microerror.Mask(err)
	}

	if addKeyPairResponse.Payload.ID == "" {
		return "", microerror.Maskf(assertionFailedError, "'id' in key pair creation response is empty")
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
	params := node_pools.NewDeleteNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = giantSwarmClient.GSClientGen.NodePools.DeleteNodePool(params, authWriter)
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
	params := node_pools.NewModifyNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID).WithBody(modifyBody)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	response, err := giantSwarmClient.GSClientGen.NodePools.ModifyNodePool(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Payload.Scaling == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling' is missing in node pool modification response"))
	} else {
		if response.Payload.Scaling.Min != int64(min) {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling.min' in node pool modification response is not %d", min))
		}
		if response.Payload.Scaling.Min != int64(max) {
			cliutil.Complain(microerror.Maskf(assertionFailedError, "'scaling.min' in node pool modification response is not %d", max))
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
	params := node_pools.NewModifyNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID).WithBody(modifyBody)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}

	response, err := giantSwarmClient.GSClientGen.NodePools.ModifyNodePool(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Payload.Name != name {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "'name' in node pool modification response is not %q", name))
	}

	cliutil.PrintSuccess("Nodepool %s/%s has been renamed", clusterID, nodePoolID)
	return nil
}

// RenameNodePool tests whether a node pool can be renamed.
func UpdateClusterToHA(giantSwarmClient *client.Client, clusterID string, highAvailability bool) error {
	modifyBody := &models.V5ModifyClusterRequest{
		MasterNodes: &models.V5ModifyClusterRequestMasterNodes{
			HighAvailability: highAvailability,
		},
	}
	params := clusters.NewModifyClusterV5Params().WithClusterID(clusterID).WithBody(modifyBody)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return microerror.Mask(err)
	}
	response, err := giantSwarmClient.GSClientGen.Clusters.ModifyClusterV5(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	if response.Payload.Master == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "Cluster master is empty"))
	} else if response.Payload.Master.AvailabilityZone == "" {
		cliutil.PrintInfo("'response.Payload.Master.AvailabilityZone' is empty.")
	} else {
		cliutil.PrintInfo("'response.Payload.Master.AvailabilityZone' is %s", response.Payload.Master.AvailabilityZone)
	}
	if response.Payload.MasterNodes == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane master is empty"))
	} else if response.Payload.MasterNodes.AvailabilityZones == nil {
		cliutil.Complain(microerror.Maskf(assertionFailedError, "ControlPlane availability_zones is empty"))
	} else {
		cliutil.PrintInfo("'response.Payload.MasterNodes.AvailabilityZones' is %s", response.Payload.MasterNodes.AvailabilityZones)
		cliutil.PrintInfo("'response.Payload.MasterNodes.HighAvailability' is %t", response.Payload.MasterNodes.HighAvailability)
		cliutil.PrintInfo("'response.Payload.MasterNodes.NumReady' is %d", response.Payload.MasterNodes.NumReady)
	}

	cliutil.PrintSuccess("Cluster %s has been updated to HA", clusterID)
	return nil
}

// GetNodePoolDetails returns details on a node pool
func GetNodePoolDetails(giantSwarmClient *client.Client, clusterID string, nodePoolID string) (*models.V5GetNodePoolResponse, error) {
	params := node_pools.NewGetNodePoolParams().WithClusterID(clusterID).WithNodepoolID(nodePoolID)
	authWriter, err := giantSwarmClient.AuthHeaderWriter()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	details, err := giantSwarmClient.GSClientGen.NodePools.GetNodePool(params, authWriter)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return details.Payload, nil
}
