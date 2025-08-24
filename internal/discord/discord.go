package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"teamacedia/discord-bot/internal/config"
	"teamacedia/discord-bot/internal/logging"
	"teamacedia/discord-bot/internal/reaction_roles"

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
	}
)

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()

	if data.Name == "help" {
		embed := &discordgo.MessageEmbed{
			Title:       "Command Help",
			Description: "Currently this bot has no commands, as we continue developing it, and adding features, more commands will be added..",
			Color:       0x00FFFF, // Cyan
		}
		replyEmbed(s, i, embed)
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
	session.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		reaction_roles.HandleReactionAdd(s, r, state)
	})

	session.AddHandler(interactionHandler)

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
