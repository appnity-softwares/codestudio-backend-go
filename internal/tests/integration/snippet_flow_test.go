package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/routes"
	"github.com/stretchr/testify/assert"
)

func setupSnippetRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Mimic main.go structure
	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		routes.RegisterAuthRoutes(auth)

		routes.RegisterSnippetRoutes(api)
		// Add others if needed for side effects, e.g. User routes for profile check
		routes.RegisterUserRoutes(api)
	}

	return r
}

func TestSnippetFlow(t *testing.T) {
	// 1. Setup
	setupTestDB(t)
	r := setupSnippetRouter()

	// 2. Create User A
	tokenA := createTestUser(t, "author", "USER")

	// 3. Create Snippet
	snippetID := createTestSnippet(t, r, tokenA)
	assert.NotEmpty(t, snippetID)

	// 4. List Snippets (Search)
	verifySnippetInList(t, r, snippetID)

	// 5. Get Snippet Detail
	verifySnippetDetail(t, r, snippetID, tokenA)

	// 6. Update Snippet
	updateTestSnippet(t, r, snippetID, tokenA)

	// 8. Delete Snippet
	deleteSnippet(t, r, snippetID, tokenA)

	// Verify Gone
	verifySnippetGone(t, r, snippetID, tokenA)
}

// --- Helpers ---

func createTestSnippet(t *testing.T, r *gin.Engine, token string) string {
	payload := map[string]interface{}{
		"title":       "Hello World",
		"description": "Prints hello world",
		"language":    "python",
		"code":        "print('Hello World')",
		"tags":        []string{"python", "test"},
		"visibility":  "public",
	}

	w := performRequest(r, "POST", "/api/snippets", payload, token)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Handler returns the snippet object directly
	return resp["id"].(string)
}

func verifySnippetInList(t *testing.T, r *gin.Engine, snippetID string) {
	w := performRequest(r, "GET", "/api/snippets?search=Hello", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)

	t.Logf("List Response: %s", w.Body.String())

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	if resp["snippets"] == nil {
		t.Fatal("snippets field is nil in response")
	}

	snippets, ok := resp["snippets"].([]interface{})
	if !ok {
		t.Fatalf("snippets is not a list: %T", resp["snippets"])
	}
	assert.True(t, len(snippets) > 0, "Snippets list should not be empty")

	found := false
	for _, s := range snippets {
		if sMap, ok := s.(map[string]interface{}); ok {
			if sMap["id"] == snippetID {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Snippet %s should be in list", snippetID)
}

func verifySnippetDetail(t *testing.T, r *gin.Engine, snippetID, token string) {
	w := performRequest(r, "GET", "/api/snippets/"+snippetID, nil, token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Hello World", resp["title"])
}

func updateTestSnippet(t *testing.T, r *gin.Engine, snippetID, token string) {
	payload := map[string]interface{}{
		"title": "Hello Universe",
	}
	w := performRequest(r, "PUT", "/api/snippets/"+snippetID, payload, token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Hello Universe", resp["title"])
}

func deleteSnippet(t *testing.T, r *gin.Engine, snippetID, token string) {
	w := performRequest(r, "DELETE", "/api/snippets/"+snippetID, nil, token)
	assert.Equal(t, http.StatusOK, w.Code)
}

func verifySnippetGone(t *testing.T, r *gin.Engine, snippetID, token string) {
	w := performRequest(r, "GET", "/api/snippets/"+snippetID, nil, token)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
