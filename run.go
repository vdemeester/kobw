package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
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

func runCommand() *cobra.Command {
	var opt runOption
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run the build and show logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Run build:", opt.name)
			c := exec.Command("oc", "start-build", opt.name, "--follow")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Start(); err != nil {
				return err
			}
			if err := c.Wait(); err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						fmt.Println("ExitStatus:", status.ExitStatus())
					}
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.name, "name", "", "build configuration name")
	return cmd
}
