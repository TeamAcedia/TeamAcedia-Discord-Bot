package db

import (
	"database/sql"
	"fmt"
	"teamacedia/discord-bot/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(path string) error {
	var err error
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	// Enable foreign keys for SQLite
	_, err = DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		text TEXT UNIQUE NOT NULL
	);

	`
	_, err = DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func AddReminder(reminder models.Reminder) error {
	_, err := DB.Exec("INSERT INTO reminders (user_id, text) VALUES (?, ?)", reminder.UserID, reminder.Text)
	return err
}

func DeleteReminder(reminder models.Reminder) error {
	_, err := DB.Exec("DELETE FROM reminders WHERE user_id = ? AND text = ?", reminder.UserID, reminder.Text)
	return err
}

func GetAllReminders() ([]models.Reminder, error) {
	rows, err := DB.Query("SELECT user_id, text FROM reminders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var r models.Reminder
		if err := rows.Scan(&r.UserID, &r.Text); err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, nil
}

func GetUserReminders(userID string) ([]models.Reminder, error) {
	rows, err := DB.Query("SELECT user_id, text FROM reminders WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []models.Reminder
	for rows.Next() {
		var r models.Reminder
		if err := rows.Scan(&r.UserID, &r.Text); err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	return reminders, nil
}
