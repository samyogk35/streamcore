package database

import (
	"time"

	"gorm.io/gorm"
)

type DBUser struct {
	gorm.Model
	Name     string `gorm:"type:varchar(100);not null;uniqueIndex"`
	Password string `gorm:"type:varchar(100);not null"`
}

type DBMarketData struct {
	gorm.Model
	Symbol    string     `gorm:"type:varchar(20);not null;index"`
	Price     float64    `gorm:"type:decimal(18,8);not null"`
	Volume    float64    `gorm:"type:decimal(18,8);not null"`
	Side      string     `gorm:"type:varchar(10);not null"`
	Timestamp *time.Time
}
