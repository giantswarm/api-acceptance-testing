package runtests

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/node-pools-acceptance-test/pkg/client"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/cliutil"
	"github.com/giantswarm/node-pools-acceptance-test/pkg/uat"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

// Run is called when the runtests command is executed.
func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Initialize client
	var apiClient *client.Client
	{
		var err error

		fmt.Printf("API Endpoint: %s\n", r.flag.Endpoint)

		apiClient, err = client.New(r.flag.Endpoint)
		cliutil.ExitIfError(err)
	}

	// test client and authentication
	err := uat.TestClient(apiClient)
	cliutil.ExitIfError(err)

	var clusterOneID string
	var clusterOneAPIEndpoint string
	var nodePoolOneID string

	if r.flag.ClusterID == "" {
		// 1. Create a cluster with one node pool based on defaults.
		fmt.Printf("\nStep 1 - Create a cluster with one node pool based on defaults - %s\n", time.Now())
		clusterOneID, clusterOneAPIEndpoint, err = uat.CreateClusterUsingDefaults(apiClient)
		cliutil.ExitIfError(err)
	} else {
		clusterOneID = r.flag.ClusterID
	}

	// Workaround until step 1 returns proper cluster info.
	if clusterOneAPIEndpoint == "" {
		fmt.Printf("\nStep 1a - Get cluster details, so we know the API endpoint - %s\n", time.Now())
		var params *clusters.GetClusterV5Params
		params = clusters.NewGetClusterV5Params().WithClusterID(clusterOneID)
		authWriter, err := apiClient.AuthHeaderWriter()
		if err != nil {
			cliutil.ExitIfError(microerror.Mask(err))
		}

		result, err := apiClient.GSClientGen.Clusters.GetClusterV5(params, authWriter)
		if err != nil {
			cliutil.ExitIfError(microerror.Mask(err))
		}

		clusterOneAPIEndpoint = result.Payload.APIEndpoint
	}

	if r.flag.FirstNodePoolID == "" {
		// 2. Create a node pool based on defaults.
		fmt.Printf("\nStep 2 - Create a node pool based on defaults\n")
		nodePoolOneID, err = uat.CreateNodePoolUsingDefaults(apiClient, clusterOneID)
		cliutil.ExitIfError(err)

		time.Sleep(1 * time.Second)
	} else {
		nodePoolOneID = r.flag.FirstNodePoolID
	}

	// Create key pair
	fmt.Printf("\nStep 3 - Create a key pair for cluster %s with k8s endpoint '%s' - %s\n", clusterOneID, clusterOneAPIEndpoint, time.Now())
	kubeconfigPath, err := uat.CreateKeyPair(apiClient, clusterOneID, clusterOneAPIEndpoint)
	cliutil.ExitIfError(err)

	// Test kubectl access
	fmt.Printf("\nStep 4 - Access cluster's K8s API %s with kubeconfig file %s - %s\n(Take your time, we wait until it succeeds.)\n", clusterOneAPIEndpoint, kubeconfigPath, time.Now())
	operation := func() error {
		return uat.RunKubectlCommandToTestKeyPair(kubeconfigPath)
	}
	err = backoff.Retry(operation, backoff.NewConstantBackOff(10*time.Second))
	cliutil.ExitIfError(err)

	// Deploy test app
	fmt.Printf("\nStep 5 - Deploy test app - %s", time.Now())
	testAppURL, err := uat.DeployTestApp(kubeconfigPath, clusterOneAPIEndpoint)
	cliutil.ExitIfError(err)

	// Create load
	fmt.Printf("\nStep 6 - Create load on test app - %s\n", time.Now())
	uat.CreateLoadOnIngress(testAppURL)

	// Increase replicas
	fmt.Printf("\nStep 7 - Increase test app replicas - %s\n", time.Now())
	uat.IncreaseTestAppReplicas(kubeconfigPath)

	// wait for nodepool scaling
	cliutil.PrintInfo("Waiting 10 minutes for node pool to adapt")
	for i := 0; i < 10; i++ {
		time.Sleep(60 * time.Second)
		details, err := uat.GetNodePoolDetails(apiClient, clusterOneID, nodePoolOneID)
		cliutil.Complain(err)

		if details != nil {
			cliutil.PrintInfo("Node pool details - nodes desired: %d, nodes in state ready: %d", details.Status.Nodes, details.Status.NodesReady)
		}
	}

	// rename only node pool
	fmt.Printf("\nStep 8 - Renaming only node pool %s - %s\n", nodePoolOneID, time.Now())
	err = uat.RenameNodePool(apiClient, clusterOneID, nodePoolOneID, "First test node pool")
	cliutil.Complain(err)

	// scale only node pool
	fmt.Printf("\nStep 9 - Scaling only node pool %s to min=2/max=2 - %s\n", nodePoolOneID, time.Now())
	err = uat.ScaleNodePool(apiClient, clusterOneID, nodePoolOneID, 2, 2)
	cliutil.Complain(err)

	// delete only node pool
	// fmt.Printf("\nStep 10 - Deleting only node pool %s - %s\n", nodePoolOneID, time.Now())
	// err = uat.DeleteNodePool(apiClient, clusterOneID, nodePoolOneID)
	// cliutil.Complain(err)

	// Delete cluster one.
	// fmt.Printf("\nStep 20 - Delete cluster - %s\n", time.Now())
	// err = uat.Test20ClusterDeletion(apiClient, clusterOneID)
	// cliutil.ExitIfError(err)

	return nil
}
