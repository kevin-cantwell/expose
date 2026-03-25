package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	// GitHubClientID is the OAuth App client ID, set at build time or via env.
	// Override with EXPOSE_GITHUB_CLIENT_ID env var.
	defaultGitHubClientID = ""
)

// LoginCmd authenticates with GitHub using the device flow and saves the token.
type LoginCmd struct {
	ClientID string `help:"GitHub OAuth App client ID" env:"EXPOSE_GITHUB_CLIENT_ID" default:""`
}

func (c *LoginCmd) Run() error {
	clientID := c.ClientID
	if clientID == "" {
		clientID = defaultGitHubClientID
	}
	if clientID == "REPLACE_WITH_GITHUB_CLIENT_ID" {
		return fmt.Errorf("GitHub OAuth App client ID not set; set EXPOSE_GITHUB_CLIENT_ID env var or rebuild with the correct client ID")
	}

	// Step 1: request device code
	resp, err := http.PostForm("https://github.com/login/device/code", url.Values{
		"client_id": {clientID},
		"scope":     {"read:user"},
	})
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return fmt.Errorf("parsing device code response: %w", err)
	}
	deviceCode := vals.Get("device_code")
	userCode := vals.Get("user_code")
	verificationURI := vals.Get("verification_uri")
	interval := 5
	if i := vals.Get("interval"); i != "" {
		fmt.Sscanf(i, "%d", &interval)
	}

	fmt.Printf("\nOpen %s and enter code: %s\n\n", verificationURI, userCode)

	// Step 2: poll for token
	timeout := time.After(15 * time.Minute)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for authorization")
		case <-time.After(time.Duration(interval) * time.Second):
			token, err := pollDeviceToken(clientID, deviceCode)
			if err != nil {
				switch err.Error() {
				case "authorization_pending":
					// Normal — user hasn't authorized yet
				case "slow_down":
					// GitHub wants us to back off
					interval += 5
				default:
					return err
				}
				continue
			}
			if token != "" {
				if err := saveToken(token); err != nil {
					return err
				}
				fmt.Println("Authenticated successfully.")
				return nil
			}
		}
	}
}

func pollDeviceToken(clientID, deviceCode string) (string, error) {
	resp, err := http.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	vals, _ := url.ParseQuery(string(body))

	if errCode := vals.Get("error"); errCode != "" {
		return "", fmt.Errorf("%s", errCode)
	}
	return vals.Get("access_token"), nil
}

func saveToken(token string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	path := filepath.Join(dir, "token")
	data, _ := json.Marshal(map[string]string{"token": token})
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	fmt.Printf("Token saved to %s\n", path)
	return nil
}

// LoadToken reads the saved GitHub OAuth token.
func LoadToken() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "token"))
	if err != nil {
		return "", fmt.Errorf("not logged in: run 'expose login' first")
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("corrupted token file: run 'expose login'")
	}
	token := m["token"]
	if token == "" {
		return "", fmt.Errorf("empty token: run 'expose login'")
	}
	return token, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Honor XDG_CONFIG_HOME if set
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "expose"), nil
}

// TunnelsDir returns the path to the tunnels state directory.
func TunnelsDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tunnels"), nil
}

