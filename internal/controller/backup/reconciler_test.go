package backup

import (
	"context"
	"reflect"

	operatortime "github.com/cloudogu/k8s-backup-operator/pkg/time"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestEventRecorder() eventRecorder {
	return record.NewFakeRecorder(100)
}

func newVeleroBackupForReconcilerTest(namespace string, name string, phase velerov1.BackupPhase) *velerov1.Backup {
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: velerov1.BackupStatus{
			Phase: phase,
		},
	}
}

func newVeleroBackupStorageLocationForReconcilerTest(phase velerov1.BackupStorageLocationPhase) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "default",
		},
		Status: velerov1.BackupStorageLocationStatus{
			Phase: phase,
		},
	}
}

type callCounter struct {
	configMapGetCount         int
	veleroBackupGetCount      int
	veleroBackupGetCallError  error
	veleroBackupCreateCount   int
	subResourcePatchCount     int
	subResourcePatchCallError error
	getCallError              error
	createCallError           error
}

func (c *callCounter) getCall(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.getCallError != nil {
		return c.getCallError
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*corev1.ConfigMap]() {
		c.configMapGetCount++
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
		if c.veleroBackupGetCallError != nil {
			return c.veleroBackupGetCallError
		}
		c.veleroBackupGetCount++
	}
	return client.Get(ctx, key, obj, opts...)
}

func (c *callCounter) subResourcePatchCall(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if c.subResourcePatchCallError != nil {
		return c.subResourcePatchCallError
	}
	c.subResourcePatchCount++
	return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
}

func (c *callCounter) createCall(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
	if c.createCallError != nil {
		return c.createCallError
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
		c.veleroBackupCreateCount++
	}
	return client.Create(ctx, obj, opts...)
}

func newRealClock() Clock {
	return &operatortime.Clock{}
}
