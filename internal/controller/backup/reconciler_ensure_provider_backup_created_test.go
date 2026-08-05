package backup

import (
	"context"
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureProviderBackupCreated(t *testing.T) {
	t.Run("If the velero backup does not exist, create it, set succeeded to unknown and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup, logr.Discard())

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionUnknown, completedCondition.Status)
		assert.Equal(t, reasonProviderBackupResourceDoesNotExist, completedCondition.Reason)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		assert.Equal(t, 1, counter.veleroBackupCreateCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup resource exists proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.Name,
				Namespace: backup.Namespace,
			},
		}
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		assert.Equal(t, 0, counter.veleroBackupCreateCount)
		assert.Equal(t, 0, counter.subResourcePatchCount)
	})

	t.Run("If retrieving the Velero backup resource fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: errors.New("get error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If patching the status fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			subResourcePatchCallError: errors.New("patch error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If creating the Velero backup resource fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			createCallError: errors.New("create error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "create error")
		assert.Equal(t, Abort, nextAction)
	})
}
