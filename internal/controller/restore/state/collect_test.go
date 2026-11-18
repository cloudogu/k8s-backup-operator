package state

import (
	"context"
	"maps"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace   = "ecosystem"
	testRestoreName = "myrestore"
	testBackupName  = "mybackup"
)

var (
	unitsAnnotation      = `[{"type":"doguv2"},{"type":"dogu","name":"redmine"},{"type":"dogu","name":"postfix"},{"type":"global-config"},{"type":"misc"}]`
	dogusV3Annotation    = `[{"name":"redmine","version":"5.1.3-1"},{"name":"postfix","version":"3.8.4-1"}]`
	dogusV2Annotation    = `[{"name":"cockpit","version":"1.2.3"},{"name":"nginx","version":"1.26.1-1"}]`
	componentsAnnotation = `[{"name":"k8s-loki","version":"3.3.2-1"},{"name":"k8s-velero","version":"10.0.1-1"}]`
)

func TestCollect_NotFound(t *testing.T) {
	foreignUnitBackup := veleroBackup("otherbackup-misc", velerov1.BackupPhaseCompleted,
		map[string]string{"k8s.cloudogu.com/backup": "otherbackup", "k8s.cloudogu.com/backup-type": "misc"}, nil)

	restore := &backupv1.Restore{
		Name: testRestoreName, Namespace: testNamespace,
		Spec: backupv1.RestoreSpec{BackupName: testBackupName},
	}
	got, err := NewCollector(fakeClientBuilder(t, []client.Object{foreignUnitBackup}...).Build()).Collect(t.Context(), restore)

	require.NoError(t, err)
	assert.Equal(t, RestoreState{Kind: KindNotFound}, got)
}

func TestCollect_Legacy(t *testing.T) {
	legacyBackup := veleroBackup(testBackupName, velerov1.BackupPhasePartiallyFailed, map[string]string{"app": "ces"}, nil)

	restore := &backupv1.Restore{
		Name: testRestoreName, Namespace: testNamespace,
		Spec: backupv1.RestoreSpec{BackupName: testBackupName},
	}
	got, err := NewCollector(fakeClientBuilder(t, []client.Object{legacyBackup}...).Build()).Collect(t.Context(), restore)

	require.NoError(t, err)
	assert.Equal(t, KindLegacy, got.Kind)
	assert.Len(t, got.Units, 1)
	assert.Equal(t, UnitTypeLegacy, got.Units[0].Type)
	assert.Equal(t, testBackupName, got.Units[0].ProviderBackup.Name)
}

func TestCollect_Units(t *testing.T) {
	globalConfig := unitBackup("global-config", map[string]string{"k8s.cloudogu.com/backup-fqdn": "ces.local"})
	misc := unitBackup("misc", map[string]string{"k8s.cloudogu.com/backup-components": componentsAnnotation})
	doguV2 := unitBackup("doguv2", nil)
	redmine := doguV3UnitBackup("redmine", "5.1.3-1")

	restore := &backupv1.Restore{
		Name: testRestoreName, Namespace: testNamespace,
		Spec: backupv1.RestoreSpec{BackupName: testBackupName},
	}
	got, err := NewCollector(fakeClientBuilder(t, []client.Object{globalConfig, misc, doguV2, redmine}...).Build()).Collect(t.Context(), restore)

	require.NoError(t, err)
	assert.Equal(t, KindUnits, got.Kind)
	// globalConf + misc + doguV2 + postfix(missing) + redmine
	assert.Len(t, got.Units, 5)
	// units are deterministically sorted by type then name
	assert.Equal(t, backupv1.UnitTypeGlobalConfig, got.Units[0].Type)
	assert.Equal(t, "mybackup-global-config", got.Units[0].ProviderBackup.Name)

	assert.Equal(t, backupv1.UnitTypeMisc, got.Units[1].Type)
	assert.Equal(t, "mybackup-misc", got.Units[1].ProviderBackup.Name)

	assert.Equal(t, backupv1.UnitTypeDoguV2, got.Units[2].Type)
	assert.Equal(t, "mybackup-doguv2", got.Units[2].ProviderBackup.Name)

	assert.Equal(t, backupv1.UnitTypeDogu, got.Units[3].Type)
	assert.Equal(t, "postfix", got.Units[3].Name)
	assert.Empty(t, got.Units[3].ProviderBackup) // Backup missing

	assert.Equal(t, backupv1.UnitTypeDogu, got.Units[4].Type)
	assert.Equal(t, "redmine", got.Units[4].Name)
	assert.Equal(t, "mybackup-dogu-redmine", got.Units[4].ProviderBackup.Name)

	assert.Equal(t, []string{"cockpit", "nginx"}, got.V2Dogus)
	assert.Equal(t, []string{"k8s-loki", "k8s-velero"}, got.Components)
}

func TestCollect_SameNamedUnitBackupIsNoLegacyBackup(t *testing.T) {
	// we detect by labels/annotations not by name
	labelled := veleroBackup(testBackupName, velerov1.BackupPhaseCompleted,
		map[string]string{"k8s.cloudogu.com/backup": testBackupName, "k8s.cloudogu.com/backup-type": "misc"},
		map[string]string{"k8s.cloudogu.com/backup-units": `[{"type":"misc"}]`})

	restore := &backupv1.Restore{
		Name: testRestoreName, Namespace: testNamespace,
		Spec: backupv1.RestoreSpec{BackupName: testBackupName},
	}
	got, err := NewCollector(fakeClientBuilder(t, []client.Object{labelled}...).Build()).Collect(t.Context(), restore)

	require.NoError(t, err)
	assert.Equal(t, KindUnits, got.Kind)
	assert.Len(t, got.Units, 1)
	assert.Equal(t, backupv1.UnitTypeMisc, got.Units[0].Type)
	assert.Equal(t, testBackupName, got.Units[0].ProviderBackup.Name)
}

