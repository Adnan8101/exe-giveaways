package bot

import (
	"bytes"
	"context"
	"discord-giveaway-bot/internal/commands"
	"discord-giveaway-bot/internal/commands/antinuke"
	"discord-giveaway-bot/internal/commands/economy"
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/commands/voice"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) Ready(s *discordgo.Session, r *discordgo.Ready) {
	// Manually populate state user since state tracking is disabled
	if s.State.User == nil {
		s.State.User = r.User
	}

	log.Printf("Logged in as: %v#%v", r.User.Username, r.User.Discriminator)
	log.Printf("Serving %d guilds", len(r.Guilds))

	// Register commands for each guild to ensure instant updates
	log.Println("Registering guild commands...")
	for _, guild := range r.Guilds {
		_, err := s.ApplicationCommandBulkOverwrite(r.User.ID, guild.ID, commands.Commands)
		if err != nil {
			log.Printf("Failed to register commands for guild %s: %v", guild.ID, err)
		} else {
			log.Printf("Registered commands for guild %s", guild.ID)
		}
	}
}

func (b *Bot) GuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	log.Printf("Guild joined/loaded: %s (%s). Starting command registration...", g.Name, g.ID)

	// Warm AntiNuke cache for this guild
	// Warm AntiNuke cache for this guild
	// if b.AntiNukeV2 != nil {
	// 	b.AntiNukeV2.WarmCache(g.ID)
	// }

	// Register main bot commands
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, g.ID, commands.Commands)
	if err != nil {
		log.Printf("Failed to register commands for guild %s: %v", g.ID, err)
	} else {
		log.Printf("Registered giveaway commands for guild %s", g.Name)
	}
}

