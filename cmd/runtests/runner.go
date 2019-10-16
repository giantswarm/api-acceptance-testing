package runtests

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff"
	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/go-openapi/runtime"
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
	var authWriter runtime.ClientAuthInfoWriter
	var giantSwarmClient *gsclient.Gsclientgen
	{
		var err error

		fmt.Printf("API Endpoint: %s\n", r.flag.Endpoint)

		authWriter, err = client.ClientAuth(r.flag.Scheme, r.flag.AuthToken)
		cliutil.ExitIfError(err)

		giantSwarmClient, err = client.NewClient(r.flag.Endpoint)
		cliutil.ExitIfError(err)
	}

	// test client and authentication
	err := uat.TestClient(giantSwarmClient, authWriter)
	cliutil.ExitIfError(err)

	// 1. Create a cluster with one node pool based on defaults.
	fmt.Printf("\nStep 1 - Create a cluster with one node pool based on defaults\n")
	clusterOneID, clusterOneAPIEndpoint, err := uat.Test01ClusterCreation(giantSwarmClient, authWriter)
	cliutil.ExitIfError(err)

	time.Sleep(3 * time.Second)

	// Workaround until step 1 returns proper cluster info.
	if clusterOneAPIEndpoint == "" {
		fmt.Printf("\nStep 1a - Get cluster details, so we know the API endpoint\n")
		var params *clusters.GetClusterV5Params
		params = clusters.NewGetClusterV5Params().WithClusterID(clusterOneID)
		result, err := giantSwarmClient.Clusters.GetClusterV5(params, authWriter)
		if err != nil {
			cliutil.ExitIfError(microerror.Mask(err))
		}

		clusterOneAPIEndpoint = result.Payload.APIEndpoint
	}

	time.Sleep(3 * time.Second)

	// Create key pair
	fmt.Printf("\nStep 6 - Create a key pair for cluster %s with k8s endpoint '%s'\n", clusterOneID, clusterOneAPIEndpoint)
	kubeconfigPath, err := uat.Test06CreateKeyPair(giantSwarmClient, authWriter, clusterOneID, clusterOneAPIEndpoint)
	cliutil.ExitIfError(err)

	fmt.Printf("\nStep 7 - Access cluster's K8s API %s with kubeconfig file %s\n(Take your time, we wait until it succeeds.)\n", clusterOneAPIEndpoint, kubeconfigPath)
	operation := func() error {
		return uat.Test07GetKubernetesNodes(kubeconfigPath)
	}
	err = backoff.Retry(operation, backoff.NewConstantBackOff(10*time.Second))
	cliutil.ExitIfError(err)

	// 2. Create a node pool based on defaults.
	// fmt.Printf("\nStep 2 - Create a node pool based on defaults\n")
	// _, err = uat.Test02NodePoolCreationUsingDefaults(giantSwarmClient, authWriter, clusterOneID)
	// cliutil.ExitIfError(err)

	//time.Sleep(1 * time.Second)

	//20. Delete cluster one.
	// fmt.Printf("\nStep 20 - Delete cluster\n")
	// err = uat.Test20ClusterDeletion(giantSwarmClient, authWriter, clusterOneID)
	// cliutil.ExitIfError(err)

	return nil
}
