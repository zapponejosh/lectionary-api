package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/config"
	"github.com/zapponejosh/lectionary-api/internal/database"
)

// =============================================================================
// TEST SETUP HELPERS
// =============================================================================

// testEnv sets up a complete test environment with database, config, and handlers
type testEnv struct {
	db       *database.DB
	cfg      *config.Config
	handlers *Handlers
	adminKey string
	cleanup  func()
}

// setupTest creates a fresh test environment
func setupTest(t *testing.T) *testEnv {
	t.Helper()

	// Create in-memory database
	dbCfg := database.Config{
		Path:            ":memory:",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	db, err := database.Open(dbCfg, logger)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	// Run migrations
	ctx := context.Background()
	if _, err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	// Create app config with admin key
	adminKey := "admin-test-key-32-characters-minimum-length"
	cfg := &config.Config{
		Port:         8080,
		Env:          config.EnvDevelopment,
		DatabasePath: ":memory:",
		AdminAPIKey:  adminKey,
		LogLevel:     "error",
		LogFormat:    "text",
	}

	// Create handlers
	handlers := NewHandlers(db, cfg, logger)

	return &testEnv{
		db:       db,
		cfg:      cfg,
		handlers: handlers,
		adminKey: adminKey,
		cleanup: func() {
			db.Close()
		},
	}
}

// createTestUser creates a user and returns their API key
func (env *testEnv) createTestUser(t *testing.T, username string) (user *database.User, apiKey string) {
	t.Helper()
	ctx := context.Background()

	email := username + "@example.com"
	fullName := "Test User: " + username

	user, err := env.db.CreateUser(ctx, username, &email, &fullName)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	keyWithPlaintext, err := env.db.CreateAPIKey(ctx, user.ID, username+" Test Device")
	if err != nil {
		t.Fatalf("create test api key: %v", err)
	}

	return user, keyWithPlaintext.PlaintextKey
}

// makeRequest is a helper to make HTTP requests with optional API key
func makeRequest(method, path string, body interface{}, apiKey string) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonData)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	return req
}

// parseResponse parses JSON response
func parseResponse(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v, body: %s", err, rr.Body.String())
	}
}

// =============================================================================
// MIDDLEWARE TESTS
// =============================================================================

