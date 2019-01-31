package plugin

import (
	"fmt"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/amadeusitgroup/kanary/pkg/apis"
)

// NewClient returns new client instance
func NewClient(clientConfig clientcmd.ClientConfig) (client.Client, error) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get rest client config, err: %v", err)
	}

	// Create the mapper provider
	mapper, err := apiutil.NewDiscoveryRESTMapper(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to to instantiate mapper, err: %v", err)
	}

	if err = apis.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("unable register kanary apis, err: %v", err)
	}
	// Create the Client for Read/Write operations.
	var newClient client.Client
	newClient, err = client.New(restConfig, client.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate client, err: %v", err)
	}
	return newClient, nil
}
