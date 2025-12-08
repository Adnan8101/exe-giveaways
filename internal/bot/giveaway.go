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
		giveaways, err := b.DB.GetAllActiveGiveaways()
		if err != nil {
			log.Printf("Error fetching active giveaways: %v", err)
			continue
		}

		// Process ending giveaways concurrently
		now := models.Now()
		var wg sync.WaitGroup

		for _, g := range giveaways {
			if g.EndTime <= now {
				wg.Add(1)
				go func(messageID string) {
					defer wg.Done()
					if err := b.Service.EndGiveaway(messageID); err != nil {
						log.Printf("Error ending giveaway %s: %v", messageID, err)
					}
				}(g.MessageID)
			}
		}

		// Wait for all concurrent endings to complete
		wg.Wait()
	}
}
