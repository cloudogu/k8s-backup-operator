package ownerreference

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"
	"testing"
)

var _ dynamic.Interface = &DynamicClientStub{}

func ListEmptyMock(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, nil
}

func listError(_ context.Context, _ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, assert.AnError
}

func (d *DynamicClientStub) createTestData(filePath string) func(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	testFile := filepath.Join("testdata", filePath)

	data, err := os.ReadFile(testFile)
	if err != nil {
		d.t.Fatalf("failed to read testdata file: %v", err)
	}

	var resourceList unstructured.UnstructuredList

	if rErr := json.Unmarshal(data, &resourceList); rErr != nil {
		d.t.Fatalf("failed to unmarshal testdata file: %v", rErr)
	}

	for _, resource := range resourceList.Items {
		d.resources[resource.GetName()] = &resource
	}

	return func(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
		return &resourceList, nil
	}
}

type DynamicClientStub struct {
	t                *testing.T
	testDataBasePath string
	resources        map[string]*unstructured.Unstructured
	listMock         func(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
	updateMock       func(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error)
	getMock          func(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error)
	listCRDErr       bool
}

func (d *DynamicClientStub) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	switch resource.Resource {
	case "customresourcedefinitions":
		if d.listCRDErr {
			d.listMock = listError
		} else {
			d.listMock = d.createTestData("crds.json")
		}
	case "services":
		d.listMock = d.createTestData(filepath.Join(d.testDataBasePath, "Service.json"))
	case "deployments":
		d.listMock = d.createTestData(filepath.Join(d.testDataBasePath, "Deployment.json"))
	case "ingresses":
		d.listMock = d.createTestData(filepath.Join(d.testDataBasePath, "Ingress.json"))
	case "dogus":
		d.listMock = d.createTestData(filepath.Join(d.testDataBasePath, "Dogu.json"))
	default:
		d.listMock = ListEmptyMock
	}

	d.getMock = func(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
		res, ok := d.resources[name]
		if !ok {
			return nil, assert.AnError
		}

		return res, nil
	}

	return d
}

func (d *DynamicClientStub) Namespace(_ string) dynamic.ResourceInterface {
	return d
}

func (d *DynamicClientStub) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return d.listMock(ctx, opts)
}

func (d *DynamicClientStub) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return d.updateMock(ctx, obj, options, subresources...)
}

func (d *DynamicClientStub) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return d.getMock(ctx, name, options, subresources...)
}

func (d *DynamicClientStub) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DynamicClientStub) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	//TODO implement me
	panic("implement me")
}
