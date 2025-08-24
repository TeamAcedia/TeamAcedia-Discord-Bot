package reaction_roles

import (
	"fmt"
	"log"
	"teamacedia/discord-bot/internal/config"
	"teamacedia/discord-bot/internal/models"
	"time"

	"github.com/bwmarrin/discordgo"
)

// State holds the posted message ID so we know what to watch
type State struct {
	MessageID string
	Roles     []models.ReactionRole
}

// InitReactionRoles clears the channel, posts an embed, adds reactions, and returns state
func InitReactionRoles(s *discordgo.Session, roles []models.ReactionRole) (*State, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles provided")
	}

	channelID := config.Config.ReactionRolesChannelID

	// 1. Delete all existing messages in the channel
	msgs, err := s.ChannelMessages(channelID, 100, "", "", "")
	if err == nil {
		for _, m := range msgs {
			_ = s.ChannelMessageDelete(channelID, m.ID)
		}
	}

	// 2. Create an embed listing all roles
	desc := ""
	for _, rr := range roles {
		desc += fmt.Sprintf("%s - %s\n", rr.Emoji, rr.Name)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Reaction Roles",
		Description: desc,
		Color:       0x00FFFF, // Cyan
	}

	msg, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return nil, fmt.Errorf("failed to send embed: %w", err)
	}

	// 3. Add reactions to the message
	for _, rr := range roles {
		if err := s.MessageReactionAdd(channelID, msg.ID, rr.Emoji); err != nil {
			log.Printf("failed to add reaction %s: %v", rr.Emoji, err)
		}
	}

	return &State{
		MessageID: msg.ID,
		Roles:     roles,
	}, nil
}

// HandleReactionAdd toggles a role, removes the user's reaction, and sends a temporary embed
func HandleReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd, state *State) {
	if s.State.User != nil && r.UserID == s.State.User.ID {
		return
	}

	if r.MessageID != state.MessageID {
		return
	}

	for _, rr := range state.Roles {
		if r.Emoji.Name == rr.Emoji {
			// Get member to check roles
			member, err := s.GuildMember(r.GuildID, r.UserID)
			if err != nil {
				break
			}

			hasRole := false
			for _, roleID := range member.Roles {
				if roleID == rr.ID {
					hasRole = true
					break
				}
			}

			var description string
			if hasRole {
				_ = s.GuildMemberRoleRemove(r.GuildID, r.UserID, rr.ID)
				description = fmt.Sprintf("<@%s> I have removed the role <@&%s> from you.", r.UserID, rr.ID)
			} else {
				_ = s.GuildMemberRoleAdd(r.GuildID, r.UserID, rr.ID)
				description = fmt.Sprintf("<@%s> I have given the role <@&%s> to you.", r.UserID, rr.ID)
			}

			// Remove user's reaction
			_ = s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID)

			// Send a temporary embed message
			embed := &discordgo.MessageEmbed{
				Title:       "Role Update",
				Description: description,
				Color:       0x00ffcc,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "This message will be deleted in 15 seconds",
				},
			}

			msg, err := s.ChannelMessageSendEmbed(r.ChannelID, embed)
			if err == nil {
				go func(channelID, messageID string) {
					time.Sleep(15 * time.Second)
					_ = s.ChannelMessageDelete(channelID, messageID)
				}(r.ChannelID, msg.ID)
			}

			break
		}
	}
}
