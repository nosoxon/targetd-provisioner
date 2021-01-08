package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/util"

	"thrymgjol.io/targetd-provisioner/pkg/targetd"
)

const (
	annoVolume     = "volume"
	annoPool       = "pool"
	annoInitiators = "initiators"
)

var accessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}

type provisioner struct {
	targetd *targetd.Client
	options *Options
}

type StorageClassParameters struct {
	DiscoveryCHAPAuth bool                `json:"chapAuthDiscovery,string,omitempty"`
	SessionCHAPAuth   bool                `json:"chapAuthSession,string,omitempty"`
	FSType            string              `json:"fsType,omitempty"`
	Initiators        string              `json:"initiators,omitempty"`
	IQN               string              `json:"iqn,omitempty"`
	ISCSIInterface    string              `json:"iscsiInterface,omitempty"`
	ReadOnly          bool                `json:"readonly,string,omitempty"`
	TargetPortal      string              `json:"targetPortal,omitempty"`
	Pool              string              `json:"pool,omitempty"`
	Portals           []string
	SecretRef         *v1.SecretReference
	Volume            string
	Storage           resource.Quantity
}

func New(options *Options) controller.Provisioner {
	return &provisioner{
		targetd: targetd.New(options.Targetd),
		options: options,
	}
}

func (pv *provisioner) Provision(_ context.Context, options controller.ProvisionOptions) (
	volume *v1.PersistentVolume, state controller.ProvisioningState, err error,
) {
	state = controller.ProvisioningFinished
	p := pv.getParameters(options)
	klog.Infof("received provisioning request from claim %v/%v for volume %v size %v",
		options.PVC.Namespace, options.PVC.Name, options.PVName, p.Storage.String())

	if !util.AccessModesContainedInAll(accessModes, options.PVC.Spec.AccessModes) {
		klog.Errorf("unsupported volume access modes specified: %v", options.PVC.Spec.AccessModes)
		err = fmt.Errorf("access modes must be a subset of %v", accessModes); return
	}

	lun, err := pv.nextLUN()
	if err != nil {
		klog.Errorf("failed to select viable LUN for volume %v: %v", options.PVName, err)
		err = fmt.Errorf("failed to select viable LUN"); return
	}
	klog.V(4).Infof("selected LUN %v for volume %v", lun, options.PVName)

	if err = pv.targetd.CreateVolume(p.Pool, p.Volume, p.Storage.Value()); err != nil {
		klog.Errorf("failed to create storage asset for volume %v: %v", options.PVName, err)
		err = fmt.Errorf("failed to create storage asset"); return
	}

	if p.Initiators != "" {
		for _, initiator := range strings.Split(p.Initiators, ",") {
			if err = pv.targetd.CreateExport(p.Pool, p.Volume, initiator, lun); err != nil {
				klog.Errorf("failed to export volume %v from pool %v for initiator %v: %v",
					options.PVName, p.Pool, initiator, err)
				err = fmt.Errorf("failed to create export on target server"); return
			}
		}
	}

	klog.Infof("volume %v successfully provisioned", options.PVName)
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   p.Volume,
			Annotations: map[string]string{
				annoVolume:     p.Volume,
				annoPool:       p.Pool,
				annoInitiators: p.Initiators,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			AccessModes:                   options.PVC.Spec.AccessModes,
			VolumeMode:                    options.PVC.Spec.VolumeMode,
			PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
			Capacity:                      v1.ResourceList{v1.ResourceStorage: p.Storage},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				ISCSI: &v1.ISCSIPersistentVolumeSource{
					TargetPortal:      p.TargetPortal,
					Portals:           p.Portals,
					IQN:               p.IQN,
					ISCSIInterface:    p.ISCSIInterface,
					Lun:               int32(lun),
					ReadOnly:          p.ReadOnly,
					FSType:            p.FSType,
					DiscoveryCHAPAuth: p.DiscoveryCHAPAuth,
					SessionCHAPAuth:   p.SessionCHAPAuth,
					SecretRef:         p.SecretRef,
				},
			},
		},
	}, controller.ProvisioningFinished, nil
}

func (pv *provisioner) Delete(_ context.Context, v *v1.PersistentVolume) error {
	pool, volume := v.Annotations[annoPool], v.Annotations[annoVolume]
	klog.Infof("received deletion request from claim %v/%v for volume %v",
		v.Spec.ClaimRef.Namespace, v.Spec.ClaimRef.Name, v.Name)

	if val, ok := v.Annotations[annoInitiators]; ok && val != "" {
		for _, initiator := range strings.Split(val, ",") {
			if err := pv.targetd.DestroyExport(pool, volume, initiator); err != nil {
				klog.Errorf("failed to destroy volume %v export from pool %v for initiator %v: %v",
					v.Name, pool, initiator, err)
			}
		}
	}

	if err := pv.targetd.DestroyVolume(pool, volume); err != nil {
		klog.Errorf("failed to destroy volume %v: %v", v.Name, err)
		return fmt.Errorf("failed to destroy volume")
	}

	return nil
}

func (pv *provisioner) SupportsBlock() bool {
	return true
}

func (pv *provisioner) nextLUN() (int, error) {
	exports, err := pv.targetd.ListExports()
	if err != nil { return 0, err }

	m := make(map[int]bool)
	luns := make([]int, 0)

	for _, e := range exports {
		if n := e.LUN; !m[n] {
			m[n] = true
			luns = append(luns, n)
		}
	}

	sort.Ints(luns)
	for i, lun := range luns {
		if i < lun { return i, nil }
	}

	return len(luns), nil
}

func (pv *provisioner) getParameters(options controller.ProvisionOptions) (scp *StorageClassParameters) {
	var portals []string
	if raw := options.StorageClass.Parameters["portals"]; len(raw) > 0 {
		portals = strings.Split(raw, ",")
	}

	data, _ := json.Marshal(options.StorageClass.Parameters)
	scp = &StorageClassParameters{
		FSType:  pv.options.DefaultFS,
		Portals: portals,
		Pool:    "vg-targetd",
		Volume:  options.PVName,
		Storage: options.PVC.Spec.Resources.Requests[v1.ResourceStorage],
	}
	scp.Storage.Format = resource.BinarySI
	json.Unmarshal(data, scp)

	if scp.DiscoveryCHAPAuth || scp.SessionCHAPAuth {
		scp.SecretRef = &v1.SecretReference{Name: pv.options.Type + "-chap"}
	}

	return
}