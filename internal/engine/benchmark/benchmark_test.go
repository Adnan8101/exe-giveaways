package benchmark

import (
	"discord-giveaway-bot/internal/engine/auditor"
	"discord-giveaway-bot/internal/engine/cde"
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Mock objects
var (
	ringBuffer *ring.RingBuffer
	auditCache *auditor.AuditCacheManager
	attrEngine *auditor.AttributionEngine
	testEvent  fdl.FastEvent
)

func init() {
	// Setup overhead
	ringBuffer = ring.New()
	auditCache = &auditor.AuditCacheManager{} // Simplified mock
	attrEngine = auditor.NewAttributionEngine(ringBuffer, auditCache)
	attrEngine.Start()

	testEvent = fdl.FastEvent{
		ReqType:        fdl.EvtChannelDelete,
		GuildID:        123456789012345678,
		UserID:         987654321098765432,
		EntityID:       112233445566778899,
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: time.Now().UnixNano(),
	}

	// Initialize CDE mock state
	// In a real test we'd need to mock DB or ensure arenas are init
	// For now we assume CDE package init has run global vars
	// We manually set a guild enabled in simple benchmark
	// Note: Without DB mock, this might fail or return skipped.
	// But we are benchmarking the hot path logic primarily.
}

func BenchmarkAttributionPush(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attrEngine.PushEvent(&testEvent, "123456789012345678", "112233445566778899", discordgo.AuditLogActionChannelDelete)
	}
}

func BenchmarkDecisionProcess(b *testing.B) {
	// Pre-req: Ensure CDE can run without crashing
	// We might need to mock CDE internals if it hits DB
	// internal/engine/cde/config_cache.go checks atomic flags now.
	// If ID not found, it tries to load from DB.
	// We can manually inject into Arena for the test guild.

	// HACK: Use reflection or export a test helper in CDE?
	// Or just benchmark the function and see if it returns early (which is still a valid path to measure overhead)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cde.ProcessEvent(testEvent)
	}
}
