package discord

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"teamacedia/discord-bot/internal/config"
	"teamacedia/discord-bot/internal/db"
	"teamacedia/discord-bot/internal/logging"
	"teamacedia/discord-bot/internal/models"
	"teamacedia/discord-bot/internal/reaction_roles"
	"teamacedia/discord-bot/internal/sticky_roles"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	session  *discordgo.Session
	cmdIDs   []*discordgo.ApplicationCommand
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Get a list of commands that work with this bot",
			Options:     []*discordgo.ApplicationCommandOption{},
		},
		{
			Name:        "remindme",
			Description: "Set a reminder for yourself that sends daily until removed. Usage: /remindme [message]",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "Reminder message",
					Required:    true,
				},
			},
		},
		{
			Name:        "removereminder",
			Description: "Remove a daily reminder set with /remindme",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "message",
					Description:  "Reminder message to remove",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	}
)

func containsIgnoreCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	// Only autocomplete the removereminder command
	if data.Name != "removereminder" {
		return
	}

	// User input so far
	input := data.Options[0].StringValue()

	// Fetch reminders for the user
	userReminders, err := db.GetUserReminders(i.Member.User.ID)
	if err != nil {
		log.Printf("Autocomplete DB error: %v", err)
		return
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	for _, r := range userReminders {
		// Filter by input
		if input == "" || containsIgnoreCase(r.Text, input) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  r.Text,
				Value: r.Text,
			})
		}

		// Discord allows max 25 choices
		if len(choices) >= 25 {
			break
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(8),
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		handleAutocomplete(s, i)
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()

	var commands_description string = "Available commands:\n\n" +
		"`/help` - Get a list of commands that work with this bot\n" +
		"`/remindme [message]` - Set a reminder for yourself that sends daily until removed.\n" +
		"`/removereminder [message]` - Remove a daily reminder set with `/remindme`\n"

	switch data.Name {
	case "help":
		embed := &discordgo.MessageEmbed{
			Title:       "Command Help",
			Description: commands_description,
			Color:       0x00FFFF, // Cyan
		}
		replyEmbed(s, i, embed)
	case "remindme":
		reminderText := data.Options[0].StringValue()
		err := db.AddReminder(models.Reminder{
			UserID: i.Member.User.ID,
			Text:   reminderText,
		})
		if err != nil {
			reply(s, i, "Failed to add reminder: "+err.Error())
		} else {
			embed := &discordgo.MessageEmbed{
				Title:       "Reminder Set",
				Description: "Your daily reminder has been set:\n" + reminderText,
				Color:       0x00FFFF, // Cyan
			}
			replyEmbed(s, i, embed)
		}
	case "removereminder":
		reminderText := data.Options[0].StringValue()
		err := db.DeleteReminder(models.Reminder{
			UserID: i.Member.User.ID,
			Text:   reminderText,
		})
		if err != nil {
			reply(s, i, "Failed to remove reminder: "+err.Error())
		} else {
			embed := &discordgo.MessageEmbed{
				Title:       "Reminder Removed",
				Description: "Your daily reminder has been removed:\n" + reminderText,
				Color:       0x00FFFF, // Cyan
			}
			replyEmbed(s, i, embed)
		}
	}
}

func reply(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func replyEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction with embed: %v", err)
	}
}

func DmUser(userID string, content string) error {
	// Create or fetch DM channel
	channel, err := session.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	// Send message to the DM channel ID
	_, err = session.ChannelMessageSend(channel.ID, content)
	if err != nil {
		return err
	}

	return nil
}

func DmUserEmbed(userID string, embed *discordgo.MessageEmbed) error {
	// Create or fetch DM channel
	channel, err := session.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	// Send message to the DM channel ID
	_, err = session.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		return err
	}

	return nil
}

// Start the bot, register commands, and block until SIGINT/SIGTERM
func Start(botToken string, appID string, guildID string) {
	var err error
	session, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	// Enable intents for message content and events
	session.Identify.Intents = discordgo.IntentsAll

	// Open the session
	err = session.Open()
	if err != nil {
		log.Fatalf("Cannot open Discord session: %v", err)
	}
	log.Println("Discord bot is running...")

	// Register commands
	for _, v := range commands {
		cmd, err := session.ApplicationCommandCreate(appID, guildID, v)
		if err != nil {
			log.Fatalf("Cannot create '%s' command: %v", v.Name, err)
		}
		cmdIDs = append(cmdIDs, cmd)
	}
	log.Println("Commands registered.")

	// Set up handlers
	state, err := reaction_roles.InitReactionRoles(session, config.Config.ReactionRoles)
	if err != nil {
		log.Fatal(err)
	}

	// Register handlers
	session.AddHandler(logging.OnMessageCreate)
	session.AddHandler(logging.OnMessageUpdate)
	session.AddHandler(logging.OnMessageDelete)
	session.AddHandler(sticky_roles.OnMemberJoin)
	session.AddHandler(sticky_roles.OnMemberUpdate)
	session.AddHandler(sticky_roles.OnRoleDelete)
	session.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		reaction_roles.HandleReactionAdd(s, r, state)
	})

	session.AddHandler(interactionHandler)

	// Log all member roles
	err = sticky_roles.SyncGuildRoles(session, guildID)
	if err != nil {
		log.Printf("Failed to sync guild roles: %v", err)
	}

	// Wait here until Ctrl+C or kill signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Cleanup before exit
	log.Println("Shutting down, removing commands...")
	for _, cmd := range cmdIDs {
		err := session.ApplicationCommandDelete(appID, guildID, cmd.ID)
		if err != nil {
			log.Printf("Cannot delete command %s: %v", cmd.Name, err)
		}
	}
	session.Close()
	log.Println("Bot stopped cleanly.")
}

func StartScheduler() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	SendReminders()

	// Create a channel to listen for OS signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			if now.Hour() == 8 { // 8 AM
				err := SendReminders()
				if err != nil {
					log.Printf("Error sending reminders: %v", err)
				}
			}
		case <-sigs:
			return
		}
	}
}

// Reminder code
func SendReminders() error {
	// Map of userID to list of models.reminder
	reminders, err := db.GetAllReminders()
	if err != nil {
		return err
	}
	reminderMap := make(map[string][]models.Reminder)
	for _, r := range reminders {
		reminderMap[r.UserID] = append(reminderMap[r.UserID], r)
	}

	// Send reminders (placeholder logic)
	for userID, rs := range reminderMap {
		var message string
		message = "You have the following reminders for today:\n\n"
		for _, r := range rs {
			message += "- " + r.Text + "\n"
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Daily Reminder",
			Description: message,
			Color:       0x00FFFF, // Cyan
		}
		DmUserEmbed(userID, embed)
	}

	return nil
}
