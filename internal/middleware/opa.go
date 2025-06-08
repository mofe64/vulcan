package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mofe64/vulkan/internal/config"
)

func NewOPAAuth(cfg config.VulkanConfig) gin.HandlerFunc {

	client := &http.Client{Timeout: cfg.OpaReqTIMEOUT}

	url := strings.TrimSuffix(cfg.OpaUrl, "/") + "/v1/" + cfg.OpaPolicy_Path

	return func(c *gin.Context) {
		body, err := buildOPAInput(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}

		allowed, err := queryOPA(c.Request.Context(), client, url, body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "policy check failed"})
			return
		}
		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// Example input document structure expected by OPA
// {
// 	"action":   "read" | "write",
// 	"resource": {
// 		"kind": "org" | "project" | "application",
// 		"org_id":     "org-123",
// 		"project_id": "proj-456",   // present for project/app
// 		"app_id":     "app-789"     // present for application
// 	},
// 	"subject": {
// 		// global roles that are not tied to a scope
// 		"roles": ["vulkan-admin"],
// 		// scoped roles that apply to a specific org / project / app
// 		"scoped_roles": [
// 			{ "role": "org-admin",     "org_id": "org-123" },
// 			{ "role": "project-read",  "project_id": "proj-456" },
// 			{ "role": "app-admin",     "app_id": "app-789" }
// 		]
// 	}
//   }

// buildOPAInput extracts data from gin.Context and marshals the input doc.
func buildOPAInput(c *gin.Context) ([]byte, error) {
	act := "read"
	if c.Request.Method != http.MethodGet {
		act = "write"
	}

	// Assume previous JWT middleware stored claims under "claims"
	claims, ok := c.Get("claims")
	if !ok {
		return nil, errors.New("claims not found in context")
	}

	input := map[string]any{
		"action": act,
		"resource": map[string]any{
			"kind":       c.Param("kind"), // e.g. "org", "project", "application"
			"org_id":     c.Param("org"),  // adjust to your router params
			"project_id": c.Param("proj"),
			"app_id":     c.Param("app"),
		},
		"subject": claims,
	}
	return json.Marshal(map[string]any{"input": input})
}

// queryOPA sends the input to OPA and returns its boolean result.
func queryOPA(ctx context.Context, client *http.Client, url string, body []byte) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		io.Copy(io.Discard, res.Body)
		return false, errors.New(res.Status)
	}
	var out struct {
		Result bool `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Result, nil
}
