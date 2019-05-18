package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	yaml "github.com/ghodss/yaml"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

var (
	generateExample = `
	# generate KanaryDeployment from "foo" Deployment
	%[1]s generate foo
	# generate KanaryDeployment from "foo" Deployment and service "bar"
	%[1]s generate foo --service bar
`
)

const (
	argOutputFormat                   = "output"
	argServiceName                    = "service"
	argScale                          = "scale"
	argTraffic                        = "traffic"
	argName                           = "name"
	argDryRun                         = "dry-run"
	argValidationPeriod               = "validation-period"
	argValidationLabelWatchPod        = "validation-labelwatch-pod"
	argValidationLabelWatchDeployment = "validation-labelwatch-dep"
	argValidationPromQLIstioQuantile  = "validation-promql-istio-quantile"
	argValidationPromQLIstioSuccess   = "validation-promql-istio-success"
)

type outputFormat string

type outputFormatArg struct {
	value outputFormat
}

func (o *outputFormatArg) String() string {
	return string(o.value)
}

func (o *outputFormatArg) Set(s string) error {
	switch s {
	case string(outputFormatYAML):
		o.value = outputFormatYAML
	case string(outputFormatJSON):
		o.value = outputFormatJSON
	default:
		return fmt.Errorf("%s not a valid value", s)
	}
	return nil
}

func (o *outputFormatArg) Type() string {
	return string("format")
}

func (o *outputFormatArg) Get() outputFormat {
	return o.value
}

var _ pflag.Value = &outputFormatArg{}

const (
	outputFormatYAML outputFormat = "yaml"
	outputFormatJSON outputFormat = "json"
)

// generateOptions provides information required to generate subcommand
type generateOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace                      string
	userDeploymentName                 string
	userServiceName                    string
	userScale                          string
	userDryRun                         bool
	userName                           string
	userTraffic                        string
	userValidationPeriod               time.Duration
	userValidationLabelWatchPod        string
	userValidationLabelWatchDeployment string
	userValidationPromQLIstioQuantile  string
	userValidationPromQLIstioSuccess   float64
	userOutputFormat                   outputFormatArg
}

