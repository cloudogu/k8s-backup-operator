package ecosystem

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RestoreInterface interface {
	// Create takes the representation of a restore and creates it.  Returns the server's representation of the restore, and an error, if there is any.
	Create(ctx context.Context, restore *v1.Restore, opts metav1.CreateOptions) (*v1.Restore, error)

	// Update takes the representation of a restore and updates it. Returns the server's representation of the restore, and an error, if there is any.
	Update(ctx context.Context, restore *v1.Restore, opts metav1.UpdateOptions) (*v1.Restore, error)

	// UpdateStatus was generated because the type contains a Status member.
	UpdateStatus(ctx context.Context, restore *v1.Restore, opts metav1.UpdateOptions) (*v1.Restore, error)

	// UpdateStatusInProgress sets the status of the restore to "in progress".
	UpdateStatusInProgress(ctx context.Context, restore *v1.Restore) (*v1.Restore, error)

	// UpdateStatusCompleted sets the status of the restore to "completed".
	UpdateStatusCompleted(ctx context.Context, restore *v1.Restore) (*v1.Restore, error)

	// UpdateStatusFailed sets the status of the restore to "failed".
	UpdateStatusFailed(ctx context.Context, restore *v1.Restore) (*v1.Restore, error)

	// UpdateStatusDeleting sets the status of the restore to "deleting".
	UpdateStatusDeleting(ctx context.Context, restore *v1.Restore) (*v1.Restore, error)

	// Delete takes name of the restore and deletes it. Returns an error if one occurs.
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error

	// DeleteCollection deletes a collection of objects.
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error

	// Get takes name of the restore, and returns the corresponding restore object, and an error if there is any.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Restore, error)

	// List takes label and field selectors, and returns the list of Restores that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (*v1.RestoreList, error)

	// Watch returns a watch.Interface that watches the requested restores.
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	// Patch applies the patch and returns the patched restore.
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Restore, err error)

	// AddFinalizer adds the given finalizer to the restore.
	AddFinalizer(ctx context.Context, restore *v1.Restore, finalizer string) (*v1.Restore, error)

	// AddLabels adds the app=ces and k8s.cloudogu.com/part-of=backup labels to the restore.
	AddLabels(ctx context.Context, restore *v1.Restore) (*v1.Restore, error)

	// RemoveFinalizer removes the given finalizer to the restore.
	RemoveFinalizer(ctx context.Context, restore *v1.Restore, finalizer string) (*v1.Restore, error)
}

type restoreClient struct {
	client rest.Interface
	ns     string
}

// UpdateStatusInProgress sets the status of the restore to "in progress".
func (d *restoreClient) UpdateStatusInProgress(ctx context.Context, restore *v1.Restore) (*v1.Restore, error) {
	return d.updateStatusWithRetry(ctx, restore, v1.RestoreStatusInProgress)
}

// UpdateStatusCompleted sets the status of the restore to "completed".
func (d *restoreClient) UpdateStatusCompleted(ctx context.Context, restore *v1.Restore) (*v1.Restore, error) {
	return d.updateStatusWithRetry(ctx, restore, v1.RestoreStatusCompleted)
}

// UpdateStatusFailed sets the status of the restore to "failed".
func (d *restoreClient) UpdateStatusFailed(ctx context.Context, restore *v1.Restore) (*v1.Restore, error) {
	return d.updateStatusWithRetry(ctx, restore, v1.RestoreStatusFailed)
}

// UpdateStatusDeleting sets the status of the restore to "deleting".
func (d *restoreClient) UpdateStatusDeleting(ctx context.Context, restore *v1.Restore) (*v1.Restore, error) {
	return d.updateStatusWithRetry(ctx, restore, v1.RestoreStatusDeleting)
}

func (d *restoreClient) updateStatusWithRetry(ctx context.Context, restore *v1.Restore, targetStatus string) (*v1.Restore, error) {
	var resultRestore *v1.Restore
	err := retry.OnConflict(func() error {
		updatedRestore, err := d.Get(ctx, restore.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		// do not overwrite the whole status, so we do not lose other values from the Status object
		// esp. a potentially set requeue time
		updatedRestore.Status.Status = targetStatus
		resultRestore, err = d.UpdateStatus(ctx, updatedRestore, metav1.UpdateOptions{})
		return err
	})

	return resultRestore, err
}

// AddFinalizer adds the given finalizer to the restore.
func (d *restoreClient) AddFinalizer(ctx context.Context, restore *v1.Restore, finalizer string) (*v1.Restore, error) {
	controllerutil.AddFinalizer(restore, finalizer)
	result, err := d.Update(ctx, restore, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s to restore: %w", finalizer, err)
	}

	return result, nil
}

// AddLabels adds the app=ces and k8s.cloudogu.com/part-of=backup labels to the restore.
func (d *restoreClient) AddLabels(ctx context.Context, restore *v1.Restore) (*v1.Restore, error) {
	if restore.Labels == nil {
		restore.Labels = make(map[string]string)
	}
	restore.Labels[appLabelKey] = appLabelValueCes
	restore.Labels[partOfLabelKey] = partOfLabelValueBackup

	result, err := d.Update(ctx, restore, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to add label app=ces and k8s.cloudogu.com/part-of=backup to restore: %w", err)
	}

	return result, nil
}

// RemoveFinalizer removes the given finalizer to the restore.
func (d *restoreClient) RemoveFinalizer(ctx context.Context, restore *v1.Restore, finalizer string) (*v1.Restore, error) {
	controllerutil.RemoveFinalizer(restore, finalizer)
	result, err := d.Update(ctx, restore, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to remove finalizer %s from restore: %w", finalizer, err)
	}

	return result, err
}

// Get takes name of the restore, and returns the corresponding restore object, and an error if there is any.
func (d *restoreClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Restore, err error) {
	result = &v1.Restore{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("restores").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Restores that match those selectors.
func (d *restoreClient) List(ctx context.Context, opts metav1.ListOptions) (result *v1.RestoreList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.RestoreList{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("restores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested restores.
func (d *restoreClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return d.client.Get().
		Namespace(d.ns).
		Resource("restores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a restore and creates it.  Returns the server's representation of the restore, and an error, if there is any.
func (d *restoreClient) Create(ctx context.Context, restore *v1.Restore, opts metav1.CreateOptions) (result *v1.Restore, err error) {
	result = &v1.Restore{}
	err = d.client.Post().
		Namespace(d.ns).
		Resource("restores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(restore).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a restore and updates it. Returns the server's representation of the restore, and an error, if there is any.
func (d *restoreClient) Update(ctx context.Context, restore *v1.Restore, opts metav1.UpdateOptions) (result *v1.Restore, err error) {
	result = &v1.Restore{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("restores").
		Name(restore.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(restore).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *restoreClient) UpdateStatus(ctx context.Context, restore *v1.Restore, opts metav1.UpdateOptions) (result *v1.Restore, err error) {
	result = &v1.Restore{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("restores").
		Name(restore.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(restore).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the restore and deletes it. Returns an error if one occurs.
func (d *restoreClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.client.Delete().
		Namespace(d.ns).
		Resource("restores").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (d *restoreClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return d.client.Delete().
		Namespace(d.ns).
		Resource("restores").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched restore.
func (d *restoreClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Restore, err error) {
	result = &v1.Restore{}
	err = d.client.Patch(pt).
		Namespace(d.ns).
		Resource("restores").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
