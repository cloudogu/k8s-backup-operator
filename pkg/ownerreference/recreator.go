package ownerreference

import (
	"context"
	"encoding/json"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
)

const (
	_CloudoguResourceGroup   = "k8s.cloudogu.com"
	_BackupOwnerReferenceKey = "backup-owner-references"
	_BackupUID               = "backup-uid"
)

type resourceWithGroup struct {
	item  unstructured.Unstructured
	group schema.GroupVersionResource
}

type Recreator struct {
	namespace          string
	dynamicClient      dynamic.Interface
	discoveryClient    discovery.DiscoveryInterface
	groupVersionParser func(gv string) (schema.GroupVersion, error)
}

func NewRecreator(cfg *rest.Config, namespace string) (*Recreator, error) {
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client: %w", err)
	}

	return &Recreator{
		namespace:          namespace,
		dynamicClient:      dynamicClient,
		discoveryClient:    discoveryClient,
		groupVersionParser: schema.ParseGroupVersion,
	}, nil
}

func (r Recreator) BackupOwnerReferences(ctx context.Context) error {
	logger := log.FromContext(ctx)

	kindGrps := []string{_CloudoguResourceGroup}
	kinds, err := r.getKindsOfGroup(ctx, kindGrps)
	if err != nil {
		return fmt.Errorf("unable to get kinds of groups %s: %w", kindGrps, err)
	}

	resources, err := r.listAllResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resource groups: %w", err)
	}

	parentUIDs := make([]types.UID, 0)
	for _, res := range resources {
		pUIDs, bErr := r.backupOwnerRefForResource(ctx, res, kinds)
		if bErr != nil {
			return fmt.Errorf("failed to backup owner reference for resource %s: %w", res.item.GetName(), bErr)
		}

		parentUIDs = append(parentUIDs, pUIDs...)
	}

	backupUIDSet := make(map[types.UID]bool)
	for _, res := range resources {
		if backupUIDSet[res.item.GetUID()] {
			continue
		}

		if !slices.Contains(parentUIDs, res.item.GetUID()) {
			continue
		}

		if uErr := r.backupUidForParent(ctx, res); uErr != nil {
			return fmt.Errorf("failed to backup uid for resource %s: %w", res.item.GetName(), uErr)
		}

		backupUIDSet[res.item.GetUID()] = true
	}

	logger.Info("Prepared resources with owner references for backup")

	return nil
}

func (r Recreator) getKindsOfGroup(ctx context.Context, grps []string) ([]string, error) {
	logger := log.FromContext(ctx)
	crdGVR := apiextensionsv1.SchemeGroupVersion.WithResource("customresourcedefinitions")
	crds, err := r.dynamicClient.Resource(crdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	kinds := make([]string, 0, len(crds.Items))

	for _, crd := range crds.Items {
		spec, found, lErr := unstructured.NestedMap(crd.Object, "spec")
		if lErr != nil || !found {
			logger.Info("Could not get spec for custom resource", "crd", crd.GetName())
			continue
		}

		group, ok := spec["group"].(string)
		if !ok {
			logger.Info("Could not get group for custom resource", "crd", crd.GetName())
			continue
		}

		if !slices.Contains(grps, group) {
			// Ignore CRDs outside the target group
			continue
		}

		names, ok := spec["names"].(map[string]any)
		if !ok {
			logger.Info("Could not get names for custom resource", "crd", crd.GetName())
			continue
		}

		kind, ok := names["kind"].(string)
		if !ok {
			logger.Info("Could not get kind for custom resource", "crd", crd.GetName())
		}

		logger.Info("found kind for group", "kind", kind, "group", group)

		kinds = append(kinds, kind)
	}

	logger.Info("Extracted kinds from group", "kinds", kinds, "groups", grps)

	return kinds, nil
}

func (r Recreator) listAllResources(ctx context.Context) ([]resourceWithGroup, error) {
	logger := log.FromContext(ctx)

	apiResourceList, err := r.discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, fmt.Errorf("failed to list preferred namespaced resource groups: %w", err)
	}

	resourceList := make([]resourceWithGroup, 0, len(apiResourceList))

	for _, apiResourceGroup := range apiResourceList {
		for _, apiResource := range apiResourceGroup.APIResources {
			gv, pErr := r.groupVersionParser(apiResourceGroup.GroupVersion)
			if pErr != nil {
				return nil, fmt.Errorf("failed to parse group version %s: %w", apiResourceGroup.GroupVersion, pErr)
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			res, lErr := r.dynamicClient.
				Resource(gvr).
				Namespace(r.namespace).
				List(ctx, metav1.ListOptions{})

			if lErr != nil {
				logger.Info("failed to list resources for group", "resource", apiResource.Name, "group", apiResource.Group)
				continue
			}

			for _, item := range res.Items {
				resourceList = append(resourceList, resourceWithGroup{
					item:  item,
					group: gvr,
				})
			}
		}
	}

	return resourceList, nil
}

func (r Recreator) backupOwnerRefForResource(ctx context.Context, res resourceWithGroup, backupKinds []string) ([]types.UID, error) {
	logger := log.FromContext(ctx)

	backupParents := make([]types.UID, 0)

	oRefs := res.item.GetOwnerReferences()
	if len(oRefs) == 0 {
		return backupParents, nil
	}

	logger.Info("Found owner references for resource", "length", len(oRefs), "ownerReferences", oRefs)

	backupORefs := make([]metav1.OwnerReference, 0, len(oRefs))
	for _, oRef := range oRefs {
		if slices.Contains(backupKinds, oRef.Kind) {
			backupORefs = append(backupORefs, oRef)
			backupParents = append(backupParents, oRef.UID)
		}
	}

	if len(backupORefs) == 0 {
		return backupParents, nil
	}

	jsonBackup, err := json.Marshal(backupORefs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ownerRefs for backup: %w", err)
	}

	annotations := res.item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[_BackupOwnerReferenceKey] = string(jsonBackup)
	res.item.SetAnnotations(annotations)

	if _, lErr := r.dynamicClient.Resource(res.group).Namespace(r.namespace).Update(ctx, &res.item, metav1.UpdateOptions{}); lErr != nil {
		return nil, fmt.Errorf("failed to update resource: %w", lErr)
	}

	return backupParents, nil
}

func (r Recreator) backupUidForParent(ctx context.Context, res resourceWithGroup) error {
	c := r.dynamicClient.Resource(res.group).Namespace(r.namespace)

	annotations := res.item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[_BackupUID] = string(res.item.GetUID())

	res.item.SetAnnotations(annotations)

	if _, err := c.Update(ctx, &res.item, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	return nil
}
