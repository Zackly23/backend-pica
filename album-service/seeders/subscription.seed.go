package seeders

import (
	"fmt"
	"time"

	"github.com/Zackly23/queue-app/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedSubscriptions(db *gorm.DB) error {
	subscriptions := []models.Subscription{
		{
			ID:               uuid.New(),
			SubscriptionType: "Basic",
			StorageCapacity:  100,        // dalam GB
			MaximumMediaSize: 0.1,      // 100 MB = 0.1 GB
			Features:         []byte(`["Basic Support", "3 GB Storage", "Max 100MB media upload"]`),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New(),
			SubscriptionType: "Advanced",
			StorageCapacity:  50,
			MaximumMediaSize: 1,        // 1 GB
			Features:         []byte(`["Priority Support", "15 GB Storage", "Max 1GB media upload"]`),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New(),
			SubscriptionType: "Pro",
			StorageCapacity:  1000,
			MaximumMediaSize: 5,
			Features:         []byte(`["Premium Support", "50 GB Storage", "Max 5GB media upload"]`),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	for _, sub := range subscriptions {
		fmt.Println("seed record : ", sub)
		var existing models.Subscription
		err := db.Where("subscription_type = ?", sub.SubscriptionType).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&sub).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
