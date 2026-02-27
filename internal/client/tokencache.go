package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cachedToken struct {
	TokenID   string    `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CloudName string    `json:"cloud_name"`
}

func tokenCachePath(cloudName string) string {
	dir, _ := os.UserCacheDir()
	return filepath.Join(dir, "ostui", "token-"+cloudName+".json")
}

func LoadCachedToken(cloudName string) (string, bool) {
	path := tokenCachePath(cloudName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var ct cachedToken
	if err := json.Unmarshal(data, &ct); err != nil {
		return "", false
	}
	// Consider token valid if it expires more than 5 minutes from now
	if time.Until(ct.ExpiresAt) < 5*time.Minute {
		return "", false
	}
	return ct.TokenID, true
}

func SaveCachedToken(cloudName, tokenID string, expiresAt time.Time) error {
	path := tokenCachePath(cloudName)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	ct := cachedToken{TokenID: tokenID, ExpiresAt: expiresAt, CloudName: cloudName}
	data, err := json.Marshal(ct)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func ClearCachedToken(cloudName string) {
	os.Remove(tokenCachePath(cloudName))
}
