package cmd

import (
	"fmt"

	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/gsclientgen/client/info"
	"github.com/giantswarm/gsclientgen/models"
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

	var giantSwarmClient *gsclient.Gsclientgen
	{
		giantSwarmClient, err := newClient(endpoint)
		exitIfError(err)

		params := info.NewGetInfoParams()

		infoResponse, err := giantSwarmClient.Info.GetInfo(params, authWriter)
		exitIfError(err)

		printSuccess("Client initialized and user authenticated")
		fmt.Printf("API endpoint: %s\n", endpoint)
		fmt.Printf("Scheme: %s\n", scheme)
		fmt.Printf("Installation name: %s\n", infoResponse.Payload.General.InstallationName)
	}

	/// 1. Create a cluster with one node pool based on defaults
	var clusterOneID string
	{
		fmt.Printf("\nStep 1 - Create a cluster with one node pool based on defaults\n")

		org := "giantswarm"
		req := &models.V5AddClusterRequest{Owner: &org}
		params := clusters.NewAddClusterV5Params().WithBody(req)
		_, err := giantSwarmClient.Clusters.AddClusterV5(params, authWriter)
		exitIfError(err)
	}

	/// 19. Delete the cluster
	fmt.Println(clusterOneID)
}
