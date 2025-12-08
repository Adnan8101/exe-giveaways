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
	CreateItem = &discordgo.ApplicationCommand{
		Name:        "create-item",
		Description: "Create a new shop item (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Name of the new item",
				Required:    false,
			},
		},
	}

	EditItem = &discordgo.ApplicationCommand{
		Name:        "edit-item",
		Description: "Edit a shop item (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Name of the item to edit (Optional - skip to jump to item)",
				Required:    false,
			},
		},
	}

	DeleteItem = &discordgo.ApplicationCommand{
		Name:        "delete-item",
		Description: "Delete a shop item (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Name of the item to delete",
				Required:    true,
			},
		},
	}

	GiveItem = &discordgo.ApplicationCommand{
		Name:        "give-item",
		Description: "Give an item to a user (Admin)",
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
	}

	SetStock = &discordgo.ApplicationCommand{
		Name:        "set-stock",
		Description: "Set stock for an item (Admin)",
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
				Description: "New stock amount",
				Required:    true,
			},
		},
	}

	ItemOptions = &discordgo.ApplicationCommand{
		Name:        "item-options",
		Description: "View all editable item options (Admin)",
	}

	CheckRedeem = &discordgo.ApplicationCommand{
		Name:        "check-redeem",
		Description: "Check information about a redeem code (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "code",
				Description: "Redeem code to check",
				Required:    true,
			},
		},
	}

	RedeemClaimed = &discordgo.ApplicationCommand{
		Name:        "redeem-claimed",
		Description: "Mark a redeem code as claimed/expired (Admin)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "code",
				Description: "Redeem code to mark as claimed",
				Required:    true,
			},
		},
	}
)
