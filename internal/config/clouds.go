package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

// LoadAuthOptions loads the authentication options for the given cloud name
// from the clouds.yaml file. If cloudsPath is empty it defaults to
// $HOME/.config/openstack/clouds.yaml.
func LoadAuthOptions(cloudName, cloudsPath string) (gophercloud.AuthOptions, error) {
	if cloudsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return gophercloud.AuthOptions{}, fmt.Errorf("cannot determine home directory: %w", err)
		}
		cloudsPath = filepath.Join(home, ".config", "openstack", "clouds.yaml")
	}

	// Set OS_CLIENT_CONFIG_FILE to point to the custom clouds.yaml
	orig := os.Getenv("OS_CLIENT_CONFIG_FILE")
	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsPath)
	defer os.Setenv("OS_CLIENT_CONFIG_FILE", orig)

	// Build client options
	clientOpts := &clientconfig.ClientOpts{Cloud: cloudName}

	// Get gophercloud.AuthOptions
	authOptsPtr, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return gophercloud.AuthOptions{}, fmt.Errorf("failed to load auth options for cloud %q: %w", cloudName, err)
	}
	return *authOptsPtr, nil
}
