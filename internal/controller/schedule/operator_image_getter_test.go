package schedule

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-backup-operator/internal/config"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const operatorImageGetterTestNamespace = "test-namespace"

var operatorImageGetterTestCtx = context.TODO()

func TestOperatorImageGetter_ImageForKey(t *testing.T) {
	t.Run("should fail on non-existing configmap", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		// given
		sut := NewOperatorImageGetter(fakeClient, operatorImageGetterTestNamespace)

		// when
		_, err := sut.ImageForKey(operatorImageGetterTestCtx, config.OperatorImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while getting configmap 'k8s-backup-operator-additional-images':")
	})
	t.Run("should fail on missing configmap key", func(t *testing.T) {
		// given
		invalidCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: operatorImageGetterTestNamespace,
			},
		}
		fakeClient := fake.NewSimpleClientset(invalidCM)
		sut := NewOperatorImageGetter(fakeClient, operatorImageGetterTestNamespace)

		// when
		_, err := sut.ImageForKey(operatorImageGetterTestCtx, config.OperatorImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "image \"operatorImage\" in configmap \"k8s-backup-operator-additional-images\" is empty")
	})
	t.Run("should fail on invalid image tag", func(t *testing.T) {
		// given
		invalidCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: operatorImageGetterTestNamespace,
			},
			Data: map[string]string{config.OperatorImageConfigmapNameKey: "busybox:::::123"},
		}
		fakeClient := fake.NewSimpleClientset(invalidCM)
		sut := NewOperatorImageGetter(fakeClient, operatorImageGetterTestNamespace)

		// when
		_, err := sut.ImageForKey(operatorImageGetterTestCtx, config.OperatorImageConfigmapNameKey)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "configmap 'k8s-backup-operator-additional-images' contains an invalid image tag: image tag 'busybox:::::123' seems invalid")
	})
	t.Run("should succeed on valid configmap", func(t *testing.T) {
		// given
		validCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OperatorAdditionalImagesConfigmapName,
				Namespace: operatorImageGetterTestNamespace,
			},
			Data: map[string]string{config.OperatorImageConfigmapNameKey: "kubectl:123"},
		}
		fakeClient := fake.NewSimpleClientset(validCM)
		sut := NewOperatorImageGetter(fakeClient, operatorImageGetterTestNamespace)

		// when
		actual, err := sut.ImageForKey(operatorImageGetterTestCtx, config.OperatorImageConfigmapNameKey)

		// then
		require.NoError(t, err)
		assert.Equal(t, "kubectl:123", actual)
	})
}

func TestVerifyImageTag(t *testing.T) {
	tests := []struct {
		name     string
		imageTag string
		wantErr  assert.ErrorAssertionFunc
	}{
		{"valid simple w/o tag", "repo/image", assert.NoError},
		{"valid simple with tag", "repo/image:latest", assert.NoError},
		{"valid simple with version", "repo/image:v1.2.3", assert.NoError},

		{"invalid ending separator", "repo/image_", assert.Error},
		{"invalid uppercase", "repo/Image", assert.Error},
		{"invalid hostname length", "superlongtagomgwhatisgoingonherethistagiswaylongerthaniexpectedbutweallknowthatatagmayconsistofupto128charachtersohwatchherewegox:8080/repo/image:v1.2.3", assert.Error},
		{"invalid tag length", "repo/image:superlongtagomgwhatisgoingonherethistagiswaylongerthaniexpectedbutweallknowthatatagmayconsistofupto128charachtersohwatchherewegox", assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, verifyImageTag(tt.imageTag), fmt.Sprintf("verifyImageTag(%v)", tt.imageTag))
		})
	}
}
