package core

import (
	"testing"
	"time"
)

func BenchmarkAtomicCache_GetConfig(b *testing.B) {
	cache := NewAtomicCache()

	// Pre-populate with test data
	cfg := &GuildConfig{
		GuildID:     "123456789",
		Enabled:     true,
		OwnerID:     "987654321",
		LogsChannel: "111222333",
	}
	cache.SetConfig(cfg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.GetConfig("123456789")
	}
}

func BenchmarkAtomicCache_IsWhitelisted(b *testing.B) {
	cache := NewAtomicCache()

	// Pre-populate with test whitelist
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = string(rune(i))
	}
	cache.SetWhitelist("guild123", ids)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.IsWhitelisted("guild123", "50")
	}
}

func BenchmarkAtomicCache_GetLimit(b *testing.B) {
	cache := NewAtomicCache()

	// Pre-populate with test limit
	limit := &LimitConfig{
		GuildID:       "guild123",
		ActionType:    "channel_create",
		Enabled:       true,
		LimitCount:    5,
		WindowSeconds: 10,
		Punishment:    "ban",
	}
	cache.SetLimit(limit)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.GetLimit("guild123", "channel_create")
	}
}

func BenchmarkRateLimiter_Check(b *testing.B) {
	limiter := NewRateLimiter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		limiter.Check("guild123", "channel_create", "user456", 10, 60)
	}
}

func BenchmarkFastRateLimiter_Check(b *testing.B) {
	limiter := NewFastRateLimiter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		limiter.Check("guild123", "channel_create", "user456", 10, 60)
	}
}

func BenchmarkRateLimiter_Concurrent(b *testing.B) {
	limiter := NewRateLimiter()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			guildID := "guild" + string(rune(i%10))
			userID := "user" + string(rune(i%100))
			limiter.Check(guildID, "channel_create", userID, 10, 60)
			i++
		}
	})
}

func TestRateLimiter_Basic(t *testing.T) {
	limiter := NewRateLimiter()

	// Should not trigger on first 5 requests
	for i := 0; i < 5; i++ {
		triggered, count := limiter.Check("guild1", "action1", "user1", 5, 10)
		if triggered {
			t.Errorf("Should not trigger on request %d", i+1)
		}
		if count != i+1 {
			t.Errorf("Expected count %d, got %d", i+1, count)
		}
	}

	// Should trigger on 6th request
	triggered, count := limiter.Check("guild1", "action1", "user1", 5, 10)
	if !triggered {
		t.Error("Should trigger on 6th request")
	}
	if count != 6 {
		t.Errorf("Expected count 6, got %d", count)
	}
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	limiter := NewRateLimiter()

	// Add 5 events
	for i := 0; i < 5; i++ {
		limiter.Check("guild1", "action1", "user1", 10, 1)
	}

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should not trigger (old events expired)
	triggered, count := limiter.Check("guild1", "action1", "user1", 10, 1)
	if triggered {
		t.Error("Should not trigger after window expired")
	}
	if count != 1 {
		t.Errorf("Expected count 1 after expiry, got %d", count)
	}
}

func TestRateLimiter_MultipleUsers(t *testing.T) {
	limiter := NewRateLimiter()

	// User1: 3 events
	for i := 0; i < 3; i++ {
		limiter.Check("guild1", "action1", "user1", 5, 10)
	}

	// User2: 4 events
	for i := 0; i < 4; i++ {
		limiter.Check("guild1", "action1", "user2", 5, 10)
	}

	// User1 should have 3 events
	triggered, count := limiter.Check("guild1", "action1", "user1", 5, 10)
	if triggered {
		t.Error("User1 should not trigger")
	}
	if count != 4 {
		t.Errorf("User1 expected count 4, got %d", count)
	}

	// User2 should have 4 events
	triggered, count = limiter.Check("guild1", "action1", "user2", 5, 10)
	if triggered {
		t.Error("User2 should not trigger")
	}
	if count != 5 {
		t.Errorf("User2 expected count 5, got %d", count)
	}
}

func TestAtomicCache_ConcurrentAccess(t *testing.T) {
	cache := NewAtomicCache()

	cfg := &GuildConfig{
		GuildID: "guild1",
		Enabled: true,
	}
	cache.SetConfig(cfg)

	// Spawn 100 goroutines reading config
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				_ = cache.GetConfig("guild1")
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestAtomicCache_WhitelistOperations(t *testing.T) {
	cache := NewAtomicCache()

	// Initially empty
	if cache.IsWhitelisted("guild1", "user1") {
		t.Error("Should not be whitelisted initially")
	}

	// Add to whitelist
	cache.AddToWhitelist("guild1", "user1")
	if !cache.IsWhitelisted("guild1", "user1") {
		t.Error("Should be whitelisted after add")
	}

	// Remove from whitelist
	cache.RemoveFromWhitelist("guild1", "user1")
	if cache.IsWhitelisted("guild1", "user1") {
		t.Error("Should not be whitelisted after remove")
	}
}
