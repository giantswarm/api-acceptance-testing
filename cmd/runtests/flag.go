package runtests

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

type flag struct {
	Endpoint      string
	Scheme        string
	AuthToken     string
	EnableLogging bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Endpoint, "endpoint", "", "Endpoint URL for the Giant Swarm API, without trailing slash.")
	cmd.Flags().StringVar(&f.Scheme, "scheme", "giantswarm", "Use 'giantswarm' for normal token auth or 'Bearer' for SSO token auth.")
	cmd.Flags().StringVar(&f.AuthToken, "token", "", "Use this flag to pass your auth token.")
	cmd.Flags().BoolVar(&f.EnableLogging, "enable-logging", false, "Set to true to enable verbose stack logging on errors.")
}

func (f *flag) Validate() error {
	if f.Endpoint == "" {
		return microerror.Maskf(invalidFlagsError, "flag --endpoint must be used")
	}
	if f.Scheme == "" {
		return microerror.Maskf(invalidFlagsError, "flag --scheme must not be empty")
	}
	if f.AuthToken == "" {
		return microerror.Maskf(invalidFlagsError, "flag --token must not be empty")
	}
	return nil
}
