package shop

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type AdminShopCommand struct {
	DB *database.Database
}

func NewAdminShopCommand(db *database.Database) *AdminShopCommand {
	return &AdminShopCommand{DB: db}
}

func (c *AdminShopCommand) isAdmin(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if i.Member.Permissions&discordgo.PermissionManageServer != 0 {
		return true
	}
	// TODO: Check for Bot Commander role if configured
	return false
}

// /create-item [name]
func (c *AdminShopCommand) CreateItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	name := "New Item"
	if len(options) > 0 {
		name = options[0].StringValue()
	}

	// Create default item
	item := &models.ShopItem{
		Name:        name,
		Description: "No description set.",
		Price:       100,
		Stock:       -1,
		Type:        "item",
		Hidden:      false, // Visible by default
	}

	id, err := c.DB.CreateShopItem(item)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to create item. Name might already exist.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(utils.EmojiTick+" Item **%s** (ID: %d) created! It is now visible in the shop.\nUse `/edit-item` to configure it.", name, id),
		},
	})
}

// ... (rest of the file)

func (c *AdminShopCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	var query string
	for _, opt := range data.Options {
		if opt.Focused {
			query = opt.StringValue()
			break
		}
	}

	items, err := c.DB.SearchShopItems(query)
	if err != nil {
		return
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, item := range items {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  item.Name,
			Value: item.Name,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// /edit-item [item-name]
func (c *AdminShopCommand) EditItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) > 0 {
		itemName := options[0].StringValue()
		item, err := c.DB.GetShopItem(itemName)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Item not found.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}
		c.sendEditEmbed(s, i.Interaction, item, true)
		return
	}

	// Interactive Mode
	items, err := c.DB.GetAdminShopItems(25, 0)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to fetch items.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if len(items) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " No items found to edit.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	var selectOptions []discordgo.SelectMenuOption
	for _, item := range items {
		selectOptions = append(selectOptions, discordgo.SelectMenuOption{
			Label:       item.Name,
			Value:       item.Name,
			Description: fmt.Sprintf("%d coins | Stock: %d", item.Price, item.Stock),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select an item to edit:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "shop_select_item",
							Placeholder: "Choose an item...",
							Options:     selectOptions,
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *AdminShopCommand) sendEditEmbed(s *discordgo.Session, i *discordgo.Interaction, item *models.ShopItem, isNew bool) {
	stockStr := "∞"
	if item.Stock != -1 {
		stockStr = strconv.Itoa(item.Stock)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ Editing Item: " + item.Name,
		Description: fmt.Sprintf("**Description:** %s\n**Type:** %s", item.Description, item.Type),
		Color:       0xF1C40F,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Price", Value: fmt.Sprintf("%d coins", item.Price), Inline: true},
			{Name: "Stock", Value: stockStr, Inline: true},
			{Name: "Role ID", Value: item.RoleID, Inline: true},
			{Name: "Duration", Value: fmt.Sprintf("%d s", item.Duration), Inline: true},
			{Name: "Req. Balance", Value: fmt.Sprintf("%d", item.RequiredBalance), Inline: true},
			{Name: "Hidden", Value: fmt.Sprintf("%t", item.Hidden), Inline: true},
		},
	}

	if item.ImageURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: item.ImageURL}
	}

	// Edit Options Select Menu
	selectOptions := []discordgo.SelectMenuOption{
		{Label: "Edit Name", Value: "name", Description: "Change item name", Emoji: &discordgo.ComponentEmoji{Name: "✏️"}},
		{Label: "Edit Description", Value: "desc", Description: "Change description", Emoji: &discordgo.ComponentEmoji{Name: "📝"}},
		{Label: "Edit Price", Value: "price", Description: "Change price", Emoji: &discordgo.ComponentEmoji{Name: "💰"}},
		{Label: "Edit Stock", Value: "stock", Description: "Change stock amount", Emoji: &discordgo.ComponentEmoji{Name: "📦"}},
		{Label: "Edit Type", Value: "type", Description: "Change item type (item/role)", Emoji: &discordgo.ComponentEmoji{Name: "🏷️"}},
		{Label: "Edit Role", Value: "role", Description: "Change assigned role", Emoji: &discordgo.ComponentEmoji{Name: "🎭"}},
		{Label: "Edit Duration", Value: "duration", Description: "Change duration (seconds)", Emoji: &discordgo.ComponentEmoji{Name: "⏱️"}},
		{Label: "Edit Req. Balance", Value: "req_bal", Description: "Change required balance", Emoji: &discordgo.ComponentEmoji{Name: "💎"}},
		{Label: "Edit Reply Message", Value: "reply", Description: "Change reply message", Emoji: &discordgo.ComponentEmoji{Name: "💬"}},
		{Label: "Edit Image URL", Value: "image", Description: "Change image URL", Emoji: &discordgo.ComponentEmoji{Name: "🖼️"}},
		{Label: "Toggle Hidden", Value: "hidden", Description: "Show/Hide in shop", Emoji: &discordgo.ComponentEmoji{Name: "👁️"}},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("shop_edit_select_%s", item.Name),
					Placeholder: "Select a field to edit...",
					Options:     selectOptions,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Save & Close",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("shop_edit_done_%s", item.Name),
				},
				discordgo.Button{
					Label:    "Delete Item",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("shop_edit_delete_%s", item.Name),
				},
			},
		},
	}

	responseType := discordgo.InteractionResponseUpdateMessage
	if isNew {
		responseType = discordgo.InteractionResponseChannelMessageWithSource
	}

	s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *AdminShopCommand) HandleEditItemSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	itemName := data.Values[0]

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Item not found.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	c.sendEditEmbed(s, i.Interaction, item, false)
}

