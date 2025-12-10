package acl

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// API endpoint pool and request optimization
var (
	// Pre-computed API endpoint strings
	banEndpointPrefix = "https://discord.com/api/v10/guilds/"
	banEndpointSuffix = "/bans/"
	banQuerySuffix    = "?delete_message_seconds=0"

	// Cached authorization header
	cachedAuthHeader string
	authHeaderOnce   sync.Once

	// Pre-allocated byte buffer pool for URL construction (EXTREME SPEED)
	urlBufferPool = sync.Pool{
		New: func() interface{} {
			// Pre-allocate 256 bytes - enough for any Discord URL
			buf := make([]byte, 0, 256)
			return &buf
		},
	}

	// Ultra-optimized HTTP client with connection pooling
	ultraFastClient     *http.Client
	ultraFastClientOnce sync.Once

	// Pre-allocated header map for requests (EXTREME SPEED)
	authHeader = http.Header{}
	headerOnce sync.Once
)

// initUltraFastClient creates an HTTP client optimized for minimum latency
func initUltraFastClient() {
	ultraFastClientOnce.Do(func() {
		transport := &http.Transport{
			// Massive connection pooling for parallel requests
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 200,
			MaxConnsPerHost:     200,
			IdleConnTimeout:     90 * time.Second,

			// Ultra-fast TCP settings
			DialContext: (&net.Dialer{
				Timeout:   2 * time.Second, // Reduced from 5s
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,

			// Enable HTTP/2 for multiplexing
			ForceAttemptHTTP2: true,

			// Aggressive timeouts
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 8 * time.Second, // Reduced from 10s
			ExpectContinueTimeout: 500 * time.Millisecond,

			// Disable compression for speed
			DisableCompression: false,

			// Large buffers for throughput
			WriteBufferSize: 64 * 1024,
			ReadBufferSize:  64 * 1024,
		}

		ultraFastClient = &http.Client{
			Transport: transport,
			Timeout:   12 * time.Second, // Reduced from 15s
		}

		// Pre-warm connections
		go func() {
			for i := 0; i < 20; i++ {
				req, _ := http.NewRequest("HEAD", "https://discord.com/api/v10/gateway", nil)
				if discordSession != nil {
					req.Header.Set("Authorization", discordSession.Token)
				}
				resp, err := ultraFastClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	})
}

// FastBanRequest performs ULTRA-optimized ban API call
// Target: <150ms total latency including Discord API RTT
func FastBanRequest(guildID, userID, reason string) error {
	// Initialize client on first call
	initUltraFastClient()

	// Build URL with string builder (faster concatenation)
	var urlBuilder strings.Builder
	urlBuilder.Grow(100) // Pre-allocate
	urlBuilder.WriteString("https://discord.com/api/v10/guilds/")
	urlBuilder.WriteString(guildID)
	urlBuilder.WriteString("/bans/")
	urlBuilder.WriteString(userID)
	url := urlBuilder.String()

	// Static JSON body reader (reusable)
	body := strings.NewReader(`{"delete_message_seconds":0}`)

	// Create request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return err
	}

	// Initialize headers once
	headerOnce.Do(func() {
		if discordSession != nil {
			cachedAuthHeader = discordSession.Token
			authHeader.Set("Authorization", cachedAuthHeader)
		}
	})

	// Clone and set headers
	req.Header = authHeader.Clone()
	req.Header.Set("Content-Type", "application/json")
	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	// Execute with ultra-fast client
	resp, err := ultraFastClient.Do(req)
	if err != nil {
		return err
	}

	// Fast success path
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Async drain for connection reuse
		go func() {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
		return nil
	}

	// Error path
	body2, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	return fmt.Errorf("ban failed: %d - %s", resp.StatusCode, string(body2))
}

// FastKickRequest performs an optimized kick request
func FastKickRequest(guildID, userID, reason string) error {
	initUltraFastClient()

	var urlBuilder strings.Builder
	urlBuilder.Grow(100)
	urlBuilder.WriteString("https://discord.com/api/v10/guilds/")
	urlBuilder.WriteString(guildID)
	urlBuilder.WriteString("/members/")
	urlBuilder.WriteString(userID)
	url := urlBuilder.String()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header = authHeader.Clone()
	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	resp, err := ultraFastClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		go func() {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	return fmt.Errorf("kick failed: %d - %s", resp.StatusCode, string(body))
}