func (b *Bot) InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "gcreate":
			b.HandleGCreate(i)
		case "gend":
			b.HandleGEnd(i)
		case "greroll":
			b.HandleGReroll(i)
		case "glist":
			b.HandleGList(i)
		case "gcancel":
			b.HandleGCancel(i)
		// Economy Commands
		case "daily":
			economy.DailyHandler(s, i, b.EconomyService)
		case "weekly":
			economy.WeeklyHandler(s, i, b.EconomyService)
		case "hourly":
			economy.HourlyHandler(s, i, b.EconomyService)
		case "coins":
			economy.CoinsHandler(s, i, b.EconomyService)
		case "leaderboard":
			economy.LeaderboardHandler(s, i, b.EconomyService)
		case "invites":
			economy.InvitesHandler(s, i, b.EconomyService)
		case "coinflip":
			economy.CoinflipHandler(s, i, b.EconomyService)
		case "economy":
			economy.EconomyHandler(s, i, b.EconomyService)
		case "admin-coins":
			economy.AdminCoinsHandler(s, i, b.EconomyService)
		case "give":
			ctx := framework.NewSlashContext(s, i)
			economy.GiveCmd(ctx, b.EconomyService)
		case "help":
			commands.HandleHelp(s, i)
		case "ping":
			commands.HandlePing(s, i, b.DB, b.Redis)
		case "stats":
			commands.HandleStats(s, i, b.StartTime)
		case "performance":
			commands.HandlePerformance(s, i, b)
		// AntiNuke Commands
		case "panic": // Renamed from panic_mode in definition
			antinuke.HandlePanicMode(s, i, b.DB)
		// Voice Commands
		case "wv":
			voice.WhereVoiceHandler(s, i)
		case "drag":
			voice.DragHandler(s, i)
		case "to":
			voice.ToHandler(s, i)
		case "muteall":
			voice.MuteAllHandler(s, i)
		case "unmuteall":
			voice.UnmuteAllHandler(s, i)
		case "deafenall":
			voice.DeafenAllHandler(s, i)
		case "undeafenall":
			voice.UndeafenAllHandler(s, i)
		case "vcclear":
			voice.VCClearHandler(s, i)
		case "autodrag":
			voice.AutoDragHandler(s, i, b.DB)
		case "autoafk":
			voice.AutoAFKHandler(s, i, b.DB)
		// Shop Commands
		case "shop":
			b.ShopCommands.Shop(s, i)
		case "buy":
			b.ShopCommands.Buy(s, i)
		case "item-info":
			b.ShopCommands.ItemInfo(s, i)
		case "inventory":
			b.ShopCommands.Inventory(s, i)
		case "create-item":
			b.AdminShopCommands.CreateItem(s, i)
		case "edit-item":
			b.AdminShopCommands.EditItem(s, i)
		case "delete-item":
			b.AdminShopCommands.DeleteItem(s, i)
		case "give-item":
			b.AdminShopCommands.GiveItem(s, i)
		case "set-stock":
			b.AdminShopCommands.SetStock(s, i)
		case "item-options":
			b.AdminShopCommands.ItemOptions(s, i)
		case "check-redeem":
			b.AdminShopCommands.CheckRedeem(s, i)
		case "redeem-claimed":
			b.AdminShopCommands.RedeemClaimed(s, i)
			// AntiNuke Commands
		case "antinuke":
			antinuke.HandleAntiNuke(s, i, b.DB)
		case "setlimit":
			antinuke.HandleSetLimit(s, i, b.DB)
		case "punishment":
			antinuke.HandlePunishment(s, i, b.DB)
		case "whitelist":
			antinuke.HandleWhitelist(s, i, b.DB)
		case "logs":
			antinuke.HandleLogs(s, i, b.DB)
		}

	case discordgo.InteractionMessageComponent:
		// Handle buttons (in DM now)
		customID := i.MessageComponentData().CustomID
		if strings.HasPrefix(customID, "captcha_") {
			parts := strings.Split(customID, "_")
			if len(parts) < 3 {
				return
			}
			giveawayID, _ := strconv.ParseInt(parts[1], 10, 64)

			// Show modal
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: fmt.Sprintf("captcha_submit_%d", giveawayID),
					Title:    "Captcha Verification",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "captcha_code",
									Label:       "Enter the code from the image above",
									Style:       discordgo.TextInputShort,
									Placeholder: "Enter captcha code...",
									Required:    true,
									MaxLength:   10,
								},
							},
						},
					},
				},
			})
		} else if customID == "select_allowed_channels" {
			economy.HandleChannelSelect(s, i, b.EconomyService)
		} else if strings.HasPrefix(customID, "give_") {
			economy.HandleGiveButton(s, i, b.EconomyService)
		} else if customID == "shop_select_item" {
			b.AdminShopCommands.HandleEditItemSelect(s, i)
		} else if strings.HasPrefix(customID, "shop_edit_") {
			b.AdminShopCommands.HandleEditButton(s, i)
		} else if strings.HasPrefix(customID, "shop_role_select_") {
			b.AdminShopCommands.HandleEditRoleSelect(s, i)
		} else if customID == "help_category_select" {
			commands.HandleHelpSelect(s, i)
			// } else if strings.HasPrefix(customID, "whitelist_add_select_") {
			// 	antinuke.HandleWhitelistSelect(s, i, b.DB)
		}

	case discordgo.InteractionModalSubmit:
		// Handle modal submit
		customID := i.ModalSubmitData().CustomID
		if strings.HasPrefix(customID, "shop_modal_") {
			b.AdminShopCommands.HandleEditModal(s, i)
		} else if strings.HasPrefix(customID, "captcha_submit_") {
			parts := strings.Split(customID, "_")
			if len(parts) < 3 {
				return
			}
			giveawayID, _ := strconv.ParseInt(parts[2], 10, 64)

			inputCode := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

			// Fix panic: Check if Member is nil (DM)
			var userID string
			if i.Member != nil {
				userID = i.Member.User.ID
			} else if i.User != nil {
				userID = i.User.ID
			} else {
				log.Println("Could not determine user ID in modal submit")
				return
			}

			isValid, err := b.DB.VerifyCaptcha(userID, giveawayID, inputCode)
			if err != nil {
				log.Printf("Error verifying captcha: %v", err)
				return
			}

			if isValid {
				g, err := b.DB.GetGiveawayByID(giveawayID)
				if err != nil || g == nil || g.Ended {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ Giveaway not found or ended.",
						},
					})
					return
				}

				// Add participant
				b.DB.AddParticipant(giveawayID, userID)
				b.Service.UpdateGiveawayMessage(g)

				// We do NOT add reaction back because we never removed it.
				// Just confirm success.

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(" Captcha verified! You've successfully entered the giveaway for **%s**!", g.Prize),
					},
				})
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "❌ Invalid captcha code. Entry declined.",
					},
				})

				// Remove reaction from original message on failure
				g, err := b.DB.GetGiveawayByID(giveawayID)
				if err == nil && g != nil {
					s.MessageReactionRemove(g.ChannelID, g.MessageID, "🎉", userID)
				}
			}
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		switch i.ApplicationCommandData().Name {
		case "edit-item":
			b.AdminShopCommands.HandleAutocomplete(s, i)
		}
	}
}

