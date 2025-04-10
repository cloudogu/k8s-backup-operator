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

// cloudoguResourceGroup is the group in which CRDs from Cloudogu are defined
const cloudoguResourceGroup = "k8s.cloudogu.com"

// searchWorkerCount defines the amount of search operations that can run concurrently
const searchWorkerCount = 5

var (
	annotationBackupOwnerReferenceKey = fmt.Sprintf("%s/backup-owner-references", cloudoguResourceGroup)
	annotationBackupUIDKey            = fmt.Sprintf("%s/backup-uid", cloudoguResourceGroup)
)

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

// Recreator is responsible for creating and restoring backups of owner references referred in the metadata
// of a resource.
//
// It leverages dynamic and discovery clients to interact with the Kubernetes API
// in a flexible and version-aware manner, enabling operations across different
// resource types and API groups.
//
// Fields:
//   - namespace: The Kubernetes namespace in which the Recreator operates.
//   - dynamicClient: A dynamic client used to interact with arbitrary Kubernetes resources.
//   - discoveryClient: A discovery interface used to fetch API resource information from the server.
//   - groupVersionParser: A function used to parse and handle GroupVersion strings for resources.
type Recreator struct {
	namespace          string
	dynamicClient      dynamic.Interface
	discoveryClient    discovery.ServerResourcesInterface
	groupVersionParser func(gv string) (schema.GroupVersion, error)
}

// NewRecreator creates a new instance of Recreator using the provided Kubernetes
// REST configuration and namespace. It initializes a dynamic client and a discovery
// client for interacting with the Kubernetes API.
//
// Parameters:
//   - cfg: A pointer to a rest.Config that contains the configuration required
//     to connect to the Kubernetes API.
//   - namespace: The Kubernetes namespace in which the Recreator will operate.
//
// Returns:
//   - A pointer to the initialized Recreator instance.
//   - An error if either the dynamic client or discovery client cannot be created.
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

// BackupOwnerReferences scans all resources in the configured namespace to identify
// which ones are referenced as owners by other resources and backs up these relationships
// into resource annotations.
//
// This method performs the following steps:
//  1. Fetches all Cloudogu CRD-based parent resources and a mapping of child resources
//     referencing them via OwnerReferences.
//  2. Starts a pool of worker goroutines to inspect each parent resource and determine
//     if it's referenced as an owner.
//  3. Collects both parent and child resources that need backup, annotates them with
//     their UID or OwnerReferences
//  4. Updates the annotated resources in the Kubernetes cluster.
//
// The backup enables later restoration of the ownership structure between resources.
func (r Recreator) BackupOwnerReferences(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// wg is a WaitGroup to track the lifecycle of worker goroutines
	var wg sync.WaitGroup
	// tasks is a WaitGroup to track individual parent resource search operations
	var tasks sync.WaitGroup

	// parentChan queues parent resources to be processed by worker goroutines
	// Each parent will be checked for any child resources referencing it via OwnerReferences
	parentChan := make(chan resourceWithGroup, 100)
	// backupChan receives parent and child resources that need their owner reference
	// data backed up into annotations
	backupChan := make(chan backupResource, 100)

	// Pre-fetch all resources in the namespace
	parentList, childMap, err := r.fetchResourcesForBackup(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch resources for backup: %w", err)
	}

	// Start a fixed-size worker pool to search for child resources referencing each parent
	for i := 0; i < searchWorkerCount; i++ {
		wg.Add(1)
		go searchChildrenForParent(childMap, parentChan, backupChan, &wg, &tasks)
	}

	// Enqueue each parent resource to trigger a search for its referencing children
	for _, parent := range parentList {
		tasks.Add(1)         // adds the search operation the ongoing tasks
		parentChan <- parent // start search operation for a parent
	}

	// Wait for all search tasks and worker goroutines to complete before processing results
	go func() {
		tasks.Wait()
		close(parentChan)
		wg.Wait()
		close(backupChan)
	}()

	backupResourceMap := make(map[types.UID]resourceWithGroup)

	// Process the backed-up results, ensuring parent UIDs and child OwnerReferences
	// are properly annotated in memory before applying updates to the cluster
	for resource := range backupChan {
		resourceUID := resource.item.GetUID()

		_, ok := backupResourceMap[resourceUID]
		if !ok {
			backupResourceMap[resourceUID] = resource.resourceWithGroup
		}

		if resource.isParent {
			backupResourceMap[resourceUID] = setResourceUIDToAnnotations(backupResourceMap[resourceUID])
		}

		if resource.isChild {
			update, uErr := setOwnerReferencesToAnnotations(backupResourceMap[resourceUID])
			if uErr != nil {
				return fmt.Errorf("could not backup owner references for child resource %s: %w", resource.item.GetName(), uErr)
			}

			backupResourceMap[resourceUID] = update
		}
	}

	// Persist the modified resources by updating them
	if uErr := r.updateResources(ctx, slices.Collect(maps.Values(backupResourceMap))); uErr != nil {
		return fmt.Errorf("unable to update resources: %w", uErr)
	}

	logger.Info("Backup owner references completed")

	return nil
}

