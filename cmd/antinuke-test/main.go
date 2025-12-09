package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Pre-configured test values
	token := "YOUR_TOKEN_HERE"
	guildID := "YOUR_GUILD_ID_HERE"

	fmt.Println("🔧 Using pre-configured test credentials")
	fmt.Printf("Token: %s...%s\n", token[:20], token[len(token)-10:])
	fmt.Printf("Guild ID: %s\n", guildID)

	// Ask for operation counts
	fmt.Print("How many channels to create/delete? ")
	channelCountStr, _ := reader.ReadString('\n')
	channelCount, err := strconv.Atoi(strings.TrimSpace(channelCountStr))
	if err != nil {
		fmt.Println("Invalid number, using default: 5")
		channelCount = 5
	}

	fmt.Print("How many roles to create/delete? ")
	roleCountStr, _ := reader.ReadString('\n')
	roleCount, err := strconv.Atoi(strings.TrimSpace(roleCountStr))
	if err != nil {
		fmt.Println("Invalid number, using default: 5")
		roleCount = 5
	}

	fmt.Print("Delay between operations (ms)? ")
	delayStr, _ := reader.ReadString('\n')
	delay, err := strconv.Atoi(strings.TrimSpace(delayStr))
	if err != nil {
		fmt.Println("Invalid number, using default: 100ms")
		delay = 100
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Open connection
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}
	defer dg.Close()

	fmt.Println("\n✅ Bot connected successfully!")
	fmt.Printf("📊 Test Configuration:\n")
	fmt.Printf("   - Guild ID: %s\n", guildID)
	fmt.Printf("   - Channels to create/delete: %d\n", channelCount)
	fmt.Printf("   - Roles to create/delete: %d\n", roleCount)
	fmt.Printf("   - Delay between operations: %dms\n\n", delay)

	// Menu
	for {
		fmt.Println("\n=== AntiNuke Speed Test Menu ===")
		fmt.Println("1. Test Channel Creation")
		fmt.Println("2. Test Channel Deletion")
		fmt.Println("3. Test Role Creation")
		fmt.Println("4. Test Role Deletion")
		fmt.Println("5. Test Mass Channel Creation (Spam)")
		fmt.Println("6. Test Mass Role Creation (Spam)")
		fmt.Println("7. Test Channel Update (Name Change)")
		fmt.Println("8. Test Role Update (Permission Change)")
		fmt.Println("9. Test All Operations (Full Spam)")
		fmt.Println("0. Exit")
		fmt.Print("\nSelect option: ")

		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		switch option {
		case "1":
			testChannelCreation(dg, guildID, channelCount, time.Duration(delay)*time.Millisecond)
		case "2":
			testChannelDeletion(dg, guildID, channelCount, time.Duration(delay)*time.Millisecond)
		case "3":
			testRoleCreation(dg, guildID, roleCount, time.Duration(delay)*time.Millisecond)
		case "4":
			testRoleDeletion(dg, guildID, roleCount, time.Duration(delay)*time.Millisecond)
		case "5":
			testMassChannelCreation(dg, guildID, channelCount)
		case "6":
			testMassRoleCreation(dg, guildID, roleCount)
		case "7":
			testChannelUpdate(dg, guildID, channelCount, time.Duration(delay)*time.Millisecond)
		case "8":
			testRoleUpdate(dg, guildID, roleCount, time.Duration(delay)*time.Millisecond)
		case "9":
			testFullSpam(dg, guildID, channelCount, roleCount)
		case "0":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid option, please try again")
		}
	}
}

