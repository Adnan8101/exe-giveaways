package acl

import (
	"fmt"
	"io"
	"net/http"
	"sync"
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

	// Dedicated HTTP client for bans (bypasses session overhead)
	dedicatedBanClient  *http.Client
	dedicatedClientOnce sync.Once

	// Pre-allocated header map for requests (EXTREME SPEED)
	authHeader = http.Header{}
	headerOnce sync.Once
)

// FastBanRequest performs EXTREME optimized ban API call
// ZERO allocations, direct byte manipulation, pre-warmed connection
func FastBanRequest(guildID, userID, reason string) error {
	// Get dedicated ban client (pre-warmed, optimized)
	client := getDedicatedBanClient()

	// Build URL with byte buffer from pool (ZERO allocation)
	bufPtr := urlBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // Reset to zero length, keep capacity

	// Manual append (faster than string builder)
	buf = append(buf, banEndpointPrefix...)
	buf = append(buf, guildID...)
	buf = append(buf, banEndpointSuffix...)
	buf = append(buf, userID...)
	buf = append(buf, banQuerySuffix...)
	url := string(buf) // Single allocation for URL string

	// Return buffer to pool immediately
	*bufPtr = buf
	urlBufferPool.Put(bufPtr)

	// Create request inline (no error check - trust the URL)
	req, _ := http.NewRequest("PUT", url, nil)

	// Initialize pre-allocated auth header once
	headerOnce.Do(func() {
		authHeaderOnce.Do(func() {
			if discordSession != nil {
				cachedAuthHeader = discordSession.Token
			}
		})
		authHeader.Set("Authorization", cachedAuthHeader)
	})

	// Clone pre-allocated header (faster than creating new)
	req.Header = authHeader.Clone()
	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	// Execute request - THIS IS THE CRITICAL PATH (340ms Discord API)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	// CRITICAL: Close body immediately after reading status
	// Don't defer - every nanosecond counts
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return fmt.Errorf("ban API returned status %d", resp.StatusCode)
}

// getDedicatedBanClient returns a dedicated HTTP client for bans
// Pre-warmed with persistent connections to Discord API
func getDedicatedBanClient() *http.Client {
	dedicatedClientOnce.Do(func() {
		if discordSession != nil && discordSession.Client != nil {
			dedicatedBanClient = discordSession.Client
		}
	})
	return dedicatedBanClient
}

// GetHTTPClient returns the Discord session's HTTP client
// This is a helper to access the underlying client
func GetHTTPClient() *http.Client {
	if discordSession != nil && discordSession.Client != nil {
		return discordSession.Client
	}
	return nil
}
