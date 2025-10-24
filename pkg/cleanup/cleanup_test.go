package cleanup

import (
	"context"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCleanUp(t *testing.T) {
	k8sNotFoundErr := k8sErr.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "error")

	t.Run("should successfully clean up dogus", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().List(mock.Anything, v1.ListOptions{}).Return(&v2.DoguList{
			Items: []v2.Dogu{
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-2"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-3"}},
			},
		}, nil)

		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-1", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-2", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-3", v1.DeleteOptions{}).Return(nil)

		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-1", v1.GetOptions{}).Return(nil, k8sNotFoundErr)
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-2", v1.GetOptions{}).Return(nil, k8sNotFoundErr)
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-3", v1.GetOptions{}).Return(nil, k8sNotFoundErr)

		sut := &DefaultCleanupManager{doguClient: mDoguClient}

		err := sut.Cleanup(context.Background())
		assert.NoError(t, err)
	})

	t.Run("should successfully clean up dogus and wait for dogus to be deleted", func(t *testing.T) {
		originalWaitTime := doguDeleteWaitTime
		doguDeleteWaitTime = 10 * time.Millisecond
		defer func() {
			doguDeleteWaitTime = originalWaitTime
		}()

		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().List(mock.Anything, v1.ListOptions{}).Return(&v2.DoguList{
			Items: []v2.Dogu{
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-2"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-3"}},
			},
		}, nil)

		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-1", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-2", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-3", v1.DeleteOptions{}).Return(nil)

		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-1", v1.GetOptions{}).Return(nil, k8sNotFoundErr)
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-3", v1.GetOptions{}).Return(nil, k8sNotFoundErr)

		counter := 0
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-2", v1.GetOptions{}).RunAndReturn(func(ctx context.Context, name string, options v1.GetOptions) (*v2.Dogu, error) {
			assert.Equal(t, "test-dogu-2", name)
			counter++

			if counter == 1 {
				return nil, assert.AnError
			}

			if counter <= 2 {
				return &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}}, nil
			}

			return nil, k8sNotFoundErr
		})

		sut := &DefaultCleanupManager{doguClient: mDoguClient}

		err := sut.Cleanup(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 3, counter)
	})

	t.Run("should fail to clean up dogus when timeout is expired", func(t *testing.T) {
		originalTimeout := cleanupTimeout
		cleanupTimeout = 1 * time.Second
		defer func() {
			cleanupTimeout = originalTimeout
		}()

		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().List(mock.Anything, v1.ListOptions{}).Return(&v2.DoguList{
			Items: []v2.Dogu{
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-2"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-3"}},
			},
		}, nil)

		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-1", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-2", v1.DeleteOptions{}).Return(nil)
		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-3", v1.DeleteOptions{}).Return(nil)

		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-1", v1.GetOptions{}).Return(nil, k8sNotFoundErr)
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-3", v1.GetOptions{}).Return(nil, k8sNotFoundErr)

		counter := 0
		mDoguClient.EXPECT().Get(mock.Anything, "test-dogu-2", v1.GetOptions{}).RunAndReturn(func(ctx context.Context, name string, options v1.GetOptions) (*v2.Dogu, error) {
			counter++
			assert.Equal(t, "test-dogu-2", name)
			assert.Less(t, counter, 3)

			return &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}}, nil
		})

		sut := &DefaultCleanupManager{doguClient: mDoguClient}

		err := sut.Cleanup(context.Background())
		assert.Error(t, err)
		assert.ErrorContains(t, err, "cleanup timed out: context deadline exceeded")
	})

	t.Run("should fail to clean up dogus on error deleting dogu", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().List(mock.Anything, v1.ListOptions{}).Return(&v2.DoguList{
			Items: []v2.Dogu{
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-1"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-2"}},
				{ObjectMeta: v1.ObjectMeta{Name: "test-dogu-3"}},
			},
		}, nil)

		mDoguClient.EXPECT().Delete(mock.Anything, "test-dogu-1", v1.DeleteOptions{}).Return(assert.AnError)

		sut := &DefaultCleanupManager{doguClient: mDoguClient}

		err := sut.Cleanup(context.Background())
		assert.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete dogu test-dogu-1:")
	})

	t.Run("should fail to clean up dogus on error listing dogus", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().List(mock.Anything, v1.ListOptions{}).Return(nil, assert.AnError)

		sut := &DefaultCleanupManager{doguClient: mDoguClient}

		err := sut.Cleanup(context.Background())
		assert.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list dogus:")
	})
}

func TestNewManager(t *testing.T) {
	t.Run("should create a new manager", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)

		sut := NewManager(mDoguClient)

		assert.NotNil(t, sut)
		assert.Equal(t, mDoguClient, sut.doguClient)
	})
}
