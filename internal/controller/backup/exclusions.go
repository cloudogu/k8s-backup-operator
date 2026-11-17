package backup

import (
	"context"
	"fmt"
	"slices"
	"strings"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

// invalidExclusions describes the excluded dogus and components of a backup which do not match the cluster.
type invalidExclusions struct {
	// missingDogus are excluded dogus which are not installed as v3 dogus.
	missingDogus []string
	// v2Dogus are excluded dogus which are v2 dogus. They can only be excluded as a whole.
	v2Dogus []string
	// missingComponents are excluded components which are not installed.
	missingComponents []string
}

func (i invalidExclusions) isEmpty() bool {
	return len(i.missingDogus) == 0 && len(i.v2Dogus) == 0 && len(i.missingComponents) == 0
}

func (i invalidExclusions) message() string {
	var parts []string
	if len(i.missingDogus) > 0 {
		parts = append(parts, fmt.Sprintf("The excluded dogus %v are not installed as v3 dogus.", i.missingDogus))
	}
	if len(i.v2Dogus) > 0 {
		parts = append(parts, fmt.Sprintf("The excluded dogus %v are v2 dogus, which can only be excluded as a whole with options.ignoreV2Dogus.", i.v2Dogus))
	}
	if len(i.missingComponents) > 0 {
		parts = append(parts, fmt.Sprintf("The excluded components %v are not installed.", i.missingComponents))
	}
	if len(parts) == 0 {
		return ""
	}
	parts = append(parts, "Create a new backup with valid exclusions.")
	return strings.Join(parts, " ")
}

// findInvalidExclusions compares the excluded dogus and components of the backup with the installed ones.
func findInvalidExclusions(spec backupv1.BackupSpec, dogus []doguv3.Dogu, components []componentv1.Component) invalidExclusions {
	doguApiVersions := make(map[string]doguv3.DoguApiVersion, len(dogus))
	for _, dogu := range dogus {
		doguApiVersions[dogu.Spec.Name] = dogu.Spec.DoguApiVersion
	}

	var result invalidExclusions
	for _, excludedDogu := range spec.ExcludedDogus {
		apiVersion, exists := doguApiVersions[excludedDogu]
		switch {
		case exists && apiVersion == doguv3.DoguApiVersionV3:
			continue
		case exists && apiVersion == doguv3.DoguApiVersionV2:
			result.v2Dogus = append(result.v2Dogus, excludedDogu)
		default:
			result.missingDogus = append(result.missingDogus, excludedDogu)
		}
	}

	for _, excludedComponent := range spec.ExcludedComponents {
		installed := slices.ContainsFunc(components, func(component componentv1.Component) bool {
			return component.Spec.Name == excludedComponent
		})
		if !installed {
			result.missingComponents = append(result.missingComponents, excludedComponent)
		}
	}

	return result
}

// getInvalidExclusions lists the installed dogus and components and returns the exclusions of the backup which do not
// match them. The cluster is only queried if the backup has exclusions.
func (c *defaultReconciler) getInvalidExclusions(ctx context.Context, backup *backupv1.Backup) (invalidExclusions, error) {
	var dogus []doguv3.Dogu
	if len(backup.Spec.ExcludedDogus) > 0 {
		var err error
		if dogus, err = c.stateCollector.ListDogus(ctx, backup.Namespace); err != nil {
			return invalidExclusions{}, fmt.Errorf("validate excluded dogus: %w", err)
		}
	}

	var components []componentv1.Component
	if len(backup.Spec.ExcludedComponents) > 0 {
		var err error
		if components, err = c.stateCollector.ListComponents(ctx, backup.Namespace); err != nil {
			return invalidExclusions{}, fmt.Errorf("validate excluded components: %w", err)
		}
	}

	return findInvalidExclusions(backup.Spec, dogus, components), nil
}
