package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

type kobwOptions struct {
	kubeconfig string
	masterURL  string
}

func debug() {
	cwd, _ := os.Getwd()
	fmt.Println("cwd", cwd)
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}
}

func main() {
	var opt kobwOptions
	rootCmd := &cobra.Command{
		Use:   "kotw",
		Short: "Run openshift builds from knative",
	}
	rootCmd.Flags().StringVar(&opt.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster")
	rootCmd.Flags().StringVar(&opt.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster")
	rootCmd.AddCommand(runCommand(opt))
	rootCmd.AddCommand(createCommand(opt))
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
