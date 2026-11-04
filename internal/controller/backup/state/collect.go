package state

import (
	"context"
	"fmt"

	"github.com/cloudogu/ces-commons-lib/errors"
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"go.yaml.in/yaml/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const configKeyName = "config.yaml"
const fqdnKeyName = "fqdn"
const domainKeyName = "domain"

const backupScopeLabelKey = "k8s.cloudogu.com/backup-scope"

type BackupStateCollector struct {
	k8sClient client.Client
}

func NewCollector(k8sClient client.Client) *BackupStateCollector {
	return &BackupStateCollector{
		k8sClient: k8sClient,
	}
}

func (b *BackupStateCollector) Collect(ctx context.Context, backup *backupv1.Backup) (BackupState, error) {
	blueprintId, err := b.getBlueprintId(ctx, backup.Namespace)
	if err != nil {
		return BackupState{}, fmt.Errorf("get blueprint id: %w", err)
	}

	v2Dogus, v3Dogus, err := b.getDogus(ctx, backup)
	if err != nil {
		return BackupState{}, fmt.Errorf("get v2 and v3 dogus: %w", err)
	}

	components, err := b.getComponents(ctx, backup)
	if err != nil {
		return BackupState{}, fmt.Errorf("get components: %w", err)
	}

	fqdn, domain, err := b.getGlobalConfigInfo(ctx, backup)
	if err != nil {
		return BackupState{}, fmt.Errorf("get fqdn and domain: %w", err)
	}

	return BackupState{
		Name:             backup.Name,
		Coverage:         b.getCoverage(backup),
		BlueprintId:      blueprintId,
		Units:            b.getUnits(backup, v2Dogus, v3Dogus),
		InstalledDogusV2: v2Dogus,
		InstalledDogusV3: v3Dogus,
		Components:       components,
		Fqdn:             fqdn,
		Domain:           domain,
	}, nil
}

func (b *BackupStateCollector) getBlueprintId(ctx context.Context, namespace string) (string, error) {
	var blueprintList = blueprintv3.BlueprintList{}
	err := b.k8sClient.List(ctx, &blueprintList, client.InNamespace(namespace))
	// no blueprint or no blueprint crd
	if err != nil {
		if !errors.IsNotFoundError(err) && !meta.IsNoMatchError(err) {
			return "", fmt.Errorf("list blueprints: %w", err)
		}
		blueprintList.Items = nil
	}

	// only take first blueprint
	if len(blueprintList.Items) > 0 {
		blueprint := blueprintList.Items[0]
		return blueprint.Spec.DisplayName, nil
	}

	return "", nil
}

func (b *BackupStateCollector) getDogus(ctx context.Context, backup *backupv1.Backup) (v2DoguVersions, v3DoguVersions []backupv1.DoguVersion, err error) {
	doguList := doguv3.DoguList{}
	err = b.k8sClient.List(ctx, &doguList, client.InNamespace(backup.Namespace))
	if err != nil {
		return nil, nil, fmt.Errorf("list dogus: %w", err)
	}

outer:
	for _, dogu := range doguList.Items {
		if dogu.Spec.DoguApiVersion == doguv3.DoguApiVersionV2 && !backup.Spec.Options.IgnoreV2Dogus {
			v2DoguVersions = append(v2DoguVersions, backupv1.DoguVersion{
				Name:    dogu.Spec.Name,
				Version: dogu.Spec.Version,
			})
			continue
		} else if dogu.Spec.DoguApiVersion != doguv3.DoguApiVersionV3 {
			continue
		}

		for _, excludedDogu := range backup.Spec.ExcludedDogus {
			if dogu.Spec.Name == excludedDogu {
				continue outer
			}
		}

		v3DoguVersions = append(v3DoguVersions, backupv1.DoguVersion{
			Name:    dogu.Spec.Name,
			Version: dogu.Spec.Version,
		})
	}
	return v2DoguVersions, v3DoguVersions, nil
}

func (b *BackupStateCollector) getComponents(ctx context.Context, backup *backupv1.Backup) ([]backupv1.ComponentStatus, error) {
	if backup.Spec.Options.IgnoreMisc {
		return nil, nil
	}

	backupableItems, err := b.collectBackupableComponentItems(ctx, backup)
	if err != nil {
		return nil, fmt.Errorf("collect backupable items: %w", err)
	}

	components, err := b.componentsFromItems(backupableItems, backup)
	if err != nil {
		return nil, fmt.Errorf("get components from items: %w", err)
	}

	return components, nil
}

func (b *BackupStateCollector) collectBackupableComponentItems(ctx context.Context, backup *backupv1.Backup) ([]unstructured.Unstructured, error) {
	configMapList := unstructured.UnstructuredList{}
	configMapList.SetAPIVersion("v1")
	configMapList.SetKind("ConfigMapList")
	err := b.k8sClient.List(ctx, &configMapList, client.InNamespace(backup.Namespace), client.HasLabels{componentv1.ComponentNameLabelKey, backupScopeLabelKey})
	if err != nil {
		return nil, fmt.Errorf("list component config maps: %w", err)
	}

	secretList := unstructured.UnstructuredList{}
	secretList.SetAPIVersion("v1")
	secretList.SetKind("SecretList")
	err = b.k8sClient.List(ctx, &secretList, client.InNamespace(backup.Namespace), client.HasLabels{componentv1.ComponentNameLabelKey, backupScopeLabelKey})
	if err != nil {
		return nil, fmt.Errorf("list component secrets: %w", err)
	}

	pvcList := unstructured.UnstructuredList{}
	pvcList.SetAPIVersion("v1")
	pvcList.SetKind("PersistentVolumeClaimList")
	err = b.k8sClient.List(ctx, &pvcList, client.InNamespace(backup.Namespace), client.HasLabels{componentv1.ComponentNameLabelKey, backupScopeLabelKey})
	if err != nil {
		return nil, fmt.Errorf("list component PVCs: %w", err)
	}

	backupableItems := make([]unstructured.Unstructured, 0, len(configMapList.Items)+len(secretList.Items)+len(pvcList.Items))
	backupableItems = append(backupableItems, configMapList.Items...)
	backupableItems = append(backupableItems, secretList.Items...)
	backupableItems = append(backupableItems, pvcList.Items...)
	return backupableItems, nil
}

func (b *BackupStateCollector) componentsFromItems(backupableItems []unstructured.Unstructured, backup *backupv1.Backup) (components []backupv1.ComponentStatus, err error) {
	componentSet := make(map[string]struct{})
outer:
	for _, item := range backupableItems {
		item.GetLabels()
		componentName, exists := item.GetLabels()[componentv1.ComponentNameLabelKey]
		if !exists {
			return nil, fmt.Errorf("%s %s does not have %s label, despite being selected", item.GetKind(), item.GetName(), componentv1.ComponentNameLabelKey)
		}

		for _, excludedComponent := range backup.Spec.ExcludedComponents {
			if componentName == excludedComponent {
				continue outer
			}
		}

		componentVersion, exists := item.GetLabels()[componentv1.ComponentVersionLabelKey]
		if !exists {
			return nil, fmt.Errorf("%s %s does not have %s label, despite being selected", item.GetKind(), item.GetName(), componentv1.ComponentVersionLabelKey)
		}

		if _, exists := componentSet[componentName]; exists {
			continue
		}
		componentSet[componentName] = struct{}{}
		components = append(components, backupv1.ComponentStatus{Name: componentName, Version: componentVersion})
	}
	return components, nil
}

func (b *BackupStateCollector) getGlobalConfigInfo(ctx context.Context, backup *backupv1.Backup) (string, string, error) {
	if backup.Spec.Options.IgnoreGlobalConfig {
		return "", "", nil
	}

	globalConfig := corev1.ConfigMap{}
	err := b.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: backup.Namespace,
		Name:      "global-config",
	}, &globalConfig)
	if err != nil {
		return "", "", fmt.Errorf("get global config: %w", err)
	}

	configYaml, ok := globalConfig.Data[configKeyName]
	if !ok {
		return "", "", fmt.Errorf("global config does not contain %s", configKeyName)
	}

	var config map[string]any
	err = yaml.Unmarshal([]byte(configYaml), &config)
	if err != nil {
		return "", "", fmt.Errorf("unmarshal global config: %w", err)
	}

	fqdn, err := getStringKey(config, fqdnKeyName)
	if err != nil {
		return "", "", fmt.Errorf("get %s from global config: %w", fqdnKeyName, err)
	}

	domain, err := getStringKey(config, domainKeyName)
	if err != nil {
		return "", "", fmt.Errorf("get %s from global config: %w", domainKeyName, err)
	}

	return fqdn, domain, nil
}

