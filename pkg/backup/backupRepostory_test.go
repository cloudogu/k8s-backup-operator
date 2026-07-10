package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBackupRepository(t *testing.T) {
	var testCtx = context.TODO()

	t.Run("", func(t *testing.T) {
		t.Skip("TODO")

		backupCr := &backupv1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup1",
				Namespace: "ns1",
			},
			Spec: backupv1.BackupSpec{
				Provider: "velero",
			},
		}
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, "backup1", mock.Anything).Return(backupCr, nil)

		backupRepo := NewBackupRespository(backupInterfaceMock)

		backup := Backup{Name: "backup1"}
		err := backupRepo.save(testCtx, backup)

		assert.NoError(t, err)
	})

}