func (c *AdminShopCommand) HandleEditOptionSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	itemName := strings.TrimPrefix(customID, "shop_edit_select_")
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}
	action := values[0]

	switch action {
	case "name":
		c.sendModal(s, i, "shop_modal_name_"+itemName, "Edit Name", "New Name", discordgo.TextInputShort, itemName)
	case "desc":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_desc_"+itemName, "Edit Description", "New Description", discordgo.TextInputParagraph, item.Description)
	case "price":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_price_"+itemName, "Edit Price", "New Price", discordgo.TextInputShort, strconv.Itoa(item.Price))
	case "stock":
		item, _ := c.DB.GetShopItem(itemName)
		stockStr := "-1"
		if item.Stock != -1 {
			stockStr = strconv.Itoa(item.Stock)
		}
		c.sendModal(s, i, "shop_modal_stock_"+itemName, "Edit Stock", "New Stock (-1 for infinite)", discordgo.TextInputShort, stockStr)
	case "type":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_type_"+itemName, "Edit Type", "New Type (item/role)", discordgo.TextInputShort, item.Type)
	case "role":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Select the role to assign:",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								CustomID:    "shop_role_select_" + itemName,
								Placeholder: "Choose a role...",
								MenuType:    discordgo.RoleSelectMenu,
								MaxValues:   1,
							},
						},
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	case "duration":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_duration_"+itemName, "Edit Duration", "Duration in seconds", discordgo.TextInputShort, strconv.Itoa(item.Duration))
	case "req_bal":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_reqbal_"+itemName, "Edit Required Balance", "Minimum Balance", discordgo.TextInputShort, strconv.Itoa(item.RequiredBalance))
	case "reply":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_reply_"+itemName, "Edit Reply Message", "Message sent on buy", discordgo.TextInputParagraph, item.ReplyMessage)
	case "image":
		item, _ := c.DB.GetShopItem(itemName)
		c.sendModal(s, i, "shop_modal_image_"+itemName, "Edit Image URL", "Image URL", discordgo.TextInputShort, item.ImageURL)
	case "hidden":
		item, err := c.DB.GetShopItem(itemName)
		if err != nil {
			return
		}
		item.Hidden = !item.Hidden
		c.DB.UpdateShopItem(item)
		c.sendEditEmbed(s, i.Interaction, item, false)
	}
}

func (c *AdminShopCommand) sendModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID, title, label string, style discordgo.TextInputStyle, value string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    title,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID: "input",
							Label:    label,
							Style:    style,
							Value:    value,
							Required: true,
						},
					},
				},
			},
		},
	})
}

func (c *AdminShopCommand) HandleEditButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.SplitN(customID, "_", 4) // shop_edit_done_name
	if len(parts) < 4 {
		return
	}
	action := parts[2]
	itemName := parts[3]

	if action == "done" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf(utils.EmojiTick+" Changes saved for **%s**.", itemName),
				Components: []discordgo.MessageComponent{},
				Embeds:     []*discordgo.MessageEmbed{},
			},
		})
	} else if action == "delete" {
		// Confirm delete? For now just delete.
		c.DB.DeleteShopItem(itemName)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    fmt.Sprintf(utils.EmojiTick+" Item **%s** deleted.", itemName),
				Components: []discordgo.MessageComponent{},
				Embeds:     []*discordgo.MessageEmbed{},
			},
		})
	}
}

func (c *AdminShopCommand) HandleEditRoleSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	itemName := strings.TrimPrefix(customID, "shop_role_select_")
	roleID := i.MessageComponentData().Values[0]

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		return
	}

	item.RoleID = roleID
	item.Type = "role" // Auto-switch to role type
	c.DB.UpdateShopItem(item)

	c.sendEditEmbed(s, i.Interaction, item, false)
}

