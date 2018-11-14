package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type runOption struct {
	name string
}

func validateRunOpts(opt runOption) error {
	if opt.name == "" {
		return errors.New("name is empty")
	}
	return nil
}

func runCommand(opts kobwOptions) *cobra.Command {
	var opt runOption
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run the build and show logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Run build:", opt.name)
			config, err := clientcmd.BuildConfigFromFlags(opts.masterURL, opts.kubeconfig)
			if err != nil {
				return errors.Wrap(err, "could not create kubernetes client config")
			}
			if err := startBuild(config, opt.name); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.name, "name", "", "build configuration name")
	return cmd
}
