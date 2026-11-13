package backup

import (
	"context"
	"fmt"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerEnsureProviderBackupCompleted(t *testing.T) {
	t.Run("fail to list provider backups", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, isVeleroBackups := list.(*velerov1.BackupList); isVeleroBackups {
					return assert.AnError
				}
				return client.List(ctx, list, opts...)
			}}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "listing unit backups")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("fail to collect backup state", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().Collect(t.Context(), backup).Return(state.BackupState{}, assert.AnError)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "collecting backup state")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("fail to unmarshal v2 dogus", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		veleroBackup.Labels[backupv1.BackupLabelKey] = "backup"
		veleroBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
		veleroBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = "invalid json"
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().Collect(t.Context(), backup).
			Return(state.BackupState{
				Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDoguV2}},
			}, nil)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorContains(t, err, "determining unit completion: unmarshal installed dogus v2 from unit")
		assert.Equal(t, Abort, nextAction)
	})

	patchStatusErrTests := []struct {
		name         string
		veleroStatus velerov1.BackupPhase
	}{
		{
			name:         "in progress",
			veleroStatus: velerov1.BackupPhaseInProgress,
		},
		{
			name:         "failed",
			veleroStatus: velerov1.BackupPhaseFailed,
		},
		{
			name:         "succeeded",
			veleroStatus: velerov1.BackupPhaseCompleted,
		},
	}
	for _, tt := range patchStatusErrTests {
		t.Run(fmt.Sprintf("fail to patch status %s", tt.name), func(t *testing.T) {
			backup := newBackupForTest("ns", "backup")
			veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", tt.veleroStatus)
			veleroBackup.Labels[backupv1.BackupLabelKey] = "backup"
			veleroBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
			veleroBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = `[{"name": "cockpit", "version": "1.2.3-1"}]`
			fakeClient := newFakeClientBuilder(t).
				WithObjects(backup, veleroBackup).
				WithStatusSubresource(backup).
				WithInterceptorFuncs(interceptor.Funcs{SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return assert.AnError
				}}).
				Build()
			reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
			stateCollector := newMockBackupStateCollector(t)
			stateCollector.EXPECT().Collect(t.Context(), backup).
				Return(state.BackupState{
					Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDoguV2}},
				}, nil)
			reconciler.stateCollector = stateCollector

			nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, "patch status of backup resource")
			assert.Equal(t, Abort, nextAction)
		})
	}

	t.Run("fail on unknown status", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseQueued)
		veleroBackup.Labels[backupv1.BackupLabelKey] = "backup"
		veleroBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
		veleroBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = `[{"name": "cockpit", "version": "1.2.3-1"}]`
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().Collect(t.Context(), backup).
			Return(state.BackupState{
				Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDoguV2}},
			}, nil)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorContains(t, err, "provider backup is in unknown status")
		assert.Equal(t, Abort, nextAction)
	})

	successTests := []struct {
		name                 string
		backup               *backupv1.Backup
		veleroBackups        *velerov1.BackupList
		backupState          state.BackupState
		expectedStatus       string
		expectedCondition    metav1.Condition
		expectedGlobalConfig *backupv1.GlobalConfigStatus
		expectedV2Dogus      *backupv1.V2DogusUnitStatus
		expectedDogus        []backupv1.DoguUnitStatus
		expectedMisc         *backupv1.UnitStatus
		expectedAction       action
	}{
		{
			name:          "no provider unit found means in progress",
			backup:        newBackupForTest("ns", "backup"),
			veleroBackups: &velerov1.BackupList{},
			backupState: state.BackupState{
				Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDoguV2}},
			},
			expectedStatus: "in progress",
			expectedCondition: metav1.Condition{
				Status:  "Unknown",
				Reason:  "ProviderBackupInProgress",
				Message: "Provider backup is in progress (running for less than 1m).",
			},
			expectedV2Dogus: &backupv1.V2DogusUnitStatus{
				State:   "Missing",
				Reason:  "ProviderBackupResourceDoesNotExist",
				Message: "the provider backup for this unit is missing or might not have been created yet",
				Dogus:   nil,
			},
			expectedAction: Retry,
		},
		{
			name:   "in progress provider backup means in progress",
			backup: newBackupForTest("ns", "backup"),
			veleroBackups: &velerov1.BackupList{
				Items: []velerov1.Backup{func() velerov1.Backup {
					providerBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFinalizing)
					providerBackup.Annotations = map[string]string{
						backupv1.FQDNAnnotationKey:   "my.fqdn",
						backupv1.DomainAnnotationKey: "my.domain",
					}
					providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
					providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeGlobalConfig)
					return *providerBackup
				}()},
			},
			backupState: state.BackupState{
				Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeGlobalConfig}},
			},
			expectedStatus: "in progress",
			expectedCondition: metav1.Condition{
				Status:  "Unknown",
				Reason:  "ProviderBackupInProgress",
				Message: "Provider backup is in progress (running for less than 1m).",
			},
			expectedGlobalConfig: &backupv1.GlobalConfigStatus{
				State:       "",
				Reason:      "ProviderBackupInProgress",
				Message:     "provider backup is in progress with status phase: Finalizing",
				ProviderRef: &backupv1.ProviderRef{Name: "backup"},
				FQDN:        "my.fqdn",
				Domain:      "my.domain",
			},
			expectedAction: Retry,
		},
		{
			name:   "failed provider backup means failed",
			backup: newBackupForTest("ns", "backup"),
			veleroBackups: &velerov1.BackupList{
				Items: []velerov1.Backup{func() velerov1.Backup {
					providerBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhasePartiallyFailed)
					providerBackup.Labels[backupv1.DoguNameLabelKey] = "baseline"
					providerBackup.Labels[backupv1.DoguVersionLabelKey] = "1.2.3-3"
					providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
					providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDogu)
					return *providerBackup
				}()},
			},
			backupState: state.BackupState{
				Units:            []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDogu, Name: "baseline"}},
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "1.2.3-3"}},
			},
			expectedStatus: "in progress",
			expectedCondition: metav1.Condition{
				Status:  "False",
				Reason:  "ProviderBackupFailed",
				Message: "Provider backup has failed.",
			},
			expectedDogus: []backupv1.DoguUnitStatus{
				{
					Name:        "baseline",
					Version:     "1.2.3-3",
					State:       "Failed",
					Reason:      "ProviderBackupFailed",
					Message:     "provider backup failed with status phase: PartiallyFailed",
					ProviderRef: &backupv1.ProviderRef{Name: "backup"}},
			},
			expectedAction: Next,
		},
		{
			name:   "completion state priority is respected",
			backup: newBackupForTest("ns", "backup"),
			veleroBackups: &velerov1.BackupList{
				Items: []velerov1.Backup{
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "baseline", velerov1.BackupPhasePartiallyFailed)
						providerBackup.Labels[backupv1.DoguNameLabelKey] = "baseline"
						providerBackup.Labels[backupv1.DoguVersionLabelKey] = "1.2.3-3"
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDogu)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "globalconfig", velerov1.BackupPhaseFinalizing)
						providerBackup.Annotations = map[string]string{
							backupv1.FQDNAnnotationKey:   "my.fqdn",
							backupv1.DomainAnnotationKey: "my.domain",
						}
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeGlobalConfig)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "misc", velerov1.BackupPhaseFailed)
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeMisc)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "doguv2", velerov1.BackupPhaseCompleted)
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
						providerBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = `[{"name": "cockpit", "version": "1.2.3-1"}]`
						return *providerBackup
					}(),
				},
			},
			backupState: state.BackupState{
				Units: []backupv1.UnitIdentifier{
					{Type: backupv1.UnitTypeGlobalConfig},
					{Type: backupv1.UnitTypeDoguV2},
					{Type: backupv1.UnitTypeMisc},
					{Type: "invalid"},
					{Type: backupv1.UnitTypeDogu, Name: "baseline"},
				},
				InstalledDogusV2: []backupv1.DoguVersion{{Name: "cockpit", Version: "1.2.3-1"}},
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "1.2.3-3"}},
			},
			expectedStatus: "in progress",
			expectedCondition: metav1.Condition{
				Status:  "Unknown",
				Reason:  "ProviderBackupInProgress",
				Message: "Provider backup is in progress (running for less than 1m).",
			},
			expectedGlobalConfig: &backupv1.GlobalConfigStatus{
				State:       "",
				Reason:      "ProviderBackupInProgress",
				Message:     "provider backup is in progress with status phase: Finalizing",
				ProviderRef: &backupv1.ProviderRef{Name: "globalconfig"},
				FQDN:        "my.fqdn",
				Domain:      "my.domain",
			},
			expectedV2Dogus: &backupv1.V2DogusUnitStatus{
				State:       "Succeeded",
				Reason:      "ProviderBackupSucceeded",
				Message:     "",
				ProviderRef: &backupv1.ProviderRef{Name: "doguv2"},
				Dogus: []backupv1.DoguVersion{{
					Name:    "cockpit",
					Version: "1.2.3-1",
				}},
			},
			expectedMisc: &backupv1.UnitStatus{
				State:       "Failed",
				Reason:      "ProviderBackupFailed",
				Message:     "provider backup failed with status phase: Failed",
				ProviderRef: &backupv1.ProviderRef{Name: "misc"},
			},
			expectedDogus: []backupv1.DoguUnitStatus{
				{
					Name:        "baseline",
					Version:     "1.2.3-3",
					State:       "Failed",
					Reason:      "ProviderBackupFailed",
					Message:     "provider backup failed with status phase: PartiallyFailed",
					ProviderRef: &backupv1.ProviderRef{Name: "baseline"},
				},
			},
			expectedAction: Retry,
		},
		{
			name:   "succeed if all is completed",
			backup: newBackupForTest("ns", "backup"),
			veleroBackups: &velerov1.BackupList{
				Items: []velerov1.Backup{
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "baseline", velerov1.BackupPhaseCompleted)
						providerBackup.Labels[backupv1.DoguNameLabelKey] = "baseline"
						providerBackup.Labels[backupv1.DoguVersionLabelKey] = "1.2.3-3"
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDogu)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "globalconfig", velerov1.BackupPhaseCompleted)
						providerBackup.Annotations = map[string]string{
							backupv1.FQDNAnnotationKey:   "my.fqdn",
							backupv1.DomainAnnotationKey: "my.domain",
						}
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeGlobalConfig)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "misc", velerov1.BackupPhaseCompleted)
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeMisc)
						return *providerBackup
					}(),
					func() velerov1.Backup {
						providerBackup := newVeleroBackupForReconcilerTest("ns", "doguv2", velerov1.BackupPhaseCompleted)
						providerBackup.Labels[backupv1.BackupLabelKey] = "backup"
						providerBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
						providerBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = `[{"name": "cockpit", "version": "1.2.3-1"}]`
						return *providerBackup
					}(),
				},
			},
			backupState: state.BackupState{
				Units: []backupv1.UnitIdentifier{
					{Type: backupv1.UnitTypeGlobalConfig},
					{Type: backupv1.UnitTypeDoguV2},
					{Type: backupv1.UnitTypeMisc},
					{Type: backupv1.UnitTypeDogu, Name: "baseline"},
				},
				InstalledDogusV2: []backupv1.DoguVersion{{Name: "cockpit", Version: "1.2.3-1"}},
				InstalledDogusV3: []backupv1.DoguVersion{{Name: "baseline", Version: "1.2.3-3"}},
			},
			expectedStatus: "in progress",
			expectedCondition: metav1.Condition{
				Status:  "True",
				Reason:  "ProviderBackupSucceeded",
				Message: "Provider backup has succeeded.",
			},
			expectedGlobalConfig: &backupv1.GlobalConfigStatus{
				State:       "Succeeded",
				Reason:      "ProviderBackupSucceeded",
				Message:     "",
				ProviderRef: &backupv1.ProviderRef{Name: "globalconfig"},
				FQDN:        "my.fqdn",
				Domain:      "my.domain",
			},
			expectedV2Dogus: &backupv1.V2DogusUnitStatus{
				State:       "Succeeded",
				Reason:      "ProviderBackupSucceeded",
				Message:     "",
				ProviderRef: &backupv1.ProviderRef{Name: "doguv2"},
				Dogus: []backupv1.DoguVersion{{
					Name:    "cockpit",
					Version: "1.2.3-1",
				}},
			},
			expectedMisc: &backupv1.UnitStatus{
				State:       "Succeeded",
				Reason:      "ProviderBackupSucceeded",
				Message:     "",
				ProviderRef: &backupv1.ProviderRef{Name: "misc"},
			},
			expectedDogus: []backupv1.DoguUnitStatus{
				{
					Name:        "baseline",
					Version:     "1.2.3-3",
					State:       "Succeeded",
					Reason:      "ProviderBackupSucceeded",
					Message:     "",
					ProviderRef: &backupv1.ProviderRef{Name: "baseline"},
				},
			},
			expectedAction: Next,
		},
	}
	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newFakeClientBuilder(t).
				WithObjects(tt.backup).
				WithLists(tt.veleroBackups).
				WithStatusSubresource(tt.backup).
				Build()
			reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
			stateCollector := newMockBackupStateCollector(t)
			stateCollector.EXPECT().Collect(t.Context(), tt.backup).
				Return(tt.backupState, nil)
			reconciler.stateCollector = stateCollector

			nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), tt.backup)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAction, nextAction)
			actualBackup := &backupv1.Backup{}

			err = fakeClient.Get(t.Context(), tt.backup.GetNamespacedName(), actualBackup)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, actualBackup.Status.Status)

			actualCondition := meta.FindStatusCondition(actualBackup.Status.Conditions, "ProviderSucceeded")
			assert.Equal(t, tt.expectedCondition.Status, actualCondition.Status)
			assert.Equal(t, tt.expectedCondition.Reason, actualCondition.Reason)
			assert.Equal(t, tt.expectedCondition.Message, actualCondition.Message)

			assert.Equal(t, tt.expectedGlobalConfig, actualBackup.Status.GlobalConfig)
			assert.Equal(t, tt.expectedV2Dogus, actualBackup.Status.V2Dogus)
			assert.Equal(t, tt.expectedDogus, actualBackup.Status.Dogus)
			assert.Equal(t, tt.expectedMisc, actualBackup.Status.Misc)
		})
	}
}
