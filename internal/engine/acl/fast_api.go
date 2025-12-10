package acl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// API endpoint pool and request optimization
var (
	// Pre-allocated buffer pools for API requests
	requestBodyPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// Pre-computed API endpoint strings
	banEndpointPrefix = "https://discord.com/api/v10/guilds/"
	banEndpointSuffix = "/bans/"

	// Cached authorization header
	cachedAuthHeader string
	authHeaderOnce   sync.Once
)

// FastBanRequest performs an ultra-optimized ban API call
// Uses pooled buffers and pre-computed strings to minimize allocations
func FastBanRequest(guildID, userID, reason string) error {
	// Get HTTP client from Discord session
	client := GetHTTPClient()
	if client == nil {
		return fmt.Errorf("no HTTP client available")
	}

	// Build URL with minimal allocations
	// Format: https://discord.com/api/v10/guilds/{guild.id}/bans/{user.id}
	url := banEndpointPrefix + guildID + banEndpointSuffix + userID + "?delete_message_seconds=0"

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}

	// Get authorization header (cached after first call)
	authHeaderOnce.Do(func() {
		if discordSession != nil {
			cachedAuthHeader = discordSession.Token
		}
	})

	// Add required headers (minimal set for speed)
	req.Header.Set("Authorization", cachedAuthHeader)
	req.Header.Set("Content-Type", "application/json")
	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Drain and discard body to reuse connection
	io.Copy(io.Discard, resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ban API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetHTTPClient returns the Discord session's HTTP client
// This is a helper to access the underlying client
func GetHTTPClient() *http.Client {
	if discordSession != nil && discordSession.Client != nil {
		return discordSession.Client
	}
	return nil
}
