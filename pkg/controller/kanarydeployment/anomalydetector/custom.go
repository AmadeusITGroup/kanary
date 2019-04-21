package anomalydetector

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	kapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

//CustomAnomalyDetector call an external service to get the list of faulty pods
type CustomAnomalyDetector struct {
	serviceURI string
	logger     logr.Logger
	client     *http.Client
	decoder    runtime.Decoder
}

func (c *CustomAnomalyDetector) init() {
	transport := new(http.Transport)
	setDefaults(transport, http.DefaultTransport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: false}
	c.client = &http.Client{
		Timeout:   time.Second,
		Transport: transport,
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(kapiv1.SchemeGroupVersion, &kapiv1.PodList{})
	c.decoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
}

//GetPodsOutOfBounds implements the anomaly detector interface
func (c *CustomAnomalyDetector) GetPodsOutOfBounds() ([]*kapiv1.Pod, error) {
	response, err := c.client.Get("http://" + c.serviceURI)
	if err != nil {
		return nil, fmt.Errorf("Error while contacting custom server: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the custom did not respond Ok (200) but %d", response.StatusCode)
	}
	bodyBytes, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil {
		return nil, fmt.Errorf("can't read response buffer %v", err2)
	}

	list := &kapiv1.PodList{}
	gvk := kapiv1.SchemeGroupVersion.WithKind("PodList")
	if _, _, err = c.decoder.Decode(bodyBytes, &gvk, list); err != nil {
		return nil, fmt.Errorf("decoding custom server response failed: %v", err)
	}

	result := []*kapiv1.Pod{}
	for i := range list.Items {
		result = append(result, &list.Items[i])
	}
	return result, nil
}

func setDefaults(a, b interface{}) {
	pt := reflect.TypeOf(a)
	t := pt.Elem()
	va := reflect.ValueOf(a).Elem()
	vb := reflect.ValueOf(b).Elem()
	for i := 0; i < t.NumField(); i++ {
		aField := va.Field(i)
		// Set a from b if it is public
		if aField.CanSet() {
			bField := vb.Field(i)
			aField.Set(bField)
		}
	}
}