func TestAuthMiddleware_ValidKey(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	user, apiKey := env.createTestUser(t, "authuser")

	// Create a test handler that uses auth middleware
	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user from context
			u := GetUser(r)
			if u == nil {
				t.Error("User not found in context")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if u.ID != user.ID {
				t.Errorf("User.ID = %d, want %d", u.ID, user.ID)
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := makeRequest("GET", "/test", nil, apiKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_MissingKey(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := makeRequest("GET", "/test", nil, "") // No API key
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := makeRequest("GET", "/test", nil, "key_invalid123456789")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminOnlyMiddleware_ValidAdminKey(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	handler := AdminOnlyMiddleware(env.cfg, slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := makeRequest("GET", "/admin/test", nil, env.adminKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAdminOnlyMiddleware_UserKey(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	_, userKey := env.createTestUser(t, "notadmin")

	handler := AdminOnlyMiddleware(env.cfg, slog.Default())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := makeRequest("GET", "/admin/test", nil, userKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d (user key should not have admin access)", rr.Code, http.StatusForbidden)
	}
}

// =============================================================================
// ADMIN ENDPOINT TESTS
// =============================================================================

func TestCreateUser_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	reqBody := map[string]interface{}{
		"username":  "newuser",
		"email":     "newuser@example.com",
		"full_name": "New User",
	}

	req := makeRequest("POST", "/api/v1/admin/users", reqBody, env.adminKey)
	rr := httptest.NewRecorder()

	env.handlers.CreateUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d, body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    database.User `json:"data"`
	}
	parseResponse(t, rr, &resp)

	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if resp.Data.Username != "newuser" {
		t.Errorf("Username = %q, want %q", resp.Data.Username, "newuser")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	// Create first user
	env.createTestUser(t, "duplicate")

	// Try to create second user with same username
	reqBody := map[string]interface{}{
		"username": "duplicate",
		"email":    "different@example.com",
	}

	req := makeRequest("POST", "/api/v1/admin/users", reqBody, env.adminKey)
	rr := httptest.NewRecorder()

	env.handlers.CreateUser(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestCreateUser_MissingUsername(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	reqBody := map[string]interface{}{
		"email": "nouser@example.com",
	}

	req := makeRequest("POST", "/api/v1/admin/users", reqBody, env.adminKey)
	rr := httptest.NewRecorder()

	env.handlers.CreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestListUsers_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	// Create some users
	env.createTestUser(t, "user1")
	env.createTestUser(t, "user2")

	req := makeRequest("GET", "/api/v1/admin/users", nil, env.adminKey)
	rr := httptest.NewRecorder()

	env.handlers.ListUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Users []database.User `json:"users"`
			Count int             `json:"count"`
		} `json:"data"`
	}
	parseResponse(t, rr, &resp)

	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if resp.Data.Count != 2 {
		t.Errorf("Count = %d, want 2", resp.Data.Count)
	}
}

func TestCreateAPIKey_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	user, _ := env.createTestUser(t, "keytest")

	reqBody := map[string]interface{}{
		"name": "Test Device 2",
	}

	// Note: Using PathValue requires Go 1.22+ mux, so we'll test with the actual router
	// For now, we'll just test the handler logic
	req := makeRequest("POST", "/api/v1/admin/users/"+fmt.Sprintf("%d", user.ID)+"/keys", reqBody, env.adminKey)
	rr := httptest.NewRecorder()

	// Set path value manually for test
	req.SetPathValue("userID", fmt.Sprintf("%d", user.ID))

	env.handlers.CreateAPIKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d, body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey  database.APIKeyWithPlaintext `json:"api_key"`
			Warning string                       `json:"warning"`
		} `json:"data"`
	}
	parseResponse(t, rr, &resp)

	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if resp.Data.APIKey.PlaintextKey == "" {
		t.Error("PlaintextKey is empty")
	}
	if resp.Data.APIKey.Name != "Test Device 2" {
		t.Errorf("Name = %q, want %q", resp.Data.APIKey.Name, "Test Device 2")
	}
}

// =============================================================================
// USER ENDPOINT TESTS
// =============================================================================

func TestGetCurrentUser_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	user, apiKey := env.createTestUser(t, "currentuser")

	// Need to wrap handler with auth middleware
	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(env.handlers.GetCurrentUser),
	)

	req := makeRequest("GET", "/api/v1/me", nil, apiKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    database.User `json:"data"`
	}
	parseResponse(t, rr, &resp)

	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if resp.Data.ID != user.ID {
		t.Errorf("User.ID = %d, want %d", resp.Data.ID, user.ID)
	}
	if resp.Data.Username != user.Username {
		t.Errorf("Username = %q, want %q", resp.Data.Username, user.Username)
	}
}

func TestGetMyAPIKeys_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	user, apiKey := env.createTestUser(t, "keysuser")

	// Create additional key
	ctx := context.Background()
	_, err := env.db.CreateAPIKey(ctx, user.ID, "Second Device")
	if err != nil {
		t.Fatalf("create second key: %v", err)
	}

	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(env.handlers.GetMyAPIKeys),
	)

	req := makeRequest("GET", "/api/v1/me/keys", nil, apiKey)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			APIKeys []database.APIKey `json:"api_keys"`
			Count   int               `json:"count"`
		} `json:"data"`
	}
	parseResponse(t, rr, &resp)

	if !resp.Success {
		t.Error("Success = false, want true")
	}
	if resp.Data.Count != 2 {
		t.Errorf("Count = %d, want 2 (created 2 keys)", resp.Data.Count)
	}
}

