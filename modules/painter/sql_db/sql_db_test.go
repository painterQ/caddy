package sql_db_test

import (
	"fmt"

	"github.com/caddyserver/caddy/v2/modules/painter/sql_db"
	"gorm.io/gorm"
)

// Example usage within another caddy module: call mysqlapp.GetDB()
func UseDBExample() error {
	db := sql_db.GetDB()
	if db == nil {
		return fmt.Errorf("mysql db not configured")
	}

	// Example: ping via underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Ping(); err != nil {
		return err
	}

	// Or use GORM normally:
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	fmt.Println("users:", count)
	return nil
}

type User struct {
	ID   uint
	Name string
}
