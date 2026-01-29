package migrations

import (
	"gorm.io/gorm"
)

// Migration001AddReplyToFK adds the foreign key constraint for message threading
// This is done via raw SQL because GORM's auto-migration can have type matching issues
// with self-referential foreign keys
func Migration001AddReplyToFK() Migration {
	return Migration{
		ID:   "001_add_reply_to_fk",
		Name: "Add foreign key constraint for message reply threading",
		Up: func(db *gorm.DB) error {
			// 1. Clean up orphans (cast to text for safety)
			cleanupSQL := `
				UPDATE messages 
				SET reply_to_id = NULL 
				WHERE reply_to_id IS NOT NULL 
				AND reply_to_id::text NOT IN (SELECT id::text FROM messages)
			`
			if err := db.Exec(cleanupSQL).Error; err != nil {
				return err
			}

			// 2. Get the exact type of the 'id' column
			var idType string
			typeQuery := `
				SELECT data_type 
				FROM information_schema.columns 
				WHERE table_name = 'messages' AND column_name = 'id'
			`
			if err := db.Raw(typeQuery).Scan(&idType).Error; err != nil {
				return err
			}

			// 3. Force reply_to_id to MATCH id's type exactly
			// This prevents SQLSTATE 42804 (datatype mismatch)
			var alterSQL string
			if idType == "text" {
				alterSQL = "ALTER TABLE messages ALTER COLUMN reply_to_id TYPE text USING reply_to_id::text"
			} else {
				// Assumes uuid or compatible
				alterSQL = "ALTER TABLE messages ALTER COLUMN reply_to_id TYPE uuid USING reply_to_id::uuid"
			}

			if err := db.Exec(alterSQL).Error; err != nil {
				return err
			}

			// 4. Check if constraint already exists
			var count int64
			checkSQL := `
				SELECT COUNT(*) 
				FROM information_schema.table_constraints 
				WHERE constraint_name = 'fk_messages_reply_to' 
				AND table_name = 'messages'
			`
			if err := db.Raw(checkSQL).Scan(&count).Error; err != nil {
				return err
			}

			if count > 0 {
				return nil
			}

			// 5. Add Constraint
			addFKSQL := `
				ALTER TABLE messages 
				ADD CONSTRAINT fk_messages_reply_to 
				FOREIGN KEY (reply_to_id) 
				REFERENCES messages(id) 
				ON DELETE SET NULL 
				ON UPDATE CASCADE
			`
			return db.Exec(addFKSQL).Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec(`
				ALTER TABLE messages 
				DROP CONSTRAINT IF EXISTS fk_messages_reply_to
			`).Error
		},
	}
}

// Migration002EnsureUUIDExtension ensures the uuid-ossp extension is available
func Migration002EnsureUUIDExtension() Migration {
	return Migration{
		ID:   "002_ensure_uuid_extension",
		Name: "Ensure uuid-ossp extension is available",
		Up: func(db *gorm.DB) error {
			return db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error
		},
		Down: func(db *gorm.DB) error {
			// Don't drop extension as other things might depend on it
			return nil
		},
	}
}
