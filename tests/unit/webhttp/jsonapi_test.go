package webhttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondOne(t *testing.T) {
	w := httptest.NewRecorder()
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "users",
		ID:         "123",
		Attributes: map[string]string{"name": "test"},
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, webhttp.ContentTypeJSONAPI, w.Header().Get("Content-Type"))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "response missing 'data' object")
	assert.Equal(t, "users", data["type"])
	assert.Equal(t, "123", data["id"])
}

func TestRespondMany_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	webhttp.RespondMany(w, http.StatusOK, nil)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data, ok := body["data"].([]interface{})
	require.True(t, ok, "response 'data' should be an array")
	assert.Empty(t, data)
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	webhttp.RespondError(w, http.StatusNotFound, "badge not found")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	errors, ok := body["errors"].([]interface{})
	require.True(t, ok, "response missing 'errors' array")
	require.Len(t, errors, 1)

	errObj := errors[0].(map[string]interface{})
	assert.Equal(t, "404", errObj["status"])
	assert.Equal(t, "badge not found", errObj["detail"])
}

func TestRespondManyPaginated(t *testing.T) {
	resources := make([]webhttp.Resource, 15)
	for i := range resources {
		resources[i] = webhttp.Resource{Type: "test", ID: "id"}
	}

	w := httptest.NewRecorder()
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.Page{Offset: 5, Limit: 5})

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	data, ok := body["data"].([]interface{})
	require.True(t, ok, "response 'data' should be an array")
	assert.Len(t, data, 5)

	meta, ok := body["meta"].(map[string]interface{})
	require.True(t, ok, "response missing 'meta' object")
	assert.Equal(t, float64(15), meta["total"])
	assert.Equal(t, float64(5), meta["offset"])
	assert.Equal(t, float64(5), meta["limit"])
}
