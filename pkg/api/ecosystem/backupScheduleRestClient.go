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

type BackupScheduleInterface interface {
	// Create takes the representation of a backup schedule and creates it.  Returns the server's representation of the backup schedule, and an error, if there is any.
	Create(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.CreateOptions) (*v1.BackupSchedule, error)

	// Update takes the representation of a backup schedule and updates it. Returns the server's representation of the backup schedule, and an error, if there is any.
	Update(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.UpdateOptions) (*v1.BackupSchedule, error)

	// UpdateStatus was generated because the type contains a Status member.
	UpdateStatus(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.UpdateOptions) (*v1.BackupSchedule, error)

	// UpdateStatusCreated sets the status of the backup schedule to "created".
	UpdateStatusCreated(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// UpdateStatusCreating sets the status of the backup schedule to "creating".
	UpdateStatusCreating(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// UpdateStatusUpdating sets the status of the backup schedule to "updating".
	UpdateStatusUpdating(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// UpdateStatusDeleting sets the status of the backup schedule to "deleting".
	UpdateStatusDeleting(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// UpdateStatusFailed sets the status of the backup schedule to "failed".
	UpdateStatusFailed(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// Delete takes name of the backup schedule and deletes it. Returns an error if one occurs.
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error

	// DeleteCollection deletes a collection of objects.
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error

	// Get takes name of the backup schedule, and returns the corresponding backup schedule object, and an error if there is any.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.BackupSchedule, error)

	// List takes label and field selectors, and returns the list of BackupSchedules that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (*v1.BackupScheduleList, error)

	// Watch returns a watch.Interface that watches the requested backup schedules.
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	// Patch applies the patch and returns the patched backup schedule.
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.BackupSchedule, err error)

	// AddFinalizer adds the given finalizer to the backup schedule.
	AddFinalizer(ctx context.Context, backupSchedule *v1.BackupSchedule, finalizer string) (*v1.BackupSchedule, error)

	// AddLabels adds the app=ces and k8s.cloudogu.com/part-of=backup labels to the backup schedule.
	AddLabels(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error)

	// RemoveFinalizer removes the given finalizer to the backup schedule.
	RemoveFinalizer(ctx context.Context, backupSchedule *v1.BackupSchedule, finalizer string) (*v1.BackupSchedule, error)
}

type backupScheduleClient struct {
	client rest.Interface
	ns     string
}

// UpdateStatusCreated sets the status of the backup schedule to "created".
func (d *backupScheduleClient) UpdateStatusCreated(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	return d.updateStatusWithRetry(ctx, backupSchedule, v1.BackupScheduleStatusCreated)
}

// UpdateStatusCreating sets the status of the backup schedule to "creating".
func (d *backupScheduleClient) UpdateStatusCreating(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	return d.updateStatusWithRetry(ctx, backupSchedule, v1.BackupScheduleStatusCreating)
}

// UpdateStatusUpdating sets the status of the backup schedule to "updating".
func (d *backupScheduleClient) UpdateStatusUpdating(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	return d.updateStatusWithRetry(ctx, backupSchedule, v1.BackupScheduleStatusUpdating)
}

// UpdateStatusDeleting sets the status of the backup schedule to "deleting".
func (d *backupScheduleClient) UpdateStatusDeleting(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	return d.updateStatusWithRetry(ctx, backupSchedule, v1.BackupScheduleStatusDeleting)
}

// UpdateStatusFailed sets the status of the backup schedule to "failed".
func (d *backupScheduleClient) UpdateStatusFailed(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	return d.updateStatusWithRetry(ctx, backupSchedule, v1.BackupScheduleStatusFailed)
}

func (d *backupScheduleClient) updateStatusWithRetry(ctx context.Context, backupSchedule *v1.BackupSchedule, targetStatus string) (*v1.BackupSchedule, error) {
	var resultBackupSchedule *v1.BackupSchedule
	err := retry.OnConflict(func() error {
		updatedBackupSchedule, err := d.Get(ctx, backupSchedule.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		// do not overwrite the whole status, so we do not lose other values from the Status object
		// esp. a potentially set requeue time
		updatedBackupSchedule.Status.Status = targetStatus
		resultBackupSchedule, err = d.UpdateStatus(ctx, updatedBackupSchedule, metav1.UpdateOptions{})
		return err
	})

	return resultBackupSchedule, err
}

// AddFinalizer adds the given finalizer to the backup schedule.
func (d *backupScheduleClient) AddFinalizer(ctx context.Context, backupSchedule *v1.BackupSchedule, finalizer string) (*v1.BackupSchedule, error) {
	controllerutil.AddFinalizer(backupSchedule, finalizer)
	result, err := d.Update(ctx, backupSchedule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s to backup schedule: %w", finalizer, err)
	}

	return result, nil
}

// AddLabels adds the app=ces and k8s.cloudogu.com/part-of=backup labels to the backup schedule.
func (d *backupScheduleClient) AddLabels(ctx context.Context, backupSchedule *v1.BackupSchedule) (*v1.BackupSchedule, error) {
	if backupSchedule.Labels == nil {
		backupSchedule.Labels = make(map[string]string)
	}
	backupSchedule.Labels[appLabelKey] = appLabelValueCes
	backupSchedule.Labels[partOfLabelKey] = partOfLabelValueBackup

	result, err := d.Update(ctx, backupSchedule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to add label app=ces and k8s.cloudogu.com/part-of=backup to backup schedule: %w", err)
	}

	return result, nil
}

// RemoveFinalizer removes the given finalizer to the backup schedule.
func (d *backupScheduleClient) RemoveFinalizer(ctx context.Context, backupSchedule *v1.BackupSchedule, finalizer string) (*v1.BackupSchedule, error) {
	controllerutil.RemoveFinalizer(backupSchedule, finalizer)
	result, err := d.Update(ctx, backupSchedule, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to remove finalizer %s from backup schedule: %w", finalizer, err)
	}

	return result, nil
}

// Get takes name of the backup schedule, and returns the corresponding backup schedule object, and an error if there is any.
func (d *backupScheduleClient) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.BackupSchedule, err error) {
	result = &v1.BackupSchedule{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("backupschedules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BackupSchedules that match those selectors.
func (d *backupScheduleClient) List(ctx context.Context, opts metav1.ListOptions) (result *v1.BackupScheduleList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.BackupScheduleList{}
	err = d.client.Get().
		Namespace(d.ns).
		Resource("backupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested backup schedules.
func (d *backupScheduleClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return d.client.Get().
		Namespace(d.ns).
		Resource("backupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a backup schedule and creates it.  Returns the server's representation of the backup schedule, and an error, if there is any.
func (d *backupScheduleClient) Create(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.CreateOptions) (result *v1.BackupSchedule, err error) {
	result = &v1.BackupSchedule{}
	err = d.client.Post().
		Namespace(d.ns).
		Resource("backupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backupSchedule).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a backup schedule and updates it. Returns the server's representation of the backup schedule, and an error, if there is any.
func (d *backupScheduleClient) Update(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.UpdateOptions) (result *v1.BackupSchedule, err error) {
	result = &v1.BackupSchedule{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("backupschedules").
		Name(backupSchedule.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backupSchedule).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (d *backupScheduleClient) UpdateStatus(ctx context.Context, backupSchedule *v1.BackupSchedule, opts metav1.UpdateOptions) (result *v1.BackupSchedule, err error) {
	result = &v1.BackupSchedule{}
	err = d.client.Put().
		Namespace(d.ns).
		Resource("backupschedules").
		Name(backupSchedule.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(backupSchedule).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the backup schedule and deletes it. Returns an error if one occurs.
func (d *backupScheduleClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return d.client.Delete().
		Namespace(d.ns).
		Resource("backupschedules").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (d *backupScheduleClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return d.client.Delete().
		Namespace(d.ns).
		Resource("backupschedules").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched backup schedule.
func (d *backupScheduleClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.BackupSchedule, err error) {
	result = &v1.BackupSchedule{}
	patch := d.client.Patch(pt)
	err = patch.
		Namespace(d.ns).
		Resource("backupschedules").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