func TestRevokeMyAPIKey_Success(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	user, apiKey := env.createTestUser(t, "revokeuser")

	// Create a second key to revoke
	ctx := context.Background()
	keyToRevoke, err := env.db.CreateAPIKey(ctx, user.ID, "Key To Revoke")
	if err != nil {
		t.Fatalf("create key to revoke: %v", err)
	}

	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(env.handlers.RevokeMyAPIKey),
	)

	req := makeRequest("DELETE", "/api/v1/me/keys/"+fmt.Sprintf("%d", keyToRevoke.ID), nil, apiKey)
	req.SetPathValue("keyID", fmt.Sprintf("%d", keyToRevoke.ID))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Verify key is revoked
	keys, err := env.db.ListUserAPIKeys(ctx, user.ID)
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}

	var found bool
	for _, k := range keys {
		if k.ID == keyToRevoke.ID {
			found = true
			if k.Active {
				t.Error("Key should be inactive after revocation")
			}
		}
	}
	if !found {
		t.Error("Revoked key not found in user's keys")
	}
}

func TestRevokeMyAPIKey_WrongUser(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	_, apiKey1 := env.createTestUser(t, "user1")
	user2, _ := env.createTestUser(t, "user2")

	// Create key for user2
	ctx := context.Background()
	user2Key, err := env.db.CreateAPIKey(ctx, user2.ID, "User2 Key")
	if err != nil {
		t.Fatalf("create user2 key: %v", err)
	}

	// User1 tries to revoke user2's key
	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(env.handlers.RevokeMyAPIKey),
	)

	req := makeRequest("DELETE", "/api/v1/me/keys/"+fmt.Sprintf("%d", user2Key.ID), nil, apiKey1)
	req.SetPathValue("keyID", fmt.Sprintf("%d", user2Key.ID))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d (user1 should not be able to revoke user2's key)", rr.Code, http.StatusNotFound)
	}

	// Verify user2's key is still active
	keys, err := env.db.ListUserAPIKeys(ctx, user2.ID)
	if err != nil {
		t.Fatalf("list user2 keys: %v", err)
	}

	for _, k := range keys {
		if k.ID == user2Key.ID && !k.Active {
			t.Error("User2's key should still be active")
		}
	}
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestFullAuthFlow(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup()

	// 1. Admin creates user
	createUserBody := map[string]interface{}{
		"username":  "flowtest",
		"email":     "flowtest@example.com",
		"full_name": "Flow Test User",
	}

	req := makeRequest("POST", "/api/v1/admin/users", createUserBody, env.adminKey)
	rr := httptest.NewRecorder()
	env.handlers.CreateUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Create user failed: %d", rr.Code)
	}

	var createUserResp struct {
		Success bool          `json:"success"`
		Data    database.User `json:"data"`
	}
	parseResponse(t, rr, &createUserResp)

	userID := createUserResp.Data.ID

	// 2. Admin creates API key for user
	createKeyBody := map[string]interface{}{
		"name": "Flow Test Device",
	}

	req = makeRequest("POST", "/api/v1/admin/users/"+fmt.Sprintf("%d", userID)+"/keys", createKeyBody, env.adminKey)
	req.SetPathValue("userID", fmt.Sprintf("%d", userID))
	rr = httptest.NewRecorder()
	env.handlers.CreateAPIKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Create key failed: %d", rr.Code)
	}

	var createKeyResp struct {
		Success bool `json:"success"`
		Data    struct {
			APIKey database.APIKeyWithPlaintext `json:"api_key"`
		} `json:"data"`
	}
	parseResponse(t, rr, &createKeyResp)

	userAPIKey := createKeyResp.Data.APIKey.PlaintextKey

	// 3. User authenticates with their key
	handler := AuthMiddleware(env.db, slog.Default())(
		http.HandlerFunc(env.handlers.GetCurrentUser),
	)

	req = makeRequest("GET", "/api/v1/me", nil, userAPIKey)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Get current user failed: %d", rr.Code)
	}

	var getUserResp struct {
		Success bool          `json:"success"`
		Data    database.User `json:"data"`
	}
	parseResponse(t, rr, &getUserResp)

	if getUserResp.Data.ID != userID {
		t.Errorf("Authenticated as wrong user: got %d, want %d", getUserResp.Data.ID, userID)
	}

	t.Logf("✓ Full auth flow test passed: admin created user, issued key, user authenticated")
}
