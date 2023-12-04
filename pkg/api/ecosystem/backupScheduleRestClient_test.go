package ecosystem

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

func Test_backupScheduleClient_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/testbackupSchedule", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "testbackupSchedule", Namespace: "test"}}
			backupScheduleBytes, err := json.Marshal(backupSchedule)
			require.NoError(t, err)
			_, err = writer.Write(backupScheduleBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.Get(testCtx, "testbackupSchedule", v1.GetOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodGet, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			backupScheduleList := k8sv1.BackupScheduleList{}
			backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "testbackupSchedule", Namespace: "test"}}
			backupScheduleList.Items = append(backupScheduleList.Items, *backupSchedule)
			backupScheduleBytes, err := json.Marshal(backupScheduleList)
			require.NoError(t, err)
			_, err = writer.Write(backupScheduleBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.List(testCtx, v1.ListOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_Watch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)
			assert.Equal(t, "labelSelector=test&watch=true", request.URL.RawQuery)

			writer.Header().Add("content-type", "application/json")
			_, err := writer.Write([]byte("egal"))
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.Watch(testCtx, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackupSchedule := &k8sv1.BackupSchedule{}
			require.NoError(t, json.Unmarshal(bytes, createdBackupSchedule))
			assert.Equal(t, "tocreate", createdBackupSchedule.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.Create(testCtx, backupSchedule, v1.CreateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/tocreate", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackupSchedule := &k8sv1.BackupSchedule{}
			require.NoError(t, json.Unmarshal(bytes, createdBackupSchedule))
			assert.Equal(t, "tocreate", createdBackupSchedule.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.Update(testCtx, backupSchedule, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/tocreate/status", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackupSchedule := &k8sv1.BackupSchedule{}
			require.NoError(t, json.Unmarshal(bytes, createdBackupSchedule))
			assert.Equal(t, "tocreate", createdBackupSchedule.Name)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(bytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		_, err = dClient.UpdateStatus(testCtx, backupSchedule, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/testbackupSchedule", request.URL.Path)

			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		err = dClient.Delete(testCtx, "testbackupSchedule", v1.DeleteOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_DeleteCollection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules", request.URL.Path)
			assert.Equal(t, "labelSelector=test", request.URL.RawQuery)
			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		// when
		err = dClient.DeleteCollection(testCtx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_Patch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPatch, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/testbackupSchedule", request.URL.Path)
			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("test"), bytes)
			result, err := json.Marshal(k8sv1.BackupSchedule{})
			require.NoError(t, err)

			writer.Header().Add("content-type", "application/json")
			_, err = writer.Write(result)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.BackupSchedules("test")

		patchData := []byte("test")

		// when
		_, err = dClient.Patch(testCtx, "testbackupSchedule", types.JSONPatchType, patchData, v1.PatchOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupScheduleClient_UpdateStatusXXX(t *testing.T) {
	for _, testCase := range []struct {
		functionName   string
		expectedStatus string
	}{
		{
			functionName:   "UpdateStatusCreated",
			expectedStatus: "created",
		},
		{
			functionName:   "UpdateStatusCreating",
			expectedStatus: "creating",
		},
		{
			functionName:   "UpdateStatusUpdating",
			expectedStatus: "updating",
		},
		{
			functionName:   "UpdateStatusDeleting",
			expectedStatus: "deleting",
		},
		{
			functionName:   "UpdateStatusFailed",
			expectedStatus: "failed",
		},
	} {
		t.Run(fmt.Sprintf("%s success", testCase.functionName), func(t *testing.T) {
			// given
			backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "testbackupSchedule", Namespace: "test"}}

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch request.Method {
				case http.MethodGet:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/testbackupSchedule", request.URL.Path)
					assert.Equal(t, http.NoBody, request.Body)

					writer.Header().Add("content-type", "application/json")
					backupSchedule := &k8sv1.BackupSchedule{ObjectMeta: v1.ObjectMeta{Name: "testbackupSchedule", Namespace: "test"}}
					backupScheduleBytes, err := json.Marshal(backupSchedule)
					require.NoError(t, err)
					_, err = writer.Write(backupScheduleBytes)
					require.NoError(t, err)
					writer.WriteHeader(200)
				case http.MethodPut:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backupschedules/testbackupSchedule/status", request.URL.Path)
					bytes, err := io.ReadAll(request.Body)
					require.NoError(t, err)

					createdBackupSchedule := &k8sv1.BackupSchedule{}
					require.NoError(t, json.Unmarshal(bytes, createdBackupSchedule))
					assert.Equal(t, "testbackupSchedule", createdBackupSchedule.Name)
					assert.Equal(t, testCase.expectedStatus, createdBackupSchedule.Status.Status)

					writer.Header().Add("content-type", "application/json")
					_, err = writer.Write(bytes)
					require.NoError(t, err)
					writer.WriteHeader(200)
				default:
					assert.Fail(t, "method should be get or put")
				}
			}))

			config := rest.Config{
				Host: server.URL,
			}
			client, err := NewForConfig(&config)
			require.NoError(t, err)
			dClient := client.BackupSchedules("test")

			// when
			returnValues := reflect.ValueOf(dClient).MethodByName(testCase.functionName).Call([]reflect.Value{reflect.ValueOf(testCtx), reflect.ValueOf(backupSchedule)})
			err, _ = returnValues[1].Interface().(error)

			// then
			require.NoError(t, err)
		})
	}
}
