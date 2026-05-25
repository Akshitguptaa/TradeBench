package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tradebench/dummy-orderbook/api"
)

func TestDummyOrderbook_HealthEndpoint(t *testing.T) {
	router := api.SetupRouter()

	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")
	assert.Equal(t, "OK", rr.Body.String(), "handler returned unexpected body")
}

func TestDummyOrderbook_OrdersEndpoint(t *testing.T) {
	router := api.SetupRouter()

	req, err := http.NewRequest("GET", "/orders", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "handler returned wrong content type")

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "dummy orders", response["status"], "handler returned unexpected JSON status")
}
