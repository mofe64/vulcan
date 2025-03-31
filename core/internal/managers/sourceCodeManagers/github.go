package sourcecodemanagers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// github repo struct
type GithubRepository struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// github repository branch struct
type GithubBranch struct {
	Name string `json:"name"`
}

// github create webhook response struct
type WebhookResponse struct {
	ID int `json:"id"`
}

// ListUserRepos retrieves the repositories of the authenticated user using the PAT.
func ListUserRepos(pat string) ([]GithubRepository, error) {
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
	var repos []GithubRepository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// ListBranches retrieves all branches for a given repository.
func ListBranches(owner, repo, pat string) ([]GithubBranch, error) {
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
	var branches []GithubBranch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// ValidateBranch checks if the specified branch exists in the repository.
func ValidateBranch(owner, repo, branchName, pat string) (bool, error) {
	branches, err := ListBranches(owner, repo, pat)
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

// CreateWebhook creates a push-event webhook for the given repository.
// It returns the webhook ID if creation is successful.
func CreateWebhook(owner, repo, pat, webhookURL, secret string) (int, error) {
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

	var webhookResp WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return 0, err
	}
	return webhookResp.ID, nil
}

// DeleteWebhook deletes a webhook from the repository using its hook ID.
func DeleteWebhook(owner, repo, pat string, hookID int) error {
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

// DownloadRepo downloads the repository as a zip archive and extracts it to localDir.
func DownloadRepo(owner, repo, branch, pat, localDir string) error {
	// GitHub's archive API
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, branch)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+pat)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download repo, status: %d", resp.StatusCode)
	}

	// Write the zip to a temporary file
	zipFilePath := filepath.Join(localDir, repo+".zip")
	out, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return err
	}

	// Extract the zip file
	return unzip(zipFilePath, localDir)
}

// unzip extracts a zip archive to a destination directory.
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		// Ensure that the path is within the destination directory
		if !filepath.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		inFile, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			inFile.Close()
			return err
		}
		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
