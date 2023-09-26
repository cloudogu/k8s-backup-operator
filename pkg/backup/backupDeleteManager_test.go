package backup

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewBackupDeleteManager(t *testing.T) {
	manager := NewBackupDeleteManager(nil, nil)

	require.NotNil(t, manager)
}

func Test_backupDeleteManager_delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().DeleteBackup(context.TODO(), backup).Return(nil)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder, namespace string) (Provider, error) {
			return providerMock, nil
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusDeleting(context.TODO(), backup).Return(backup, nil)
		clientMock.EXPECT().RemoveFinalizer(context.TODO(), backup, v1.BackupFinalizer).Return(backup, nil)
		recorderMock := newMockEventRecorder(t)
		sut := &backupDeleteManager{client: clientMock, recorder: recorderMock}

		// when
		err := sut.delete(context.TODO(), backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on status update error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusDeleting(context.TODO(), backup).Return(nil, assert.AnError)
		recorderMock := newMockEventRecorder(t)
		sut := &backupDeleteManager{client: clientMock, recorder: recorderMock}

		// when
		err := sut.delete(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [deleting] in backup resource")
	})

	t.Run("should return error on unknown provider", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "unknown"}}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusDeleting(context.TODO(), backup).Return(backup, nil)
		recorderMock := newMockEventRecorder(t)
		sut := &backupDeleteManager{client: clientMock, recorder: recorderMock}

		// when
		err := sut.delete(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete backup: failed to get backup provider: unknown backup provider unknown")
	})

	t.Run("should, return error on backup deletion error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().DeleteBackup(context.TODO(), backup).Return(assert.AnError)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder, namespace string) (Provider, error) {
			return providerMock, nil
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusDeleting(context.TODO(), backup).Return(backup, nil)
		recorderMock := newMockEventRecorder(t)
		sut := &backupDeleteManager{client: clientMock, recorder: recorderMock}

		// when
		err := sut.delete(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete backup")
	})
}
