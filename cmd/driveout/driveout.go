package main

import (
	"context"
	"flag"
	"fmt"
	apisv1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/local-storage/v1alpha1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	driveoutassistant "github.com/hwameistor/hwameistor/pkg/driveout"

	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	lockName                       = "driveout-assistant"
	storageServerDefaultPort       = 80
	driveoutCoolDownDefaultMinutes = 15
)

var BUILDVERSION, BUILDTIME, GOVERSION string

func printVersion() {
	log.Info(fmt.Sprintf("GitCommit:%q, BuildDate:%q, GoVersion:%q", BUILDVERSION, BUILDTIME, GOVERSION))
}

func setupLogging(debug bool) {

	if debug {
		log.SetLevel(log.DebugLevel)
	}

}

func main() {

	var debug bool
	var driveoutCooldownDurationMinutes time.Duration
	var storageServerPort int
	flag.BoolVar(&debug, "debug", true, "debug mode")
	flag.DurationVar(&driveoutCooldownDurationMinutes, "driveout-cooldown-duration-minutes", driveoutCoolDownDefaultMinutes, "node driveout cooldown duration in minutes")
	flag.IntVar(&storageServerPort, "storage-server-service-port", storageServerDefaultPort, "storage server service port")
	flag.Parse()

	setupLogging(debug)
	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to get kubernetes cluster config")
	}

	leClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create client set")
	}

	// Set default manager options
	options := manager.Options{}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup Scheme for all resources of Local Storage Member
	if err := apisv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.WithError(err).Error("Failed to setup scheme for all resources")
		os.Exit(1)
	}

	run := func(ctx context.Context) {
		stopCh := signals.SetupSignalHandler()
		// Start the resource controllers manager
		go func() {
			log.Info("Starting the manager of the resources.")
			if err := mgr.Start(stopCh); err != nil {
				log.WithError(err).Error("Failed to run resources manager")
				os.Exit(1)
			}
		}()
		if err := driveoutassistant.New(driveoutCooldownDurationMinutes*time.Minute, mgr).Run(stopCh); err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Error("failed to run driveout assistant")
			os.Exit(1)
		}
	}

	le := leaderelection.NewLeaderElection(leClientset, lockName, run)
	opNamespace, _ := k8sutil.GetOperatorNamespace()
	le.WithNamespace(opNamespace)

	if err := le.Run(); err != nil {
		log.Fatalf("failed to initialize leader election: %v", err)
	}
}
