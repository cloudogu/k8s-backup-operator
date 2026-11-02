package garbagecollection

import (
	"context"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/retention"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "test-ns"

var testCtx = context.Background()

func TestNewManager(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()

	manager := NewManager(k8sClient, testNamespace, "keepAll")
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

		k8sClient := fake.NewClientBuilder().
			WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
				return assert.AnError
			}}).Build()

		sut := &manager{
			client:         k8sClient,
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
		backup1 := v1.Backup{Name: "backup-1", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{Name: "backup-2", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(&backup1, &backup2).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
					return k8sClient.List(ctx, list, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if obj.GetName() == "backup-1" {
						return assert.AnError
					}
					assert.Equal(t, "backup-2", obj.GetName())
					return client.Delete(ctx, obj, opts...)
				},
			}).Build()

		sut := &manager{
			client:         k8sClient,
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
		backup1 := v1.Backup{Name: "backup-1", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{Name: "backup-2", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(&backup1, &backup2).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
					return k8sClient.List(ctx, list, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if obj.GetName() == "backup-1" {
						return assert.AnError
					}
					if obj.GetName() == "backup-2" {
						return assert.AnError
					}
					assert.Fail(t, "unexpected backup deletion", "backup name: %s", obj.GetName())
					return nil
				},
			}).Build()

		sut := &manager{
			client:         k8sClient,
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
		backup1 := v1.Backup{Name: "backup-1", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{Name: "backup-2", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(retention.RemovedBackups{backup2}, retention.RetainedBackups{backup1})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(&backup1, &backup2).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
					return k8sClient.List(ctx, list, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					assert.Equal(t, "backup-2", obj.GetName())
					return client.Delete(ctx, obj, opts...)
				},
			}).Build()

		sut := &manager{
			client:         k8sClient,
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
		backup1 := v1.Backup{Name: "backup-1", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{Name: "backup-2", Namespace: testNamespace, Status: v1.BackupStatus{Status: v1.BackupStatusFailed}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval([]v1.Backup{backup1}).Return([]v1.Backup{backup1}, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(&backup1, &backup2).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
					return k8sClient.List(ctx, list, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					assert.Equal(t, "backup-1", obj.GetName())
					return client.Delete(ctx, obj, opts...)
				},
			}).Build()

		sut := &manager{
			client:         k8sClient,
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
		backup1 := v1.Backup{Name: "backup-1", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backup2 := v1.Backup{Name: "backup-2", Namespace: testNamespace, ResourceVersion: "999", Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}
		backups := &v1.BackupList{Items: []v1.Backup{backup1, backup2}}

		strategyMock := newMockStrategy(t)
		strategyMock.EXPECT().GetName().Return(retention.KeepAllStrategy)
		strategyMock.EXPECT().FilterForRemoval(backups.Items).Return(backups.Items, retention.RetainedBackups{})
		strategyGetterMock := newMockStrategyGetter(t)
		strategyGetterMock.EXPECT().Get(retention.KeepAllStrategy).Return(strategyMock, nil)

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithObjects(&backup1, &backup2).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, k8sClient client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					assert.Equal(t, testNamespace, opts[0].(*client.ListOptions).Namespace)
					return k8sClient.List(ctx, list, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if obj.GetName() == "backup-1" || obj.GetName() == "backup-2" {
						return client.Delete(ctx, obj, opts...)
					}
					assert.Fail(t, "unexpected backup deletion", "backup name: %s", obj.GetName())
					return nil
				},
			}).Build()

		sut := &manager{
			client:         k8sClient,
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

func testScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	builder := runtime.NewSchemeBuilder(v1.AddToScheme)
	err := builder.AddToScheme(scheme)
	assert.NoError(t, err)
	return scheme
}
