package economy

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type BlackjackCommand struct {
	DB      *database.Database
	Service *services.EconomyService
}

func NewBlackjackCommand(db *database.Database, service *services.EconomyService) *BlackjackCommand {
	return &BlackjackCommand{DB: db, Service: service}
}

// Card represents a playing card
type Card struct {
	Suit  string
	Value string
	Score int
}

// Deck represents a deck of cards
type Deck []Card

func NewDeck() Deck {
	suits := []string{"♠️", "♥️", "♣️", "♦️"}
	values := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	deck := make(Deck, 0, 52)

	for _, suit := range suits {
		for _, value := range values {
			score := 0
			switch value {
			case "A":
				score = 11
			case "K", "Q", "J":
				score = 10
			default:
				score, _ = strconv.Atoi(value)
			}
			deck = append(deck, Card{Suit: suit, Value: value, Score: score})
		}
	}
	return deck
}

func (d Deck) Shuffle() Deck {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	return d
}

func (d *Deck) Draw() Card {
	card := (*d)[0]
	*d = (*d)[1:]
	return card
}

type Hand []Card

func (h Hand) String() string {
	var s []string
	for _, card := range h {
		s = append(s, fmt.Sprintf("`%s%s`", card.Suit, card.Value))
	}
	return strings.Join(s, " ")
}

func (h Hand) Score() int {
	score := 0
	aces := 0
	for _, card := range h {
		score += card.Score
		if card.Value == "A" {
			aces++
		}
	}
	for score > 21 && aces > 0 {
		score -= 10
		aces--
	}
	return score
}

// !bj <amount/all>
func (c *BlackjackCommand) Handle(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, utils.EmojiCross+" Usage: `!bj <amount/all>`")
		return
	}

	// Help command check
	if strings.ToLower(args[0]) == "help" {
		c.SendHelp(s, m.ChannelID)
		return
	}

	amountStr := args[0]
	var bet int64

	user, err := c.DB.GetEconomyUser(m.GuildID, m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, utils.EmojiCross+" Failed to fetch your balance.")
		return
	}

	if strings.ToLower(amountStr) == "all" {
		bet = user.Balance
	} else {
		parsed, err := parseAmount(amountStr)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, utils.EmojiCross+" Invalid amount.")
			return
		}
		bet = parsed
	}

	if bet <= 0 {
		s.ChannelMessageSend(m.ChannelID, utils.EmojiCross+" Bet must be positive.")
		return
	}

	if user.Balance < bet {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s You don't have enough coins. Balance: %d %s", utils.EmojiCross, user.Balance, utils.EmojiCoin))
		return
	}

	// Start Game
	deck := NewDeck().Shuffle()
	playerHand := Hand{deck.Draw(), deck.Draw()}
	dealerHand := Hand{deck.Draw()} // Only draw one visible card for dealer initially

	// Deduct bet immediately (refund on push/win)
	err = c.Service.RemoveCoins(m.GuildID, m.Author.ID, bet)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, utils.EmojiCross+" Transaction failed.")
		return
	}

	// Check for immediate Blackjack
	playerScore := playerHand.Score()
	if playerScore == 21 {
		// Payout 3:2
		winnings := int64(float64(bet) * 2.5) // Return bet + 1.5x
		c.Service.AddCoins(m.GuildID, m.Author.ID, winnings)

		// Draw dealer's full hand for display (simulated)
		dealerHand = append(dealerHand, deck.Draw())

		embed := &discordgo.MessageEmbed{
			Title: "🃏 Blackjack!",
			Color: utils.ColorGreen,
			Fields: []*discordgo.MessageEmbedField{
				{Name: fmt.Sprintf("%s's Hand", m.Author.Username), Value: fmt.Sprintf("%s\nScore: **21**", playerHand), Inline: true},
				{Name: "Dealer's Hand", Value: fmt.Sprintf("%s\nScore: **%d**", dealerHand, dealerHand.Score()), Inline: true},
			},
			Description: fmt.Sprintf("You got Blackjack! You won **%d** %s", winnings-bet, utils.EmojiCoin),
		}
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	embed := c.createGameEmbed(m.Author, playerHand, dealerHand, bet, false)

	msg, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		return
	}

	// Add Reactions
	s.MessageReactionAdd(m.ChannelID, msg.ID, "🃏") // Hit
	s.MessageReactionAdd(m.ChannelID, msg.ID, "🛑") // Stand
}