func testChannelCreation(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Creating %d channels with %v delay...\n", count, delay)
	startTime := time.Now()

	for i := 0; i < count; i++ {
		channelName := fmt.Sprintf("test-channel-%d", i+1)
		ch, err := dg.GuildChannelCreate(guildID, channelName, discordgo.ChannelTypeGuildText)
		if err != nil {
			fmt.Printf("❌ Error creating channel %d: %v\n", i+1, err)
		} else {
			fmt.Printf("✅ Created channel: %s (ID: %s)\n", ch.Name, ch.ID)
		}
		time.Sleep(delay)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n⏱️  Total time: %v (Avg: %v per channel)\n", elapsed, elapsed/time.Duration(count))
}

func testChannelDeletion(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Looking for test channels to delete...\n")

	channels, err := dg.GuildChannels(guildID)
	if err != nil {
		fmt.Printf("❌ Error fetching channels: %v\n", err)
		return
	}

	deleted := 0
	startTime := time.Now()

	for _, ch := range channels {
		if strings.HasPrefix(ch.Name, "test-channel-") && deleted < count {
			_, err := dg.ChannelDelete(ch.ID)
			if err != nil {
				fmt.Printf("❌ Error deleting channel %s: %v\n", ch.Name, err)
			} else {
				fmt.Printf("✅ Deleted channel: %s (ID: %s)\n", ch.Name, ch.ID)
				deleted++
			}
			time.Sleep(delay)
		}
	}

	elapsed := time.Since(startTime)
	if deleted > 0 {
		fmt.Printf("\n⏱️  Total time: %v (Avg: %v per channel)\n", elapsed, elapsed/time.Duration(deleted))
	}
	fmt.Printf("Deleted %d channels\n", deleted)
}

func testRoleCreation(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Creating %d roles with %v delay...\n", count, delay)
	startTime := time.Now()

	for i := 0; i < count; i++ {
		roleName := fmt.Sprintf("Test Role %d", i+1)
		role, err := dg.GuildRoleCreate(guildID, &discordgo.RoleParams{
			Name:  roleName,
			Color: new(int),
		})
		if err != nil {
			fmt.Printf("❌ Error creating role %d: %v\n", i+1, err)
		} else {
			fmt.Printf("✅ Created role: %s (ID: %s)\n", role.Name, role.ID)
		}
		time.Sleep(delay)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n⏱️  Total time: %v (Avg: %v per role)\n", elapsed, elapsed/time.Duration(count))
}

func testRoleDeletion(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Looking for test roles to delete...\n")

	roles, err := dg.GuildRoles(guildID)
	if err != nil {
		fmt.Printf("❌ Error fetching roles: %v\n", err)
		return
	}

	deleted := 0
	startTime := time.Now()

	for _, role := range roles {
		if strings.HasPrefix(role.Name, "Test Role ") && deleted < count {
			err := dg.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				fmt.Printf("❌ Error deleting role %s: %v\n", role.Name, err)
			} else {
				fmt.Printf("✅ Deleted role: %s (ID: %s)\n", role.Name, role.ID)
				deleted++
			}
			time.Sleep(delay)
		}
	}

	elapsed := time.Since(startTime)
	if deleted > 0 {
		fmt.Printf("\n⏱️  Total time: %v (Avg: %v per role)\n", elapsed, elapsed/time.Duration(deleted))
	}
	fmt.Printf("Deleted %d roles\n", deleted)
}

func testMassChannelCreation(dg *discordgo.Session, guildID string, count int) {
	fmt.Printf("\n⚠️  MASS SPAM: Creating %d channels as fast as possible!\n", count)
	fmt.Println("   This will trigger antinuke protections!")
	fmt.Print("   Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	startTime := time.Now()
	successful := 0
	failed := 0

	for i := 0; i < count; i++ {
		channelName := fmt.Sprintf("spam-channel-%d", i+1)
		_, err := dg.GuildChannelCreate(guildID, channelName, discordgo.ChannelTypeGuildText)
		if err != nil {
			failed++
			fmt.Printf("❌ Failed #%d: %v\n", i+1, err)
		} else {
			successful++
			fmt.Printf("✅ Created #%d\n", i+1)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("   - Successful: %d\n", successful)
	fmt.Printf("   - Failed: %d\n", failed)
	fmt.Printf("   - Total time: %v\n", elapsed)
	if successful > 0 {
		fmt.Printf("   - Avg per channel: %v\n", elapsed/time.Duration(successful))
	}
}

func testMassRoleCreation(dg *discordgo.Session, guildID string, count int) {
	fmt.Printf("\n⚠️  MASS SPAM: Creating %d roles as fast as possible!\n", count)
	fmt.Println("   This will trigger antinuke protections!")
	fmt.Print("   Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	startTime := time.Now()
	successful := 0
	failed := 0

	for i := 0; i < count; i++ {
		roleName := fmt.Sprintf("Spam Role %d", i+1)
		_, err := dg.GuildRoleCreate(guildID, &discordgo.RoleParams{
			Name:  roleName,
			Color: new(int),
		})
		if err != nil {
			failed++
			fmt.Printf("❌ Failed #%d: %v\n", i+1, err)
		} else {
			successful++
			fmt.Printf("✅ Created #%d\n", i+1)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("   - Successful: %d\n", successful)
	fmt.Printf("   - Failed: %d\n", failed)
	fmt.Printf("   - Total time: %v\n", elapsed)
	if successful > 0 {
		fmt.Printf("   - Avg per role: %v\n", elapsed/time.Duration(successful))
	}
}

func testChannelUpdate(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Testing channel updates...\n")

	channels, err := dg.GuildChannels(guildID)
	if err != nil {
		fmt.Printf("❌ Error fetching channels: %v\n", err)
		return
	}

	updated := 0
	startTime := time.Now()

	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildText && updated < count {
			newName := fmt.Sprintf("updated-%d", time.Now().Unix())
			_, err := dg.ChannelEdit(ch.ID, &discordgo.ChannelEdit{
				Name: newName,
			})
			if err != nil {
				fmt.Printf("❌ Error updating channel %s: %v\n", ch.Name, err)
			} else {
				fmt.Printf("✅ Updated channel: %s -> %s\n", ch.Name, newName)
				updated++
			}
			time.Sleep(delay)
		}
	}

	elapsed := time.Since(startTime)
	if updated > 0 {
		fmt.Printf("\n⏱️  Total time: %v (Avg: %v per update)\n", elapsed, elapsed/time.Duration(updated))
	}
	fmt.Printf("Updated %d channels\n", updated)
}

func testRoleUpdate(dg *discordgo.Session, guildID string, count int, delay time.Duration) {
	fmt.Printf("\n🔄 Testing role updates...\n")

	roles, err := dg.GuildRoles(guildID)
	if err != nil {
		fmt.Printf("❌ Error fetching roles: %v\n", err)
		return
	}

	updated := 0
	startTime := time.Now()

	for _, role := range roles {
		if !role.Managed && updated < count {
			newPerms := int64(0)
			_, err := dg.GuildRoleEdit(guildID, role.ID, &discordgo.RoleParams{
				Permissions: &newPerms,
			})
			if err != nil {
				fmt.Printf("❌ Error updating role %s: %v\n", role.Name, err)
			} else {
				fmt.Printf("✅ Updated role: %s (permissions cleared)\n", role.Name)
				updated++
			}
			time.Sleep(delay)
		}
	}

	elapsed := time.Since(startTime)
	if updated > 0 {
		fmt.Printf("\n⏱️  Total time: %v (Avg: %v per update)\n", elapsed, elapsed/time.Duration(updated))
	}
	fmt.Printf("Updated %d roles\n", updated)
}

func testFullSpam(dg *discordgo.Session, guildID string, channelCount, roleCount int) {
	fmt.Printf("\n⚠️⚠️⚠️  FULL SPAM TEST ⚠️⚠️⚠️\n")
	fmt.Printf("   This will spam ALL operations simultaneously!\n")
	fmt.Printf("   - %d channels\n", channelCount)
	fmt.Printf("   - %d roles\n", roleCount)
	fmt.Println("   This WILL trigger antinuke!")
	fmt.Print("   Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	startTime := time.Now()

	// Channels
	fmt.Println("\n🔥 Spamming channels...")
	for i := 0; i < channelCount; i++ {
		go func(idx int) {
			channelName := fmt.Sprintf("spam-%d", idx)
			dg.GuildChannelCreate(guildID, channelName, discordgo.ChannelTypeGuildText)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	// Roles
	fmt.Println("🔥 Spamming roles...")
	for i := 0; i < roleCount; i++ {
		go func(idx int) {
			roleName := fmt.Sprintf("Spam %d", idx)
			dg.GuildRoleCreate(guildID, &discordgo.RoleParams{
				Name:  roleName,
				Color: new(int),
			})
		}(i)
	}

	time.Sleep(2 * time.Second)

	elapsed := time.Since(startTime)
	fmt.Printf("\n⏱️  Total spam duration: %v\n", elapsed)
	fmt.Println("✅ Check your antinuke logs to see detection speed!")
}
