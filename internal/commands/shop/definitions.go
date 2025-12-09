package shop

import "github.com/bwmarrin/discordgo"

var (
	// User Commands
	Shop = &discordgo.ApplicationCommand{
		Name:        "shop",
		Description: "View the server shop",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "page",
				Description: "Page number to view",
				Required:    false,
			},
		},
	}

	Buy = &discordgo.ApplicationCommand{
		Name:        "buy",
		Description: "Buy an item from the shop",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Name of the item to buy",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Amount to buy (default 1)",
				Required:    false,
			},
		},
	}

	ItemInfo = &discordgo.ApplicationCommand{
		Name:        "item-info",
		Description: "View details about a shop item",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Name of the item",
				Required:    true,
			},
		},
	}

	Inventory = &discordgo.ApplicationCommand{
		Name:        "inventory",
		Description: "View your inventory",
	}

	// Admin Commands
	AdminShop = &discordgo.ApplicationCommand{
		Name:        "admin-shop",
		Description: "Manage shop items (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "create",
				Description: "Create a new shop item",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "name",
						Description: "Name of the new item",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "edit",
				Description: "Edit a shop item",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "item",
						Description: "Name of the item to edit",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "delete",
				Description: "Delete a shop item",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "item",
						Description: "Name of the item to delete",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "give",
				Description: "Give an item to a user",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "User to give item to",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "item",
						Description: "Name of the item",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "quantity",
						Description: "Amount to give",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "stock",
				Description: "Set item stock",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "item",
						Description: "Name of the item",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "amount",
						Description: "New stock amount (-1 for infinite)",
						Required:    true,
					},
				},
			},
		},
	}
)
