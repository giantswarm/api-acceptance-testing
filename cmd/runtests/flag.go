package runtests

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

type flag struct {
	AuthToken       string
	ClusterID       string
	EnableLogging   bool
	Endpoint        string
	FirstNodePoolID string
	Scheme          string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Endpoint, "endpoint", "", "Endpoint URL for the Giant Swarm API, without trailing slash.")
	cmd.Flags().StringVar(&f.Scheme, "scheme", "giantswarm", "Use 'giantswarm' for normal token auth or 'Bearer' for SSO token auth.")
	cmd.Flags().StringVar(&f.AuthToken, "token", "", "Use this flag to pass your auth token.")
	cmd.Flags().StringVar(&f.ClusterID, "cluster-id", "", "Use this cluster instead of creating a new one, to take a shortcut.")
	cmd.Flags().StringVar(&f.FirstNodePoolID, "first-nodepool-id", "", "Use this node pool as the first one instead of creating a new one, to take a shortcut.")
	cmd.Flags().BoolVar(&f.EnableLogging, "enable-logging", false, "Set to true to enable verbose stack logging on errors.")
}

func (f *flag) Validate() error {
	if f.Endpoint == "" {
		return microerror.Maskf(invalidFlagsError, "flag --endpoint must be used")
	}
	if f.Scheme != "giantswarm" && f.Scheme != "Bearer" {
		return microerror.Maskf(invalidFlagsError, "flag --scheme must be used and must be 'Bearer' or 'giantswarm' (case sensitive)")
	}

	return nil
}
