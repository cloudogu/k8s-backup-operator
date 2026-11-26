package restore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

// unitChildInPhase returns the provider restore of the given unit and Restore in the specified phase.
func unitChildInPhase(parent *k8sv1.Restore, unit k8sv1.UnitIdentifier, phase velerov1.RestorePhase) *velerov1.Restore {
	child := ownedChild(parent, unit)
	child.Status.Phase = phase

	return child
}

// unitBackups are the completed provider backups for the given units.
func unitBackups(t *testing.T, units ...k8sv1.UnitIdentifier) []client.Object {
	t.Helper()

	annotation, err := json.Marshal(units)
	require.NoError(t, err)
	var backups []client.Object
	for _, unit := range units {
		backup := completedUnitBackup(unit)
		backup.Annotations[k8sv1.UnitsAnnotationKey] = string(annotation)
		backups = append(backups, backup)
	}

	return backups
}

// expectProviderRestoreStart expects the event of starting a provider restore once per expected start.
func expectProviderRestoreStart(t *testing.T, starts int) *mockEventRecorder {
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeNormal, ReasonProviderRestoreRunning, actionCreateProviderRestore, "Start provider restore process").Times(starts)

	return recorderMock
}

func TestProviderRestoreStageRestoresTheUnitsOneAfterAnother(t *testing.T) {
	tests := []struct {
		name         string
		children     func(restore *k8sv1.Restore) []client.Object
		wantCreated  string
		wantBackup   string
		wantChildren []string
	}{
		{
			name:         "without a child the first unit is started",
			children:     func(*k8sv1.Restore) []client.Object { return nil },
			wantCreated:  "test-restore-global-config",
			wantBackup:   "test-backup-global-config",
			wantChildren: []string{"test-restore-global-config"},
		},
		{
			name: "a completed unit starts the next one",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{unitChildInPhase(restore, testGlobalConfigUnit, velerov1.RestorePhaseCompleted)}
			},
			wantCreated:  "test-restore-dogu-ldap",
			wantBackup:   "test-backup-dogu-ldap",
			wantChildren: []string{"test-restore-dogu-ldap", "test-restore-global-config"},
		},
		{
			name: "a failed unit starts the next one",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{unitChildInPhase(restore, testGlobalConfigUnit, velerov1.RestorePhasePartiallyFailed)}
			},
			wantCreated:  "test-restore-dogu-ldap",
			wantBackup:   "test-backup-dogu-ldap",
			wantChildren: []string{"test-restore-dogu-ldap", "test-restore-global-config"},
		},
		{
			name: "a vanished child is recreated even if a later unit ran",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{unitChildInPhase(restore, testLdapUnit, velerov1.RestorePhaseCompleted)}
			},
			wantCreated:  "test-restore-global-config",
			wantBackup:   "test-backup-global-config",
			wantChildren: []string{"test-restore-dogu-ldap", "test-restore-global-config"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := startableRestore()
			objects := append(unitBackups(t, testGlobalConfigUnit, testLdapUnit), tt.children(restore)...)
			k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore, objects...)
			reconciler := NewRestoreReconciler(k8sClient, expectProviderRestoreStart(t, 1), testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment)

			_, outcome := reconciler.ensureProviderRestore(testCtx, restore)

			assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
			assert.Equal(t, tt.wantChildren, childNames(t, k8sClient))
			created := &velerov1.Restore{}
			require.NoError(t, k8sClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: tt.wantCreated}, created))
			assert.Equal(t, tt.wantBackup, created.Spec.BackupName)
		})
	}
}

func TestProviderRestoreStageCreatesNoChildWhenNoUnitIsDue(t *testing.T) {
	tests := []struct {
		name     string
		children func(restore *k8sv1.Restore) []client.Object
	}{
		{
			name: "a unit is still running",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{unitChildInPhase(restore, testGlobalConfigUnit, velerov1.RestorePhaseInProgress)}
			},
		},
		{
			name: "a unit reports an unknown phase",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{unitChildInPhase(restore, testGlobalConfigUnit, "SomeFuturePhase")}
			},
		},
		{
			name: "all units are finished",
			children: func(restore *k8sv1.Restore) []client.Object {
				return []client.Object{
					unitChildInPhase(restore, testGlobalConfigUnit, velerov1.RestorePhaseCompleted),
					unitChildInPhase(restore, testLdapUnit, velerov1.RestorePhaseFailed),
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := startableRestore()
			children := tt.children(restore)
			objects := append(unitBackups(t, testGlobalConfigUnit, testLdapUnit), children...)
			k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore, objects...)
			reconciler := NewRestoreReconciler(k8sClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment)

			_, outcome := reconciler.ensureProviderRestore(testCtx, restore)

			assert.Equal(t, next(), outcome)
			assert.Len(t, childNames(t, k8sClient), len(children))
		})
	}
}

func TestProviderRestoreStageSkipsExcludedUnits(t *testing.T) {
	restore := startableRestore()
	restore.Spec.Options = &k8sv1.UnitOptions{IgnoreGlobalConfig: true}
	// no restore units, yet
	k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore, unitBackups(t, testGlobalConfigUnit, testLdapUnit)...)
	reconciler := NewRestoreReconciler(k8sClient, expectProviderRestoreStart(t, 1), testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment)

	_, outcome := reconciler.ensureProviderRestore(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	// global config was skipped
	assert.Equal(t, []string{"test-restore-dogu-ldap"}, childNames(t, k8sClient))
}

func childNames(t *testing.T, k8sClient client.Client) []string {
	t.Helper()

	children := &velerov1.RestoreList{}
	require.NoError(t, k8sClient.List(testCtx, children))
	var names []string
	for _, child := range children.Items {
		names = append(names, child.Name)
	}

	return names
}