func (r Recreator) getCloudoguCRDKinds(ctx context.Context) ([]string, error) {
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
			return nil, fmt.Errorf("failed to find spec for CRD %s: %w", crd.GetName(), lErr)
		}

		group, ok := spec["group"].(string)
		if !ok {
			return nil, fmt.Errorf("failed to find group for CRD %s", crd.GetName())
		}

		if group != cloudoguResourceGroup {
			// Ignore CRDs outside the cloudogu group
			continue
		}

		names, ok := spec["names"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("failed to find names for CRD %s", crd.GetName())
		}

		kind, ok := names["kind"].(string)
		if !ok {
			return nil, fmt.Errorf("failed to find kind for CRD %s", crd.GetName())
		}

		kinds = append(kinds, kind)
	}

	logger.Info("Extracted kinds from cloudogu resource group", "kinds", kinds)

	return kinds, nil
}

func (r Recreator) fetchResourcesForBackup(ctx context.Context) ([]resourceWithGroup, map[types.UID][]resourceWithGroup, error) {
	logger := log.FromContext(ctx)

	cloudoguCRDKinds, err := r.getCloudoguCRDKinds(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get kinds of cloudogu resource group: %w", err)
	}

	apiResourceList, err := r.discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list preferred namespaced resources: %w", err)
	}

	// parentList contains resources that are considered top-level "parents"
	// in the context of the Cloudogu ecosystem. These are typically custom
	// resource types like Dogu, Component, or Backup, and are identified by
	// checking whether the resource kind matches known Cloudogu CRD kinds.
	//
	// These parent resources will later be inspected to determine whether
	// they are being referenced as owners by other Kubernetes resources.
	parentList := make([]resourceWithGroup, 0, len(apiResourceList))

	// childMap maps the UID of a parent resource to a list of other resources
	// (children) that reference it via an OwnerReference in their metadata.
	//
	// This mapping enables the system to identify which child resources
	// are associated with which parent, so that both can be considered
	// during backup or restore operations.
	childMap := make(map[types.UID][]resourceWithGroup)

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

			if slices.Contains(cloudoguCRDKinds, apiResource.Kind) {
				appendResourceToParentList(gvr, &parentList, res)
				continue
			}

			appendResourceToChildMap(gvr, childMap, res)
		}
	}

	return parentList, childMap, nil
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

// searchChildrenForParent processes tasks by checking for immediate child resources.
//
// searchChildrenForParent iterates parent nodes which are provided by the parent channel. Basically it inspects top level resources like
// dogus, backups, components, etc. but only to the first level of its children in order to save metadata later-on.
// The reconciling mechanism of the lower level children like ingress etc. are reconciled properly and will be overwritten
// even if they will be restored with a previously saved owner reference. One can think of the inspected data structure
// like this:
//
//	dogu --> service    --> ingress (not regarded)
//	     \-> deployment --> replica set (not regarded)
func searchChildrenForParent(childMap map[types.UID][]resourceWithGroup, parentChan chan resourceWithGroup, backupChan chan<- backupResource, wg, tasks *sync.WaitGroup) {
	defer wg.Done()

	for p := range parentChan {
		// Check if the UID is parent of a child group
		children, isParent := childMap[p.item.GetUID()]
		if !isParent {
			tasks.Done()
			continue
		}

		for _, child := range children {
			backupChan <- backupResource{
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

		backupChan <- backupResource{
			resourceWithGroup: p,
			isParent:          true,
			isChild:           false,
		}

		tasks.Done()
	}
}

func appendResourceToParentList(gvr schema.GroupVersionResource, parentList *[]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		*parentList = append(*parentList, resourceWithGroup{
			item:  item,
			group: gvr,
		})
	}
}

func appendResourceToChildMap(gvr schema.GroupVersionResource, resultMap map[types.UID][]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		for _, ownerRef := range item.GetOwnerReferences() {
			resultMap[ownerRef.UID] = append(resultMap[ownerRef.UID], resourceWithGroup{
				item:  item,
				group: gvr,
			})
		}
	}
}

func setOwnerReferencesToAnnotations(res resourceWithGroup) (resourceWithGroup, error) {
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

	annotations[annotationBackupOwnerReferenceKey] = string(jsonBackup)
	res.item.SetAnnotations(annotations)

	return res, nil
}

func setResourceUIDToAnnotations(res resourceWithGroup) resourceWithGroup {
	annotations := res.item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[annotationBackupUIDKey] = string(res.item.GetUID())

	res.item.SetAnnotations(annotations)

	return res
}

