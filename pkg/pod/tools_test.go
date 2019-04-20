package pod

import (
	"reflect"
	"testing"

	test "github.com/amadeusitgroup/kanary/test"
	kapiv1 "k8s.io/api/core/v1"
)

func TestPurgeNotReadyPods(t *testing.T) {
	type args struct {
		pods []*kapiv1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    []*kapiv1.Pod
		wantErr bool
	}{
		{
			name: "empty",
			want: []*kapiv1.Pod{},
		},
		{
			name: "all ok",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, nil, true, true),
					test.PodGen("B", "test-ns", nil, nil, true, true),
					test.PodGen("C", "test-ns", nil, nil, true, true),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("A", "test-ns", nil, nil, true, true),
				test.PodGen("B", "test-ns", nil, nil, true, true),
				test.PodGen("C", "test-ns", nil, nil, true, true),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, nil, false, true),
					test.PodGen("B", "test-ns", nil, nil, true, false),
					test.PodGen("C", "test-ns", nil, nil, false, false),
					test.PodGen("D", "test-ns", nil, nil, true, true),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("D", "test-ns", nil, nil, true, true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := PurgeNotReadyPods(tt.args.pods)
			if tt.wantErr && gotErr == nil {
				t.Errorf("PurgeNotReadyPods().error want error, current: %v", gotErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PurgeNotReadyPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeepRunningPods(t *testing.T) {
	type args struct {
		pods []*kapiv1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    []*kapiv1.Pod
		wantErr error
	}{
		{
			name: "empty",
			want: []*kapiv1.Pod{},
		},
		{
			name: "all ok",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, nil, true, true),
					test.PodGen("B", "test-ns", nil, nil, true, true),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("A", "test-ns", nil, nil, true, true),
				test.PodGen("B", "test-ns", nil, nil, true, true),
			},
		},
		{
			name: "mix",
			args: args{
				pods: []*kapiv1.Pod{
					test.PodGen("A", "test-ns", nil, nil, false, true),
					test.PodGen("B", "test-ns", nil, nil, true, false),
				},
			},
			want: []*kapiv1.Pod{
				test.PodGen("B", "test-ns", nil, nil, true, false),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := KeepRunningPods(tt.args.pods)
			if gotErr != tt.wantErr {
				t.Errorf("KeepRunningPods().error = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KeepRunningPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeFromSlice(t *testing.T) {
	pod1 := test.PodGen("pod1", "test-ns", nil, nil, true, false)
	pod2 := test.PodGen("pod2", "test-ns", nil, nil, true, false)
	pod3 := test.PodGen("pod3", "test-ns", nil, nil, true, false)
	pod4 := test.PodGen("pod4", "test-ns", nil, nil, true, false)
	pod5 := test.PodGen("pod5", "test-ns", nil, nil, true, false)

	type args struct {
		fromSlice []*kapiv1.Pod
		inSlice   []*kapiv1.Pod
	}
	tests := []struct {
		name string
		args args
		want []*kapiv1.Pod
	}{
		{
			name: "similar slice",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3},
			},
			want: []*kapiv1.Pod{},
		},
		{
			name: "missing pods",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3},
			},
			want: []*kapiv1.Pod{pod4, pod5},
		},
		{
			name: "additional pods",
			args: args{
				fromSlice: []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
				inSlice:   []*kapiv1.Pod{pod1, pod2, pod3, pod4, pod5},
			},
			want: []*kapiv1.Pod{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExcludeFromSlice(tt.args.fromSlice, tt.args.inSlice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExcludeFromSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