func (c *AdminShopCommand) HandleEditModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	parts := strings.SplitN(customID, "_", 4) // shop_modal_price_name
	if len(parts) < 4 {
		return
	}
	action := parts[2]
	itemName := parts[3]

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		return
	}

	input := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	switch action {
	case "name":
		// Check if name exists?
		if input != item.Name {
			err := c.DB.RenameShopItem(item.Name, input)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Name already exists or invalid.", Flags: discordgo.MessageFlagsEphemeral},
				})
				return
			}
			item.Name = input
		}
	case "price":
		val, _ := strconv.Atoi(input)
		item.Price = val
	case "stock":
		val, _ := strconv.Atoi(input)
		item.Stock = val
	case "desc":
		item.Description = input
	case "type":
		item.Type = input
	case "duration":
		val, _ := strconv.Atoi(input)
		item.Duration = val
	case "reqbal":
		val, _ := strconv.Atoi(input)
		item.RequiredBalance = val
	case "reply":
		item.ReplyMessage = input
	case "image":
		item.ImageURL = input
	}

	err = c.DB.UpdateShopItem(item)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to update item.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	c.sendEditEmbed(s, i.Interaction, item, false)
}

// /delete-item <item-name>
func (c *AdminShopCommand) DeleteItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	itemName := options[0].StringValue()

	err := c.DB.DeleteShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to delete item.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf(utils.EmojiTick+" Item **%s** deleted.", itemName)},
	})
}

// /give-item <user> <item-name> [quantity]
func (c *AdminShopCommand) GiveItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	user := options[0].UserValue(s)
	itemName := options[1].StringValue()
	quantity := 1
	if len(options) > 2 {
		quantity = int(options[2].IntValue())
	}

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Item not found.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	err = c.DB.GiveItem(user.ID, i.GuildID, item.ID, quantity)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to give item.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// If role item, give role
	if item.Type == "role" && item.RoleID != "" {
		s.GuildMemberRoleAdd(i.GuildID, user.ID, item.RoleID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(utils.EmojiTick+" Gave **%d x %s** to **%s**.", quantity, item.Name, user.Username),
		},
	})
}

// /set-stock <item-name> <amount>
func (c *AdminShopCommand) SetStock(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	itemName := options[0].StringValue()
	amount := int(options[1].IntValue())

	item, err := c.DB.GetShopItem(itemName)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Item not found.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	item.Stock = amount
	err = c.DB.UpdateShopItem(item)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to update stock.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(utils.EmojiTick+" Stock for **%s** set to **%d**.", item.Name, amount),
		},
	})
}

// /item-options
func (c *AdminShopCommand) ItemOptions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ Item Edit Options",
		Description: "Use `/edit-item <option> <item-name> <value>` with these options:",
		Color:       0x9B59B6, // Purple
		Fields: []*discordgo.MessageEmbedField{
			{Name: "name", Value: "Change item name (Unique)", Inline: true},
			{Name: "price", Value: "Set price (Integer)", Inline: true},
			{Name: "description", Value: "Set item description", Inline: true},
			{Name: "stock", Value: "Set stock (-1 for infinite)", Inline: true},
			{Name: "type", Value: "'item', 'role', or 'boost'", Inline: true},
			{Name: "role-given", Value: "Role ID/Mention to give", Inline: true},
			{Name: "duration", Value: "Role duration (e.g. 30d, 1h)", Inline: true},
			{Name: "required-balance", Value: "Min balance to buy", Inline: true},
			{Name: "reply", Value: "Custom reply message on buy", Inline: true},
			{Name: "hidden", Value: "'true' or 'false' to hide/show", Inline: true},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// /check-redeem <code>
func (c *AdminShopCommand) CheckRedeem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	code := options[0].StringValue()

	redeemCode, err := c.DB.GetRedeemCode(code)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Invalid redeem code.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	statusEmoji := utils.EmojiTick
	statusText := "Active"
	statusColor := utils.ColorGreen
	if redeemCode.IsClaimed {
		statusEmoji = "🔒"
		statusText = "Claimed/Expired"
		statusColor = utils.ColorRed
	}

	// Get user info
	user, err := s.User(redeemCode.UserID)
	username := "Unknown User"
	if err == nil {
		username = user.Username
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎫 Redeem Code Information",
		Description: fmt.Sprintf("**Code:** `%s`\n**Status:** %s %s", code, statusEmoji, statusText),
		Color:       statusColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📦 Item", Value: redeemCode.ItemName, Inline: true},
			{Name: "💰 Price", Value: fmt.Sprintf("%d %s", redeemCode.ItemPrice, utils.EmojiCoin), Inline: true},
			{Name: "👤 Buyer", Value: username, Inline: true},
			{Name: "📝 Description", Value: redeemCode.ItemDescription, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Created at %s", time.Unix(redeemCode.CreatedAt/1000, 0).Format("2006-01-02 15:04:05")),
		},
	}

	// Send publicly in channel (visible to all)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// /redeem-claimed <code>
func (c *AdminShopCommand) RedeemClaimed(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !c.isAdmin(s, i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " You need Manage Server permission.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	code := options[0].StringValue()

	// Check if code exists
	redeemCode, err := c.DB.GetRedeemCode(code)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Invalid redeem code.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	if redeemCode.IsClaimed {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "⚠️ This redeem code is already claimed/expired.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	// Mark as claimed
	err = c.DB.MarkRedeemCodeClaimed(code)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to mark redeem code as claimed.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s Redeem code `%s` has been marked as claimed/expired.", utils.EmojiTick, code),
		},
	})
}
