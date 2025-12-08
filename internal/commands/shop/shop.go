package shop

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type ShopCommand struct {
	DB      *database.Database
	Service *services.EconomyService
}

func NewShopCommand(db *database.Database, service *services.EconomyService) *ShopCommand {
	return &ShopCommand{
		DB:      db,
		Service: service,
	}
}

// /shop [page]
func (c *ShopCommand) Shop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	page := 1
	options := i.ApplicationCommandData().Options
	if len(options) > 0 {
		page = int(options[0].IntValue())
	}
	if page < 1 {
		page = 1
	}

	limit := 10
	offset := (page - 1) * limit

	items, err := c.DB.GetShopItems(limit, offset)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Failed to fetch shop items.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	totalItems, _ := c.DB.GetTotalShopItems()
	totalPages := (totalItems + limit - 1) / limit

	// Premium Embed Design
	embed := &discordgo.MessageEmbed{
		Title:       "🛒  **P R E M I U M   S T O R E**",
		Description: "Welcome to the exclusive server store. Browse our collection of premium items below.\n\n**How to Buy:**\nSelect an item from the dropdown menu below to purchase instantly.",
		Color:       0x2B2D31, // Dark sleek background color
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://media.discordapp.net/attachments/1234567890/1234567890/shop_icon.png", // Placeholder or use a nice default
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "https://media.discordapp.net/attachments/1311640533042106368/1311640685651857468/shop_banner.png?ex=6749961e&is=6748449e&hm=c5e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5e6e5", // Use a nice banner if available, or remove
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Page %d of %d • %d Items Available", page, totalPages, totalItems),
			IconURL: i.Member.User.AvatarURL(""),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Remove Image if not valid, just keeping it as a placeholder idea for "World Class"
	embed.Image = nil
	embed.Thumbnail = nil

	var selectOptions []discordgo.SelectMenuOption

	if len(items) == 0 {
		embed.Description = "🚫 **The shop is currently closed.**\nCheck back later for new items!"
		embed.Color = utils.ColorRed
	} else {
		for _, item := range items {
			stockStr := "∞"
			if item.Stock != -1 {
				stockStr = fmt.Sprintf("%d", item.Stock)
			}

			priceStr := fmt.Sprintf("%d %s", item.Price, utils.EmojiCoin)

			// Add field for each item
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("> **%s**", item.Name),
				Value:  fmt.Sprintf("` 🏷️ ` **Price:** %s\n` 📦 ` **Stock:** %s\n` 📝 ` *%s*", priceStr, stockStr, item.Description),
				Inline: false,
			})

			// Add to select menu
			selectOptions = append(selectOptions, discordgo.SelectMenuOption{
				Label:       item.Name,
				Value:       item.Name,
				Description: fmt.Sprintf("Buy for %d coins", item.Price),
				Emoji:       &discordgo.ComponentEmoji{Name: "🛒"},
			})
		}
	}

	components := []discordgo.MessageComponent{}

	// Navigation Buttons
	navRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Previous",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("shop_page_%d", page-1),
				Disabled: page <= 1,
			},
			discordgo.Button{
				Label:    "Next",
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("shop_page_%d", page+1),
				Disabled: page >= totalPages,
			},
			discordgo.Button{
				Label:    "Dashboard",
				Style:    discordgo.LinkButton,
				URL:      "https://example.com/dashboard", // Replace with actual dashboard if exists
				Disabled: true,                            // Disabled for now
			},
		},
	}
	components = append(components, navRow)

	// Buy Menu
	if len(selectOptions) > 0 {
		buyRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "shop_buy_select",
					Placeholder: "🛒 Select an item to buy...",
					Options:     selectOptions,
				},
			},
		}
		// Prepend buy row so it's above navigation or below? Usually below items is better.
		// Let's put Buy Row FIRST, then Navigation.
		components = []discordgo.MessageComponent{buyRow, navRow}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// /buy <item-name> [quantity]
