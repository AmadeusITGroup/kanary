package plugin

import (
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// KanaryOptions provides information required to manage Kanary
type KanaryOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewKanaryOptions provides an instance of KanaryOptions with default values
func NewKanaryOptions(streams genericclioptions.IOStreams) *KanaryOptions {
	return &KanaryOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdKanary provides a cobra command wrapping KanaryOptions
func NewCmdKanary(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewKanaryOptions(streams)

	cmd := &cobra.Command{
		Use: "kanary [subcommand] [flags]",
	}

	cmd.AddCommand(NewCmdGenerate(streams))
	cmd.AddCommand(NewCmdGet(streams))

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for processing the command
func (o *KanaryOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *KanaryOptions) Validate() error {
	return nil
}

// Run use to run the command
func (o *KanaryOptions) Run() error {
	return nil
}
