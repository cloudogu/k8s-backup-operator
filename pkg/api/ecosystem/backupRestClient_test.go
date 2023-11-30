package ecosystem

import (
	"context"
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

var testCtx = context.Background()

func Test_backupClient_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/testbackup", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "testbackup", Namespace: "test"}}
			backupBytes, err := json.Marshal(backup)
			require.NoError(t, err)
			_, err = writer.Write(backupBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Backups("test")

		// when
		_, err = dClient.Get(testCtx, "testbackup", v1.GetOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodGet, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			backupList := k8sv1.BackupList{}
			backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "testbackup", Namespace: "test"}}
			backupList.Items = append(backupList.Items, *backup)
			backupBytes, err := json.Marshal(backupList)
			require.NoError(t, err)
			_, err = writer.Write(backupBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Backups("test")

		// when
		_, err = dClient.List(testCtx, v1.ListOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_Watch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups", request.URL.Path)
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
		dClient := client.Backups("test")

		// when
		_, err = dClient.Watch(testCtx, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackup := &k8sv1.Backup{}
			require.NoError(t, json.Unmarshal(bytes, createdBackup))
			assert.Equal(t, "tocreate", createdBackup.Name)

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
		dClient := client.Backups("test")

		// when
		_, err = dClient.Create(testCtx, backup, v1.CreateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/tocreate", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackup := &k8sv1.Backup{}
			require.NoError(t, json.Unmarshal(bytes, createdBackup))
			assert.Equal(t, "tocreate", createdBackup.Name)

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
		dClient := client.Backups("test")

		// when
		_, err = dClient.Update(testCtx, backup, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/tocreate/status", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdBackup := &k8sv1.Backup{}
			require.NoError(t, json.Unmarshal(bytes, createdBackup))
			assert.Equal(t, "tocreate", createdBackup.Name)

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
		dClient := client.Backups("test")

		// when
		_, err = dClient.UpdateStatus(testCtx, backup, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/testbackup", request.URL.Path)

			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Backups("test")

		// when
		err = dClient.Delete(testCtx, "testbackup", v1.DeleteOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_DeleteCollection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups", request.URL.Path)
			assert.Equal(t, "labelSelector=test", request.URL.RawQuery)
			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Backups("test")

		// when
		err = dClient.DeleteCollection(testCtx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_Patch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPatch, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/testbackup", request.URL.Path)
			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("test"), bytes)
			result, err := json.Marshal(k8sv1.Backup{})
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
		dClient := client.Backups("test")

		patchData := []byte("test")

		// when
		_, err = dClient.Patch(testCtx, "testbackup", types.JSONPatchType, patchData, v1.PatchOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_backupClient_UpdateStatusXXX(t *testing.T) {
	for _, testCase := range []struct {
		functionName   string
		expectedStatus string
	}{
		{
			functionName:   "UpdateStatusInProgress",
			expectedStatus: "in progress",
		},
		{
			functionName:   "UpdateStatusCompleted",
			expectedStatus: "completed",
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
			backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "testbackup", Namespace: "test"}}

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch request.Method {
				case http.MethodGet:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/testbackup", request.URL.Path)
					assert.Equal(t, http.NoBody, request.Body)

					writer.Header().Add("content-type", "application/json")
					backup := &k8sv1.Backup{ObjectMeta: v1.ObjectMeta{Name: "testbackup", Namespace: "test"}}
					backupBytes, err := json.Marshal(backup)
					require.NoError(t, err)
					_, err = writer.Write(backupBytes)
					require.NoError(t, err)
					writer.WriteHeader(200)
				case http.MethodPut:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/backups/testbackup/status", request.URL.Path)
					bytes, err := io.ReadAll(request.Body)
					require.NoError(t, err)

					createdBackup := &k8sv1.Backup{}
					require.NoError(t, json.Unmarshal(bytes, createdBackup))
					assert.Equal(t, "testbackup", createdBackup.Name)
					assert.Equal(t, testCase.expectedStatus, createdBackup.Status.Status)

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
			dClient := client.Backups("test")

			// when
			returnValues := reflect.ValueOf(dClient).MethodByName(testCase.functionName).Call([]reflect.Value{reflect.ValueOf(testCtx), reflect.ValueOf(backup)})
			err, _ = returnValues[1].Interface().(error)

			// then
			require.NoError(t, err)
		})
	}
}
