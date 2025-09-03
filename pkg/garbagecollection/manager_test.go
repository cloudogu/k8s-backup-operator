package garbagecollection

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const testNamespace = "test-ns"

var testCtx = context.Background()

func TestNewManager(t *testing.T) {
	clientSetMock := newMockEcosystemClientSet(t)

	manager := NewManager(clientSetMock, testNamespace, "keepAll")
	assert.NotEmpty(t, manager)
}

func Test_manager_CollectGarbage(t *testing.T) {
	t.Run("should fail to get retention strategy", func(t *testing.T) {
		// given
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(nil, assert.AnError)

		sut := &manager{
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get retention strategy")
	})
	t.Run("should fail to list backups", func(t *testing.T) {
		// given
		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list backups")
	})
	t.Run("should fail to delete one backup", func(t *testing.T) {
		// given
		backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-2"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(backups, nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-1", metav1.DeleteOptions{}).Return(assert.AnError)
		backupClientMock.EXPECT().Delete(testCtx, "backup-2", metav1.DeleteOptions{}).Return(nil)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete backup \"backup-1\"")
	})
	t.Run("should fail to delete two backups", func(t *testing.T) {
		// given
		backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-2"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(backups, nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-1", metav1.DeleteOptions{}).Return(assert.AnError)
		backupClientMock.EXPECT().Delete(testCtx, "backup-2", metav1.DeleteOptions{}).Return(assert.AnError)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete backup \"backup-1\"")
		assert.ErrorContains(t, err, "failed to delete backup \"backup-2\"")
	})
	t.Run("should only delete backups filtered for deletion", func(t *testing.T) {
		// given
		backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-2"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(retention.RemovedBackups{backup2}, retention.RetainedBackups{backup1})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(backups, nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-2", metav1.DeleteOptions{}).Return(nil)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should only delete completed backups", func(t *testing.T) {
		// given
		backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-2"}, Status: v1.BackupStatus{Status: v1.BackupStatusFailed}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval([]v1.Backup{backup1}).Return([]v1.Backup{backup1}, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(backups, nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-1", metav1.DeleteOptions{}).Return(nil)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		backup1 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-2"}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		backupClientMock := newMockBackupClient(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(backups, nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-1", metav1.DeleteOptions{}).Return(nil)
		backupClientMock.EXPECT().Delete(testCtx, "backup-2", metav1.DeleteOptions{}).Return(nil)
		v1alpha1Mock := newMockEcosystemV1Alpha1(t)
		v1alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

		sut := &manager{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			strategyName:   retention.StrategyId("keepAll"),
			strategyGetter: strategyGetterMock,
		}

		// when
		err := sut.CollectGarbage(testCtx)

		// then
		require.NoError(t, err)
	})
}
