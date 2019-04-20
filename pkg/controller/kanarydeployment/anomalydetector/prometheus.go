package anomalydetector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type configDiscreteValueOutOfList struct {
	Key              string   // Key for the metrics. For the previous example it will be "code"
	GoodValues       []string // Good Values ["200","201"]. If empty means that BadValues should be used to do exclusion instead of inclusion.
	BadValues        []string // Bad Values ["500","404"].
	TolerancePercent uint
}

type promDiscreteValueOutOfListAnalyser struct {
	config           configDiscreteValueOutOfList
	PodNameKey       string
	Query            string
	queyrAPI         promApi.API
	logger           logr.Logger
	valueCheckerFunc func(value string) (ok bool)
}

func (p *promDiscreteValueOutOfListAnalyser) doAnalysis() (okkoByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	// p.config.PodNameKey should be "kubernetes_pod_name"
	// p.config.Key should be "code"
	m, err := p.queyrAPI.Query(ctx, p.Query, tsNow)
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
		podName := string(metrics[model.LabelName(p.PodNameKey)])
		counters := countersByPods[podName]

		discreteValue := metrics[model.LabelName(p.config.Key)]
		if p.valueCheckerFunc(string(discreteValue)) {
			counters.ok += uint(sample.Value)
		} else {
			counters.ko += uint(sample.Value)
		}
		countersByPods[podName] = counters
	}
	return countersByPods
}

type configContinuousValueDeviation struct {
	MaxDeviationPercent float64
}

type promContinuousValueDeviationAnalyser struct {
	config     configContinuousValueDeviation
	PodNameKey string
	Query      string
	queryAPI   promApi.API
	logger     logr.Logger
}

func (p *promContinuousValueDeviationAnalyser) doAnalysis() (deviationByPodName, error) {
	ctx := context.Background()
	tsNow := time.Now()

	// promQL example: (rate(solution_price_sum{}[1m])/rate(solution_price_count{}[1m]) and delta(solution_price_count{}[1m])>70) / scalar(sum(rate(solution_price_sum{}[1m]))/sum(rate(solution_price_count{}[1m])))
	// p.PodNameKey should point to the label containing the pod name
	m, err := p.queryAPI.Query(ctx, p.Query, tsNow)
	if err != nil {
		return nil, fmt.Errorf("error processing prometheus query: %s", err)
	}

	vector, ok := m.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("the prometheus query did not return a result in the form of expected type 'model.Vector': %s", err)
	}

	result := deviationByPodName{}
	for _, sample := range vector {
		metrics := sample.Metric
		podName := string(metrics[model.LabelName(p.PodNameKey)])
		deviation := sample.Value
		result[podName] = float64(deviation)
	}
	return result, nil
}
