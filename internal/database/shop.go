package database

import (
	"database/sql"
	"discord-giveaway-bot/internal/models"
	"fmt"
)

// Shop Item Operations

func (d *Database) CreateShopItem(item *models.ShopItem) (int64, error) {
	query := `
		INSERT INTO shop_items (
			name, description, price, stock, type, role_id, duration,
			required_balance, role_required, reply_message, image_url, hidden, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	var id int64
	err := d.db.QueryRow(query,
		item.Name, item.Description, item.Price, item.Stock, item.Type, item.RoleID, item.Duration,
		item.RequiredBalance, item.RoleRequired, item.ReplyMessage, item.ImageURL, item.Hidden, models.Now(),
	).Scan(&id)
	return id, err
}

func (d *Database) GetShopItem(name string) (*models.ShopItem, error) {
	query := `
		SELECT id, name, description, price, stock, type, role_id, duration,
		       required_balance, role_required, reply_message, image_url, hidden, created_at
		FROM shop_items WHERE name = $1
	`
	return d.scanShopItem(d.db.QueryRow(query, name))
}

func (d *Database) GetShopItemByID(id int64) (*models.ShopItem, error) {
	query := `
		SELECT id, name, description, price, stock, type, role_id, duration,
		       required_balance, role_required, reply_message, image_url, hidden, created_at
		FROM shop_items WHERE id = $1
	`
	return d.scanShopItem(d.db.QueryRow(query, id))
}

func (d *Database) UpdateShopItem(item *models.ShopItem) error {
	query := `
		UPDATE shop_items SET
			description = $1, price = $2, stock = $3, type = $4, role_id = $5, duration = $6,
			required_balance = $7, role_required = $8, reply_message = $9, image_url = $10, hidden = $11
		WHERE name = $12
	`
	_, err := d.db.Exec(query,
		item.Description, item.Price, item.Stock, item.Type, item.RoleID, item.Duration,
		item.RequiredBalance, item.RoleRequired, item.ReplyMessage, item.ImageURL, item.Hidden, item.Name,
	)
	return err
}

func (d *Database) DeleteShopItem(name string) error {
	_, err := d.db.Exec("DELETE FROM shop_items WHERE name = $1", name)
	return err
}

func (d *Database) RenameShopItem(oldName, newName string) error {
	_, err := d.db.Exec("UPDATE shop_items SET name = $1 WHERE name = $2", newName, oldName)
	return err
}

func (d *Database) GetShopItems(limit, offset int) ([]*models.ShopItem, error) {
	query := `
		SELECT id, name, description, price, stock, type, role_id, duration,
		       required_balance, role_required, reply_message, image_url, hidden, created_at
		FROM shop_items
		WHERE hidden = FALSE
		ORDER BY price ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := d.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ShopItem
	for rows.Next() {
		item, err := d.scanShopItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) GetAdminShopItems(limit, offset int) ([]*models.ShopItem, error) {
	query := `
		SELECT id, name, description, price, stock, type, role_id, duration,
		       required_balance, role_required, reply_message, image_url, hidden, created_at
		FROM shop_items
		ORDER BY name ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := d.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ShopItem
	for rows.Next() {
		item, err := d.scanShopItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) SearchShopItems(query string) ([]*models.ShopItem, error) {
	sqlQuery := `
		SELECT id, name, description, price, stock, type, role_id, duration,
		       required_balance, role_required, reply_message, image_url, hidden, created_at
		FROM shop_items
		WHERE name ILIKE $1
		ORDER BY name ASC
		LIMIT 25
	`
	rows, err := d.db.Query(sqlQuery, query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.ShopItem
	for rows.Next() {
		item, err := d.scanShopItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *Database) GetTotalShopItems() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM shop_items WHERE hidden = FALSE").Scan(&count)
	return count, err
}

// Inventory Operations

func (d *Database) PurchaseItem(userID, guildID string, itemID int64, quantity int, cost int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Check and deduct balance
	res, err := tx.Exec("UPDATE economy_users SET balance = balance - $1, total_spent = total_spent + $1 WHERE user_id = $2 AND guild_id = $3 AND balance >= $1", cost, userID, guildID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("insufficient funds")
	}

	// 2. Update stock if not infinite
	var stock int
	err = tx.QueryRow("SELECT stock FROM shop_items WHERE id = $1", itemID).Scan(&stock)
	if err != nil {
		return err
	}

	if stock != -1 {
		if stock < quantity {
			return fmt.Errorf("insufficient stock")
		}
		_, err = tx.Exec("UPDATE shop_items SET stock = stock - $1 WHERE id = $2", quantity, itemID)
		if err != nil {
			return err
		}
	}

	// 3. Add to inventory
	// Check if item exists in inventory
	var currentQty int
	err = tx.QueryRow("SELECT quantity FROM user_inventory WHERE user_id = $1 AND guild_id = $2 AND item_id = $3", userID, guildID, itemID).Scan(&currentQty)

	if err == sql.ErrNoRows {
		// Insert new
		_, err = tx.Exec(`
			INSERT INTO user_inventory (user_id, guild_id, item_id, quantity, acquired_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, guildID, itemID, quantity, models.Now())
	} else if err == nil {
		// Update existing
		_, err = tx.Exec("UPDATE user_inventory SET quantity = quantity + $1 WHERE user_id = $2 AND guild_id = $3 AND item_id = $4", quantity, userID, guildID, itemID)
	} else {
		return err
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateStockAndInventory updates stock and adds item to inventory (no coin deduction)
func (d *Database) UpdateStockAndInventory(userID, guildID string, itemID int64, quantity int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Update stock if not infinite
	var stock int
	err = tx.QueryRow("SELECT stock FROM shop_items WHERE id = $1", itemID).Scan(&stock)
	if err != nil {
		return err
	}

	if stock != -1 {
		if stock < quantity {
			return fmt.Errorf("insufficient stock")
		}
		_, err = tx.Exec("UPDATE shop_items SET stock = stock - $1 WHERE id = $2", quantity, itemID)
		if err != nil {
			return err
		}
	}

	// 2. Add to inventory
	var currentQty int
	err = tx.QueryRow("SELECT quantity FROM user_inventory WHERE user_id = $1 AND guild_id = $2 AND item_id = $3", userID, guildID, itemID).Scan(&currentQty)

	if err == sql.ErrNoRows {
		// Insert new
		_, err = tx.Exec(`
			INSERT INTO user_inventory (user_id, guild_id, item_id, quantity, acquired_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, guildID, itemID, quantity, models.Now())
	} else if err == nil {
		// Update existing
		_, err = tx.Exec("UPDATE user_inventory SET quantity = quantity + $1 WHERE user_id = $2 AND guild_id = $3 AND item_id = $4", quantity, userID, guildID, itemID)
	} else {
		return err
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) GetUserInventory(userID, guildID string) ([]*models.InventoryItem, error) {
	query := `
		SELECT i.id, i.user_id, i.guild_id, i.item_id, i.quantity, i.acquired_at, i.expires_at,
		       s.name, s.type
		FROM user_inventory i
		JOIN shop_items s ON i.item_id = s.id
		WHERE i.user_id = $1 AND i.guild_id = $2
		ORDER BY i.acquired_at DESC
	`
	rows, err := d.db.Query(query, userID, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.InventoryItem
	for rows.Next() {
		var i models.InventoryItem
		err := rows.Scan(
			&i.ID, &i.UserID, &i.GuildID, &i.ItemID, &i.Quantity, &i.AcquiredAt, &i.ExpiresAt,
			&i.ItemName, &i.ItemType,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	return items, nil
}

func (d *Database) GiveItem(userID, guildID string, itemID int64, quantity int) error {
	// Admin bypass: just add to inventory, ignore cost/stock
	var currentQty int
	err := d.db.QueryRow("SELECT quantity FROM user_inventory WHERE user_id = $1 AND guild_id = $2 AND item_id = $3", userID, guildID, itemID).Scan(&currentQty)

	if err == sql.ErrNoRows {
		_, err = d.db.Exec(`
			INSERT INTO user_inventory (user_id, guild_id, item_id, quantity, acquired_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, guildID, itemID, quantity, models.Now())
	} else if err == nil {
		_, err = d.db.Exec("UPDATE user_inventory SET quantity = quantity + $1 WHERE user_id = $2 AND guild_id = $3 AND item_id = $4", quantity, userID, guildID, itemID)
	} else {
		return err
	}
	return err
}

// Helpers

func (d *Database) scanShopItem(row *sql.Row) (*models.ShopItem, error) {
	var i models.ShopItem
	var roleID, roleRequired, replyMessage, imageURL sql.NullString

	err := row.Scan(
		&i.ID, &i.Name, &i.Description, &i.Price, &i.Stock, &i.Type,
		&roleID, &i.Duration, &i.RequiredBalance, &roleRequired,
		&replyMessage, &imageURL, &i.Hidden, &i.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	i.RoleID = roleID.String
	i.RoleRequired = roleRequired.String
	i.ReplyMessage = replyMessage.String
	i.ImageURL = imageURL.String

	return &i, nil
}

func (d *Database) scanShopItemRows(rows *sql.Rows) (*models.ShopItem, error) {
	var i models.ShopItem
	var roleID, roleRequired, replyMessage, imageURL sql.NullString

	err := rows.Scan(
		&i.ID, &i.Name, &i.Description, &i.Price, &i.Stock, &i.Type,
		&roleID, &i.Duration, &i.RequiredBalance, &roleRequired,
		&replyMessage, &imageURL, &i.Hidden, &i.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	i.RoleID = roleID.String
	i.RoleRequired = roleRequired.String
	i.ReplyMessage = replyMessage.String
	i.ImageURL = imageURL.String

	return &i, nil
}

// Redeem Code Operations

func (d *Database) CreateRedeemCode(code string, itemID int64, userID, guildID string) error {
	query := `
		INSERT INTO redeem_codes (code, item_id, user_id, guild_id, is_claimed, created_at)
		VALUES ($1, $2, $3, $4, FALSE, $5)
	`
	_, err := d.db.Exec(query, code, itemID, userID, guildID, models.Now())
	return err
}

func (d *Database) GetRedeemCode(code string) (*models.RedeemCode, error) {
	query := `
		SELECT r.code, r.item_id, r.user_id, r.guild_id, r.is_claimed, r.created_at,
		       s.name, s.description, s.price
		FROM redeem_codes r
		JOIN shop_items s ON r.item_id = s.id
		WHERE r.code = $1
	`
	var r models.RedeemCode
	err := d.db.QueryRow(query, code).Scan(
		&r.Code, &r.ItemID, &r.UserID, &r.GuildID, &r.IsClaimed, &r.CreatedAt,
		&r.ItemName, &r.ItemDescription, &r.ItemPrice,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *Database) MarkRedeemCodeClaimed(code string) error {
	_, err := d.db.Exec("UPDATE redeem_codes SET is_claimed = TRUE WHERE code = $1", code)
	return err
}
