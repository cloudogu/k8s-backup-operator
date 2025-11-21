package annotations

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BlueprintIdAnnotation = "backup.cloudogu.com/blueprintId"
	DogusAnnotation       = "backup.cloudogu.com/dogus"
)

func GetBackupAnnotations(objectMeta metav1.ObjectMeta) map[string]string {
	result := make(map[string]string)
	for key, val := range objectMeta.Annotations {
		if key == BlueprintIdAnnotation || key == DogusAnnotation {
			result[key] = val
		}
	}

	return result
}