func (c *ShopCommand) Buy(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	itemName := options[0].StringValue()
	quantity := 1
	if len(options) > 1 {
		quantity = int(options[1].IntValue())
	}

	if quantity < 1 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Quantity must be at least 1.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Item not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Check requirements
	if item.RequiredBalance > 0 {
		user, err := c.DB.GetEconomyUser(i.GuildID, i.Member.User.ID)
		if err != nil {
			// Handle error
			return
		}
		if user.Balance < int64(item.RequiredBalance) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("%s You need at least %d %s to buy this item.", utils.EmojiCross, item.RequiredBalance, utils.EmojiCoin),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	if item.RoleRequired != "" {
		hasRole := false
		for _, r := range i.Member.Roles {
			if r == item.RoleRequired {
				hasRole = true
				break
			}
		}
		if !hasRole {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: utils.EmojiCross + " You do not have the required role to buy this item.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	totalCost := int64(item.Price * quantity)

	// Use EconomyService to remove coins (this handles cache properly)
	err = c.Service.RemoveCoins(i.GuildID, i.Member.User.ID, totalCost)
	if err != nil {
		msg := utils.EmojiCross + " Transaction failed."
		if err.Error() == "insufficient funds" {
			msg = fmt.Sprintf("%s You do not have enough %s. Cost: %d", utils.EmojiCross, utils.EmojiCoin, totalCost)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Now update stock and add to inventory (no coin deduction here)
	err = c.DB.UpdateStockAndInventory(i.Member.User.ID, i.GuildID, item.ID, quantity)
	if err != nil {
		// Critical: coins deducted but inventory failed - try to refund
		_ = c.Service.AddCoins(i.GuildID, i.Member.User.ID, totalCost)

		msg := utils.EmojiCross + " Transaction failed."
		if err.Error() == "insufficient stock" {
			msg = utils.EmojiCross + " Not enough stock available."
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Handle Role Items
	if item.Type == "role" && item.RoleID != "" {
		err := s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, item.RoleID)
		if err != nil {
			// Log error but don't fail transaction (user paid)
			fmt.Printf("Failed to add role %s to user %s: %v\n", item.RoleID, i.Member.User.ID, err)
		}
	}

	// Generate redeem code for non-role items
	var redeemCode string
	if item.Type != "role" {
		redeemCode = generateRedeemCode()
		err = c.DB.CreateRedeemCode(redeemCode, item.ID, i.Member.User.ID, i.GuildID)
		if err != nil {
			fmt.Printf("Failed to create redeem code: %v\n", err)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       utils.EmojiTick + " Purchase Successful!",
		Description: fmt.Sprintf("You bought **%d x %s** for **%d** %s.", quantity, item.Name, totalCost, utils.EmojiCoin),
		Color:       utils.ColorGreen,
	}

	if item.ReplyMessage != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "📝 Message",
			Value: item.ReplyMessage,
		})
	}

	// Send redeem code
	if redeemCode != "" {
		// Send to DM (simple message without "view once" text)
		dm, err := s.UserChannelCreate(i.Member.User.ID)
		if err == nil {
			redeemEmbed := &discordgo.MessageEmbed{
				Title:       "🎫 Your Redeem Code",
				Description: fmt.Sprintf("**Item:** %s\n**Quantity:** %d\n**Price:** %d %s\n\n**Redeem Code:**\n```%s```\n\n💾 Make sure to save this code!", item.Name, quantity, totalCost, utils.EmojiCoin, redeemCode),
				Color:       0x3498DB,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Use this code to claim your purchase",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}
			s.ChannelMessageSendEmbed(dm.ID, redeemEmbed)
		}

		// Show code in channel (ephemeral with "view once" warning)
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "🎫 Redeem Code (View Once)",
			Value: fmt.Sprintf("```%s```\n⚠️ **IMPORTANT:** Save this code now! It's also sent to your DM.", redeemCode),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// /item-info <item-name>
func (c *ShopCommand) ItemInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	itemName := options[0].StringValue()

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Item not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	stockStr := "∞"
	if item.Stock != -1 {
		stockStr = strconv.Itoa(item.Stock)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "ℹ️ Item Information: " + item.Name,
		Description: item.Description,
		Color:       0x3498DB, // Blue
		Fields: []*discordgo.MessageEmbedField{
			{Name: "💰 Price", Value: fmt.Sprintf("%d %s", item.Price, utils.EmojiCoin), Inline: true},
			{Name: "📦 Stock", Value: stockStr, Inline: true},
			{Name: "🏷️ Type", Value: strings.Title(item.Type), Inline: true},
		},
	}

	if item.RoleID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Gives Role", Value: fmt.Sprintf("<@&%s>", item.RoleID), Inline: true,
		})
	}
	if item.Duration > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Duration", Value: fmt.Sprintf("%ds", item.Duration), Inline: true,
		})
	}
	if item.RequiredBalance > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "💎 Required Balance", Value: fmt.Sprintf("%d %s", item.RequiredBalance, utils.EmojiCoin), Inline: true,
		})
	}
	if item.ImageURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: item.ImageURL}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// /inventory
