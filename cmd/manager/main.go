package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/blang/semver"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/amadeusitgroup/kanary/pkg/apis"
	kanaryConfig "github.com/amadeusitgroup/kanary/pkg/config"
	"github.com/amadeusitgroup/kanary/pkg/controller"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	//auto discover if subresource  will work or not
	if os.Getenv(kanaryConfig.KanaryStatusSubresourceDisabledEnvVar) == "" {
		discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(cfg)
		var serverVersion *version.Info
		if serverVersion, err = discoveryClient.ServerVersion(); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		minServerVersion, err := semver.Make("1.10.0")
		if err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		currentServerVersion, err := semver.Make(strings.TrimPrefix(serverVersion.String(), "v"))
		if err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		if currentServerVersion.Compare(minServerVersion) < 0 {
			if err = os.Setenv(kanaryConfig.KanaryStatusSubresourceDisabledEnvVar, "1"); err != nil {
				log.Error(err, "")
				os.Exit(1)
			}
		}
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), "kanary-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	defer func() {
		err = r.Unset()
		if err != nil {
			log.Error(err, "")
		}
	}()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}
}