func (b *Bot) MessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}

	if r.Emoji.Name != "🎉" {
		return
	}

	log.Printf("Reaction added by %s on message %s", r.UserID, r.MessageID)

	g, err := b.DB.GetGiveaway(r.MessageID)
	if err != nil {
		log.Printf("Error getting giveaway: %v", err)
		return
	}
	if g == nil {
		log.Printf("Giveaway not found for message %s", r.MessageID)
		return
	}
	if g.Ended {
		log.Printf("Giveaway %d ended", g.ID)
		return
	}

	log.Printf("Processing entry for giveaway %d (Fee: %d)", g.ID, g.EntryFee)

	// Check if already a participant (using fast prepared statement with context)
	ctx := context.Background()
	isParticipant, _ := b.DB.IsParticipantFast(ctx, g.ID, r.UserID)
	if isParticipant {
		log.Printf("User %s is already a participant", r.UserID)
		return
	}

	// Check requirements
	res, err := utils.CheckAllRequirements(s, b.DB, r.GuildID, r.UserID, g)
	if err != nil {
		log.Printf("Error checking requirements: %v", err)
		return
	}

	if !res.Passed {
		log.Printf("User %s failed requirements: %s", r.UserID, res.Reason)
		s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)
		dm, err := s.UserChannelCreate(r.UserID)
		if err == nil {
			s.ChannelMessageSend(dm.ID, fmt.Sprintf("❌ You cannot enter the giveaway for **%s**: %s", g.Prize, res.Reason))
		}
		return
	}

	// Captcha handling
	if g.CaptchaRequirement {
		// Do NOT remove reaction yet.

		captcha, err := utils.GenerateCaptcha()
		if err != nil {
			log.Printf("Error generating captcha: %v", err)
			return
		}

		b.DB.CreateCaptchaSession(r.UserID, g.ID, captcha.Code)

		// Create DM channel
		dm, err := s.UserChannelCreate(r.UserID)
		if err != nil {
			log.Printf("Failed to create DM channel: %v", err)
			// If we can't DM, we must remove reaction and maybe try to tell them?
			s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)
			return
		}

		btn := discordgo.Button{
			Label:    "Enter Captcha Code",
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("captcha_%d_%s", g.ID, r.UserID),
			Emoji:    &discordgo.ComponentEmoji{Name: "✍️"},
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Captcha Verification",
			Description: fmt.Sprintf("Please solve the captcha to enter the giveaway for **%s**.\nYou have **1 minute**.", g.Prize),
			Color:       0x0000FF,
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://captcha.png",
			},
		}

		msg := &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Files: []*discordgo.File{
				{
					Name:   "captcha.png",
					Reader: bytes.NewReader(captcha.Image),
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{btn},
				},
			},
		}

		m, err := s.ChannelMessageSendComplex(dm.ID, msg)
		if err != nil {
			log.Printf("Failed to send captcha DM: %v", err)
			// Remove reaction if we can't send DM
			s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)
			return
		}

		// 1 minute timeout handler
		go func() {
			time.Sleep(1 * time.Minute)

			isPart, _ := b.DB.IsParticipant(g.ID, r.UserID)
			if !isPart {
				// Timeout!
				s.ChannelMessageDelete(dm.ID, m.ID)
				embed := &discordgo.MessageEmbed{
					Description: fmt.Sprintf("❌ Time expired! Entry declined for **%s**.", g.Prize),
					Color:       0xFF0000,
				}
				s.ChannelMessageSendEmbed(dm.ID, embed)

				// Remove reaction
				s.MessageReactionRemove(r.ChannelID, r.MessageID, "🎉", r.UserID)
			}
		}()

		return
	}

	// Entry Fee Handling
	if g.EntryFee > 0 {
		log.Printf("Checking entry fee for user %s (Balance check)", r.UserID)
		// Use Service to get balance (checks cache)
		balance, err := b.EconomyService.GetUserBalance(r.GuildID, r.UserID)
		if err != nil {
			log.Printf("Error getting economy user: %v", err)
			return
		}

		if balance < int64(g.EntryFee) {
			log.Printf("User %s has insufficient funds: %d < %d", r.UserID, balance, g.EntryFee)
			s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)

			config, err := b.EconomyService.GetConfig(r.GuildID)
			emoji := "<:Cash:1443554334670327848>"
			if err == nil {
				emoji = config.CurrencyEmoji
			}

			dm, err := s.UserChannelCreate(r.UserID)
			if err == nil {
				embed := &discordgo.MessageEmbed{
					Description: fmt.Sprintf("❌ You need **%d** %s to enter this giveaway. You have **%d**.", g.EntryFee, emoji, balance),
					Color:       0xFF0000,
				}
				s.ChannelMessageSendEmbed(dm.ID, embed)
			}
			return
		}

		// Deduct fee using Service (updates cache)
		log.Printf("Deducting %d coins from user %s", g.EntryFee, r.UserID)
		err = b.EconomyService.RemoveCoins(r.GuildID, r.UserID, int64(g.EntryFee))
		if err != nil {
			log.Printf("Failed to deduct fee: %v", err)
			return
		}

		// Notify user of deduction
		dm, err := s.UserChannelCreate(r.UserID)
		if err == nil {
			config, err := b.EconomyService.GetConfig(r.GuildID)
			emoji := "<:Cash:1443554334670327848>"
			if err == nil {
				emoji = config.CurrencyEmoji
			}

			embed := &discordgo.MessageEmbed{
				Description: fmt.Sprintf("✅ **%d** %s have been deducted for entering the giveaway **%s**.", g.EntryFee, emoji, g.Prize),
				Color:       0x00FF00,
			}
			s.ChannelMessageSendEmbed(dm.ID, embed)
		}
	}

	// Add participant (using fast prepared statement with context)
	log.Printf("Adding participant %s to giveaway %d", r.UserID, g.ID)
	ctx = context.Background()
	err = b.DB.AddParticipantFast(ctx, g.ID, r.UserID)
	if err != nil {
		log.Printf("Failed to add participant: %v", err)
	} else {
		log.Printf("Successfully added participant %s", r.UserID)

		// Assign Role Handling
		if g.AssignRole != "" {
			err := s.GuildMemberRoleAdd(r.GuildID, r.UserID, g.AssignRole)
			if err != nil {
				log.Printf("Failed to assign role %s to user %s: %v", g.AssignRole, r.UserID, err)
			}
		}
	}
	b.Service.UpdateGiveawayMessage(g)
}

