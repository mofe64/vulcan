package sourcecodemanager_helpers

// Package sourcecodemanager_helpers provides functions to interact with GitHub's API

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	githubResponses "github.com/mofe64/vulcan/pkg/responses/github"
)

// ListGithubRepositoriesForUser retrieves the repositories of the authenticated user using the PAT.
func ListGithubRepositoriesForUser(pat string) ([]githubResponses.Repository, error) {
	url := "https://api.github.com/user/repos"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+pat)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list repos, status: %d", resp.StatusCode)
	}
	var repos []githubResponses.Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// ListBranchesInGithubRepository: retrieves all branches for a given github repository.
func ListBranchesInGithubRepository(owner, repo, pat string) ([]githubResponses.Branch, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+pat)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list branches, status: %d", resp.StatusCode)
	}
	var branches []githubResponses.Branch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// ValidateGithubBranchName checks if the specified branch exists in the repository.
func ValidateGithubBranchName(owner, repo, branchName, pat string) (bool, error) {
	branches, err := ListBranchesInGithubRepository(owner, repo, pat)
	if err != nil {
		return false, err
	}
	for _, branch := range branches {
		if branch.Name == branchName {
			return true, nil
		}
	}
	return false, nil
}

// CreateWebhookForGithubRepositorycreates a push-event webhook for the given repository.
// It returns the webhook ID if creation is successful.
func CreateWebhookForGithubRepository(owner, repo, pat, webhookURL, secret string) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", owner, repo)
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "token "+pat)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create webhook, status: %d, response: %s", resp.StatusCode, string(respBody))
	}

	var webhookResp githubResponses.WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return 0, err
	}
	return webhookResp.ID, nil
}

// DeleteWebhookForGithubRepository deletes a webhook from the repository using its hook ID.
func DeleteWebhookForGithubRepository(owner, repo, pat string, hookID int) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks/%d", owner, repo, hookID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+pat)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete webhook, status: %d", resp.StatusCode)
	}
	return nil
}
