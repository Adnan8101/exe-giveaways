package database

import (
	"database/sql"
)

func (d *Database) GetGuildPrefix(guildID string) (string, error) {
	var prefix string
	err := d.db.QueryRow("SELECT prefix FROM guild_settings WHERE guild_id = $1", guildID).Scan(&prefix)
	if err == sql.ErrNoRows {
		return "!", nil
	}
	if err != nil {
		return "!", err
	}
	return prefix, nil
}

func (d *Database) SetGuildPrefix(guildID, prefix string) error {
	query := `
		INSERT INTO guild_settings (guild_id, prefix) VALUES ($1, $2)
		ON CONFLICT(guild_id) DO UPDATE SET prefix = EXCLUDED.prefix
	`
	_, err := d.db.Exec(query, guildID, prefix)
	return err
}
