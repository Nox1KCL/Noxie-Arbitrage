package database

import (
	"fmt"
	"os"

	"github.com/Nox1KCL/Arbitrage/internal/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PWD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return db, nil
}

func LoadSubscriptions(db *gorm.DB, subs []*models.Subscription) (*models.CachedMaps, error) {
	db.Find(&subs)
	var interestedSymbols = make(map[string]bool)
	var subsBySymbol = make(map[string][]models.Subscription)
	for _, sub := range subs {
		if !interestedSymbols[sub.Symbol] {
			interestedSymbols[sub.Symbol] = true
			subsBySymbol[sub.Symbol] = []models.Subscription{*sub}
		} else {
			subsBySymbol[sub.Symbol] = append(subsBySymbol[sub.Symbol], *sub)
		}
	}

	maps := &models.CachedMaps{
		InterestedSymbols: interestedSymbols,
		SubsBySymbol:      subsBySymbol,
	}

	return maps, nil
}
