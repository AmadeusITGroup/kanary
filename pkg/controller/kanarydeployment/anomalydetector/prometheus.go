package anomalydetector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	promClient "github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/prometheus/common/model"
)

const (
	GlobalQueryKey = "__**GlobalKeyQuery**__"
)

//ConfigPrometheusAnomalyDetector configuration to connect to prometheus
type ConfigPrometheusAnomalyDetector struct {
	PrometheusService string
	PodNameKey        string
	AllPodsQuery      bool
	Query             string
	queryAPI          promApi.API
	logger            logr.Logger
}

//===== DiscreteValueOutOfListAnalyser =====

type promDiscreteValueOutOfListAnalyser struct {
	promConfig ConfigPrometheusAnomalyDetector
	config     DiscreteValueOutOfListConfig
}

func (p *promDiscreteValueOutOfListAnalyser) doAnalysis() (okkoByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	// p.config.PodNameKey should be "kubernetes_pod_name"
	// p.config.Key should be "code"
	m, err := p.promConfig.queryAPI.Query(ctx, p.promConfig.Query, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
	}

	return p.buildCounters(vector), nil
}

func (p *promDiscreteValueOutOfListAnalyser) buildCounters(vector model.Vector) okkoByPodName {
	countersByPods := okkoByPodName{}

	for _, sample := range vector {
		metrics := sample.Metric
		podName, err := extractPodNameFromMetric(metrics, p.promConfig)
		if err != nil {
			continue // TODO: this analyser does not fail when problem with podKeyName. To Fix
		}

		counters := countersByPods[podName]

		discreteValue := metrics[model.LabelName(p.config.Key)]
		if p.config.valueCheckerFunc(string(discreteValue)) {
			counters.ok += uint(sample.Value)
		} else {
			counters.ko += uint(sample.Value)
		}
		countersByPods[podName] = counters
	}
	return countersByPods
}

func newPromDiscreteValueOutOfListAnalyser(promConfig ConfigPrometheusAnomalyDetector, config DiscreteValueOutOfListConfig) (*promDiscreteValueOutOfListAnalyser, error) {
	good, bad := config.GoodValues, config.BadValues
	valueCheckerFunc := func(value string) bool { return ContainsString(good, value) }
	if len(good) == 0 && len(bad) != 0 {
		valueCheckerFunc = func(value string) bool { return !ContainsString(bad, value) }
	}

	config.valueCheckerFunc = valueCheckerFunc
	promconfig := promClient.Config{Address: "http://" + promConfig.PrometheusService}
	prometheusClient, err := promClient.NewClient(promconfig)
	if err != nil {
		return nil, err
	}
	promConfig.queryAPI = promApi.NewAPI(prometheusClient)

	return &promDiscreteValueOutOfListAnalyser{config: config, promConfig: promConfig}, nil
}

//===== ContinuousValueDeviationAnalyser =====

type promContinuousValueDeviationAnalyser struct {
	promConfig ConfigPrometheusAnomalyDetector
	config     ContinuousValueDeviationConfig
}

//newPromContinuousValueDeviationAnalyser new amnalyser for ContinuousValueDeviation backed by prometheus
func newPromContinuousValueDeviationAnalyser(promConfig ConfigPrometheusAnomalyDetector, config ContinuousValueDeviationConfig) (*promContinuousValueDeviationAnalyser, error) {

	promconfig := promClient.Config{Address: "http://" + promConfig.PrometheusService}
	prometheusClient, err := promClient.NewClient(promconfig)
	if err != nil {
		return nil, err
	}
	promConfig.queryAPI = promApi.NewAPI(prometheusClient)
	return &promContinuousValueDeviationAnalyser{promConfig: promConfig, config: config}, nil
}

func (p *promContinuousValueDeviationAnalyser) doAnalysis() (deviationByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: (rate(solution_price_sum{}[1m])/rate(solution_price_count{}[1m]) and delta(solution_price_count{}[1m])>70) / scalar(sum(rate(solution_price_sum{}[1m]))/sum(rate(solution_price_count{}[1m])))
	// p.PodNameKey should point to the label containing the pod name (if the query is not for all pods)
	m, err := p.promConfig.queryAPI.Query(ctx, p.promConfig.Query, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
	}

	result := deviationByPodName{}
	for _, sample := range vector {
		podName, err := extractPodNameFromMetric(sample.Metric, p.promConfig)
		if err != nil {
			return nil, err
		}

		deviation := sample.Value
		result[podName] = float64(deviation)
	}
	return result, nil
}

// ===== ValueInRangeAnalyser =====

type promValueInRangeAnalyser struct {
	promConfig ConfigPrometheusAnomalyDetector
	config     ValueInRangeConfig
}

//newPromValueInRangeAnalyser new amnalyser for ValueInRange backed by prometheus
func newPromValueInRangeAnalyser(promConfig ConfigPrometheusAnomalyDetector, config ValueInRangeConfig) (*promValueInRangeAnalyser, error) {

	promconfig := promClient.Config{Address: "http://" + promConfig.PrometheusService}
	prometheusClient, err := promClient.NewClient(promconfig)
	if err != nil {
		return nil, err
	}
	promConfig.queryAPI = promApi.NewAPI(prometheusClient)
	return &promValueInRangeAnalyser{promConfig: promConfig, config: config}, nil
}

func (p *promValueInRangeAnalyser) doAnalysis() (inRangeByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: (rate(solution_price_sum{}[1m])/rate(solution_price_count{}[1m]) and delta(solution_price_count{}[1m])>70) / scalar(sum(rate(solution_price_sum{}[1m]))/sum(rate(solution_price_count{}[1m])))
	// p.PodNameKey should point to the label containing the pod name (if the query is not for all pods)
	m, err := p.promConfig.queryAPI.Query(ctx, p.promConfig.Query, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
	}

	result := inRangeByPodName{}
	for _, sample := range vector {
		podName, err := extractPodNameFromMetric(sample.Metric, p.promConfig)
		if err != nil {
			return nil, err
		}
		if float64(sample.Value) >= p.config.Min && float64(sample.Value) <= p.config.Max {
			result[podName] = true
		} else {
			result[podName] = false
		}
	}
	return result, nil
}

func extractPodNameFromMetric(metrics model.Metric, promConfig ConfigPrometheusAnomalyDetector) (string, error) {
	podName := string(metrics[model.LabelName(promConfig.PodNameKey)])
	if promConfig.AllPodsQuery {
		podName = GlobalQueryKey
	}
	if podName == "" && !promConfig.AllPodsQuery {
		return "", fmt.Errorf("The metric returned is missing the podName dimension '%s', while the query is not marked to be global", podName)
	}
	return podName, nil
}
