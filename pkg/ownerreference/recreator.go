package ownerreference

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"maps"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
	"sync"
)

const (
	_CloudoguResourceGroup   = "k8s.cloudogu.com"
	_BackupOwnerReferenceKey = "backup-owner-references"
	_BackupUID               = "backup-uid"
)

const workerCount = 5

type resourceWithGroup struct {
	item  unstructured.Unstructured
	group schema.GroupVersionResource
}

type backupResource struct {
	resourceWithGroup
	isParent bool
	isChild  bool
}

type restoreResource struct {
	resourceWithGroup
	ownerRefs map[types.UID]metav1.OwnerReference
}

type Recreator struct {
	namespace          string
	dynamicClient      dynamic.Interface
	discoveryClient    discovery.ServerResourcesInterface
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

	var wg, tasks sync.WaitGroup

	// Channels
	taskChan := make(chan resourceWithGroup, 100)
	resultChan := make(chan backupResource, 100)

	rootGroups := []string{_CloudoguResourceGroup}
	parentKinds, err := r.getKindsOfGroup(ctx, rootGroups)
	if err != nil {
		return fmt.Errorf("unable to get kinds of groups %s: %w", rootGroups, err)
	}

	// Pre-fetch all resources in a namespace
	parents, children, err := r.fetchBackupResources(ctx, parentKinds)
	if err != nil {
		return fmt.Errorf("unable to fetch resources for kinds %s: %w", parentKinds, err)
	}

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(children, taskChan, resultChan, &wg, &tasks)
	}

	// Add root parents to task queue
	for _, parent := range parents {
		tasks.Add(1)
		taskChan <- parent
	}

	// Close taskChan when all tasks are processed
	go func() {
		tasks.Wait()
		close(taskChan)
		wg.Wait()
		close(resultChan)
	}()

	backupResourceMap := make(map[types.UID]resourceWithGroup)

	// Process results
	for result := range resultChan {
		_, ok := backupResourceMap[result.item.GetUID()]
		if !ok {
			backupResourceMap[result.item.GetUID()] = result.resourceWithGroup
		}

		if result.isParent {
			backupResourceMap[result.item.GetUID()] = backupUidForParent(backupResourceMap[result.item.GetUID()])
		}

		if result.isChild {
			update, uErr := backupOwnerRefForResource(backupResourceMap[result.item.GetUID()])
			if uErr != nil {
				return fmt.Errorf("could not backup owner references for child resource %s: %w", result.item.GetName(), uErr)
			}

			backupResourceMap[result.item.GetUID()] = update
		}
	}

	if uErr := r.updateResources(ctx, slices.Collect(maps.Values(backupResourceMap))); uErr != nil {
		return fmt.Errorf("unable to update resources: %w", uErr)
	}

	logger.Info("Backup owner references completed")

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

		kinds = append(kinds, kind)
	}

	logger.Info("Extracted kinds from groups", "kinds", kinds, "groups", grps)

	return kinds, nil
}

func (r Recreator) fetchBackupResources(ctx context.Context, parentKinds []string) ([]resourceWithGroup, map[types.UID][]resourceWithGroup, error) {
	logger := log.FromContext(ctx)

	apiResourceList, err := r.discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list preferred namespaced resource groups: %w", err)
	}

	parentResources := make([]resourceWithGroup, 0, len(apiResourceList))
	childResourceMap := make(map[types.UID][]resourceWithGroup)

	for _, apiResourceGroup := range apiResourceList {
		for _, apiResource := range apiResourceGroup.APIResources {
			gv, pErr := r.groupVersionParser(apiResourceGroup.GroupVersion)
			if pErr != nil {
				return nil, nil, fmt.Errorf("failed to parse group version %s: %w", apiResourceGroup.GroupVersion, pErr)
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

			if res == nil {
				continue
			}

			if slices.Contains(parentKinds, apiResource.Kind) {
				appendParents(gvr, &parentResources, res)
				continue
			}

			appendChildren(gvr, childResourceMap, res)
		}
	}

	return parentResources, childResourceMap, nil
}

