package provisioner

import (
	"time"

	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller"

	"thrymgjol.io/targetd-provisioner/pkg/targetd"
)

const (
	defaultTargetdPort     = 18700
	defaultProvisionerType = "targetd"
	defaultFileSystem      = "ext4"
	defaultLogLevel        = "info"
)

type Options struct {
	Targetd    *targetd.Options
	Controller *ControllerOptions
	APIServer  string
	KubeConfig string
	Type       string
	DefaultFS  string
	LogLevel   string
}

type ControllerOptions struct {
	ResyncPeriod             time.Duration
	LeaseDuration            time.Duration
	RenewDeadline            time.Duration
	RetryPeriod              time.Duration
	FailedProvisionThreshold int
	ExponentialBackoff       bool
}

func DefaultOptions() *Options {
	return &Options{
		Targetd: &targetd.Options{
			Insecure: true,
			Port:     defaultTargetdPort,
		},
		Controller: &ControllerOptions{
			ResyncPeriod:             controller.DefaultResyncPeriod,
			LeaseDuration:            controller.DefaultLeaseDuration,
			RenewDeadline:            controller.DefaultRenewDeadline,
			RetryPeriod:              controller.DefaultRetryPeriod,
			FailedProvisionThreshold: controller.DefaultFailedProvisionThreshold,
			ExponentialBackoff:       controller.DefaultExponentialBackOffOnError,
		},
		Type:      defaultProvisionerType,
		DefaultFS: defaultFileSystem,
		LogLevel:  defaultLogLevel,
	}
}
