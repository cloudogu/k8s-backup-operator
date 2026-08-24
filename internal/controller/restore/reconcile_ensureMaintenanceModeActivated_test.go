package restore

import (
	"testing"

	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

func TestMaintenanceModeActivationSkipsAnAlreadyActiveMode(t *testing.T) {
	restore := newParentRestore()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
}

func TestMaintenanceModeActivationActivatesAnInactiveMode(t *testing.T) {
	restore := newParentRestore()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Maintenance mode activated").Once()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(nil).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock, recorder: recorderMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
}

func TestMaintenanceModeActivationStillActivatesWhenStatusCheckFailsAndReportsInactive(t *testing.T) {
	restore := newParentRestore()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().
		GetStatus(testCtx).
		Return(repository.MaintenanceModeDescription{}, false, assert.AnError).
		Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(nil).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(
		restore,
		corev1.EventTypeNormal,
		ReasonMaintenanceModeActivated,
		"Could not get maintenance mode status; continuing restore.",
	).Once()
	recorderMock.EXPECT().Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Maintenance mode activated").Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock, recorder: recorderMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
	assert.NoError(t, outcome.err)
}

func TestMaintenanceModeActivationSkipsActivationWhenStatusCheckReportsActiveWithAnError(t *testing.T) {
	restore := newParentRestore()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().
		GetStatus(testCtx).
		Return(repository.MaintenanceModeDescription{}, true, assert.AnError).
		Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(
		restore,
		corev1.EventTypeNormal,
		ReasonMaintenanceModeActivated,
		"Could not get maintenance mode status; continuing restore.",
	).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock, recorder: recorderMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
	assert.NoError(t, outcome.err)
}

func TestMaintenanceModeActivationFailureIsReportedButDoesNotStopTheRestore(t *testing.T) {
	restore := newParentRestore()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(assert.AnError).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(
		restore,
		corev1.EventTypeNormal,
		ReasonMaintenanceModeActivated,
		"Could not activate maintenance mode; continuing restore.",
	).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock, recorder: recorderMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
	assert.NoError(t, outcome.err)
}

func TestMaintenanceModeActivationSkipsAfterTheWorkflowDeactivatedIt(t *testing.T) {
	recovered := map[string]metav1.Condition{
		ReasonMaintenanceModeDeactivated: {Status: metav1.ConditionUnknown, Reason: ReasonMaintenanceModeDeactivated},
		ReasonWorkloadRecoveryCompleted:  {Status: metav1.ConditionTrue, Reason: ReasonWorkloadRecoveryCompleted},
	}

	for name, recovery := range recovered {
		t.Run(name, func(t *testing.T) {
			restore := newParentRestore()
			restore.Status.Conditions = []metav1.Condition{{
				Type:               k8sv1.ConditionWorkloadsRecovered,
				Status:             recovery.Status,
				Reason:             recovery.Reason,
				LastTransitionTime: metav1.Now(),
			}}
			// A mock without expectations fails the test as soon as the switch is touched at all.
			sut := &restoreReconciler{maintenanceModeSwitch: newMockMaintenanceModeSwitch(t)}

			actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

			require.Same(t, restore, actual)
			assert.Equal(t, actionNext, outcome.action)
		})
	}
}

// Every state before the deliberate deactivation keeps re-asserting the maintenance mode.
func TestMaintenanceModeActivationStillActivatesWhileTheRestoreIsRunning(t *testing.T) {
	restore := newParentRestore()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Maintenance mode activated").Once()
	restore.Status.Conditions = []metav1.Condition{{
		Type:               k8sv1.ConditionWorkloadsRecovered,
		Status:             metav1.ConditionUnknown,
		Reason:             ReasonWaitingForWorkloads,
		LastTransitionTime: metav1.Now(),
	}}
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(nil).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock, recorder: recorderMock}
	actual, outcome := sut.ensureMaintenanceModeActivated(testCtx, restore)

	require.Same(t, restore, actual)
	assert.Equal(t, actionNext, outcome.action)
}
