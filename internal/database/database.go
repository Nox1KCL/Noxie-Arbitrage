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

func SaveSubscription(db *gorm.DB, sub *models.Subscription) error {
	result := db.Create(sub)
	if result.Error != nil {
		return fmt.Errorf("creating field for %d: %w", sub.TelegramChatID, result.Error)
	}
	return nil
}

func GetSubscriptions(db *gorm.DB, chatID int64) ([]*models.Subscription, error) {
	var subs []*models.Subscription
	result := db.Where("telegram_chat_id = ?", chatID).Find(&subs)
	if result.Error != nil {
		return nil, fmt.Errorf("looking for fields with ID-%d: %w", chatID, result.Error)
	}
	return subs, nil
}

func DeleteSubscription(db *gorm.DB, chatID int64, symbol string) error {
	result := db.Where("telegram_chat_id = ? AND symbol = ?", chatID, symbol).Delete(&models.Subscription{})
	if result.Error != nil {
		return fmt.Errorf("deleting sub with ID-%d | %s: %w", chatID, symbol, result.Error)
	}
	return nil
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
