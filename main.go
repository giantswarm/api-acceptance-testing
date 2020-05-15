package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/giantswarm/api-acceptance-test/cmd"
	"github.com/giantswarm/api-acceptance-test/pkg/project"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

func main() {
	err := mainE(context.Background())
	if err != nil {
		panic(fmt.Sprintf("%#v\n", err))
	}
}

func mainE(ctx context.Context) error {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var rootCommand *cobra.Command
	{
		c := cmd.Config{
			Logger: logger,

			GitCommit: project.GitSHA(),
			Source:    project.Source(),
		}

		rootCommand, err = cmd.New(c)
	}

	err = rootCommand.Execute()
	if err != nil {
		fmt.Println(color.RedString("\nTests could not be started"))
		fmt.Println("Please check the error details below.")

		if mErr, ok := err.(*microerror.Error); ok {
			fmt.Printf("Kind: %s\n", mErr.Kind)
			if mErr.Docs != "" {
				fmt.Printf("Documentation: %s\n", mErr.Docs)
			}
		}

		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	return nil
}