func (c *ShopCommand) Inventory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	items, err := c.DB.GetUserInventory(i.Member.User.ID, i.GuildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Failed to fetch inventory.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:     "🎒 Your Inventory",
		Color:     0xF1C40F, // Gold
		Timestamp: time.Now().Format(time.RFC3339),
		Footer:    &discordgo.MessageEmbedFooter{Text: i.Member.User.Username},
	}

	if len(items) == 0 {
		embed.Description = "You don't own any items yet. Use `/shop` to buy some!"
	} else {
		for _, item := range items {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s (x%d)", item.ItemName, item.Quantity),
				Value:  fmt.Sprintf("Type: %s | Acquired: <t:%d:R>", strings.Title(item.ItemType), item.AcquiredAt/1000),
				Inline: true,
			})
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *ShopCommand) HandleShopBuySelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}
	itemName := values[0]

	c.processBuy(s, i, itemName, 1)
}

func (c *ShopCommand) processBuy(s *discordgo.Session, i *discordgo.InteractionCreate, itemName string, quantity int) {
	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " Item not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Check requirements
	if item.RequiredBalance > 0 {
		user, err := c.DB.GetEconomyUser(i.GuildID, i.Member.User.ID)
		if err != nil {
			return
		}
		if user.Balance < int64(item.RequiredBalance) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("%s You need at least %d %s to buy this item.", utils.EmojiCross, item.RequiredBalance, utils.EmojiCoin),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	if item.RoleRequired != "" {
		hasRole := false
		for _, r := range i.Member.Roles {
			if r == item.RoleRequired {
				hasRole = true
				break
			}
		}
		if !hasRole {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: utils.EmojiCross + " You do not have the required role to buy this item.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	totalCost := int64(item.Price * quantity)

	// Use EconomyService to remove coins
	err = c.Service.RemoveCoins(i.GuildID, i.Member.User.ID, totalCost)
	if err != nil {
		msg := utils.EmojiCross + " Transaction failed."
		if err.Error() == "insufficient funds" {
			msg = fmt.Sprintf("%s You do not have enough %s. Cost: %d", utils.EmojiCross, utils.EmojiCoin, totalCost)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Update stock and inventory
	err = c.DB.UpdateStockAndInventory(i.Member.User.ID, i.GuildID, item.ID, quantity)
	if err != nil {
		_ = c.Service.AddCoins(i.GuildID, i.Member.User.ID, totalCost)
		msg := utils.EmojiCross + " Transaction failed."
		if err.Error() == "insufficient stock" {
			msg = utils.EmojiCross + " Not enough stock available."
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Handle Role Items
	if item.Type == "role" && item.RoleID != "" {
		err := s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, item.RoleID)
		if err != nil {
			fmt.Printf("Failed to add role %s to user %s: %v\n", item.RoleID, i.Member.User.ID, err)
		}
	}

	// Generate redeem code
	var redeemCode string
	if item.Type != "role" {
		redeemCode = generateRedeemCode()
		err = c.DB.CreateRedeemCode(redeemCode, item.ID, i.Member.User.ID, i.GuildID)
		if err != nil {
			fmt.Printf("Failed to create redeem code: %v\n", err)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       utils.EmojiTick + " Purchase Successful!",
		Description: fmt.Sprintf("You bought **%d x %s** for **%d** %s.", quantity, item.Name, totalCost, utils.EmojiCoin),
		Color:       utils.ColorGreen,
	}

	if item.ReplyMessage != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "📝 Message",
			Value: item.ReplyMessage,
		})
	}

	if redeemCode != "" {
		dm, err := s.UserChannelCreate(i.Member.User.ID)
		if err == nil {
			redeemEmbed := &discordgo.MessageEmbed{
				Title:       "🎫 Your Redeem Code",
				Description: fmt.Sprintf("**Item:** %s\n**Quantity:** %d\n**Price:** %d %s\n\n**Redeem Code:**\n```%s```\n\n💾 Make sure to save this code!", item.Name, quantity, totalCost, utils.EmojiCoin, redeemCode),
				Color:       0x3498DB,
				Footer:      &discordgo.MessageEmbedFooter{Text: "Use this code to claim your purchase"},
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			s.ChannelMessageSendEmbed(dm.ID, redeemEmbed)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "🎫 Redeem Code (View Once)",
			Value: fmt.Sprintf("```%s```\n⚠️ **IMPORTANT:** Save this code now! It's also sent to your DM.", redeemCode),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
