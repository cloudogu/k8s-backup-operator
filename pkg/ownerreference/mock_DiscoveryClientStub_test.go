package ownerreference

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"os"
	"path/filepath"
)

var _ discovery.ServerResourcesInterface = &ServerResourcesStub{}

type ServerResourcesStub struct {
	serverPreferredNamespacedResourcesErr bool
}

func (s ServerResourcesStub) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	//TODO implement me
	panic("implement me")
}

func (s ServerResourcesStub) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	//TODO implement me
	panic("implement me")
}

func (s ServerResourcesStub) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	//TODO implement me
	panic("implement me")
}

func (s ServerResourcesStub) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	if s.serverPreferredNamespacedResourcesErr {
		return nil, assert.AnError
	}

	testFile := filepath.Join("testdata", "apiResourceList.json")

	data, err := os.ReadFile(testFile)
	if err != nil {
		return nil, assert.AnError
	}

	var apiResourceList []*metav1.APIResourceList

	if rErr := json.Unmarshal(data, &apiResourceList); rErr != nil {
		return nil, assert.AnError
	}

	return apiResourceList, nil
}
