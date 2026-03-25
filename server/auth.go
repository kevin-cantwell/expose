package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type githubUser struct {
	Login string `json:"login"`
}

// verifyGitHubToken calls the GitHub API to validate a token and return the username.
func verifyGitHubToken(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, body)
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decoding GitHub user: %w", err)
	}
	return user.Login, nil
}
