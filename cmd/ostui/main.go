package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/v2"
	openstackV2 "github.com/gophercloud/gophercloud/v2/openstack"
	"log"
	"time"

	"golang.org/x/sync/errgroup"

	"ostui/internal/client"
	"ostui/internal/config"
	"ostui/internal/ui"
)

var (
	cloudName   string
	projectName string
	debug       bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ostui",
		Short: "OSTUI – OpenStack TUI management tool",
		RunE:  run,
	}

	rootCmd.PersistentFlags().StringVar(&cloudName, "cloud", os.Getenv("OS_CLOUD"), "Name of the cloud configuration in clouds.yaml")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVar(&projectName, "project", "", "Name of the project (optional)")
	_ = rootCmd.MarkPersistentFlagRequired("cloud")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if debug {
		fmt.Println("debug mode enabled")
	}

	// Load authentication options for the selected cloud
	cloudsPath := os.Getenv("OS_CLIENT_CONFIG_FILE")
	authOpts, err := config.LoadAuthOptions(cloudName, cloudsPath)
	if err != nil {
		return fmt.Errorf("failed to load cloud config: %w", err)
	}

	// Try to load cached token
	usedCache := false
	if tokenID, ok := client.LoadCachedToken(cloudName); ok {
		authOpts.TokenID = tokenID
		usedCache = true
	}

	// Authenticate with OpenStack (placeholder – further service clients can be created from this provider)
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil && usedCache {
		// Cached token likely invalid, clear and retry
		client.ClearCachedToken(cloudName)
		authOpts.TokenID = ""
		provider, err = openstack.AuthenticatedClient(authOpts)
	}
	if err != nil {
		return fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}

	// Create a v2 provider for DNS and Load Balancer services.
	var providerV2 *gophercloud.ProviderClient
	// Convert v1 AuthOptions to v2 AuthOptions.
	v2AuthOpts := gophercloud.AuthOptions{
		IdentityEndpoint: authOpts.IdentityEndpoint,
		Username:         authOpts.Username,
		UserID:           authOpts.UserID,
		Password:         authOpts.Password,
		Passcode:         authOpts.Passcode,
		DomainID:         authOpts.DomainID,
		DomainName:       authOpts.DomainName,
		TenantID:         authOpts.TenantID,
		TenantName:       authOpts.TenantName,
		AllowReauth:      authOpts.AllowReauth,
		TokenID:          authOpts.TokenID,
		// Scope omitted for simplicity.
	}
	providerV2, err = openstackV2.AuthenticatedClient(context.Background(), v2AuthOpts)
	if err != nil {
		log.Printf("warning: failed to create v2 provider for DNS/LB: %v", err)
		// Continue with nil DNS/LB clients.
	}

	// Create other service clients in parallel.
	var (
		computeClient  client.ComputeClient
		networkClient  client.NetworkClient
		storageClient  client.StorageClient
		identityClient client.IdentityClient
		imageClient    client.ImageClient
		limitsClient   client.LimitsClient
	)

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		var err error
		computeClient, err = client.NewComputeClient(authOpts)
		return err
	})
	g.Go(func() error {
		var err error
		networkClient, err = client.NewNetworkClient(authOpts)
		return err
	})
	g.Go(func() error {
		var err error
		storageClient, err = client.NewStorageClient(authOpts)
		return err
	})
	g.Go(func() error {
		var err error
		identityClient, err = client.NewIdentityClient(authOpts)
		return err
	})
	g.Go(func() error {
		var err error
		imageClient, err = client.NewImageClient(authOpts)
		return err
	})
	g.Go(func() error {
		var err error
		limitsClient, err = client.NewLimitsClient(authOpts)
		return err
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to create service clients: %w", err)
	}

	// Start the Bubble Tea TUI
	// Initialize DNS and Load Balancer clients, handling errors gracefully.
	var dnsClient client.DNSClient
	var lbClient client.LoadBalancerClient

	dnsClient, err = client.NewDNSClient(providerV2, gophercloud.EndpointOpts{})
	if err != nil {
		log.Printf("warning: failed to create DNS client: %v", err)
		dnsClient = nil
	}
	lbClient, err = client.NewLoadBalancerClient(providerV2, gophercloud.EndpointOpts{})
	if err != nil {
		log.Printf("warning: failed to create Load Balancer client: %v", err)
		lbClient = nil
	}

	// Save token to cache
	if tokenID := providerV2.Token(); tokenID != "" {
		expiresAt := time.Now().Add(1 * time.Hour) // fallback
		if tokenInfo, err := identityClient.GetTokenInfo(); err == nil && tokenInfo != nil {
			expiresAt = tokenInfo.ExpiresAt
		} else {
			log.Printf("warning: failed to get token expiry, using fallback: %v", err)
		}
		if err := client.SaveCachedToken(cloudName, tokenID, expiresAt); err != nil {
			log.Printf("warning: failed to save token cache: %v", err)
		}
	}
	// Start the Bubble Tea TUI
	p := tea.NewProgram(ui.NewModel(provider, computeClient, networkClient, storageClient, identityClient, imageClient, limitsClient, dnsClient, lbClient))

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	return nil
}

// UI model definitions
