package uat

import (
	"fmt"
	"time"

	gsclient "github.com/giantswarm/gsclientgen/client"
	"github.com/giantswarm/gsclientgen/client/clusters"
	"github.com/giantswarm/gsclientgen/client/info"
	"github.com/giantswarm/gsclientgen/models"
	"github.com/giantswarm/microerror"
	"github.com/go-openapi/runtime"

	"github.com/giantswarm/node-pools-acceptance-test/pkg/cliutil"
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

// TestMore runs all further tests.
// TODO: break this down further.
func TestMore(giantSwarmClient *gsclient.Gsclientgen, authWriter runtime.ClientAuthInfoWriter) {

	/// 1. Create a cluster with one node pool based on defaults
	fmt.Printf("\nStep 1 - Create a cluster with one node pool based on defaults\n")

	var clusterOneID string
	var creationResult *clusters.AddClusterV5Created
	{
		var err error

		org := "giantswarm"
		req := &models.V5AddClusterRequest{Owner: &org}
		params := clusters.NewAddClusterV5Params().WithBody(req)
		creationResult, err = giantSwarmClient.Clusters.AddClusterV5(params, authWriter)
		cliutil.ExitIfError(err)

		{
			// Verify cluster details
			if creationResult.Payload.Name == "" {
				cliutil.Complain(microerror.New("Cluster name is empty"))
			}
			if creationResult.Payload.APIEndpoint == "" {
				cliutil.Complain(microerror.New("Cluster api_endpoint is empty"))
			}
			if creationResult.Payload.Master == nil {
				cliutil.Complain(microerror.New("Cluster master is empty"))
			} else if creationResult.Payload.Master.AvailabilityZone == "" {
				cliutil.Complain(microerror.New("Cluster master.availability_zone is is empty"))
			}
			if creationResult.Payload.ReleaseVersion == "" {
				cliutil.Complain(microerror.New("Cluster release_version is is empty"))
			}
			if creationResult.Payload.CreateDate == "" {
				cliutil.Complain(microerror.New("Cluster create_date is is empty"))
			}
			if creationResult.Payload.Owner == "" {
				cliutil.Complain(microerror.New("Cluster owner is is empty"))
			}

			if creationResult.Payload.ID == "" {
				// we can't continue without this
				cliutil.ExitIfError(microerror.New("Cluster ID is is empty"))
			}
		}

		clusterOneID = creationResult.Payload.ID
		cliutil.PrintSuccess("Cluster created with ID %s", clusterOneID)
	}

	/// 20. Delete the cluster
	fmt.Printf("\nStep 19 - Delete cluster created in step 1\n")
	{
		// wait for some time
		time.Sleep(5 * time.Second)

		deleteClusterOneParams := clusters.NewDeleteClusterParams().WithClusterID(clusterOneID)
		_, err := giantSwarmClient.Clusters.DeleteCluster(deleteClusterOneParams, authWriter)
		cliutil.ExitIfError(err)

		cliutil.PrintSuccess("Cluster %s has been deleted", clusterOneID)
	}
}
