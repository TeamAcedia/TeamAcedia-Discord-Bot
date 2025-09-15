package sticky_roles

import (
	"database/sql"
	"log"
	"slices"
	"teamacedia/discord-bot/internal/config"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

// OnMemberJoin handles re-joining members and assigns their previous roles or the default one.
func OnMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	// Try to fetch stored roles from DB
	oldRoles, err := getStoredRoles(m.User.ID, m.GuildID)
	if err != nil {
		log.Printf("DB error while fetching stored roles for %s: %v", m.User.Username, err)
		return
	}

	// Check if user has joined before
	joinedBefore, err := hasUserJoined(m.User.ID, m.GuildID)
	if err != nil {
		log.Printf("DB error while checking join history for %s: %v", m.User.Username, err)
		return
	}
	if !joinedBefore {
		// Register this first join
		err := registerUserJoined(m.User.ID, m.GuildID)
		if err != nil {
			log.Printf("DB error while registering join for %s: %v", m.User.Username, err)
		}
		// First time join give default member role
		if !hasRole(m.Member, config.Config.MemberRoleID) {
			err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, config.Config.MemberRoleID)
			if err != nil {
				log.Printf("Failed to add default role to %s: %v", m.User.Username, err)
			} else {
				log.Printf("Assigned default role to new member: %s", m.User.Username)
			}
		}
		return
	}

	if len(oldRoles) > 0 {
		// Restore previous roles
		for _, roleID := range oldRoles {
			err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleID)
			if err != nil {
				log.Printf("Failed to restore role %s for %s: %v", roleID, m.User.Username, err)
			}
		}
		log.Printf("Restored %d roles for returning member: %s", len(oldRoles), m.User.Username)
	}
}

// OnMemberUpdate is called whenever a member's roles are updated (add/remove).
func OnMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	userID := m.User.ID
	guildID := m.GuildID
	roles := m.Roles

	// Store updated roles
	err := storeRoles(userID, guildID, roles)
	if err != nil {
		log.Printf("Failed to update roles for %s: %v", m.User.Username, err)
	} else {
		log.Printf("Updated roles for member %s: %v", m.User.Username, roles)
	}
}

// OnRoleDelete cleans up sticky_roles DB when a role is deleted from the guild.
func OnRoleDelete(s *discordgo.Session, r *discordgo.GuildRoleDelete) {
	roleID := r.RoleID
	guildID := r.GuildID

	_, err := db.Exec(
		"DELETE FROM sticky_roles WHERE guild_id = ? AND role_id = ?",
		guildID, roleID,
	)
	if err != nil {
		log.Printf("Failed to cleanup deleted role %s from guild %s: %v", roleID, guildID, err)
	} else {
		log.Printf("Cleaned up deleted role %s from guild %s", roleID, guildID)
	}
}

// hasRole checks if a member already has a specific role.
func hasRole(member *discordgo.Member, roleID string) bool {
	return slices.Contains(member.Roles, roleID)
}

var db *sql.DB

// InitDB opens (or creates) the SQLite database and sets up the schema.
func InitDB(path string) error {
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	// Create table if it doesn't exist
	schema := `
	CREATE TABLE IF NOT EXISTS sticky_roles (
		user_id  TEXT NOT NULL,
		guild_id TEXT NOT NULL,
		role_id  TEXT NOT NULL,
		PRIMARY KEY (user_id, guild_id, role_id)
	);
	
	CREATE TABLE IF NOT EXISTS joins (
   		user_id  TEXT NOT NULL,
    	guild_id TEXT NOT NULL,
		PRIMARY KEY (user_id, guild_id)
	);

	`
	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	log.Printf("StickyRoles DB initialized at %s", path)
	return nil
}

// getStoredRoles retrieves roles for a user in a guild.
func getStoredRoles(userID, guildID string) ([]string, error) {
	rows, err := db.Query(
		"SELECT role_id FROM sticky_roles WHERE user_id = ? AND guild_id = ?",
		userID, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// registerUserJoined logs a user's join event.
func registerUserJoined(userID, guildID string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO joins (user_id, guild_id) VALUES (?, ?)",
		userID, guildID,
	)
	return err
}

// hasUserJoined checks if a user has joined before.
func hasUserJoined(userID, guildID string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM joins WHERE user_id = ? AND guild_id = ?",
		userID, guildID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// storeRoles saves roles for a user in a guild.
func storeRoles(userID, guildID string, roles []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear old roles
	_, err = tx.Exec(
		"DELETE FROM sticky_roles WHERE user_id = ? AND guild_id = ?",
		userID, guildID,
	)
	if err != nil {
		return err
	}

	// Insert new roles
	stmt, err := tx.Prepare(
		"INSERT INTO sticky_roles (user_id, guild_id, role_id) VALUES (?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, role := range roles {
		_, err := stmt.Exec(userID, guildID, role)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SyncGuildRoles iterates through every member in a guild and saves their current roles.
func SyncGuildRoles(s *discordgo.Session, guildID string) error {
	after := "" // used for pagination
	limit := 100

	for {
		members, err := s.GuildMembers(guildID, after, limit)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			break
		}

		for _, m := range members {
			if len(m.Roles) == 0 {
				continue
			}
			err := storeRoles(m.User.ID, guildID, m.Roles)
			if err != nil {
				log.Printf("Failed to sync roles for %s: %v", m.User.Username, err)
			} else {
				log.Printf("Synced roles for %s: %v", m.User.Username, m.Roles)
			}
		}

		// Pagination: Discord API returns up to `limit` users after the given ID
		after = members[len(members)-1].User.ID

		// If fewer than limit returned, we reached the end
		if len(members) < limit {
			break
		}
	}

	log.Printf("Finished syncing roles for guild %s", guildID)
	return nil
}
