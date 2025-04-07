package ownerreference

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"slices"
	"testing"
)

func TestNewRecreator(t *testing.T) {
	t.Run("Create new Recreator", func(t *testing.T) {
		r, err := NewRecreator(&rest.Config{}, "test")
		assert.NoError(t, err)
		assert.NotNil(t, r)
	})

	t.Run("Invalid config", func(t *testing.T) {
		cfg := rest.Config{
			AuthProvider: &clientcmdapi.AuthProviderConfig{},
			ExecProvider: &clientcmdapi.ExecConfig{},
		}

		r, err := NewRecreator(&cfg, "test")
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestRecreator_BackupOwnerReferences(t *testing.T) {

	t.Run("should backup owner references", func(t *testing.T) {

		validateBackup := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			// Deployments and Services should have backup of ownerReference
			if slices.Contains([]string{"Deployment", "Service"}, obj.GetKind()) {
				bRef, ok := obj.GetAnnotations()[annotationBackupOwnerReferenceKey]
				assert.True(t, ok)

				oRef := obj.GetOwnerReferences()
				assert.NotNil(t, oRef)
				assert.True(t, len(oRef) > 0)

				oRefJson, err := json.Marshal(oRef)
				assert.NoError(t, err)

				assert.Equal(t, string(oRefJson), bRef)
			}

			// Dogu should have backup of UID
			if obj.GetKind() == "Dogu" {
				bUID, ok := obj.GetAnnotations()[annotationBackupUIDKey]
				assert.True(t, ok)

				assert.Equal(t, string(obj.GetUID()), bUID)
			}

			if obj.GetKind() == "Ingress" {
				_, ok := obj.GetAnnotations()[annotationBackupOwnerReferenceKey]
				assert.False(t, ok)
			}

			return &unstructured.Unstructured{}, nil
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "backup",
			updateMock:       validateBackup,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.BackupOwnerReferences(context.TODO())
		assert.NoError(t, err)
	})

	t.Run("Error - getKindsOfGroup", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "backup",
			updateMock:       failUpdate,
			listCRDErr:       true,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.BackupOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - ServerPreferredNamespacedResources", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "backup",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{serverPreferredNamespacedResourcesErr: true},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.BackupOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - groupVersionParser", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "backup",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:       "test",
			dynamicClient:   dynamicClientStub,
			discoveryClient: ServerResourcesStub{},
			groupVersionParser: func(gv string) (schema.GroupVersion, error) {
				return schema.GroupVersion{}, assert.AnError
			},
		}

		err := recreator.BackupOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - updateResources", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "backup",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.BackupOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

}

func TestRecreator_RestoreOwnerReferences(t *testing.T) {
	t.Run("should restore owner references", func(t *testing.T) {

		validateRestore := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			// Deployments and Services should have backup of ownerReference
			if slices.Contains([]string{"Deployment", "Service"}, obj.GetKind()) {
				_, ok := obj.GetAnnotations()[annotationBackupOwnerReferenceKey]
				assert.False(t, ok)

				oRef := obj.GetOwnerReferences()
				assert.NotNil(t, oRef)
				assert.True(t, len(oRef) > 0)
			}

			// Dogu should have backup of UID
			if obj.GetKind() == "Dogu" {
				_, ok := obj.GetAnnotations()[annotationBackupUIDKey]
				assert.False(t, ok)
			}

			if obj.GetKind() == "Ingress" {
				_, ok := obj.GetAnnotations()[annotationBackupOwnerReferenceKey]
				assert.False(t, ok)
			}

			return &unstructured.Unstructured{}, nil
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "restore",
			updateMock:       validateRestore,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.RestoreOwnerReferences(context.TODO())
		assert.NoError(t, err)
	})

	t.Run("Error - getKindsOfGroup", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "restore",
			updateMock:       failUpdate,
			listCRDErr:       true,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.RestoreOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - ServerPreferredNamespacedResources", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "restore",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{serverPreferredNamespacedResourcesErr: true},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.RestoreOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - groupVersionParser", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			assert.Fail(t, "update resource should not be called")

			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "restore",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:       "test",
			dynamicClient:   dynamicClientStub,
			discoveryClient: ServerResourcesStub{},
			groupVersionParser: func(gv string) (schema.GroupVersion, error) {
				return schema.GroupVersion{}, assert.AnError
			},
		}

		err := recreator.RestoreOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Error - updateResources", func(t *testing.T) {

		failUpdate := func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
			return nil, assert.AnError
		}

		dynamicClientStub := &DynamicClientStub{
			t:                t,
			testDataBasePath: "restore",
			updateMock:       failUpdate,
		}

		recreator := &Recreator{
			namespace:          "test",
			dynamicClient:      dynamicClientStub,
			discoveryClient:    ServerResourcesStub{},
			groupVersionParser: schema.ParseGroupVersion,
		}

		err := recreator.RestoreOwnerReferences(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
	})
}
