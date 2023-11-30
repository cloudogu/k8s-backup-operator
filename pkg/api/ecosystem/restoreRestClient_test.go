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

func Test_restoreClient_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/testrestore", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "testrestore", Namespace: "test"}}
			restoreBytes, err := json.Marshal(restore)
			require.NoError(t, err)
			_, err = writer.Write(restoreBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Restores("test")

		// when
		_, err = dClient.Get(testCtx, "testrestore", v1.GetOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodGet, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores", request.URL.Path)
			assert.Equal(t, http.NoBody, request.Body)

			writer.Header().Add("content-type", "application/json")
			restoreList := k8sv1.RestoreList{}
			restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "testrestore", Namespace: "test"}}
			restoreList.Items = append(restoreList.Items, *restore)
			restoreBytes, err := json.Marshal(restoreList)
			require.NoError(t, err)
			_, err = writer.Write(restoreBytes)
			require.NoError(t, err)
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Restores("test")

		// when
		_, err = dClient.List(testCtx, v1.ListOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_Watch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "GET", request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores", request.URL.Path)
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
		dClient := client.Restores("test")

		// when
		_, err = dClient.Watch(testCtx, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdRestore := &k8sv1.Restore{}
			require.NoError(t, json.Unmarshal(bytes, createdRestore))
			assert.Equal(t, "tocreate", createdRestore.Name)

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
		dClient := client.Restores("test")

		// when
		_, err = dClient.Create(testCtx, restore, v1.CreateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/tocreate", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdRestore := &k8sv1.Restore{}
			require.NoError(t, json.Unmarshal(bytes, createdRestore))
			assert.Equal(t, "tocreate", createdRestore.Name)

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
		dClient := client.Restores("test")

		// when
		_, err = dClient.Update(testCtx, restore, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "tocreate", Namespace: "test"}}

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPut, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/tocreate/status", request.URL.Path)

			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)

			createdRestore := &k8sv1.Restore{}
			require.NoError(t, json.Unmarshal(bytes, createdRestore))
			assert.Equal(t, "tocreate", createdRestore.Name)

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
		dClient := client.Restores("test")

		// when
		_, err = dClient.UpdateStatus(testCtx, restore, v1.UpdateOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/testrestore", request.URL.Path)

			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Restores("test")

		// when
		err = dClient.Delete(testCtx, "testrestore", v1.DeleteOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_DeleteCollection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodDelete, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores", request.URL.Path)
			assert.Equal(t, "labelSelector=test", request.URL.RawQuery)
			writer.Header().Add("content-type", "application/json")
			writer.WriteHeader(200)
		}))

		config := rest.Config{
			Host: server.URL,
		}
		client, err := NewForConfig(&config)
		require.NoError(t, err)
		dClient := client.Restores("test")

		// when
		err = dClient.DeleteCollection(testCtx, v1.DeleteOptions{}, v1.ListOptions{LabelSelector: "test"})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_Patch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPatch, request.Method)
			assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/testrestore", request.URL.Path)
			bytes, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte("test"), bytes)
			result, err := json.Marshal(k8sv1.Restore{})
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
		dClient := client.Restores("test")

		patchData := []byte("test")

		// when
		_, err = dClient.Patch(testCtx, "testrestore", types.JSONPatchType, patchData, v1.PatchOptions{})

		// then
		require.NoError(t, err)
	})
}

func Test_restoreClient_UpdateStatusXXX(t *testing.T) {
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
			restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "testrestore", Namespace: "test"}}

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch request.Method {
				case http.MethodGet:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/testrestore", request.URL.Path)
					assert.Equal(t, http.NoBody, request.Body)

					writer.Header().Add("content-type", "application/json")
					restore := &k8sv1.Restore{ObjectMeta: v1.ObjectMeta{Name: "testrestore", Namespace: "test"}}
					restoreBytes, err := json.Marshal(restore)
					require.NoError(t, err)
					_, err = writer.Write(restoreBytes)
					require.NoError(t, err)
					writer.WriteHeader(200)
				case http.MethodPut:
					assert.Equal(t, "/apis/k8s.cloudogu.com/v1/namespaces/test/restores/testrestore/status", request.URL.Path)
					bytes, err := io.ReadAll(request.Body)
					require.NoError(t, err)

					createdRestore := &k8sv1.Restore{}
					require.NoError(t, json.Unmarshal(bytes, createdRestore))
					assert.Equal(t, "testrestore", createdRestore.Name)
					assert.Equal(t, testCase.expectedStatus, createdRestore.Status.Status)

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
			dClient := client.Restores("test")

			// when
			returnValues := reflect.ValueOf(dClient).MethodByName(testCase.functionName).Call([]reflect.Value{reflect.ValueOf(testCtx), reflect.ValueOf(restore)})
			err, _ = returnValues[1].Interface().(error)

			// then
			require.NoError(t, err)
		})
	}
}