// RestoreOwnerReferences restores the OwnerReferences metadata on Kubernetes resources
// within the configured namespace using backup annotations previously stored by
// the BackupOwnerReferences method.
//
// This method performs the following steps:
//  1. Fetches all relevant parent and child resources in the namespace that
//     contain backup annotations.
//  2. Reconstructs the original OwnerReferences for each child resource based
//     on the annotation data and appends them to a list for update.
//  3. Cleans up parent resource annotations (e.g., backup UIDs) to restore them
//     to their pre-backup state.
//  4. Applies the changes to the cluster by updating each resource.
func (r Recreator) RestoreOwnerReferences(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// Fetch parent resources (those that were marked with backup UID annotations)
	// and child resources (those whose owner references were backed up into annotations)
	parentMap, childList, err := r.fetchResourcesForRestore(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}

	// restoreList accumulates all modified resources (parents and children)
	// that need to be updated in the cluster after restoring owner references
	restoreList := make([]resourceWithGroup, 0, len(childList))

	// Rebuild the original owner references for each child from backup annotations
	// and append them to the restore list
	for _, child := range childList {
		restoreList = append(restoreList, restoreOwnerReferencesFromAnnotations(ctx, child, parentMap))
	}

	// Remove the backup UID annotations from parent resources to clean up metadata
	for _, parent := range parentMap {
		restoreList = append(restoreList, deleteBackupUIDFromAnnotations(parent))
	}

	if uErr := r.updateResources(ctx, restoreList); uErr != nil {
		return fmt.Errorf("unable to update resources: %w", uErr)
	}

	logger.Info("Restore owner references completed")

	return nil
}

func (r Recreator) fetchResourcesForRestore(ctx context.Context) (map[types.UID]resourceWithGroup, []restoreResource, error) {
	logger := log.FromContext(ctx)

	cloudoguCRDKinds, err := r.getCloudoguCRDKinds(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get kinds of cloudogu resource group: %w", err)
	}

	apiResourceList, err := r.discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list preferred namespaced resource groups: %w", err)
	}

	// childList contains all resources with backed up owner references in their annotations
	childList := make([]restoreResource, 0, len(apiResourceList))
	// parentMap contains all resources with a backed up UID in their annotations
	parentMap := make(map[types.UID]resourceWithGroup)

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

			if slices.Contains(cloudoguCRDKinds, apiResource.Kind) {
				appendResourceToParentMap(gvr, parentMap, res)
				continue
			}

			err = appendResourceToChildList(gvr, &childList, res)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch children for group %s: %w", apiResourceGroup.GroupVersion, err)
			}
		}
	}

	return parentMap, childList, nil
}

func appendResourceToChildList(gvr schema.GroupVersionResource, childList *[]restoreResource, items *unstructured.UnstructuredList) error {
	for _, item := range items.Items {
		annotations := item.GetAnnotations()
		if len(annotations) == 0 {
			continue
		}

		oRefs, ok := annotations[annotationBackupOwnerReferenceKey]
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

		*childList = append(*childList, restoreResource{
			resourceWithGroup: resourceWithGroup{
				item:  item,
				group: gvr,
			},
			ownerRefs: ownerRefMap,
		})
	}

	return nil
}

func appendResourceToParentMap(gvr schema.GroupVersionResource, parentMap map[types.UID]resourceWithGroup, items *unstructured.UnstructuredList) {
	for _, item := range items.Items {
		annotations := item.GetAnnotations()
		if len(annotations) == 0 {
			continue
		}

		restoredUID, ok := annotations[annotationBackupUIDKey]
		if !ok {
			continue
		}

		parentMap[types.UID(restoredUID)] = resourceWithGroup{
			item:  item,
			group: gvr,
		}
	}
}

func restoreOwnerReferencesFromAnnotations(ctx context.Context, res restoreResource, parents map[types.UID]resourceWithGroup) resourceWithGroup {
	logger := log.FromContext(ctx)

	for _, ownerRef := range res.ownerRefs {
		parent, ok := parents[ownerRef.UID]
		if !ok {
			logger.Info("Could not find parent for owner reference", "parentUID", ownerRef.UID, "resource", res.item.GetName(), "kind", res.item.GetKind())
			continue
		}

		// this is kind of ugly because the deployment controller copies our owner reference annotation from the deployment
		// into the replica set. Avoid setting this during the restore by checking explicitly for the RS kind.
		if res.item.GetKind() != "ReplicaSet" {
			ownerRef.UID = parent.item.GetUID()
			res.item.SetOwnerReferences(append(res.item.GetOwnerReferences(), ownerRef))
		}

		annotations := res.item.GetAnnotations()
		delete(annotations, annotationBackupOwnerReferenceKey)
		res.item.SetAnnotations(annotations)
	}

	return res.resourceWithGroup
}

func deleteBackupUIDFromAnnotations(res resourceWithGroup) resourceWithGroup {
	annotations := res.item.GetAnnotations()
	delete(annotations, annotationBackupUIDKey)
	res.item.SetAnnotations(annotations)

	return res
}
