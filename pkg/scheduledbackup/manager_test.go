package scheduledbackup

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

const testNamespace = "my-namespace"

var testCtx = context.Background()

func TestNewManager(t *testing.T) {
	// given
	// when
	manager := NewManager(nil, Options{})

	// then
	assert.NotEmpty(t, manager)
}

func TestDefaultManager_ScheduleBackup(t *testing.T) {
	t.Run("should fail to apply backup", func(t *testing.T) {
		// given
		expectedBackupName := "banana-2023-11-30T10.30.00"
		expectedBackup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      expectedBackupName,
				Namespace: testNamespace,
				Labels: map[string]string{
					"app":                          "ces",
					"k8s.cloudogu.com/part-of":     "backup",
					"app.kubernetes.io/name":       "backup",
					"app.kubernetes.io/part-of":    "k8s-backup-operator",
					"app.kubernetes.io/created-by": "k8s-backup-operator",
				},
			},
			Spec: v1.BackupSpec{
				Provider: v1.Provider("velero"),
			},
		}

		clockMock := newMockTimeProvider(t)
		clockMock.EXPECT().Now().Return(time.Date(2023, time.November, 30, 10, 30, 00, 00, time.Local))

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Create(testCtx, expectedBackup, metav1.CreateOptions{}).Return(nil, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		options := Options{Name: "banana", Namespace: testNamespace, Provider: "velero"}

		sut := &DefaultManager{clientSet: clientSetMock, options: options, clock: clockMock}

		// when
		err := sut.ScheduleBackup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to apply backup \"banana-2023-11-30T10.30.00\"")
	})
	t.Run("should succeed to apply backup", func(t *testing.T) {
		// given
		expectedBackupName := "banana-2023-11-30T10.30.00"
		expectedBackup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      expectedBackupName,
				Namespace: testNamespace,
				Labels: map[string]string{
					"app":                          "ces",
					"k8s.cloudogu.com/part-of":     "backup",
					"app.kubernetes.io/name":       "backup",
					"app.kubernetes.io/part-of":    "k8s-backup-operator",
					"app.kubernetes.io/created-by": "k8s-backup-operator",
				},
			},
			Spec: v1.BackupSpec{
				Provider: v1.Provider("velero"),
			},
		}

		clockMock := newMockTimeProvider(t)
		clockMock.EXPECT().Now().Return(time.Date(2023, time.November, 30, 10, 30, 00, 00, time.Local))

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Create(testCtx, expectedBackup, metav1.CreateOptions{}).Return(expectedBackup, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		clientSetMock := newMockEcosystemClientSet(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		options := Options{Name: "banana", Namespace: testNamespace, Provider: "velero"}

		sut := &DefaultManager{clientSet: clientSetMock, options: options, clock: clockMock}

		// when
		err := sut.ScheduleBackup(testCtx)

		// then
		require.NoError(t, err)
	})
}
