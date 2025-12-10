package auditor

import (
	"discord-giveaway-bot/internal/engine/ring"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AuditLogMonitor manages antinuke event detection
type AuditLogMonitor struct {
	session       *discordgo.Session
	eventRing     *ring.RingBuffer
	eventHandlers *EventHandlers
}

// New creates a new audit log monitor
func New(session *discordgo.Session, eventRing *ring.RingBuffer) *AuditLogMonitor {
	return &AuditLogMonitor{
		session:       session,
		eventRing:     eventRing,
		eventHandlers: NewEventHandlers(session, eventRing),
	}
}

// Start begins monitoring events
func (m *AuditLogMonitor) Start() {
	startTime := time.Now()
	log.Println("🚀 Starting Antinuke Event Monitor...")

	// Register all event handlers
	m.eventHandlers.RegisterAll()

	elapsed := time.Since(startTime)
	log.Printf("✅ Antinuke Event Monitor started in %v", elapsed)
	log.Println("📡 Now listening for:")
	log.Println("   • Channel operations (Create/Delete/Update)")
	log.Println("   • Role operations (Create/Delete/Update)")
	log.Println("   • Member actions (Ban/Kick)")
	log.Println("   • Webhook operations")
	log.Println("   • Guild modifications")
	log.Println("")
	log.Println("⚡ Detection mode: Real-time gateway events")
	log.Println("🎯 Target latency: <3ms end-to-end")
}