func (c *BlackjackCommand) SendHelp(s *discordgo.Session, channelID string) {
	embed := &discordgo.MessageEmbed{
		Title: "🃏 Blackjack Help",
		Color: 0x3498DB,
		Description: `**How to Play Blackjack**

1. **Objective**: Beat the dealer's hand without going over 21.
2. **Card Values**:
   - Number cards (2-10): Face value
   - Face cards (J, Q, K): 10
   - Ace (A): 1 or 11
3. **Controls**:
   - React with 🃏 to **Hit** (take another card).
   - React with 🛑 to **Stand** (hold your hand).
   - *Note: Adding or removing the reaction triggers the action.*
4. **Payouts**:
   - Win: 1:1 (Double your bet)
   - Blackjack: 3:2 (2.5x your bet)
   - Push: Bet returned`,
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

func (c *BlackjackCommand) createGameEmbed(user *discordgo.User, playerHand, dealerHand Hand, bet int64, reveal bool) *discordgo.MessageEmbed {
	dealerString := fmt.Sprintf("`%s%s` `??`", dealerHand[0].Suit, dealerHand[0].Value)
	dealerScore := dealerHand[0].Score // Only show score of visible card

	if reveal {
		dealerString = dealerHand.String()
		dealerScore = dealerHand.Score()
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s's Blackjack Game", user.Username),
			IconURL: user.AvatarURL(""),
		},
		Color: 0x2B2D31,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("Your Hand %s", scoreToEmoji(playerHand.Score())),
				Value:  playerHand.String(),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Dealer's Hand %s", scoreToEmoji(dealerScore)),
				Value:  dealerString,
				Inline: true,
			},
			{
				Name:   "Bet",
				Value:  fmt.Sprintf("%d %s", bet, utils.EmojiCoin),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("PlayerID: %s | Hit 🃏 to take another card, Stand 🛑 to hold.", user.ID),
		},
	}
	return embed
}

func scoreToEmoji(score int) string {
	s := strconv.Itoa(score)
	var res string
	for _, char := range s {
		switch char {
		case '0':
			res += "0️⃣"
		case '1':
			res += "1️⃣"
		case '2':
			res += "2️⃣"
		case '3':
			res += "3️⃣"
		case '4':
			res += "4️⃣"
		case '5':
			res += "5️⃣"
		case '6':
			res += "6️⃣"
		case '7':
			res += "7️⃣"
		case '8':
			res += "8️⃣"
		case '9':
			res += "9️⃣"
		}
	}
	return res
}

func (c *BlackjackCommand) HandleReaction(s *discordgo.Session, r *discordgo.MessageReaction, add bool) {
	// Get Message
	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		return
	}

	if len(msg.Embeds) == 0 {
		return
	}
	embed := msg.Embeds[0]

	// Check if it's a Blackjack game
	if embed.Author == nil || !strings.Contains(embed.Author.Name, "Blackjack Game") {
		return
	}

	// Verify Player
	if embed.Footer == nil || !strings.Contains(embed.Footer.Text, "PlayerID: "+r.UserID) {
		return
	}

	// Check if game is already over (no footer instructions or different color?)
	// If the footer text doesn't contain "Hit", it's likely over.
	if !strings.Contains(embed.Footer.Text, "Hit") {
		return
	}

	// Parse State
	playerHand := parseHand(embed.Fields[0].Value)
	dealerHand := parseHand(strings.Split(embed.Fields[1].Value, " ")[0]) // Only first card is visible/known

	// Parse Bet
	betStr := strings.Split(embed.Fields[2].Value, " ")[0]
	bet, _ := strconv.ParseInt(betStr, 10, 64)

	action := ""
	if r.Emoji.Name == "🃏" {
		action = "hit"
	} else if r.Emoji.Name == "🛑" {
		action = "stand"
	} else {
		return
	}

	if action == "hit" {
		// Draw new card
		deck := NewDeck().Shuffle() // Infinite deck
		newCard := deck.Draw()
		playerHand = append(playerHand, newCard)

		score := playerHand.Score()
		if score > 21 {
			// Bust
			c.endGame(s, msg, playerHand, dealerHand, bet, "bust", r.UserID, r.GuildID)
		} else if score == 21 {
			c.stand(s, msg, playerHand, dealerHand, bet, r.UserID, r.GuildID)
		} else {
			// Continue
			user, _ := s.User(r.UserID)
			newEmbed := c.createGameEmbed(user, playerHand, dealerHand, bet, false)
			s.ChannelMessageEditEmbed(r.ChannelID, r.MessageID, newEmbed)
		}
	} else if action == "stand" {
		c.stand(s, msg, playerHand, dealerHand, bet, r.UserID, r.GuildID)
	}
}