// newGenerateOptions provides an instance of KanaryOptions with default values
func newGenerateOptions(streams genericclioptions.IOStreams) *generateOptions {
	return &generateOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdGenerate provides a cobra command wrapping generateOptions
func NewCmdGenerate(streams genericclioptions.IOStreams) *cobra.Command {
	o := newGenerateOptions(streams)

	cmd := &cobra.Command{
		Use:          "generate [deployment-name] [flags]",
		Short:        "generate a KanaryDeployment artifact from a Deployment",
		Example:      fmt.Sprintf(generateExample, "kubectl"),
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

	cmd.Flags().StringVarP(&o.userName, argName, "", "", "kanary name")
	cmd.Flags().StringVarP(&o.userServiceName, argServiceName, "", "", "service name")
	cmd.Flags().StringVarP(&o.userScale, argScale, "", "static", "kanary scale strategy [static|hpa]")
	cmd.Flags().BoolVarP(&o.userDryRun, argDryRun, "", false, "dry run prevent quto,qtic deployment in case of success")
	cmd.Flags().StringVarP(&o.userTraffic, argTraffic, "", "none", "kanary traffic strategy [none|service|both|mirror]")
	cmd.Flags().StringVarP(&o.userValidationLabelWatchPod, argValidationLabelWatchPod, "", "", "kanary validation labelwatch: string representation of label-selector for pod invalidation")
	cmd.Flags().StringVarP(&o.userValidationLabelWatchDeployment, argValidationLabelWatchDeployment, "", "", "kanary validation labelwatch: string representation of label-selector for deployment invalidation")
	cmd.Flags().StringVarP(&o.userValidationPromQLIstioQuantile, argValidationPromQLIstioQuantile, "", "", "kanary validation using promql on top of istio response time monitoring. format(percentile 90 lower or equal 150 ms) P90<150  ")
	cmd.Flags().Float64VarP(&o.userValidationPromQLIstioSuccess, argValidationPromQLIstioSuccess, "", -1, "kanary validation using promql on top of istio success rate. ")

	cmd.Flags().DurationVarP(&o.userValidationPeriod, argValidationPeriod, "", 15*time.Minute, "kanary validation periode")
	cmd.Flags().VarP(&o.userOutputFormat, argOutputFormat, "o", "generation output format (json or yaml)")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for processing the command
func (o *generateOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args
	var err error
	clientConfig := o.configFlags.ToRawKubeConfigLoader()
	o.client, err = NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to instantiate client, err: %v", err)
	}

	o.userNamespace, _, err = clientConfig.Namespace()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		o.userDeploymentName = args[0]
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *generateOptions) Validate() error {

	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	if o.userDeploymentName == "" {
		return fmt.Errorf("the deployment name is mandatory")
	}

	return nil
}

// Run use to run the command
func (o *generateOptions) Run() error {

	if o.userName == "" {
		o.userName = o.userDeploymentName
	}

	dep := &appsv1beta1.Deployment{}
	err := o.client.Get(context.TODO(), client.ObjectKey{Name: o.userDeploymentName, Namespace: o.userNamespace}, dep)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("deployment %s/%s didn't exist", o.userNamespace, o.userDeploymentName)
	} else if err != nil {
		return fmt.Errorf("unable to get deployment %s/%s, err: %v", o.userNamespace, o.userDeploymentName, err)
	}

	newKanaryDeployment := &v1alpha1.KanaryDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KanaryDeployment",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.userName,
			Namespace: o.userNamespace,
		},
		Spec: v1alpha1.KanaryDeploymentSpec{
			ServiceName:    o.userServiceName,
			DeploymentName: o.userDeploymentName,
			Template: v1alpha1.DeploymentTemplate{
				Spec: dep.Spec,
			},
			Validations: v1alpha1.KanaryDeploymentSpecValidationList{
				ValidationPeriod: &metav1.Duration{Duration: o.userValidationPeriod},
			},
		},
	}

	switch o.userScale {
	case "static":
		newKanaryDeployment.Spec.Scale.Static = &v1alpha1.KanaryDeploymentSpecScaleStatic{
			Replicas: v1alpha1.NewInt32(1),
		}
	case "hpa":
		newKanaryDeployment.Spec.Scale.HPA = &v1alpha1.HorizontalPodAutoscalerSpec{}
	default:
		return fmt.Errorf("wrong value for 'scale' parameter, current value:%s", o.userScale)
	}

	switch v1alpha1.KanaryDeploymentSpecTrafficSource(o.userTraffic) {
	case v1alpha1.ServiceKanaryDeploymentSpecTrafficSource:
		newKanaryDeployment.Spec.Traffic.Source = v1alpha1.ServiceKanaryDeploymentSpecTrafficSource
	case v1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource:
		newKanaryDeployment.Spec.Traffic.Source = v1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource
	case v1alpha1.BothKanaryDeploymentSpecTrafficSource:
		newKanaryDeployment.Spec.Traffic.Source = v1alpha1.BothKanaryDeploymentSpecTrafficSource
	case v1alpha1.MirrorKanaryDeploymentSpecTrafficSource:
		newKanaryDeployment.Spec.Traffic.Source = v1alpha1.MirrorKanaryDeploymentSpecTrafficSource
	case v1alpha1.NoneKanaryDeploymentSpecTrafficSource:
		newKanaryDeployment.Spec.Traffic.Source = v1alpha1.NoneKanaryDeploymentSpecTrafficSource
	default:
		return fmt.Errorf("wrong value for 'traffic' parameter, current value:%s", o.userTraffic)
	}

	if o.userValidationLabelWatchDeployment == "" && o.userValidationLabelWatchPod == "" && o.userValidationPromQLIstioQuantile == "" && o.userValidationPromQLIstioSuccess >= 0 {
		newKanaryDeployment.Spec.Validations.Items = append(newKanaryDeployment.Spec.Validations.Items, v1alpha1.KanaryDeploymentSpecValidation{Manual: &v1alpha1.KanaryDeploymentSpecValidationManual{}})
	}

	if o.userValidationLabelWatchPod != "" || o.userValidationLabelWatchDeployment != "" {
		newLabelWatch := &v1alpha1.KanaryDeploymentSpecValidationLabelWatch{}
		if o.userValidationLabelWatchPod != "" {
			var selector *metav1.LabelSelector
			selector, err = metav1.ParseToLabelSelector(o.userValidationLabelWatchPod)
			if err != nil {
				return fmt.Errorf("unable to parse %s=%s, err:%v", argValidationLabelWatchPod, o.userValidationLabelWatchPod, err)
			}
			newLabelWatch.PodInvalidationLabels = selector
		}
		if o.userValidationLabelWatchDeployment != "" {
			var selector *metav1.LabelSelector
			selector, err = metav1.ParseToLabelSelector(o.userValidationLabelWatchDeployment)
			if err != nil {
				return fmt.Errorf("unable to parse %s=%s, err:%v", argValidationLabelWatchDeployment, o.userValidationLabelWatchDeployment, err)
			}
			newLabelWatch.DeploymentInvalidationLabels = selector
		}
		newKanaryDeployment.Spec.Validations.Items = append(newKanaryDeployment.Spec.Validations.Items, v1alpha1.KanaryDeploymentSpecValidation{LabelWatch: newLabelWatch})
	}

	if o.userValidationPromQLIstioQuantile != "" {
		ok, err := regexp.MatchString("^P[0-9]{2}<[0-9]+$", o.userValidationPromQLIstioQuantile)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("Bad format for validation-promql-istio-quantile, should be something like P90<150")
		}

		p := o.userValidationPromQLIstioQuantile[1:3]
		ms, _ := strconv.Atoi(o.userValidationPromQLIstioQuantile[4:])
		d := time.Duration(ms) * time.Millisecond

		newKanaryDeployment.Spec.Validations.Items = append(newKanaryDeployment.Spec.Validations.Items, v1alpha1.KanaryDeploymentSpecValidation{PromQL: &v1alpha1.KanaryDeploymentSpecValidationPromQL{
			Query:             "histogram_quantile(0." + p + ", sum(rate(istio_request_duration_seconds_bucket{reporter=\"destination\",destination_workload=\"" + utils.GetCanaryDeploymentName(newKanaryDeployment) + "\"}[1m])) by (le))",
			PrometheusService: "prometheus.istio-system:9090",
			AllPodsQuery:      true,
			ValueInRange: &v1alpha1.ValueInRange{
				Max: v1alpha1.NewFloat64(d.Seconds()),
			},
		}})
		newKanaryDeployment.Spec.Validations.InitialDelay = &metav1.Duration{Duration: 20 * time.Second}
		newKanaryDeployment.Spec.Validations.MaxIntervalPeriod = &metav1.Duration{Duration: 10 * time.Second}
	}

	if o.userValidationPromQLIstioSuccess >= 0 {
		newKanaryDeployment.Spec.Validations.Items = append(newKanaryDeployment.Spec.Validations.Items, v1alpha1.KanaryDeploymentSpecValidation{PromQL: &v1alpha1.KanaryDeploymentSpecValidationPromQL{
			Query:             "sum(rate(istio_requests_total{reporter=\"destination\", destination_workload_namespace=~\"" + o.userNamespace + "\", destination_workload=~\"" + utils.GetCanaryDeploymentName(newKanaryDeployment) + "\",response_code!~\"5.*\"}[1m]))/sum(rate(istio_requests_total{reporter=\"destination\", destination_workload_namespace=~\"" + o.userNamespace + "\", destination_workload=~\"" + utils.GetCanaryDeploymentName(newKanaryDeployment) + "\"}[1m]))",
			PrometheusService: "prometheus.istio-system:9090",
			AllPodsQuery:      true,
			ValueInRange: &v1alpha1.ValueInRange{
				Min: v1alpha1.NewFloat64(o.userValidationPromQLIstioSuccess),
				Max: v1alpha1.NewFloat64(1),
			},
		}})
		newKanaryDeployment.Spec.Validations.InitialDelay = &metav1.Duration{Duration: 20 * time.Second}
		newKanaryDeployment.Spec.Validations.MaxIntervalPeriod = &metav1.Duration{Duration: 10 * time.Second}
	}

	if o.userDryRun {
		newKanaryDeployment.Spec.Validations.NoUpdate = true
	}

	newKanaryDeployment = v1alpha1.DefaultKanaryDeployment(newKanaryDeployment)

	var bytes []byte
	bytes, err = json.Marshal(newKanaryDeployment)
	if err != nil {
		_, err = fmt.Fprintln(o.Out, fmt.Sprintln("error during json marshalling, err:", err))
		if err != nil {
			return err
		}
	}
	if o.userOutputFormat.Get() == outputFormatYAML {
		bytes, err = yaml.JSONToYAML(bytes)
	}

	if err != nil {
		_, err = fmt.Fprintln(o.Out, fmt.Sprintln("error during yaml marshalling, err:", err))
		if err != nil {
			return err
		}
	}
	_, err = o.Out.Write(bytes)

	return err
}
