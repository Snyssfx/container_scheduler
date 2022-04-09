package api

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Snyssfx/container_scheduler/internal/api/mock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestServer_calculateHandler(t *testing.T) {
	cm := mock.NewContainersMapMock(t)
	cm.CalculateMock.Set(func(ctx context.Context, seed int, input int) (i1 int, err error) {
		assert.Equal(t, 1234, seed)
		assert.Equal(t, 4321, input)
		return 3412, nil
	})

	s := &Server{containersMap: cm}
	req := httptest.NewRequest("GET", "/calculate/1234/4321", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/calculate/{seed}/{user_input}", s.calculateHandler)

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, []byte(`3412`), w.Body.Bytes())
}