func (r Recreator) updateResources(ctx context.Context, resList []resourceWithGroup) error {
	for _, res := range resList {
		dClient := r.dynamicClient.Resource(res.group).Namespace(r.namespace)

		err := retry.OnConflict(func() error {
			currentRes, err := dClient.Get(ctx, res.item.GetName(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get resource %s, Kind: %s: %w", res.item.GetName(), res.item.GetKind(), err)
			}

			currentRes.SetAnnotations(res.item.GetAnnotations())
			currentRes.SetOwnerReferences(res.item.GetOwnerReferences())

			if _, err = r.dynamicClient.Resource(res.group).Namespace(r.namespace).Update(ctx, currentRes, metav1.UpdateOptions{}); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to update resource: %w", err)
		}
	}

	return nil
}

// worker processes tasks by checking for immediate child resources.
//
// worker iterates parent nodes which are provided by the parent channel. Basically it inspects top level resources like
// dogus, backups, components, etc. but only to the first level of its children in order to save metadata later-on.
// The reconciling mechanism of the lower level children like ingress etc. are reconciled properly and will be overwritten
// even if they will be restored with a previously saved owner reference. One can think of the inspected data structure
// like this:
//
//	dogu --> service    --> ingress (not regarded)
//	     \-> deployment --> replica set (not regarded)
func worker(childMap map[types.UID][]resourceWithGroup, parentChan chan resourceWithGroup, resultChan chan<- backupResource, wg, tasks *sync.WaitGroup) {
	defer wg.Done()

	for p := range parentChan {
		// Check if the UID is parent of a child group
		children, isParent := childMap[p.item.GetUID()]
		if !isParent {
			tasks.Done()
			continue
		}

		for _, child := range children {
			resultChan <- backupResource{
				resourceWithGroup: child,
				isParent:          false,
				isChild:           true,
			}

			// Currently we only investigate direct descendants of the parent, however a child can also be parent for another resource
			/*
				tasks.Add(1)
				parentChan <- child
			*/
		}

		resultChan <- backupResource{
			resourceWithGroup: p,
			isParent:          true,
			isChild:           false,
		}

		tasks.Done()
	}
}

func appendParents(gvr schema.GroupVersionResource, resultList *[]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		*resultList = append(*resultList, resourceWithGroup{
			item:  item,
			group: gvr,
		})
	}
}

func appendChildren(gvr schema.GroupVersionResource, resultMap map[types.UID][]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		for _, ownerRef := range item.GetOwnerReferences() {
			resultMap[ownerRef.UID] = append(resultMap[ownerRef.UID], resourceWithGroup{
				item:  item,
				group: gvr,
			})
		}
	}
}

func backupOwnerRefForResource(res resourceWithGroup) (resourceWithGroup, error) {
	ownerRefs := res.item.GetOwnerReferences()

	if len(ownerRefs) == 0 {
		return res, nil
	}

	jsonBackup, err := json.Marshal(ownerRefs)
	if err != nil {
		return resourceWithGroup{}, fmt.Errorf("failed to marshal ownerRefs for backup: %w", err)
	}

	annotations := res.item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[_BackupOwnerReferenceKey] = string(jsonBackup)
	res.item.SetAnnotations(annotations)

	return res, nil
}

func backupUidForParent(res resourceWithGroup) resourceWithGroup {
	annotations := res.item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[_BackupUID] = string(res.item.GetUID())

	res.item.SetAnnotations(annotations)

	return res
}

func (r Recreator) RestoreOwnerReferences(ctx context.Context) error {
	logger := log.FromContext(ctx)

	rootGroups := []string{_CloudoguResourceGroup}
	parentKinds, err := r.getKindsOfGroup(ctx, rootGroups)
	if err != nil {
		return fmt.Errorf("unable to get kinds of groups %s: %w", rootGroups, err)
	}

	// Pre-fetch all resources in a namespace
	parents, children, err := r.fetchRestoreResources(ctx, parentKinds)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}

	restoreList := make([]resourceWithGroup, 0, len(children))

	for _, child := range children {
		restoreList = append(restoreList, restoreOwnerRefForResource(ctx, child, parents))
	}

	for _, parent := range parents {
		restoreList = append(restoreList, restoreParent(parent))
	}

	if uErr := r.updateResources(ctx, restoreList); uErr != nil {
		return fmt.Errorf("unable to update resources: %w", uErr)
	}

	logger.Info("Restore owner references completed")

	return nil
}

