package state

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

// RestoreStateCollector collects the restore state from the provider backups.
type RestoreStateCollector struct {
	k8sClient client.Client
}

func NewCollector(k8sClient client.Client) *RestoreStateCollector {
	return &RestoreStateCollector{k8sClient: k8sClient}
}

// Collect returns the restore state of the legacy backup or the backup units.
func (c *RestoreStateCollector) Collect(ctx context.Context, restore *backupv1.Restore) (RestoreState, error) {
	legacyBackup, err := c.getLegacyBackup(ctx, restore)
	if err != nil {
		return RestoreState{}, fmt.Errorf("get legacy backup: %w", err)
	}
	if legacyBackup != nil {
		return RestoreState{
			Kind:  KindLegacy,
			Units: []UnitBackup{{Type: UnitTypeLegacy, ProviderBackup: toProviderBackup(legacyBackup)}},
		}, nil
	}

	unitBackups := velerov1.BackupList{}
	err = c.k8sClient.List(ctx, &unitBackups, client.InNamespace(restore.Namespace),
		client.MatchingLabels{backupv1.BackupLabelKey: backupv1.GetTruncatedName(restore.Spec.BackupName)})
	if err != nil {
		return RestoreState{}, fmt.Errorf("list unit backups: %w", err)
	}
	if len(unitBackups.Items) == 0 {
		return RestoreState{Kind: KindNotFound}, nil
	}

	result, err := collectUnitState(unitBackups.Items)
	if err != nil {
		return RestoreState{}, &MalformedBackupError{Err: fmt.Errorf("collect units: %w", err)}
	}
	return result, nil
}

// MalformedBackupError marks unit backups whose annotations cannot be read; retrying does not help.
type MalformedBackupError struct {
	Err error
}

func (e *MalformedBackupError) Error() string { return e.Err.Error() }

func (e *MalformedBackupError) Unwrap() error { return e.Err }

func collectUnitState(unitBackups []velerov1.Backup) (RestoreState, error) {
	// every unit backup carries the common annotations
	commonAnnotations := unitBackups[0].Annotations
	if _, ok := commonAnnotations[backupv1.UnitsAnnotationKey]; !ok {
		return RestoreState{}, fmt.Errorf("provider backup %s has no annotation %s", unitBackups[0].Name, backupv1.UnitsAnnotationKey)
	}
	unitIDs, err := unmarshalAnnotation[[]backupv1.UnitIdentifier](commonAnnotations, backupv1.UnitsAnnotationKey)
	if err != nil {
		return RestoreState{}, fmt.Errorf("get units: %w", err)
	}
	v2Dogus, err := unmarshalAnnotation[[]backupv1.DoguVersion](commonAnnotations, backupv1.InstalledDogusV2AnnotationKey)
	if err != nil {
		return RestoreState{}, fmt.Errorf("get installed V2 dogus: %w", err)
	}
	v3Dogus, err := unmarshalAnnotation[[]backupv1.DoguVersion](commonAnnotations, backupv1.InstalledDogusV3AnnotationKey)
	if err != nil {
		return RestoreState{}, fmt.Errorf("get V3 dogus: %w", err)
	}

	result := RestoreState{Kind: KindUnits}
	slices.SortFunc(unitIDs, compareUnitsByTypeThenName)
	for _, unitID := range unitIDs {
		result.Units = append(result.Units, UnitBackup{UnitIdentifier: unitID, ProviderBackup: findUnitBackup(unitBackups, unitID)})
	}

	for _, dogu := range v2Dogus {
		result.V2Dogus = append(result.V2Dogus, dogu.Name)
	}

	for _, dogu := range v3Dogus {
		result.V3Dogus = append(result.V3Dogus, dogu.Name)
	}

	result.Components, err = componentNames(unitBackups)
	if err != nil {
		return RestoreState{}, fmt.Errorf("get components: %w", err)
	}

	return result, nil
}

// componentNames returns the components of the first unit backup that carries the components annotation.
func componentNames(unitBackups []velerov1.Backup) ([]string, error) {
	index := slices.IndexFunc(unitBackups, func(unitBackup velerov1.Backup) bool {
		_, ok := unitBackup.Annotations[backupv1.ComponentsAnnotationKey]
		return ok
	})
	if index < 0 {
		return nil, nil
	}
	components, err := unmarshalAnnotation[[]backupv1.ComponentVersion](unitBackups[index].Annotations, backupv1.ComponentsAnnotationKey)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, component := range components {
		names = append(names, component.Name)
	}
	return names, nil
}

// unmarshalAnnotation returns the zero value if the annotation is absent.
func unmarshalAnnotation[T any](annotations map[string]string, key string) (T, error) {
	var value T
	raw, ok := annotations[key]
	if !ok {
		return value, nil
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return value, fmt.Errorf("unmarshal annotation %s: %w", key, err)
	}
	return value, nil
}

var unitTypeOrder = []backupv1.UnitType{
	backupv1.UnitTypeGlobalConfig,
	backupv1.UnitTypeComponents,
	backupv1.UnitTypeDoguV2,
	backupv1.UnitTypeDogu,
}

func compareUnitsByTypeThenName(a, b backupv1.UnitIdentifier) int {
	return cmp.Or(
		cmp.Compare(slices.Index(unitTypeOrder, a.Type), slices.Index(unitTypeOrder, b.Type)),
		cmp.Compare(a.Name, b.Name),
	)
}

// findUnitBackup returns nil if the unit's provider backup is missing.
func findUnitBackup(unitBackups []velerov1.Backup, unit backupv1.UnitIdentifier) *ProviderBackup {
	for _, backup := range unitBackups {
		if backup.Labels[backupv1.BackupTypeLabelKey] == string(unit.Type) && backup.Labels[backupv1.DoguNameLabelKey] == unit.Name {
			return toProviderBackup(&backup)
		}
	}
	return nil
}

func toProviderBackup(backup *velerov1.Backup) *ProviderBackup {
	return &ProviderBackup{Name: backup.Name, Phase: backup.Status.Phase}
}

// getLegacyBackup returns the provider backup named like the backup if it is no unit backup, else nil.
func (c *RestoreStateCollector) getLegacyBackup(ctx context.Context, restore *backupv1.Restore) (*velerov1.Backup, error) {
	backup, err := velero.GetBackup(ctx, c.k8sClient, restore.Namespace, restore.Spec.BackupName)
	if err != nil || backup == nil {
		return nil, err
	}

	if _, isUnit := backup.Labels[backupv1.BackupLabelKey]; isUnit {
		return nil, nil
	}
	return backup, nil
}
