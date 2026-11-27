package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
)

// unitChildInPhase returns the provider restore of the given unit and Restore in the specified phase.
func unitChildInPhase(parent *k8sv1.Restore, unit k8sv1.UnitIdentifier, phase velerov1.RestorePhase) *velerov1.Restore {
	child := ownedChild(parent, unit)
	child.Status.Phase = phase

	return child
}

// newUnitRestoreReconciler returns a reconciler whose collector reports the global-config and ldap units.
func newUnitRestoreReconciler(t *testing.T, k8sClient k8sClient, recorder eventRecorder, restore *k8sv1.Restore) *restoreReconciler {
	t.Helper()

	collectorMock := newMockRestoreStateCollector(t)
	collectorMock.EXPECT().Collect(testCtx, restore).Return(globalConfigAndLdapUnits(), nil)
	reconciler := NewRestoreReconciler(k8sClient, recorder, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment)
	reconciler.stateCollector = collectorMock

	return reconciler
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
			k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore, tt.children(restore)...)
			reconciler := newUnitRestoreReconciler(t, k8sClient, expectProviderRestoreStart(t, 1), restore)

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
			k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore, children...)
			reconciler := newUnitRestoreReconciler(t, k8sClient, nil, restore)

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
	k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
	reconciler := newUnitRestoreReconciler(t, k8sClient, expectProviderRestoreStart(t, 1), restore)

	_, outcome := reconciler.ensureProviderRestore(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	// global config was skipped
	assert.Equal(t, []string{"test-restore-dogu-ldap"}, childNames(t, k8sClient))
}

func TestProviderRestoreStageRetriesAFailedBackupCollection(t *testing.T) {
	restore := startableRestore()
	collectorMock := newMockRestoreStateCollector(t)
	collectorMock.EXPECT().Collect(testCtx, restore).Return(state.RestoreState{}, assert.AnError)
	k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
	reconciler := &restoreReconciler{k8sClient: k8sClient, stateCollector: collectorMock}

	_, outcome := reconciler.ensureProviderRestore(testCtx, restore)

	require.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to collect the backup of restore test-restore")
	assert.Empty(t, childNames(t, k8sClient), "no child may be started without the collected units")
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