func TestCollect_Errors(t *testing.T) {
	unitLabels := func(unitType string) map[string]string {
		return map[string]string{"k8s.cloudogu.com/backup": testBackupName, "k8s.cloudogu.com/backup-type": unitType}
	}
	withoutUnits := veleroBackup("mybackup-misc", velerov1.BackupPhaseCompleted, unitLabels("misc"), nil)
	brokenUnits := veleroBackup("mybackup-misc", velerov1.BackupPhaseCompleted, unitLabels("misc"),
		map[string]string{"k8s.cloudogu.com/backup-units": `{broken`})
	brokenComponents := veleroBackup("mybackup-misc", velerov1.BackupPhaseCompleted, unitLabels("misc"),
		map[string]string{"k8s.cloudogu.com/backup-units": `[{"type":"misc"}]`, "k8s.cloudogu.com/backup-components": `{broken`})
	brokenV2Dogus := veleroBackup("mybackup-doguv2", velerov1.BackupPhaseCompleted, unitLabels("doguv2"),
		map[string]string{"k8s.cloudogu.com/backup-units": `[{"type":"doguv2"}]`, "k8s.cloudogu.com/backup-installed-dogus-v2": `{broken`})
	doguV2 := unitBackup("doguv2", nil)

	tests := []struct {
		name         string
		builder      *fake.ClientBuilder
		wantContains string
		wantIs       error
	}{
		{
			name: "get legacy backup",
			builder: fakeClientBuilder(t, doguV2).WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return assert.AnError
				},
			}),
			wantContains: "get legacy backup: get velero backup mybackup",
			wantIs:       assert.AnError,
		},
		{
			name: "list unit backups",
			builder: fakeClientBuilder(t, doguV2).WithInterceptorFuncs(interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return assert.AnError
				},
			}),
			wantContains: "list unit backups",
			wantIs:       assert.AnError,
		},
		{name: "missing units annotation", builder: fakeClientBuilder(t, withoutUnits), wantContains: "collect units: provider backup mybackup-misc has no annotation k8s.cloudogu.com/backup-units"},
		{name: "invalid units annotation", builder: fakeClientBuilder(t, brokenUnits), wantContains: "collect units: get units: unmarshal annotation k8s.cloudogu.com/backup-units"},
		{name: "invalid v2 dogus annotation", builder: fakeClientBuilder(t, brokenV2Dogus), wantContains: "collect units: get installed V2 dogus: unmarshal annotation k8s.cloudogu.com/backup-installed-dogus-v2"},
		{name: "invalid components annotation", builder: fakeClientBuilder(t, brokenComponents), wantContains: "collect units: get misc components: unmarshal annotation k8s.cloudogu.com/backup-components"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCollector(tt.builder.Build()).Collect(t.Context(), &backupv1.Restore{
				Name: testRestoreName, Namespace: testNamespace,
				Spec: backupv1.RestoreSpec{BackupName: testBackupName},
			})

			require.ErrorContains(t, err, tt.wantContains)
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
		})
	}
}

// doguV3UnitBackup returns a velero backup of a V3 dogu unit like the backup operator creates it.
func doguV3UnitBackup(doguName, doguVersion string) *velerov1.Backup {
	backup := unitBackup("dogu", nil)
	backup.Name += "-" + doguName
	backup.Labels["k8s.cloudogu.com/dogu.name"] = doguName
	backup.Labels["k8s.cloudogu.com/dogu.version"] = doguVersion
	return backup
}

// unitBackup returns a velero backup of a unit without dogu name like the backup operator creates it.
func unitBackup(unitType string, additionalAnnotations map[string]string) *velerov1.Backup {
	labels := map[string]string{
		"app":                              "ces",
		"k8s.cloudogu.com/part-of":         "backup",
		"k8s.cloudogu.com/backup":          testBackupName,
		"k8s.cloudogu.com/backup-coverage": "full",
		"k8s.cloudogu.com/backup-type":     unitType,
	}

	annotations := map[string]string{
		"backup.cloudogu.com/blueprintId":            "myBlueprint",
		"k8s.cloudogu.com/backup-units":              unitsAnnotation,
		"k8s.cloudogu.com/backup-installed-dogus-v3": dogusV3Annotation,
		"k8s.cloudogu.com/backup-installed-dogus-v2": dogusV2Annotation,
	}
	maps.Copy(annotations, additionalAnnotations)

	return veleroBackup(testBackupName+"-"+unitType, velerov1.BackupPhaseCompleted, labels, annotations)
}

func veleroBackup(name string, phase velerov1.BackupPhase, labels, annotations map[string]string) *velerov1.Backup {
	return &velerov1.Backup{
		Name: name, Namespace: testNamespace, Labels: labels, Annotations: annotations,
		Status: velerov1.BackupStatus{Phase: phase},
	}
}

func fakeClientBuilder(t *testing.T, objects ...client.Object) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))
	require.NoError(t, velerov1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...)
}
