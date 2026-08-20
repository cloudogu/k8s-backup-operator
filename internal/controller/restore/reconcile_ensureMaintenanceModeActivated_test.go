package restore

import (
	"testing"

	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(nil).Once()

	sut := &restoreReconciler{maintenanceModeSwitch: maintenanceMock}
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
