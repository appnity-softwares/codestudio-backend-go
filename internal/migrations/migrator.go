package migrations

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID        string // Unique identifier (e.g., "001_add_reply_to_fk")
	Name      string // Human-readable name
	Up        func(db *gorm.DB) error
	Down      func(db *gorm.DB) error
	DependsOn []string // IDs of migrations this depends on
}

// MigrationRecord tracks which migrations have been applied
type MigrationRecord struct {
	ID        string    `gorm:"primaryKey;type:text"`
	Name      string    `gorm:"type:text"`
	AppliedAt time.Time `gorm:"autoUpdateTime:nano"`
}

// TableName overrides the table name
func (MigrationRecord) TableName() string {
	return "schema_migrations"
}

// Migrator handles database migrations
type Migrator struct {
	db         *gorm.DB
	migrations []Migration
}

// NewMigrator creates a new migrator
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: GetMigrations(),
	}
}

// Run executes all pending migrations
func (m *Migrator) Run() error {
	// Ensure migrations table exists
	if err := m.db.AutoMigrate(&MigrationRecord{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	var applied []MigrationRecord
	if err := m.db.Find(&applied).Error; err != nil {
		return fmt.Errorf("failed to fetch applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, r := range applied {
		appliedMap[r.ID] = true
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if appliedMap[migration.ID] {
			continue
		}

		log.Info().Str("migration", migration.ID).Str("name", migration.Name).Msg("🔄 Running migration")

		// Check dependencies
		for _, dep := range migration.DependsOn {
			if !appliedMap[dep] {
				return fmt.Errorf("migration %s depends on %s which is not applied", migration.ID, dep)
			}
		}

		// Run migration in transaction
		if err := m.db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return err
			}

			// Record migration
			return tx.Create(&MigrationRecord{
				ID:   migration.ID,
				Name: migration.Name,
			}).Error
		}); err != nil {
			log.Error().Err(err).Str("migration", migration.ID).Msg("❌ Migration failed")
			return fmt.Errorf("migration %s failed: %w", migration.ID, err)
		}

		log.Info().Str("migration", migration.ID).Msg("✅ Migration completed")
	}

	return nil
}

// GetMigrations returns all registered migrations in order
func GetMigrations() []Migration {
	return []Migration{
		Migration001AddReplyToFK(),
		Migration002EnsureUUIDExtension(),
		Migration003AddPerformanceIndexes(),
	}
}
