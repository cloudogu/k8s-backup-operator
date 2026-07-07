package backup

import (
	"testing"

	annotationsPkg "github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/cloudogu/k8s-backup-operator/pkg/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	t.Run("init backup with label", func(t *testing.T) {
		t.Skip("TODO")
		backupCr := createBackup("ns1", "name1")
		service := &ServiceImpl{}

		service.initBackupCr(backupCr, nil)

		labels := backupCr.GetLabels()
		assert.Contains(t, labels, "k8s.cloudogu.com/part-of")
		assert.Equal(t, labels["k8s.cloudogu.com/part-of"], "backup")

		assert.Contains(t, labels, appLabelKey)
		assert.Equal(t, labels["app"], "ces")
	})

	t.Run("init backup with blueprint annotations", func(t *testing.T) {
		t.Skip("TODO")
		backupCr := createBackup("ns1", "name1")
		service := &ServiceImpl{}

		blueprintWithDogus := &blueprint.BlueprintWithDogus{
			DisplayName: "MyBlueprint",
			DogusAsJson: "{dogus:[]}",
		}

		service.initBackupCr(backupCr, blueprintWithDogus)

		annotations := backupCr.GetAnnotations()

		assert.Contains(t, annotations, annotationsPkg.BlueprintIdAnnotation)
		assert.Equal(t, annotations[annotationsPkg.BlueprintIdAnnotation], blueprintWithDogus.DisplayName)

		assert.Contains(t, annotations, annotationsPkg.DogusAnnotation)
		assert.Equal(t, annotations[annotationsPkg.DogusAnnotation], blueprintWithDogus.DogusAsJson)
	})

}
