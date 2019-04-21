package anomalydetector

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func Test_promDiscreteValueOutOfListAnalyser_buildCounters(t *testing.T) {

	type fields struct {
		config     DiscreteValueOutOfListConfig
		PodNameKey string
		Query      string
		//valueCheckerFunc func(value string) (ok bool)
	}
	type args struct {
		vector model.Vector
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   okkoByPodName
	}{
		{
			name: "empty",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) }},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{},
			},
			want: okkoByPodName{},
		},
		{
			name: "one ok element; inclusion",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", TolerancePercent: 50, valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) }},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {1.0, 0.0}},
		},
		{
			name: "one ko element; inclusion",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) }, TolerancePercent: 50},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {0.0, 1.0}},
		},
		{
			name: "one ok element; exclusion",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", valueCheckerFunc: func(value string) bool { return !ContainsString([]string{"500"}, value) }, TolerancePercent: 50},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {1.0, 0.0}},
		},
		{
			name: "one ko element; exclusion",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", valueCheckerFunc: func(value string) bool { return !ContainsString([]string{"500"}, value) }},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  1.0,
					},
				},
			},
			want: okkoByPodName{"david": {0.0, 1.0}},
		},
		{
			name: "complex; inclusion",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{Key: "code", valueCheckerFunc: func(value string) bool { return ContainsString([]string{"200"}, value) }},
				PodNameKey: "podname",
			},
			args: args{
				vector: model.Vector{
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "david"},
						Value:  10.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "cedric"},
						Value:  20.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "david"},
						Value:  3.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "404", "podname": "david"},
						Value:  6.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "500", "podname": "cedric"},
						Value:  8.0,
					},
					&model.Sample{
						Metric: model.Metric{"code": "200", "podname": "dario"},
						Value:  30.0,
					},
				},
			},
			want: okkoByPodName{"david": {10.0, 9.0}, "cedric": {20.0, 8.0}, "dario": {30.0, 0.0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promDiscreteValueOutOfListAnalyser{
				config: tt.fields.config,
				promConfig: ConfigPrometheusAnomalyDetector{
					PodNameKey: tt.fields.PodNameKey,
					logger:     logf.Log,
				},
			}
			if got := p.buildCounters(tt.args.vector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promDiscreteValueOutOfListAnalyser.buildCounters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_promDiscreteValueOutOfListAnalyser_doAnalysis(t *testing.T) {
	type fields struct {
		config     DiscreteValueOutOfListConfig
		PodNameKey string
		qAPI       promApi.API
	}
	tests := []struct {
		name    string
		fields  fields
		want    okkoByPodName
		wantErr bool
	}{
		{
			name: "caseErrorQuery",
			fields: fields{
				config: DiscreteValueOutOfListConfig{},
				qAPI: &testPrometheusAPI{
					err: fmt.Errorf("A prom Error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "okEmpty",
			fields: fields{
				config: DiscreteValueOutOfListConfig{
					Key: "code",
				},
				PodNameKey: "pod",
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: model.Vector([]*model.Sample{}),
				},
			},
			want:    okkoByPodName{},
			wantErr: false,
		},
		{
			name: "badCast",
			fields: fields{
				config:     DiscreteValueOutOfListConfig{},
				PodNameKey: "pod",
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promDiscreteValueOutOfListAnalyser{
				config: tt.fields.config,
				promConfig: ConfigPrometheusAnomalyDetector{
					PodNameKey: tt.fields.PodNameKey,
					queryAPI:   tt.fields.qAPI,
					logger:     logf.Log,
				},
			}
			got, err := p.doAnalysis()
			if (err != nil) != tt.wantErr {
				t.Errorf("promDiscreteValueOutOfListAnalyser.doAnalysis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promDiscreteValueOutOfListAnalyser.doAnalysis() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testPrometheusAPI struct {
	value  model.Value
	lvalue model.LabelValues
	err    error
}

// Query performs a query for the given time.
func (tAPI *testPrometheusAPI) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	return tAPI.value, tAPI.err
}

// QueryRange performs a query for the given range.
func (tAPI *testPrometheusAPI) QueryRange(ctx context.Context, query string, r promApi.Range) (model.Value, error) {
	return tAPI.value, tAPI.err
}

// LabelValues performs a query for the values of the given label.
func (tAPI *testPrometheusAPI) LabelValues(ctx context.Context, label string) (model.LabelValues, error) {
	return tAPI.lvalue, tAPI.err
}

// AlertManagers returns an overview of the current state of the Prometheus alert manager discovery.
func (tAPI *testPrometheusAPI) AlertManagers(ctx context.Context) (promApi.AlertManagersResult, error) {
	return promApi.AlertManagersResult{}, nil
}

// CleanTombstones removes the deleted data from disk and cleans up the existing tombstones.
func (tAPI *testPrometheusAPI) CleanTombstones(ctx context.Context) error {
	return nil
}

// Config returns the current Prometheus configuration.
func (tAPI *testPrometheusAPI) Config(ctx context.Context) (promApi.ConfigResult, error) {
	return promApi.ConfigResult{}, nil
}

// DeleteSeries deletes data for a selection of series in a time range.
func (tAPI *testPrometheusAPI) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	return nil
}

// Flags returns the flag values that Prometheus was launched with.
func (tAPI *testPrometheusAPI) Flags(ctx context.Context) (promApi.FlagsResult, error) {
	return promApi.FlagsResult{}, nil
}

// Series finds series by label matchers.
func (tAPI *testPrometheusAPI) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, error) {
	return nil, nil
}

// Snapshot creates a snapshot of all current data into snapshots/<datetime>-<rand>
// under the TSDB's data directory and returns the directory as response.
func (tAPI *testPrometheusAPI) Snapshot(ctx context.Context, skipHead bool) (promApi.SnapshotResult, error) {
	return promApi.SnapshotResult{}, nil
}

// Targets returns an overview of the current state of the Prometheus target discovery.
func (tAPI *testPrometheusAPI) Targets(ctx context.Context) (promApi.TargetsResult, error) {
	return promApi.TargetsResult{}, nil
}
func Test_promContinuousValueDeviationAnalyser_doAnalysis(t *testing.T) {
	type fields struct {
		config     ContinuousValueDeviationConfig
		PodNameKey string
		qAPI       promApi.API
	}
	tests := []struct {
		name    string
		fields  fields
		want    deviationByPodName
		wantErr bool
	}{
		{
			name: "caseErrorQuery",
			fields: fields{
				config: ContinuousValueDeviationConfig{},
				qAPI: &testPrometheusAPI{
					err: fmt.Errorf("A prom Error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "podA",
			fields: fields{
				config:     ContinuousValueDeviationConfig{},
				PodNameKey: "pod",
				qAPI: &testPrometheusAPI{
					err: nil,
					value: model.Vector([]*model.Sample{
						{
							Metric: model.Metric(model.LabelSet(map[model.LabelName]model.LabelValue{"pod": "podA"})),
							Value:  model.SampleValue(42.0),
						},
					}),
				},
			},
			want:    map[string]float64{"podA": 42.0},
			wantErr: false,
		},
		{
			name: "badCast",
			fields: fields{
				config:     ContinuousValueDeviationConfig{},
				PodNameKey: "pod",
				qAPI: &testPrometheusAPI{
					err:   nil,
					value: nil,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &promContinuousValueDeviationAnalyser{
				config: tt.fields.config,
				promConfig: ConfigPrometheusAnomalyDetector{
					PodNameKey: tt.fields.PodNameKey,
					queryAPI:   tt.fields.qAPI,
					logger:     logf.Log,
				},
			}
			got, err := p.doAnalysis()
			if (err != nil) != tt.wantErr {
				t.Errorf("promContinuousValueDeviationAnalyser.doAnalysis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("promContinuousValueDeviationAnalyser.doAnalysis() = %v, want %v", got, tt.want)
			}
		})
	}
}