func (b *Bot) MessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	if r.Emoji.Name != "🎉" {
		return
	}

	g, err := b.DB.GetGiveaway(r.MessageID)
	if err != nil || g == nil || g.Ended {
		return
	}

	// Check if user was a participant (using fast prepared statement with context)
	ctx := context.Background()
	isParticipant, err := b.DB.IsParticipantFast(ctx, g.ID, r.UserID)
	if err != nil {
		log.Printf("Error checking participant status: %v", err)
		return
	}
	if !isParticipant {
		return // User was not a participant, so no refund needed
	}

	// Remove participant (using fast prepared statement with context)
	ctx = context.Background()
	err = b.DB.RemoveParticipantFast(ctx, g.ID, r.UserID)
	if err != nil {
		log.Printf("Error removing participant: %v", err)
		return
	}

	// Remove Assigned Role
	if g.AssignRole != "" {
		err := s.GuildMemberRoleRemove(r.GuildID, r.UserID, g.AssignRole)
		if err != nil {
			log.Printf("Failed to remove role %s from user %s: %v", g.AssignRole, r.UserID, err)
		}
	}

	// Refund Fee Handling
	if g.EntryFee > 0 {
		refundCount, err := b.DB.GetRefundCount(g.ID, r.UserID)
		if err != nil {
			log.Printf("Error getting refund count: %v", err)
		}

		if refundCount < 3 {
			user, err := b.DB.GetEconomyUser(r.GuildID, r.UserID)
			if err == nil {
				// 80% Refund Logic
				refundAmount := int64(float64(g.EntryFee) * 0.8)

				user.Balance += refundAmount
				user.TotalSpent -= refundAmount // Revert spend (partially?) Or maybe just add to balance?
				// Usually TotalSpent tracks actual spend. If we refund 80%, they effectively spent 20%.
				// So we should decrease TotalSpent by the refundAmount too?
				// Or maybe we shouldn't touch TotalSpent?
				// Let's decrease TotalSpent by refundAmount so it reflects net spend.
				user.TotalSpent -= refundAmount

				b.DB.UpdateEconomyUser(user)
				b.DB.IncrementRefundCount(g.ID, r.UserID)

				dm, err := s.UserChannelCreate(r.UserID)
				if err == nil {
					config, err := b.EconomyService.GetConfig(r.GuildID)
					emoji := "<:Cash:1443554334670327848>"
					if err == nil {
						emoji = config.CurrencyEmoji
					}

					msg := fmt.Sprintf("✅ You left the giveaway for **%s**. **%d** %s (80%%) have been refunded. (%d/3 refunds used)", g.Prize, refundAmount, emoji, refundCount+1)
					if refundCount+1 == 3 {
						msg += "\n⚠️ **Warning:** This was your last refund. No refund will be provided next time."
					}
					embed := &discordgo.MessageEmbed{
						Description: msg,
						Color:       0x00FF00,
					}
					s.ChannelMessageSendEmbed(dm.ID, embed)
				}
			}
		} else {
			dm, err := s.UserChannelCreate(r.UserID)
			if err == nil {
				embed := &discordgo.MessageEmbed{
					Description: fmt.Sprintf("⚠️ You left the giveaway for **%s**, but you have exceeded the refund limit (3/3). No coins refunded.", g.Prize),
					Color:       0xFFA500, // Orange
				}
				s.ChannelMessageSendEmbed(dm.ID, embed)
			}
		}
	}

	b.Service.UpdateGiveawayMessage(g)
}
