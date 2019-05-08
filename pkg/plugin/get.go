package plugin

import (
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/spf13/cobra"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
)

var (
	getExample = `
	# view all kanary deployement
	%[1]s get in the current namespace
	# view kanary deployment foo
	%[1]s get foo
`
)

// GetOptions provides information required to manage Kanary
type GetOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace            string
	userKanaryDeploymentName string
}

// NewGetOptions provides an instance of GetOptions with default values
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	return &GetOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdGet provides a cobra command wrapping GetOptions
func NewCmdGet(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:          "get [kanaryDeployment name]",
		Short:        "get kanary deployment(s)",
		Example:      fmt.Sprintf(getExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.AddCommand(NewCmdGenerate(streams))

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for processing the command
func (o *GetOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args
	var err error

	clientConfig := o.configFlags.ToRawKubeConfigLoader()
	// Create the Client for Read/Write operations.
	o.client, err = NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to instantiate client, err: %v", err)
	}

	o.userNamespace, _, err = clientConfig.Namespace()
	if err != nil {
		return err
	}

	ns, err2 := cmd.Flags().GetString("namespace")
	if err2 != nil {
		return err
	}
	if ns != "" {
		o.userNamespace = ns
	}

	if len(args) > 0 {
		o.userKanaryDeploymentName = args[0]
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *GetOptions) Validate() error {

	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	return nil
}

// Run use to run the command
func (o *GetOptions) Run() error {
	kanaryList := &v1alpha1.KanaryDeploymentList{}

	if o.userKanaryDeploymentName == "" {
		err := o.client.List(context.TODO(), &client.ListOptions{Namespace: o.userNamespace}, kanaryList)
		if err != nil {
			return fmt.Errorf("unable to list KanaryDeployment, err: %v", err)
		}
	} else {
		kanary := &v1alpha1.KanaryDeployment{}
		err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userKanaryDeploymentName}, kanary)
		if err != nil && errors.IsNotFound(err) {
			return fmt.Errorf("KanartDeployment %s/%s not found", o.userNamespace, o.userKanaryDeploymentName)
		} else if err != nil {
			return fmt.Errorf("unable to get KanaryDeployment, err: %v", err)
		}
		kanaryList.Items = append(kanaryList.Items, *kanary)
	}

	table := newTable(o.Out)
	for _, item := range kanaryList.Items {
		data := []string{item.Namespace, item.Name, getStatus(&item), item.Spec.DeploymentName, item.Spec.ServiceName, getScale(&item), getTraffic(&item), getValidation(&item), getDuration(&item)}
		table.Append(data)
	}

	table.Render() // Send output

	return nil
}

func getScale(kd *v1alpha1.KanaryDeployment) string {
	return kd.Status.Report.Scale
}

func getTraffic(kd *v1alpha1.KanaryDeployment) string {
	return kd.Status.Report.Traffic
}

func getValidation(kd *v1alpha1.KanaryDeployment) string {
	return kd.Status.Report.Validation
}
func getStatus(kd *v1alpha1.KanaryDeployment) string {
	return kd.Status.Report.Status
}

func getDuration(kd *v1alpha1.KanaryDeployment) string {
	duration := time.Duration(0)
	if kd.Spec.Validations.InitialDelay != nil {
		duration += kd.Spec.Validations.InitialDelay.Duration
	}
	if kd.Spec.Validations.ValidationPeriod != nil {
		duration += kd.Spec.Validations.ValidationPeriod.Duration
	}
	since := time.Since(kd.ObjectMeta.CreationTimestamp.Time)
	return fmt.Sprintf("%s/%s", since, duration)
}

func newTable(out io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Namespace", "Name", "Status", "Deployment", "Service", "Scale", "Traffic", "Validation", "Duration"})
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)

	return table
}
