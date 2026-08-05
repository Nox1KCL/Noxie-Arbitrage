package models

type Subscription struct {
	ID                    uint     `gorm:"primaryKey"`
	TelegramChatID        int64    `gorm:"not null;index;uniqueIndex:idx_user_sym"`
	Symbol                string   `gorm:"not null;index;uniqueIndex:idx_user_sym"`
	MinSpreadPercent      float64  `gorm:"not null"`
	MinVolume             float64  `gorm:"not null"`
	MinPriceChangePercent float64  `gorm:"not null"`
	// Exchanges             []string `gorm:"text[]"`
}

type CachedMaps struct {
	InterestedSymbols map[string]bool
	SubsBySymbol      map[string][]Subscription
}
