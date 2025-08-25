package member_role

import (
	"log"
	"time"

	"teamacedia/discord-bot/internal/config"

	"github.com/bwmarrin/discordgo"
)

// AssignRoleToAll assigns the specified role to all members of the guild,
// but only if they don't already have it.
func AssignRoleToAll(s *discordgo.Session, guildID string, roleID string) {
	members := []*discordgo.Member{}
	after := ""

	for {
		batch, err := s.GuildMembers(guildID, after, 1000) // Fetch in batches
		if err != nil {
			log.Printf("Error fetching members: %v\n", err)
			return
		}
		if len(batch) == 0 {
			break
		}

		members = append(members, batch...)
		after = batch[len(batch)-1].User.ID
	}

	for _, member := range members {
		if hasRole(member, roleID) {
			continue // Skip members who already have the role
		}

		err := s.GuildMemberRoleAdd(guildID, member.User.ID, roleID)
		if err != nil {
			log.Printf("Failed to add role to %s: %v\n", member.User.Username, err)
		} else {
			log.Printf("Assigned role to %s\n", member.User.Username)
		}
		time.Sleep(250 * time.Millisecond) // Avoid hitting rate limits
	}

	log.Println("Finished assigning roles to all members!")
}

// OnMemberJoin automatically assigns the role to new members if they don't already have it.
func OnMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if hasRole(m.Member, config.Config.MemberRoleID) {
		return
	}

	err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, config.Config.MemberRoleID)
	if err != nil {
		log.Printf("Failed to add role to %s: %v\n", m.User.Username, err)
	} else {
		log.Printf("Assigned role to new member: %s\n", m.User.Username)
	}
}

// hasRole checks if a member already has a specific role
func hasRole(member *discordgo.Member, roleID string) bool {
	for _, r := range member.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}
