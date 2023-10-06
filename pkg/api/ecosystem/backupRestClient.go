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

type BackupInterface interface {
	// Create takes the representation of a backup and creates it.  Returns the server's representation of the backup, and an error, if there is any.
	Create(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions) (*v1.Backup, error)

	// Update takes the representation of a backup and updates it. Returns the server's representation of the backup, and an error, if there is any.
	Update(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error)

	// UpdateStatus was generated because the type contains a Status member.
	UpdateStatus(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error)

	// UpdateStatusInProgress sets the status of the backup to "in progress".
	UpdateStatusInProgress(ctx context.Context, backup *v1.Backup) (*v1.Backup, error)

	// UpdateStatusCompleted sets the status of the backup to "completed".
	UpdateStatusCompleted(ctx context.Context, backup *v1.Backup) (*v1.Backup, error)

	// UpdateStatusDeleting sets the status of the backup to "deleting".
	UpdateStatusDeleting(ctx context.Context, backup *v1.Backup) (*v1.Backup, error)

	// UpdateStatusFailed sets the status of the backup to "failed".
	UpdateStatusFailed(ctx context.Context, backup *v1.Backup) (*v1.Backup, error)

	// Delete takes name of the backup and deletes it. Returns an error if one occurs.
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error

	// DeleteCollection deletes a collection of objects.
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error

	// Get takes name of the backup, and returns the corresponding backup object, and an error if there is any.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Backup, error)

	// List takes label and field selectors, and returns the list of Backups that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (*v1.BackupList, error)

	// Watch returns a watch.Interface that watches the requested backups.
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	// Patch applies the patch and returns the patched backup.
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Backup, err error)

	// AddFinalizer adds the given finalizer to the backup.
	AddFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error)

	// RemoveFinalizer removes the given finalizer to the backup.
	RemoveFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error)
}

type backupClient struct {
	client rest.Interface
	ns     string
}

// UpdateStatusInProgress sets the status of the backup to "in progress".
func (d *backupClient) UpdateStatusInProgress(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	return d.updateStatusWithRetry(ctx, backup, v1.BackupStatusInProgress)
}

// UpdateStatusCompleted sets the status of the backup to "completed".
func (d *backupClient) UpdateStatusCompleted(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	return d.updateStatusWithRetry(ctx, backup, v1.BackupStatusCompleted)
}

// UpdateStatusDeleting sets the status of the backup to "deleting".
func (d *backupClient) UpdateStatusDeleting(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	return d.updateStatusWithRetry(ctx, backup, v1.BackupStatusDeleting)
}

// UpdateStatusFailed sets the status of the backup to "failed".
func (d *backupClient) UpdateStatusFailed(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	return d.updateStatusWithRetry(ctx, backup, v1.BackupStatusFailed)
}

func (d *backupClient) updateStatusWithRetry(ctx context.Context, backup *v1.Backup, targetStatus string) (*v1.Backup, error) {
	var resultBackup *v1.Backup
	err := retry.OnConflict(func() error {
		updatedBackup, err := d.Get(ctx, backup.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		// do not overwrite the whole status, so we do not lose other values from the Status object
		// esp. a potentially set requeue time
		updatedBackup.Status.Status = targetStatus
		resultBackup, err = d.UpdateStatus(ctx, updatedBackup, metav1.UpdateOptions{})
		return err
	})

	return resultBackup, err
}

// AddFinalizer adds the given finalizer to the backup.
func (d *backupClient) AddFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error) {
	controllerutil.AddFinalizer(backup, finalizer)
	result, err := d.Update(ctx, backup, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s to backup: %w", finalizer, err)
	}

	return result, nil
}

// RemoveFinalizer removes the given finalizer to the backup.
func (d *backupClient) RemoveFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error) {
	controllerutil.RemoveFinalizer(backup, finalizer)
	result, err := d.Update(ctx, backup, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to remove finalizer %s from backup: %w", finalizer, err)
	}

	return result, err
}

// Get takes name of the backup, and returns the corresponding backup object, and an error if there is any.
func (d *backupClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Backup, err error) {
	result = &v1.Backup{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("backups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Backups that match those selectors.
func (d *backupClient) List(ctx context.Context, opts metav1.ListOptions) (result *v1.BackupList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.BackupList{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("backups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested backups.
func (d *backupClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return d.client.Get().
		Namespace(d.ns).
		Resource("backups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a backup and creates it.  Returns the server's representation of the backup, and an error, if there is any.
func (d *backupClient) Create(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions) (result *v1.Backup, err error) {
	result = &v1.Backup{}
	err = d.client.Post().
		Namespace(d.ns).
		Resource("backups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backup).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a backup and updates it. Returns the server's representation of the backup, and an error, if there is any.
func (d *backupClient) Update(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (result *v1.Backup, err error) {
	result = &v1.Backup{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("backups").
		Name(backup.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backup).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *backupClient) UpdateStatus(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (result *v1.Backup, err error) {
	result = &v1.Backup{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("backups").
		Name(backup.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backup).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the backup and deletes it. Returns an error if one occurs.
func (d *backupClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.client.Delete().
		Namespace(d.ns).
		Resource("backups").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (d *backupClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return d.client.Delete().
		Namespace(d.ns).
		Resource("backups").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched backup.
func (d *backupClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Backup, err error) {
	result = &v1.Backup{}
	err = d.client.Patch(pt).
		Namespace(d.ns).
		Resource("backups").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
