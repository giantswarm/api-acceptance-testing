package runtests

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

type flag struct {
	AuthToken         string
	ClusterID         string
	EnableLogging     bool
	Endpoint          string
	FirstNodePoolID   string
	OwnerOrganization string
	ReleaseVersion    string
	Scheme            string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.EnableLogging, "enable-logging", false, "Set to true to enable verbose stack logging on errors.")
	cmd.Flags().StringVar(&f.AuthToken, "token", "", "Use this flag to pass your auth token.")
	cmd.Flags().StringVar(&f.ClusterID, "cluster-id", "", "Use this cluster instead of creating a new one, to take a shortcut.")
	cmd.Flags().StringVar(&f.Endpoint, "endpoint", "", "Endpoint URL for the Giant Swarm API, without trailing slash.")
	cmd.Flags().StringVar(&f.FirstNodePoolID, "first-nodepool-id", "", "Use this node pool as the first one instead of creating a new one, to take a shortcut.")
	cmd.Flags().StringVar(&f.OwnerOrganization, "owner-org", "giantswarm", "Name of the organization owning created clusters.")
	cmd.Flags().StringVar(&f.ReleaseVersion, "release-version", "", "Release version to test with, without 'v' prefix ('X.Y.Z'). Leave empty to use latest.")
	cmd.Flags().StringVar(&f.Scheme, "scheme", "giantswarm", "Use 'giantswarm' for normal token auth or 'Bearer' for SSO token auth.")
}

func (f *flag) Validate() error {
	if f.Endpoint == "" {
		return microerror.Maskf(invalidFlagsError, "flag --endpoint must be set to specify an API to test against")
	}
	if f.Scheme != "giantswarm" && f.Scheme != "Bearer" {
		return microerror.Maskf(invalidFlagsError, "flag --scheme must be either 'Bearer' or 'giantswarm' (case sensitive!)")
	}

	return nil
}
