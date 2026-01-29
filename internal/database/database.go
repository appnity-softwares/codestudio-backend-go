package database

import (
	"log"
	"sync"
	"time"

	"github.com/pushp314/devconnect-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Feature Flag Cache
var (
	featureCache      = make(map[string]bool)
	featureCacheMutex sync.RWMutex
	lastCacheUpdate   time.Time
	cacheTTL          = 1 * time.Minute
)

func Connect() {
	dsn := config.AppConfig.DatabaseURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Configure connection pool for production performance
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// Production-grade connection pool settings
	sqlDB.SetMaxOpenConns(25)                 // Max open connections to DB
	sqlDB.SetMaxIdleConns(10)                 // Max idle connections in pool
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Connection max lifetime

	DB = db
	log.Println("Connected to PostgreSQL with connection pooling (max: 25, idle: 10)")
}

// IsFeatureEnabled checks if a system setting (feature flag) is set to "true"
// Uses in-memory caching with 1-minute TTL to reduce DB load (Performance Optimization)
func IsFeatureEnabled(key string) bool {
	if DB == nil {
		return false
	}

	featureCacheMutex.RLock()
	if time.Since(lastCacheUpdate) < cacheTTL {
		val, exists := featureCache[key]
		featureCacheMutex.RUnlock()
		if exists {
			return val
		}
	} else {
		featureCacheMutex.RUnlock() // Release read lock to acquire write lock
		// Refresh Cache
		refreshCache()
		featureCacheMutex.RLock() // Re-acquire read lock
		val, exists := featureCache[key]
		featureCacheMutex.RUnlock()
		if exists {
			return val
		}
	}

	// Fallback for uncached keys during TTL
	var setting struct {
		Value string
	}
	if err := DB.Table("system_settings").Select("value").Where("key = ?", key).First(&setting).Error; err != nil {
		return false
	}
	return setting.Value == "true"
}

func refreshCache() {
	featureCacheMutex.Lock()
	defer featureCacheMutex.Unlock()

	// Double check time in case another routine refreshed it waiting for lock
	if time.Since(lastCacheUpdate) < cacheTTL {
		return
	}

	type Setting struct {
		Key   string
		Value string
	}
	var settings []Setting
	if err := DB.Table("system_settings").Select("key, value").Find(&settings).Error; err != nil {
		log.Printf("Error refreshing feature cache: %v", err)
		return
	}

	// Rebuild map
	featureCache = make(map[string]bool)
	for _, s := range settings {
		featureCache[s.Key] = (s.Value == "true")
	}
	lastCacheUpdate = time.Now()
}
