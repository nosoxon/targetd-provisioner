package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller"

	"thrymgjol.io/targetd-provisioner/pkg/provisioner"
)

var cmd = &cobra.Command{
	Use:   "targetd-provisioner",
	Short: "dynamic volume provisioner for Kubernetes backed by targetd",
	Long:  "",
	RunE:  Run,
}

var options = provisioner.DefaultOptions()

func main() {
	if err := cmd.Execute(); err != nil {
		klog.Fatalf("execution failed: %v", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	cmd.Flags().AddFlagSet(flags(options))
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}

func initConfig() {
	cmd.Flags().VisitAll(func (f *pflag.Flag) {
		ev := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		if val, ok := os.LookupEnv(ev); ok {
			if err := f.Value.Set(val); err != nil {
				klog.Warningf("unable to parse environment variable %v: %v", ev, err)
			}
		}
	})
}

func Run(_ *cobra.Command, _ []string) (err error) {
	config, err := rest.InClusterConfig()
	if err == rest.ErrNotInCluster {
		config, err = clientcmd.BuildConfigFromFlags(options.APIServer, options.KubeConfig)
	}
	if err != nil {
		klog.Fatalf("failed to load kubernetes configuration: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("failed to create ClientSet: %v", err)
	}

	version, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("failed to determine kubernetes server version: %v", err)
	}

	pv := provisioner.New(options)

	co := options.Controller
	pc := controller.NewProvisionController(clientSet, options.Type, pv, version.GitVersion,
		controller.Threadiness(1),
		controller.ResyncPeriod(co.ResyncPeriod),
		controller.ExponentialBackOffOnError(co.ExponentialBackoff),
		controller.FailedProvisionThreshold(co.FailedProvisionThreshold),
		controller.FailedDeleteThreshold(co.FailedProvisionThreshold),
		controller.LeaseDuration(co.LeaseDuration),
		controller.RenewDeadline(co.RenewDeadline),
		controller.RetryPeriod(co.RetryPeriod))
	pc.Run(context.Background())
	return
}

func flags(o *provisioner.Options) *pflag.FlagSet {
	fs := &pflag.FlagSet{}

	fs.StringVar(&o.Type, "type", o.Type, "provisioner type defined in storage class")
	fs.StringVar(&o.DefaultFS, "default-filesystem", o.DefaultFS, "")
	fs.StringVar(&o.APIServer, "api-server", o.APIServer, "Kubernetes API endpoint")
	fs.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "path to kubeconfig")

	fs.BoolVar(&o.Targetd.Insecure, "targetd-insecure", o.Targetd.Insecure, "")
	fs.StringVar(&o.Targetd.Address, "targetd-address", o.Targetd.Address, "")
	fs.IntVar(&o.Targetd.Port, "targetd-port", o.Targetd.Port, "")
	fs.StringVar(&o.Targetd.Username, "targetd-username", o.Targetd.Username, "")
	fs.StringVar(&o.Targetd.Password, "targetd-password", o.Targetd.Password, "")

	fs.DurationVar(&o.Controller.ResyncPeriod, "resync-period", o.Controller.ResyncPeriod, "how often PVCs, PVs, and storage classes are reenumerated")
	fs.DurationVar(&o.Controller.LeaseDuration, "lease-duration", o.Controller.LeaseDuration, "")
	fs.DurationVar(&o.Controller.RenewDeadline, "renew-deadline", o.Controller.RenewDeadline, "")
	fs.DurationVar(&o.Controller.RetryPeriod, "retry-period", o.Controller.RetryPeriod, "")
	fs.BoolVar(&o.Controller.ExponentialBackoff, "exponential-backoff", o.Controller.ExponentialBackoff, "exponentially back off from provisioning failures")
	fs.IntVar(&o.Controller.FailedProvisionThreshold, "max-retries", o.Controller.FailedProvisionThreshold, "maximum provisioning attempts for each PVC")

	return fs
}
