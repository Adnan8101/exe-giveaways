package acl

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// API endpoint pool and request optimization
var (
	// Fasthttp client optimized for high concurrency
	fastClient *fasthttp.Client
	clientOnce sync.Once

	// Cached authorization header
	cachedAuthHeader []byte
	authHeaderOnce   sync.Once
)

// initFastClient creates an HTTP client optimized for minimum latency
func initFastClient() {
	clientOnce.Do(func() {
		fastClient = &fasthttp.Client{
			Name:                "AntiNuke-Bot",
			MaxConnsPerHost:     1000,
			MaxIdleConnDuration: 60 * time.Second,
			ReadTimeout:         5 * time.Second,
			WriteTimeout:        5 * time.Second,
			MaxResponseBodySize: 1024, // We don't expect large responses for bans
			// Optimize for speed
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		}
	})
}

// FastBanRequest performs ULTRA-optimized ban API call
// Target: <150ms total latency including Discord API RTT
func FastBanRequest(guildID, userID uint64, reason string) error {
	// Initialize client on first call
	initFastClient()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Construct URL using Append functions to avoid allocation
	// https://discord.com/api/v10/guilds/{guild.id}/bans/{user.id}
	uri := req.URI()
	uri.SetScheme("https")
	uri.SetHost("discord.com")
	
	// Build path manually to avoid allocations
	path := uri.PathOriginal()
	path = append(path[:0], "/api/v10/guilds/"...)
	path = strconv.AppendUint(path, guildID, 10)
	path = append(path, "/bans/"...)
	path = strconv.AppendUint(path, userID, 10)
	uri.SetPathBytes(path)

	req.Header.SetMethod("PUT")

	// Initialize headers once
	authHeaderOnce.Do(func() {
		if discordSession != nil {
			cachedAuthHeader = []byte(discordSession.Token)
		}
	})

	if len(cachedAuthHeader) > 0 {
		req.Header.SetBytesKV([]byte("Authorization"), cachedAuthHeader)
	}

	req.Header.SetContentType("application/json")
	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	// Body
	req.SetBodyString(`{"delete_message_seconds":0}`)

	// Execute with ultra-fast client
	err := fastClient.Do(req, resp)
	if err != nil {
		return err
	}

	// Fast success path
	statusCode := resp.StatusCode()
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Error path
	return fmt.Errorf("ban failed: %d - %s", statusCode, resp.Body())
}

// FastKickRequest performs an optimized kick request
func FastKickRequest(guildID, userID uint64, reason string) error {
	initFastClient()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Construct URL
	uri := req.URI()
	uri.SetScheme("https")
	uri.SetHost("discord.com")
	
	path := uri.PathOriginal()
	path = append(path[:0], "/api/v10/guilds/"...)
	path = strconv.AppendUint(path, guildID, 10)
	path = append(path, "/members/"...)
	path = strconv.AppendUint(path, userID, 10)
	uri.SetPathBytes(path)

	req.Header.SetMethod("DELETE")

	authHeaderOnce.Do(func() {
		if discordSession != nil {
			cachedAuthHeader = []byte(discordSession.Token)
		}
	})

	if len(cachedAuthHeader) > 0 {
		req.Header.SetBytesKV([]byte("Authorization"), cachedAuthHeader)
	}

	if reason != "" {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}

	err := fastClient.Do(req, resp)
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode()
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	return fmt.Errorf("kick failed: %d - %s", statusCode, resp.Body())
}
