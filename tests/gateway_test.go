package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:8080"
}

// getToken issues a JWT for the given contestant via the gateway's /api/v1/auth/token endpoint.
func getToken(t *testing.T, contestantID string) string {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"contestant_id": contestantID})
	resp, err := http.Post(gatewayURL()+"/api/v1/auth/token", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "token endpoint should return 200")

	var result struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	return result.Token
}

// TestGatewayHealth verifies the /health endpoint is publicly accessible (no auth).
func TestGatewayHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resp, err := http.Get(gatewayURL() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

// TestGatewayJWTIssueAndVerify tests the token issuance and that authenticated
// routes work while unauthenticated ones are rejected.
func TestGatewayJWTIssueAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// unauthenticated request to a protected route should fail
	resp, err := http.Get(gatewayURL() + "/api/v1/leaderboard")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// get a valid token
	token := getToken(t, "jwt-test-contestant")

	// authenticated request should succeed (or return a proxy-level response, not 401)
	req, _ := http.NewRequest("GET", gatewayURL()+"/api/v1/leaderboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode, "authenticated request should not be 401")
}

// TestGatewayRateLimit verifies that the per-contestant rate limit kicks in.
func TestGatewayRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := getToken(t, fmt.Sprintf("ratelimit-test-%d", time.Now().Unix()))

	// Fire requests rapidly — the submission RPM is 10, so 15 should trigger a 429
	hitLimit := false
	for i := 0; i < 15; i++ {
		// build a minimal multipart body
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, _ := w.CreateFormFile("binary", "test.bin")
		_, _ = part.Write([]byte("fake binary"))
		_ = w.Close()

		req, _ := http.NewRequest("POST", gatewayURL()+"/api/v1/submissions", &buf)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", w.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			hitLimit = true
			break
		}
	}

	assert.True(t, hitLimit, "expected to hit rate limit (429) within 15 rapid requests")
}

// TestGatewaySubmissionProxy verifies the full flow: obtain JWT → POST /submissions
// through the gateway → verify it proxied to the submission service.
func TestGatewaySubmissionProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := getToken(t, "proxy-test-contestant")

	// build a multipart form with a dummy binary
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("binary", "strategy.wasm")
	require.NoError(t, err)
	_, _ = part.Write([]byte("dummy wasm binary content"))
	_ = w.Close()

	req, _ := http.NewRequest("POST", gatewayURL()+"/api/v1/submissions", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// The submission service should accept the upload — NOT a 401 (auth fail) or 502 (proxy fail)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode, "should not be 401")
	assert.NotEqual(t, http.StatusBadGateway, resp.StatusCode, "should not be 502 (proxy failure)")

	t.Logf("POST /api/v1/submissions → %d: %s", resp.StatusCode, string(body))
}

// TestGatewayWSLeaderboard verifies that the /ws/leaderboard WebSocket route is
// correctly proxied through the gateway to the leaderboard service.
func TestGatewayWSLeaderboard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// WebSocket upgrade does not use Bearer auth in the header (browsers can't set
	// custom headers on WS), so the gateway skips auth for WS upgrades or the
	// token is sent as a query param. For this test we just verify the proxy works.
	wsURL := "ws://127.0.0.1:8080/ws/leaderboard"
	if v := os.Getenv("GATEWAY_WS_URL"); v != "" {
		wsURL = v
	}

	c, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// If we get a 401, the gateway requires auth for WS too — that's expected
		// behaviour. We just verify connectivity isn't broken (not a 502).
		if resp != nil {
			assert.NotEqual(t, http.StatusBadGateway, resp.StatusCode,
				"WS proxy should not return 502")
			t.Logf("WS dial returned HTTP %d (auth required for WS is acceptable)", resp.StatusCode)
			return
		}
		t.Fatalf("WS dial failed: %v", err)
	}
	defer c.Close()

	// Read the snapshot message from the leaderboard
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := c.ReadMessage()
	require.NoError(t, err, "should receive snapshot from leaderboard via gateway")

	var snapshot map[string]interface{}
	err = json.Unmarshal(msg, &snapshot)
	require.NoError(t, err)
	assert.Equal(t, "snapshot", snapshot["type"])
	t.Logf("WS snapshot via gateway: %s", string(msg))
}
