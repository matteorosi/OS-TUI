package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAuthOptions_ValidCloud(t *testing.T) {
	tmpDir := t.TempDir()
	cloudsPath := filepath.Join(tmpDir, "clouds.yaml")
	yamlContent := `
clouds:
  testcloud:
    auth:
      auth_url: http://example.com:5000/v3
      username: testuser
      password: testpass
      project_name: testproject
      domain_name: default
`
	if err := os.WriteFile(cloudsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("write clouds.yaml: %v", err)
	}

	opts, err := LoadAuthOptions("testcloud", cloudsPath)
	if err != nil {
		t.Fatalf("LoadAuthOptions returned error: %v", err)
	}
	if opts.IdentityEndpoint != "http://example.com:5000/v3" {
		t.Errorf("unexpected AuthURL: %s", opts.IdentityEndpoint)
	}
	if opts.Username != "testuser" {
		t.Errorf("unexpected Username: %s", opts.Username)
	}
	if opts.Password != "testpass" {
		t.Errorf("unexpected Password: %s", opts.Password)
	}
	if opts.TenantName != "testproject" {
		t.Errorf("unexpected ProjectName: %s", opts.TenantName)
	}
	if opts.DomainName != "default" {
		t.Errorf("unexpected DomainName: %s", opts.DomainName)
	}
}

func TestLoadAuthOptions_InvalidCloud(t *testing.T) {
	tmpDir := t.TempDir()
	cloudsPath := filepath.Join(tmpDir, "clouds.yaml")
	yamlContent := `
clouds:
  othercloud:
    auth:
      auth_url: http://example.com:5000/v3
      username: user
      password: pass
      project_name: proj
      domain_name: default
`
	if err := os.WriteFile(cloudsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("write clouds.yaml: %v", err)
	}

	_, err := LoadAuthOptions("testcloud", cloudsPath)
	if err == nil {
		t.Fatalf("expected error for unknown cloud, got nil")
	}
}

func TestLoadAuthOptions_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Set HOME to temporary directory
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Create .config/openstack/clouds.yaml
	configDir := filepath.Join(tmpDir, ".config", "openstack")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cloudsPath := filepath.Join(configDir, "clouds.yaml")
	yamlContent := `
clouds:
  testcloud:
    auth:
      auth_url: http://example.com:5000/v3
      username: testuser
      password: testpass
      project_name: testproject
      domain_name: default
`
	if err := os.WriteFile(cloudsPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("write clouds.yaml: %v", err)
	}

	opts, err := LoadAuthOptions("testcloud", "")
	if err != nil {
		t.Fatalf("LoadAuthOptions returned error: %v", err)
	}
	if opts.IdentityEndpoint != "http://example.com:5000/v3" {
		t.Errorf("unexpected AuthURL: %s", opts.IdentityEndpoint)
	}
}
