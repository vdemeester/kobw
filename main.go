package main

//go:generate esc -o static.go -pkg main assets
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "kotw",
		Short: "Run openshift builds from knative",
	}
)

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
	rootCmd.AddCommand(runCommand())
	rootCmd.AddCommand(createCommand())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
