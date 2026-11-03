package state

import (
	"encoding/json/v2"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

type BackupState struct {
	BlueprintId      string
	Units            []backupv1.UnitIdentifier
	InstalledDogusV2 []backupv1.DoguVersion
	InstalledDogusV3 []backupv1.DoguVersion
	Components       []backupv1.ComponentStatus
	Fqdn             string
	Domain           string
}

func (s BackupState) CommonUnitAnnotations() (map[string]string, error) {
	unitsJson, err := json.Marshal(s.Units)
	if err != nil {
		return nil, fmt.Errorf("marshal backup units: %w", err)
	}

	v2DogusJson, err := json.Marshal(s.InstalledDogusV2)
	if err != nil {
		return nil, fmt.Errorf("marshal installed dogus v2: %w", err)
	}

	v3DogusJson, err := json.Marshal(s.InstalledDogusV3)
	if err != nil {
		return nil, fmt.Errorf("marshal installed dogus v3: %w", err)
	}

	annotation := map[string]string{
		backupv1.UnitsAnnotationKey:            string(unitsJson),
		backupv1.InstalledDogusV3AnnotationKey: string(v3DogusJson),
	}
	if s.BlueprintId != "" {
		annotation[backupv1.BlueprintIDAnnotationKey] = s.BlueprintId
	}
	if len(s.InstalledDogusV2) > 0 {
		annotation[backupv1.InstalledDogusV2AnnotationKey] = string(v2DogusJson)
	}
	return annotation, nil
}

func (s BackupState) GlobalConfigUnitAnnotations() map[string]string {
	return map[string]string{
		backupv1.FQDNAnnotationKey:   s.Fqdn,
		backupv1.DomainAnnotationKey: s.Domain,
	}
}

func (s BackupState) MiscUnitAnnotations() (map[string]string, error) {
	componentsJson, err := json.Marshal(s.Components)
	if err != nil {
		return nil, fmt.Errorf("marshal components: %w", err)
	}

	return map[string]string{
		backupv1.ComponentsAnnotationKey: string(componentsJson),
	}, nil
}
