package runtests

import (
	"context"
	"fmt"
	"io"

	gsclient "github.com/giantswarm/gsclientgen/client"
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
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
