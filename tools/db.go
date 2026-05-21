package tools

import (
	"fmt"
	"log"
	"sync"

	"lumor_puls/config"
	"lumor_puls/types"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbOnce sync.Once
	db     *gorm.DB
	dbErr  error
)

// InitDB opens MySQL and auto-migrates tables.
func InitDB(cfg config.Config) error {
	dbOnce.Do(func() {
		instance, err := gorm.Open(mysql.Open(cfg.MySQLDsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			dbErr = fmt.Errorf("open mysql: %w", err)
			return
		}
		if err := instance.AutoMigrate(
			&types.MonitorTask{},
			&types.Snapshot{},
			&types.Signal{},
		); err != nil {
			dbErr = fmt.Errorf("migrate: %w", err)
			return
		}
		db = instance
		log.Println("mysql: connected and migrated")
	})
	return dbErr
}

// DB returns the shared GORM instance.
func DB() *gorm.DB {
	return db
}

// CloseDB closes the connection pool.
func CloseDB() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
