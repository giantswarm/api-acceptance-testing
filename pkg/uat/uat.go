package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/gsclientgen/client/info"
	"github.com/giantswarm/gsclientgen/client/key_pairs"
	"github.com/giantswarm/gsclientgen/client/nodepools"
	"github.com/giantswarm/gsclientgen/models"
	"github.com/giantswarm/microerror"
	"github.com/go-openapi/runtime"
	"github.com/spf13/afero"

	"github.com/giantswarm/node-pools-acceptance-test/pkg/cliutil"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/kubeconfig"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/shell"
)

// TestClient verifies whether the given client can authenticate.
func TestClient(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter) error {
	params := info.NewGetInfoParams()
	infoResponse, err := giantSwarmClient.Info.GetInfo(params, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("Client initialized and user authenticated")
	fmt.Printf("Installation name: %s\n", infoResponse.Payload.General.InstallationName)

	return nil
}

// Test01ClusterCreation tests
// - whether we can create a cluster
// - whether defaults are applied as expected.
func Test01ClusterCreation(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter) (string, string, error) {
	var creationResult *clusters.AddClusterV5Created
	var err error

	org := "giantswarm"
	req := &models.V5AddClusterRequest{Owner: &org}
	params := clusters.NewAddClusterV5Params().WithBody(req)
	creationResult, err = giantSwarmClient.Clusters.AddClusterV5(params, authWriter)
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

// Test02NodePoolCreationUsingDefaults ensures that a node pool can be created with minimal spec and defaults apply.
func Test02NodePoolCreationUsingDefaults(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter, clusterID string) (string, error) {
	var err error
	var creationResult *nodepools.AddNodePoolCreated

	req := &models.V5AddNodePoolRequest{}
	params := nodepools.NewAddNodePoolParams().WithBody(req)
	creationResult, err = giantSwarmClient.Nodepools.AddNodePool(params, authWriter)
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

// Test06CreateKeyPair tests key pair creation for the new cluster
// and stores away the key pair in a kubectl config file for later use.
func Test06CreateKeyPair(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter, clusterID string, clusterAPIEndpoint string) (string, error) {
	description := "test key pair"
	req := &models.V4AddKeyPairRequest{
		TTLHours:                 12,
		Description:              &description,
		CertificateOrganizations: "system:masters",
		CnPrefix:                 "user@giantswarm.io",
	}
	params := key_pairs.NewAddKeyPairParams().WithClusterID(clusterID).WithBody(req)

	addKeyPairResponse, err := giantSwarmClient.KeyPairs.AddKeyPair(params, authWriter)
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

// Test07GetKubernetesNodes used kubectl to get a list of cluster nodes and returns an error if that fails.
func Test07GetKubernetesNodes(kubeconfigPath string) error {
	out, exitCode, err := shell.RunCommand(context.Background(), "kubectl", []string{}, "--kubeconfig", kubeconfigPath, "get", "nodes")
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("kubectl get nodes exited with code %d and printed:\n\n", exitCode)
	cliutil.PrintInfo(out)
	return nil
}

// Test08DeployTestApp attempts to deploy a helloworld app on the cluster.
func Test08DeployTestApp(kubeconfigPath string, clusterAPIEndpoint string) (string, error) {
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

	endpoint := "http://test." + clusterBaseDomain

	// Wait for the ingress to be reachable.
	start := time.Now()
	operation := func() error {
		_, err := http.Get(endpoint)
		return err
	}
	err = backoff.Retry(operation, backoff.NewConstantBackOff(1*time.Second))
	if err != nil {
		return "", microerror.Mask(err)
	}

	duration := time.Now().Sub(start)
	cliutil.PrintInfo("Ingress at %s reached after %s", endpoint, duration)

	cliutil.PrintSuccess("kubectl apply exited with code %d and printed:\n\n", exitCode)
	cliutil.PrintInfo(out)
	return endpoint, nil
}

// Test20ClusterDeletion tests whether a cluster gets deleted okay.
func Test20ClusterDeletion(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter, clusterID string) error {
	deleteClusterOneParams := clusters.NewDeleteClusterParams().WithClusterID(clusterID)
	_, err := giantSwarmClient.Clusters.DeleteCluster(deleteClusterOneParams, authWriter)
	if err != nil {
		return microerror.Mask(err)
	}

	cliutil.PrintSuccess("Cluster %s has been deleted", clusterID)
	return nil
}
