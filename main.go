package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	nodeutil "k8s.io/component-helpers/node/util"
	"k8s.io/klog/v2"
)

var (
	hostnameOverride string
	kubeconfig       string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file.")
	flag.StringVar(&hostnameOverride, "hostname-override", "", "If non-empty, will be used as the name of the Node the driver is running on. If unset, the node name is assumed to be the same as the node's hostname.")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: dra-network-driver [options]\n\n")
		flag.PrintDefaults()
	}
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	flag.VisitAll(func(f *flag.Flag) {
		klog.Infof("FLAG: --%s=%q", f.Name, f.Value)
	})
	os.Exit(run())
}

func run() int {
	var config *rest.Config
	var err error
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Infof("can not create client-go configuration: %v", err)
		return 255
	}

	// use protobuf for better performance at scale
	// https://kubernetes.io/docs/reference/using-api/api-concepts/#alternate-representations-of-resources
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("can not create client-go client: %v", err)
	}

	nodeName, err := nodeutil.GetHostname(hostnameOverride)
	if err != nil {
		klog.Fatalf("can not obtain the node name, use the hostname-override flag if you want to set it to a specific value: %v", err)
	}

	// trap Ctrl+C and call cancel on the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Enable signal handler
	signalCh := make(chan os.Signal, 2)
	defer func() {
		close(signalCh)
		cancel()
	}()
	signal.Notify(signalCh, os.Interrupt, unix.SIGINT)

	driver, err := Start(ctx, driverName, clientset, nodeName)
	if err != nil {
		klog.Infof("driver failed to start: %v", err)
		return 1
	}
	defer driver.Stop()

	klog.Info("driver started")

	select {
	case <-signalCh:
		klog.Infof("Exiting: received signal")
		cancel()
	case <-ctx.Done():
	}

	return 0
}