func (b *BackupStateCollector) getUnits(backup *backupv1.Backup, v2Dogus []backupv1.DoguVersion, v3Dogus []backupv1.DoguVersion) (units []backupv1.UnitIdentifier) {
	if len(v2Dogus) > 0 {
		units = append(units, backupv1.UnitIdentifier{
			Type: backupv1.UnitTypeDoguV2,
		})
	}
	for _, dogu := range v3Dogus {
		units = append(units, backupv1.UnitIdentifier{
			Type: backupv1.UnitTypeDogu,
			Name: dogu.Name,
		})
	}
	if !backup.Spec.Options.IgnoreGlobalConfig {
		units = append(units, backupv1.UnitIdentifier{
			Type: backupv1.UnitTypeGlobalConfig,
		})
	}
	if !backup.Spec.Options.IgnoreMisc {
		units = append(units, backupv1.UnitIdentifier{
			Type: backupv1.UnitTypeMisc,
		})
	}
	return units
}

func (b *BackupStateCollector) getCoverage(backup *backupv1.Backup) backupv1.BackupCoverage {
	if len(backup.Spec.ExcludedDogus) == 0 &&
		len(backup.Spec.ExcludedComponents) == 0 &&
		!backup.Spec.Options.IgnoreMisc &&
		!backup.Spec.Options.IgnoreGlobalConfig &&
		!backup.Spec.Options.IgnoreV2Dogus {
		return backupv1.BackupCoverageFull
	}
	return backupv1.BackupCoveragePartial
}

func getStringKey(m map[string]any, key string) (string, error) {
	rawValue, ok := m[key]
	if !ok {
		return "", fmt.Errorf("map does not contain %s", key)
	}

	stringValue, ok := rawValue.(string)
	if !ok {
		return "", fmt.Errorf("map value for %s is not a string", key)
	}

	return stringValue, nil
}
