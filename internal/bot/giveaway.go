package bot

import (
	"discord-giveaway-bot/internal/models"
	"log"
	"sync"
	"time"
)

func (b *Bot) GiveawayTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Get giveaways that have ended
		now := models.Now()
		messageIDs, err := b.Redis.GetDueGiveaways(now)
		if err != nil {
			log.Printf("Error fetching due giveaways: %v", err)
			continue
		}

		if len(messageIDs) == 0 {
			continue
		}

		// Process ending giveaways concurrently
		var wg sync.WaitGroup

		for _, msgID := range messageIDs {
			wg.Add(1)
			go func(messageID string) {
				defer wg.Done()

				// Remove from queue first to prevent double processing
				if err := b.Redis.RemoveFromEndingQueue(messageID); err != nil {
					log.Printf("Error removing giveaway %s from queue: %v", messageID, err)
				}

				if err := b.Service.EndGiveaway(messageID); err != nil {
					log.Printf("Error ending giveaway %s: %v", messageID, err)
				}
			}(msgID)
		}

		// Wait for all concurrent endings to complete
		wg.Wait()
	}
}
