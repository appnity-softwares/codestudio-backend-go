package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMVP_Minimal_e2e(t *testing.T) {
	// 1. Setup
	setupTestDB(t)
	r := setupSnippetRouter()

	// 2. Login (Create Test User returns token)
	token := createTestUser(t, "mvp_user", "USER")
	assert.NotEmpty(t, token, "Login/Register should return token")

	// 3. Create Snippet
	payload := map[string]interface{}{
		"title":       "MVP Deployment Check",
		"description": "Verifying core flow",
		"language":    "markdown",
		"code":        "# Ready",
		"tags":        []string{"deployment"},
		"visibility":  "public",
	}

	w := performRequest(r, "POST", "/api/snippets", payload, token)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	snippet := createResp["snippet"].(map[string]interface{})
	snippetID := snippet["id"].(string)

	// 4. Fetch Snippet List
	wList := performRequest(r, "GET", "/api/snippets", nil, "")
	assert.Equal(t, http.StatusOK, wList.Code)

	var listResp map[string]interface{}
	json.Unmarshal(wList.Body.Bytes(), &listResp)
	snippets := listResp["snippets"].([]interface{})

	// 5. Assert Snippet Exists
	found := false
	for _, s := range snippets {
		sMap := s.(map[string]interface{})
		if sMap["id"] == snippetID {
			found = true
			break
		}
	}
	assert.True(t, found, "Newly created snippet should be in the list")
}
