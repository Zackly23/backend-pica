package jobs

import (
	"log"
	"time"

	"github.com/Zackly23/queue-app/models"
	"gorm.io/gorm"
)

func UpdateSubscriptionType(db *gorm.DB) {
	threshold := time.Now()

	var users []models.User

	// Ambil semua user dengan subscription aktif dan preload subscription-nya
	if err := db.Preload("UserSubscriptions").Where("subscription_free_status = ?", "active").Find(&users).Error; err != nil {
		log.Println("Gagal Mendapatkan Users")
		return
	}

	for _, user := range users {
		for _, sub := range user.UserSubscriptions {
			if sub.EndDate.Before(threshold) {
				// Subscription expired, ubah status
				db.Model(&user).Update("subscription_free_status", "inactive")
				break
			}
		}
	}
}
