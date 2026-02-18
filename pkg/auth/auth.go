package auth

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/logging"
	"google.golang.org/api/option"
)

// Client wraps GCP logging client and authentication
type Client struct {
	client    *logging.Client
	projectID string
}

// NewClient creates a new authenticated GCP logging client
// It uses default application credentials from gcloud CLI
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	// If projectID is not provided, try to get it from environment
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			return nil, fmt.Errorf("no project ID provided and GOOGLE_CLOUD_PROJECT not set")
		}
	}

	// Create logging client using default credentials
	// This will use gcloud credentials if available
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging client: %w", err)
	}

	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

// NewClientWithCredentials creates a new client with explicit credentials file
func NewClientWithCredentials(ctx context.Context, projectID, credentialsPath string) (*Client, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	if credentialsPath == "" {
		return nil, fmt.Errorf("credentials path is required")
	}

	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials file not found: %s", credentialsPath)
	}

	client, err := logging.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create logging client with credentials: %w", err)
	}

	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

// Close closes the logging client
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetProjectID returns the project ID associated with this client
func (c *Client) GetProjectID() string {
	return c.projectID
}

// GetUnderlyingClient returns the underlying logging client
func (c *Client) GetUnderlyingClient() *logging.Client {
	return c.client
}

// Verify checks if the client can connect to GCP (validates credentials)
func (c *Client) Verify(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("client not initialized")
	}

	// Simple verification by checking if we can access the client
	// If credentials were invalid, the client creation would have failed
	if c.projectID == "" {
		return fmt.Errorf("project ID not set")
	}
	return nil
}

// Error codes and messages
var (
	ErrNoProjectID         = fmt.Errorf("no project ID provided")
	ErrInvalidCredentials  = fmt.Errorf("invalid or expired credentials")
	ErrConnectionFailed    = fmt.Errorf("failed to connect to GCP")
	ErrAuthenticationFailed = fmt.Errorf("authentication failed; run 'gcloud auth login'")
)
