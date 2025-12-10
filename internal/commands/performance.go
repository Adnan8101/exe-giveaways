package commands

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Performance command definition
var PerformanceCommand = &discordgo.ApplicationCommand{
	Name:        "performance",
	Description: "Display bot performance metrics and system stats (Admin only)",
}

// HandlePerformance displays bot performance metrics
func HandlePerformance(s *discordgo.Session, i *discordgo.InteractionCreate, bot interface{}) error {
	// Type assert to get the bot
	type PerfBot interface {
		GetPerfMonitor() interface{}
		GetSession() *discordgo.Session
	}

	var stats map[string]interface{}
	var wsLatency time.Duration

	// Try to get performance stats from bot if available
	if perfBot, ok := bot.(PerfBot); ok {
		monitor := perfBot.GetPerfMonitor()
		if statsGetter, ok := monitor.(interface{ GetStats() map[string]interface{} }); ok {
			stats = statsGetter.GetStats()
		}
		wsLatency = perfBot.GetSession().HeartbeatLatency()
	}

	// If stats not available, create from runtime
	if stats == nil {
		stats = getCurrentStats()
		wsLatency = s.HeartbeatLatency()
	}

	// Update WebSocket latency
	stats["ws_latency_ms"] = wsLatency.Milliseconds()

	// Calculate uptime
	uptimeDuration := time.Duration(stats["uptime_seconds"].(float64) * float64(time.Second))
	uptimeStr := formatDuration(uptimeDuration)

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Build performance embed
	embed := &discordgo.MessageEmbed{
		Title:       "🚀 Bot Performance Dashboard",
		Description: "Real-time performance metrics and system statistics",
		Color:       0x00ff00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏱️ Uptime",
				Value:  uptimeStr,
				Inline: false,
			},
			{
				Name:   "📊 Latency Metrics",
				Value:  formatLatencyMetrics(stats),
				Inline: false,
			},
			{
				Name:   "📈 Throughput",
				Value:  formatThroughput(stats),
				Inline: false,
			},
			{
				Name:   "💾 System Resources",
				Value:  formatSystemResources(stats, &m),
				Inline: false,
			},
			{
				Name:   "⚙️ Runtime Configuration",
				Value:  formatRuntimeConfig(&m),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Refresh this command to see updated metrics",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral, // Only visible to command user
		},
	})
}

func getCurrentStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"uptime_seconds":     0.0,
		"command_count":      uint64(0),
		"command_latency_us": int64(0),
		"event_count":        uint64(0),
		"event_latency_us":   int64(0),
		"rest_call_count":    uint64(0),
		"rest_latency_ms":    int64(0),
		"ws_latency_ms":      int64(0),
		"goroutines":         runtime.NumGoroutine(),
		"memory_alloc_mb":    m.Alloc / 1024 / 1024,
		"memory_sys_mb":      m.Sys / 1024 / 1024,
		"gc_count":           m.NumGC,
		"cpu_cores":          runtime.NumCPU(),
	}
}

func formatLatencyMetrics(stats map[string]interface{}) string {
	wsLatency := stats["ws_latency_ms"].(int64)
	restLatency := stats["rest_latency_ms"].(int64)
	cmdLatency := stats["command_latency_us"].(int64)
	eventLatency := stats["event_latency_us"].(int64)

	wsStatus := getStatusEmoji(wsLatency, 20, 10)
	restStatus := getStatusEmoji(restLatency, 150, 100)
	cmdStatus := getStatusEmoji(cmdLatency/1000, 5, 2)
	eventStatus := getStatusEmoji(eventLatency/1000, 1, 1)

	return fmt.Sprintf(
		"```"+
			"WebSocket:     %3dms  %s  (Target: <20ms)\n"+
			"REST API:      %3dms  %s  (Target: <150ms)\n"+
			"Command Exec:  %.2fms %s  (Target: <5ms)\n"+
			"Event Process: %.2fms %s  (Target: <1ms)"+
			"```",
		wsLatency, wsStatus,
		restLatency, restStatus,
		float64(cmdLatency)/1000.0, cmdStatus,
		float64(eventLatency)/1000.0, eventStatus,
	)
}

func formatThroughput(stats map[string]interface{}) string {
	commandCount := stats["command_count"].(uint64)
	eventCount := stats["event_count"].(uint64)
	restCallCount := stats["rest_call_count"].(uint64)

	return fmt.Sprintf(
		"```"+
			"Commands:      %10d\n"+
			"Events:        %10d\n"+
			"REST Calls:    %10d"+
			"```",
		commandCount, eventCount, restCallCount,
	)
}

func formatSystemResources(stats map[string]interface{}, m *runtime.MemStats) string {
	memAlloc := stats["memory_alloc_mb"].(uint64)
	memSys := stats["memory_sys_mb"].(uint64)
	goroutines := stats["goroutines"].(int)
	gcCount := stats["gc_count"].(uint32)

	// Get actual memory limit from runtime
	memLimit := debug.SetMemoryLimit(-1) / (1024 * 1024) // Convert to MB
	memPercent := float64(memAlloc) / float64(memLimit) * 100

	memStatus := "🟢"
	if memPercent > 80 {
		memStatus = "🔴"
	} else if memPercent > 60 {
		memStatus = "🟡"
	}

	return fmt.Sprintf(
		"```"+
			"Memory Alloc:  %5d MB  %s  (%.1f%% of %d MB)\n"+
			"Memory Sys:    %5d MB\n"+
			"Goroutines:    %5d\n"+
			"GC Count:      %5d\n"+
			"GC Pause:      %.2fms (last)"+
			"```",
		memAlloc, memStatus, memPercent, memLimit,
		memSys,
		goroutines,
		gcCount,
		float64(m.PauseNs[(m.NumGC+255)%256])/1000000,
	)
}

func formatRuntimeConfig(m *runtime.MemStats) string {
	gcPercent := debug.SetGCPercent(-1) // Get current value without changing
	debug.SetGCPercent(gcPercent)       // Restore it
	numCPU := runtime.NumCPU()
	gomaxprocs := runtime.GOMAXPROCS(0)
	goVersion := runtime.Version()
	memLimit := debug.SetMemoryLimit(-1) / (1024 * 1024 * 1024) // GB

	return fmt.Sprintf(
		"```"+
			"Go Version:    %s\n"+
			"CPU Cores:     %d\n"+
			"GOMAXPROCS:    %d\n"+
			"GC Percent:    %d\n"+
			"Memory Limit:  %d GB\n"+
			"Platform:      %s/%s"+
			"```",
		goVersion,
		numCPU,
		gomaxprocs,
		gcPercent,
		memLimit,
		runtime.GOOS, runtime.GOARCH,
	)
}

func getStatusEmoji(value int64, bad int64, good int64) string {
	if value > bad {
		return "🔴"
	} else if value > good {
		return "🟡"
	}
	return "🟢"
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
