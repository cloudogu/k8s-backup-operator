package restore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

func TestProviderRestoreRequest(t *testing.T) {
	unitLabelsOf := func(unitType string) map[string]string {
		return map[string]string{"k8s.cloudogu.com/backup": testBackup, "k8s.cloudogu.com/backup-type": unitType}
	}

	tests := []struct {
		name string
		spec k8sv1.RestoreSpec
		unit k8sv1.UnitIdentifier
		want velero.RestoreRequest
	}{
		{
			name: "a legacy restore is named like the restore and restores the whole backup",
			unit: k8sv1.UnitIdentifier{Type: state.UnitTypeLegacy},
			want: velero.RestoreRequest{Name: testRestore, BackupName: testBackup},
		},
		{
			name: "global-config",
			unit: k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeGlobalConfig},
			want: velero.RestoreRequest{
				Name: "test-restore-global-config", BackupName: "test-backup-global-config",
				Labels: unitLabelsOf("global-config"),
			},
		},
		{
			name: "components without exclusions",
			unit: k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeComponents},
			want: velero.RestoreRequest{
				Name: "test-restore-components", BackupName: "test-backup-components",
				Labels: unitLabelsOf("components"),
			},
		},
		{
			name: "components with excluded components",
			spec: k8sv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki"}},
			unit: k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeComponents},
			want: velero.RestoreRequest{
				Name: "test-restore-components", BackupName: "test-backup-components",
				Labels: unitLabelsOf("components"),
				LabelSelector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "k8s.cloudogu.com/component.name", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"k8s-loki"}},
				}},
			},
		},
		{
			// the V2 blob contains no global config, so ignoring it needs no selector here
			name: "doguv2 restores the whole unit even with ignoreGlobalConfig",
			spec: k8sv1.RestoreSpec{Options: &k8sv1.UnitOptions{IgnoreGlobalConfig: true}},
			unit: k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeDoguV2},
			want: velero.RestoreRequest{
				Name: "test-restore-doguv2", BackupName: "test-backup-doguv2",
				Labels: unitLabelsOf("doguv2"),
			},
		},
		{
			name: "dogu carries the dogu name",
			spec: k8sv1.RestoreSpec{Options: &k8sv1.UnitOptions{IgnoreGlobalConfig: true}, ExcludedComponents: []string{"k8s-loki"}},
			unit: k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeDogu, Name: "redmine"},
			want: velero.RestoreRequest{
				Name: "test-restore-dogu-redmine", BackupName: "test-backup-dogu-redmine",
				Labels: map[string]string{
					"k8s.cloudogu.com/backup":      testBackup,
					"k8s.cloudogu.com/backup-type": "dogu",
					"k8s.cloudogu.com/dogu.name":   "redmine",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := newParentRestore()
			restore.Spec.ExcludedComponents = tt.spec.ExcludedComponents
			restore.Spec.Options = tt.spec.Options

			assert.Equal(t, tt.want, providerRestoreRequest(restore, tt.unit))
		})
	}
}

func TestProviderRestoreRequestTruncatesTheBackupLabel(t *testing.T) {
	restore := newParentRestore()
	restore.Spec.BackupName = strings.Repeat("b", 70)

	request := providerRestoreRequest(restore, k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeGlobalConfig})

	// the unit backups carry the same truncated label
	assert.Equal(t, k8sv1.GetTruncatedName(restore.Spec.BackupName), request.Labels["k8s.cloudogu.com/backup"])
	assert.Len(t, request.Labels["k8s.cloudogu.com/backup"], 63)
}
