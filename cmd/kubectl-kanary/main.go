package main

import (
	"os"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/amadeusitgroup/kanary/pkg/plugin"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-kanary", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := plugin.NewCmdKanary(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
