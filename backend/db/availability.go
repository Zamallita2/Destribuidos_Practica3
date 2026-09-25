package db

import (
	"context"
	"gorm.io/gorm"
	"time"
)

func IsAvailable(conn *gorm.DB) bool {
	if conn == nil {
		return false
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	return sqlDB.PingContext(ctx) == nil
}
