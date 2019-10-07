package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/node-pools-acceptance-test/cmd"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "acceptancetest",
		Short: "Runs our acceptance tests",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}

	rootCmd.PersistentFlags().StringP("endpoint", "", "", "Endpoint URL for the Giant Swarm API, without trailing slash.")
	rootCmd.PersistentFlags().StringP("scheme", "", "giantswarm", "Use 'giantswarm' for normal token auth or 'Bearer' for SSO token auth.")
	rootCmd.PersistentFlags().StringP("token", "", "", "Use this flag to pass your auth token.")

	rootCmd.AddCommand(cmd.RunCommand)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
