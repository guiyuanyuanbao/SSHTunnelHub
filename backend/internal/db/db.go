package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sshtunnelhub/internal/model"
)

var DB *gorm.DB

// InitDB initializes SQLite database using pure-Go driver
func InitDB(dataDir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "sshtunnelhub.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Auto migrate schemas
	if err := db.AutoMigrate(&model.Host{}, &model.Tunnel{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database schemas: %w", err)
	}

	DB = db
	log.Printf("[DB] SQLite database initialized at %s", dbPath)
	return db, nil
}
