package velero

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestDefaultProvider_CheckReady(t *testing.T) {
	t.Run("should fail to get bsl", func(t *testing.T) {
		// given
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      "default",
		}, &velerov1.BackupStorageLocation{}).Return(assert.AnError)

		sut := &defaultProvider{
			k8sClient: mockK8sWatchClient,
			namespace: testNamespace,
		}

		// when
		err := sut.CheckReady(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get backup storage location from cluster")
	})
	t.Run("should fail if bsl is unavailable", func(t *testing.T) {
		// given
		bsl := &velerov1.BackupStorageLocation{
			Status: velerov1.BackupStorageLocationStatus{
				Phase:   velerov1.BackupStorageLocationPhaseUnavailable,
				Message: "could not reach minio storage location",
			},
		}
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      "default",
		}, &velerov1.BackupStorageLocation{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			*object.(*velerov1.BackupStorageLocation) = *bsl
			return nil
		})

		sut := &defaultProvider{
			k8sClient: mockK8sWatchClient,
			namespace: testNamespace,
		}

		// when
		err := sut.CheckReady(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "velero is unable to reach the default backup storage location")
		assert.ErrorContains(t, err, "could not reach minio storage location")
	})
	t.Run("should succeed if bsl is available", func(t *testing.T) {
		// given
		bsl := &velerov1.BackupStorageLocation{
			Status: velerov1.BackupStorageLocationStatus{
				Phase: velerov1.BackupStorageLocationPhaseAvailable,
			},
		}
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      "default",
		}, &velerov1.BackupStorageLocation{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			*object.(*velerov1.BackupStorageLocation) = *bsl
			return nil
		})

		sut := &defaultProvider{
			k8sClient: mockK8sWatchClient,
			namespace: testNamespace,
		}

		// when
		err := sut.CheckReady(testCtx)

		// then
		require.NoError(t, err)
	})
}
