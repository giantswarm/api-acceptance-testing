package cmd

import (
	"fmt"
	"time"

	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/gsclientgen/client/info"
	"github.com/giantswarm/gsclientgen/models"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

// RunCommand is the main command of this CLI.
var RunCommand = &cobra.Command{
	Use:   "run",
	Short: "Run acceptance test",
	Run:   ExecuteRunCommand,
}

// ExecuteRunCommand is called when the user triggers this command.
func ExecuteRunCommand(cmd *cobra.Command, args []string) {
	endpoint, err := cmd.Flags().GetString("endpoint")
	exitIfError(err)

	scheme, err := cmd.Flags().GetString("scheme")
	exitIfError(err)

	token, err := cmd.Flags().GetString("token")
	exitIfError(err)

	authWriter, err := getClientAuth(scheme, token)
	exitIfError(err)

	//var giantSwarmClient *gsclient.Gsclientgen
	giantSwarmClient, err := newClient(endpoint)
	exitIfError(err)
	{
		params := info.NewGetInfoParams()

		infoResponse, err := giantSwarmClient.Info.GetInfo(params, authWriter)
		exitIfError(err)

		printSuccess("Client initialized and user authenticated")
		fmt.Printf("API endpoint: %s\n", endpoint)
		fmt.Printf("Scheme: %s\n", scheme)
		fmt.Printf("Installation name: %s\n", infoResponse.Payload.General.InstallationName)
	}

	/// 1. Create a cluster with one node pool based on defaults
	fmt.Printf("\nStep 1 - Create a cluster with one node pool based on defaults\n")

	org := "giantswarm"
	req := &models.V5AddClusterRequest{Owner: &org}
	params := clusters.NewAddClusterV5Params().WithBody(req)
	creationResult, err := giantSwarmClient.Clusters.AddClusterV5(params, authWriter)
	exitIfError(err)

	clusterOneID := creationResult.Payload.ID
	printSuccess("Cluster created with ID %s", clusterOneID)

	{
		// Verify cluster details
		if creationResult.Payload.Name == "" {
			complain(microerror.New("Cluster name is empty"))
		}
		if creationResult.Payload.APIEndpoint == "" {
			complain(microerror.New("Cluster api_endpoint is empty"))
		}
		if creationResult.Payload.Master == nil {
			complain(microerror.New("Cluster master is empty"))
		} else {
			if creationResult.Payload.Master.AvailabilityZone == "" {
				complain(microerror.New("Cluster master.availability_zone is is empty"))
			}
		}
	}

	/// 19. Delete the cluster
	fmt.Printf("\nStep 19 - Delete cluster created in step 1\n")

	// wait for some time
	time.Sleep(5 * time.Second)

	deleteClusterOneParams := clusters.NewDeleteClusterParams().WithClusterID(clusterOneID)
	_, err = giantSwarmClient.Clusters.DeleteCluster(deleteClusterOneParams, authWriter)
	exitIfError(err)

	printSuccess("Cluster %s has been deleted", clusterOneID)
}