func (c *BlackjackCommand) stand(s *discordgo.Session, msg *discordgo.Message, playerHand, dealerHand Hand, bet int64, userID, guildID string) {
	// Dealer plays
	deck := NewDeck().Shuffle() // Infinite deck for dealer hits

	// Dealer draws until 17
	for dealerHand.Score() < 17 {
		dealerHand = append(dealerHand, deck.Draw())
	}

	playerScore := playerHand.Score()
	dealerScore := dealerHand.Score()

	result := "lose"
	if dealerScore > 21 {
		result = "win" // Dealer bust
	} else if playerScore > dealerScore {
		result = "win"
	} else if playerScore == dealerScore {
		result = "push"
	}

	c.endGame(s, msg, playerHand, dealerHand, bet, result, userID, guildID)
}

func parseAmount(s string) (int64, error) {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "k", "000")
	s = strings.ReplaceAll(s, "m", "000000")
	return strconv.ParseInt(s, 10, 64)
}

func (c *BlackjackCommand) endGame(s *discordgo.Session, msg *discordgo.Message, playerHand, dealerHand Hand, bet int64, result string, userID, guildID string) {
	color := utils.ColorRed
	title := "❌ You Lost"
	desc := fmt.Sprintf("You lost **%d** %s", bet, utils.EmojiCoin)

	if result == "win" {
		color = utils.ColorGreen
		title = "🎉 You Won!"
		winnings := bet * 2
		err := c.Service.AddCoins(guildID, userID, winnings)
		if err != nil {
			fmt.Printf("Error adding coins: %v (Guild: %s, User: %s, Amount: %d)\n", err, guildID, userID, winnings)
		} else {
			fmt.Printf("Added %d coins to %s in guild %s\n", winnings, userID, guildID)
		}
		desc = fmt.Sprintf("You won **%d** %s", bet, utils.EmojiCoin)
	} else if result == "push" {
		color = 0xF1C40F // Yellow
		title = "🤝 Push"
		err := c.Service.AddCoins(guildID, userID, bet) // Refund
		if err != nil {
			fmt.Printf("Error refunding coins: %v\n", err)
		}
		desc = "Your bet has been returned."
	} else if result == "bust" {
		title = "💥 Bust!"
		desc = fmt.Sprintf("You went over 21 and lost **%d** %s", bet, utils.EmojiCoin)
	}

	user, _ := s.User(userID)
	embed := c.createGameEmbed(user, playerHand, dealerHand, bet, true)
	embed.Title = title
	embed.Description = desc
	embed.Color = color
	embed.Footer = nil // Remove footer to indicate game over

	s.ChannelMessageEditEmbed(msg.ChannelID, msg.ID, embed)
	s.MessageReactionsRemoveAll(msg.ChannelID, msg.ID)
}

// Helpers to parse "SuitValue" string back to Card
func parseCard(s string) Card {
	suit := ""
	val := ""

	if strings.HasPrefix(s, "♠️") {
		suit = "♠️"
		val = strings.TrimPrefix(s, "♠️")
	}
	if strings.HasPrefix(s, "♥️") {
		suit = "♥️"
		val = strings.TrimPrefix(s, "♥️")
	}
	if strings.HasPrefix(s, "♣️") {
		suit = "♣️"
		val = strings.TrimPrefix(s, "♣️")
	}
	if strings.HasPrefix(s, "♦️") {
		suit = "♦️"
		val = strings.TrimPrefix(s, "♦️")
	}

	score := 0
	switch val {
	case "A":
		score = 11
	case "K", "Q", "J":
		score = 10
	default:
		score, _ = strconv.Atoi(val)
	}

	return Card{Suit: suit, Value: val, Score: score}
}

func parseHand(s string) Hand {
	// s is like "`♠️10` `♥️A`"
	s = strings.ReplaceAll(s, "`", "")
	parts := strings.Split(s, " ")
	var h Hand
	for _, p := range parts {
		if p == "??" || p == "" {
			continue
		}
		h = append(h, parseCard(p))
	}
	return h
}
