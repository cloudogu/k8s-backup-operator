package velero

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_defaultReadinessChecker_CheckReady(t *testing.T) {
	t.Run("should fail to get bsl", func(t *testing.T) {
		// given
		mockBslInterface := newMockVeleroBackupStorageLocationInterface(t)
		mockBslInterface.EXPECT().Get(testCtx, "default", metav1.GetOptions{}).Return(nil, assert.AnError)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().BackupStorageLocations(testNamespace).Return(mockBslInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultReadinessChecker{
			veleroClientSet: mockVeleroClient,
			namespace:       testNamespace,
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
		mockBslInterface := newMockVeleroBackupStorageLocationInterface(t)
		mockBslInterface.EXPECT().Get(testCtx, "default", metav1.GetOptions{}).Return(bsl, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().BackupStorageLocations(testNamespace).Return(mockBslInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultReadinessChecker{
			veleroClientSet: mockVeleroClient,
			namespace:       testNamespace,
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
		mockBslInterface := newMockVeleroBackupStorageLocationInterface(t)
		mockBslInterface.EXPECT().Get(testCtx, "default", metav1.GetOptions{}).Return(bsl, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().BackupStorageLocations(testNamespace).Return(mockBslInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultReadinessChecker{
			veleroClientSet: mockVeleroClient,
			namespace:       testNamespace,
		}

		// when
		err := sut.CheckReady(testCtx)

		// then
		require.NoError(t, err)
	})
}

func Test_newReadinessChecker(t *testing.T) {
	// given
	clientSet := newMockVeleroClientSet(t)

	// when
	actual := newReadinessChecker(clientSet, testNamespace)

	// then
	assert.NotEmpty(t, actual)
}