func (r Recreator) fetchRestoreResources(ctx context.Context, parentKinds []string) (map[types.UID]resourceWithGroup, []restoreResource, error) {
	logger := log.FromContext(ctx)

	apiResourceList, err := r.discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list preferred namespaced resource groups: %w", err)
	}

	childResourceList := make([]restoreResource, 0, len(apiResourceList))
	parentResourcesMap := make(map[types.UID]resourceWithGroup)

	for _, apiResourceGroup := range apiResourceList {
		for _, apiResource := range apiResourceGroup.APIResources {
			gv, pErr := r.groupVersionParser(apiResourceGroup.GroupVersion)
			if pErr != nil {
				return nil, nil, fmt.Errorf("failed to parse group version %s: %w", apiResourceGroup.GroupVersion, pErr)
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

			if res == nil {
				continue
			}

			if slices.Contains(parentKinds, apiResource.Kind) {
				appendRestoreParents(gvr, parentResourcesMap, res)
				continue
			}

			err = appendRestoreChildren(gvr, &childResourceList, res)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch children for group %s: %w", apiResourceGroup.GroupVersion, err)
			}
		}
	}

	return parentResourcesMap, childResourceList, nil
}

func appendRestoreChildren(gvr schema.GroupVersionResource, resultList *[]restoreResource, items *unstructured.UnstructuredList) error {
	for _, item := range items.Items {
		annotations := item.GetAnnotations()
		if len(annotations) == 0 {
			continue
		}

		oRefs, ok := annotations[_BackupOwnerReferenceKey]
		if !ok {
			continue
		}

		var restoredOwnerRefs []metav1.OwnerReference

		if err := json.Unmarshal([]byte(oRefs), &restoredOwnerRefs); err != nil {
			return fmt.Errorf("failed to unmarshal ownerRefs from backup: %w", err)
		}

		ownerRefMap := make(map[types.UID]metav1.OwnerReference)

		for _, ownerRef := range restoredOwnerRefs {
			ownerRefMap[ownerRef.UID] = ownerRef
		}

		*resultList = append(*resultList, restoreResource{
			resourceWithGroup: resourceWithGroup{
				item:  item,
				group: gvr,
			},
			ownerRefs: ownerRefMap,
		})
	}

	return nil
}

func appendRestoreParents(gvr schema.GroupVersionResource, parentMap map[types.UID]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		annotations := item.GetAnnotations()
		if len(annotations) == 0 {
			continue
		}

		restoredUID, ok := annotations[_BackupUID]
		if !ok {
			continue
		}

		parentMap[types.UID(restoredUID)] = resourceWithGroup{
			item:  item,
			group: gvr,
		}
	}
}

func restoreOwnerRefForResource(ctx context.Context, res restoreResource, parents map[types.UID]resourceWithGroup) resourceWithGroup {
	logger := log.FromContext(ctx)

	for _, ownerRef := range res.ownerRefs {
		parent, ok := parents[ownerRef.UID]
		if !ok {
			logger.Info("Could not find parent for owner reference", "parentUID", ownerRef.UID, "resource", res.item.GetName(), "kind", res.item.GetKind())
			continue
		}

		if res.item.GetKind() != "ReplicaSet" {
			ownerRef.UID = parent.item.GetUID()
			res.item.SetOwnerReferences(append(res.item.GetOwnerReferences(), ownerRef))
		}

		annotations := res.item.GetAnnotations()
		delete(annotations, _BackupOwnerReferenceKey)
		res.item.SetAnnotations(annotations)
	}

	return res.resourceWithGroup
}

func restoreParent(res resourceWithGroup) resourceWithGroup {
	annotations := res.item.GetAnnotations()
	delete(annotations, _BackupUID)
	res.item.SetAnnotations(annotations)

	return res
}
