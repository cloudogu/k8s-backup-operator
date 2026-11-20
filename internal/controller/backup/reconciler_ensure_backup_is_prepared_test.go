package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

func TestReconcilerEnsureBackupIsPrepared(t *testing.T) {
	t.Run("If the velero backup storage location was not found set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationNotFound, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup storage is not available set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationNotAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup storage is available set prepared to true and proceed to the next stage", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroProviderReady, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If a provider backup of another run is still running set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		orphanedVeleroBackup := newVeleroBackupForReconcilerTest("ns", "previous-backup", velerov1.BackupPhaseWaitingForPluginOperations)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1), orphanedVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonOtherProviderBackupInProgress, preparedCondition.Reason)
		assert.Contains(t, preparedCondition.Message, "previous-backup")
		assert.Contains(t, preparedCondition.Message, velerov1.BackupPhaseWaitingForPluginOperations)
	})

	t.Run("An existing own unit backup skips the check", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionPrepared,
			Status:  metav1.ConditionTrue,
			Reason:  veleroprovider.ReasonVeleroProviderReady,
			Message: "prepared in an earlier pass",
		})
		// the unit is assigned by its label, not by its name
		ownVeleroBackup := newUnitVeleroBackupForReconcilerTest("ns", "backup-dogux", "backup", velerov1.BackupPhaseInProgress)
		// no backup storage location and no velero deployment: the provider is unready
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, ownVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status, "the gate must not flip an already running backup back to false")
		assert.Equal(t, 0, counter.subResourcePatchCount)
	})

	t.Run("A running unit backup of another backup sets prepared to false and retries", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		foreignUnitBackup := newUnitVeleroBackupForReconcilerTest("ns", "other-backup-dogux", "other-backup", velerov1.BackupPhaseInProgress)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1), foreignUnitBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonOtherProviderBackupInProgress, preparedCondition.Reason)
		assert.Contains(t, preparedCondition.Message, "other-backup-dogux")
	})

	t.Run("A running velero backup of another run does not block a backup that already started", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.StartTimestamp = metav1.Time{Time: time.Now()}
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		foreignVeleroBackup := newVeleroBackupForReconcilerTest("ns", "foreign-backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1), foreignVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
	})

	t.Run("A finished velero backup of another run does not block the preparation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		previousVeleroBackup := newVeleroBackupForReconcilerTest("ns", "previous-backup", velerov1.BackupPhaseCompleted)
		failedVeleroBackup := newVeleroBackupForReconcilerTest("ns", "failed-backup", velerov1.BackupPhasePartiallyFailed)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1), previousVeleroBackup, failedVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
	})

	t.Run("A velero backup of another namespace does not block the preparation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		foreignVeleroBackup := newVeleroBackupForReconcilerTest("other-ns", "previous-backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1), foreignVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If listing the velero backups failed then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{
			veleroBackupListCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("An unready provider is reported once, not on every retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		recorder := events.NewFakeRecorder(100)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		// second call to verify only one report
		nextAction, err = reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		require.Len(t, recorder.Events, 1, "the unchanged provider state must be reported only once")
		assert.Contains(t, <-recorder.Events, veleroprovider.ReasonVeleroBackupStorageLocationNotFound)
	})

	t.Run("If an error occurred while getting the backup storage location resource then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage is unavailable and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage is available and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage resource was not found and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)

	})

	t.Run("If the velero deployment has no ready replica set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(0)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroDeploymentNotReady, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero deployment was not found set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroDeploymentNotFound, preparedCondition.Reason)
	})
	t.Run("Invalid exclusions fail the backup, report an event and abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedDogus = []string{"ldap", "cas", "scm"}
		backup.Spec.ExcludedComponents = []string{"k8s-dogu-operator", "k8s-longhorn"}
		ldap := &doguv3.Dogu{Name: "ldap", Namespace: "ns", Spec: doguv3.DoguSpec{Name: "ldap", DoguApiVersion: doguv3.DoguApiVersionV3}}
		cas := &doguv3.Dogu{Name: "cas", Namespace: "ns", Spec: doguv3.DoguSpec{Name: "cas", DoguApiVersion: doguv3.DoguApiVersionV2}}
		doguOperator := &componentv1.Component{Name: "k8s-dogu-operator", Namespace: "ns", Spec: componentv1.ComponentSpec{Name: "k8s-dogu-operator"}}
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, ldap, cas, doguOperator, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		recorder := events.NewFakeRecorder(100)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonInvalidExclusions, preparedCondition.Reason)
		assert.Contains(t, preparedCondition.Message, "[scm] are not installed as v3 dogus")
		assert.Contains(t, preparedCondition.Message, "[cas] are v2 dogus")
		assert.Contains(t, preparedCondition.Message, "[k8s-longhorn] are not installed")

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionFalse, succeededCondition.Status)
		assert.Equal(t, reasonInvalidExclusions, succeededCondition.Reason)
		assert.Equal(t, preparedCondition.Message, succeededCondition.Message)
		assert.Equal(t, operationIgnore, requiredOperation(backup), "the failed backup must not be reconciled again")

		assert.Equal(t, 1, counter.subResourcePatchCount)
		require.Len(t, recorder.Events, 1)
		event := <-recorder.Events
		assert.Contains(t, event, corev1.EventTypeWarning)
		assert.Contains(t, event, reasonInvalidExclusions)
	})

	t.Run("Valid exclusions continue with the provider check", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedDogus = []string{"ldap"}
		backup.Spec.ExcludedComponents = []string{"k8s-dogu-operator"}
		ldap := &doguv3.Dogu{Name: "ldap", Namespace: "ns", Spec: doguv3.DoguSpec{Name: "ldap", DoguApiVersion: doguv3.DoguApiVersionV3}}
		doguOperator := &componentv1.Component{Name: "k8s-dogu-operator", Namespace: "ns", Spec: componentv1.ComponentSpec{Name: "k8s-dogu-operator"}}
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, ldap, doguOperator, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroProviderReady, preparedCondition.Reason)
	})

	t.Run("An existing own unit backup skips the validation of the exclusions", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedDogus = []string{"uninstalled-meanwhile"}
		ownVeleroBackup := newUnitVeleroBackupForReconcilerTest("ns", "backup-dogux", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, ownVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		// the mock fails the test if the dogus or components are listed
		reconciler.stateCollector = newMockBackupStateCollector(t)

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Nil(t, meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded))
	})

	t.Run("No exclusions do not list dogus and components", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, providerDeployment(1)).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		reconciler.stateCollector = newMockBackupStateCollector(t)

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If listing the dogus failed then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedDogus = []string{"ldap"}
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().ListDogus(mock.Anything, "ns").Return(nil, assert.AnError)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "validate excluded dogus")
		assert.Equal(t, Abort, nextAction)
		assert.Empty(t, backup.Status.Conditions)
	})

	t.Run("If listing the components failed then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedComponents = []string{"k8s-dogu-operator"}
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().ListComponents(mock.Anything, "ns").Return(nil, assert.AnError)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "validate excluded components")
		assert.Equal(t, Abort, nextAction)
		assert.Empty(t, backup.Status.Conditions)
	})

	t.Run("If patching the status of invalid exclusions failed then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.ExcludedDogus = []string{"scm"}
		counter := &callCounter{subResourcePatchCallError: assert.AnError}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		recorder := events.NewFakeRecorder(100)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "report invalid exclusions")
		assert.Equal(t, Abort, nextAction)
		assert.Empty(t, recorder.Events)
	})

}

func newUnitVeleroBackupForReconcilerTest(namespace string, name string, backupName string, phase velerov1.BackupPhase) *velerov1.Backup {
	veleroBackup := newVeleroBackupForReconcilerTest(namespace, name, phase)
	veleroBackup.Labels = map[string]string{backupv1.BackupLabelKey: backupName}
	return veleroBackup

}
