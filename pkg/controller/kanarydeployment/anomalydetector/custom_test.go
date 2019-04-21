package anomalydetector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	test "github.com/amadeusitgroup/kanary/test"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type testHandler struct {
	pods       []*kapiv1.Pod
	t          *testing.T
	returnCode int
	badcontent bool
}

func marshalToJSONForCodecs(obj runtime.Object, gv schema.GroupVersion, codecs serializer.CodecFactory) ([]byte, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, fmt.Errorf("unsupported media type %q", mediaType)
	}

	encoder := codecs.EncoderForVersion(info.Serializer, gv)
	return runtime.Encode(encoder, obj)
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(kapiv1.SchemeGroupVersion, &kapiv1.PodList{}, &kapiv1.Pod{})

	if h.badcontent {
		p := kapiv1.Pod{}
		p.SetGroupVersionKind(kapiv1.SchemeGroupVersion.WithKind("Pod"))
		b, err := marshalToJSONForCodecs(&p, kapiv1.SchemeGroupVersion, serializer.NewCodecFactory(scheme))
		if err != nil {
			h.t.Fatalf("%v", err)
		}
		if _, err := w.Write(b); err != nil {
			h.t.Fatalf("%v", err)
		}
		return
	}

	l := kapiv1.PodList{}
	l.SetGroupVersionKind(kapiv1.SchemeGroupVersion.WithKind("PodList"))
	for _, p := range h.pods {
		l.Items = append(l.Items, *p)
	}
	b, err := marshalToJSONForCodecs(&l, kapiv1.SchemeGroupVersion, serializer.NewCodecFactory(scheme))
	if err != nil {
		h.t.Fatalf("%v", err)
	}

	if h.returnCode != 0 && h.returnCode != 200 {
		w.WriteHeader(h.returnCode)
		return
	}

	if _, err := w.Write(b); err != nil {
		h.t.Fatalf("%v", err)
	}
}

func TestCustomAnomalyDetector_GetPodsOutOfBounds(t *testing.T) {

	pods := []*kapiv1.Pod{
		test.PodGen("A", "test-ns", map[string]string{"app": "foo", "phase": "prd"}, nil, true, true),
		test.PodGen("B", "test-ns", map[string]string{"app": "bar", "phase": "prd"}, nil, true, true),
	}

	handler := &testHandler{
		t:    t,
		pods: pods,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	type fields struct {
		serviceURI string
	}
	tests := []struct {
		name       string
		fields     fields
		returnCode int
		badContent bool
		want       []*kapiv1.Pod
		wantErr    bool
	}{
		{
			name:       "ok",
			returnCode: 200,
			badContent: false,
			fields:     fields{serviceURI: server.URL[len("http://"):]},
			want:       pods,
		},
		{
			name:       "kocontent",
			returnCode: 200,
			badContent: true,
			fields:     fields{serviceURI: server.URL[len("http://"):]},
			want:       pods,
			wantErr:    true,
		},
		{
			name:       "ko404",
			returnCode: 404,
			badContent: false,
			fields:     fields{serviceURI: server.URL[len("http://"):]},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CustomAnomalyDetector{
				serviceURI: tt.fields.serviceURI,
				logger:     logf.Log,
			}
			c.init()
			handler.returnCode = tt.returnCode
			handler.badcontent = tt.badContent
			got, err := c.GetPodsOutOfBounds()
			if (err != nil) != tt.wantErr {
				t.Errorf("CustomAnomalyDetector.GetPodsOutOfBounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == true {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CustomAnomalyDetector.GetPodsOutOfBounds()\ngot = %v\nwant= %v\n", got, tt.want)
			}
		})
	}
}
