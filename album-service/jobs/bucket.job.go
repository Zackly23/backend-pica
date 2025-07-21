package jobs

import (
	"log"
	"time"

	"github.com/Zackly23/queue-app/config"
	"github.com/Zackly23/queue-app/models"
	"github.com/Zackly23/queue-app/utils"
	"gorm.io/gorm"
)

func CleanUpUnusedFiles(db *gorm.DB) {
	bucketName := config.S3Bucket.BucketName
	threshold := time.Now().AddDate(0, 0, -60) // 60 hari terakhir

	var albums []models.Album
	if err := db.Preload("AlbumImages").Preload("AlbumVideos").
		Where("updated_at < ?", threshold).
		Find(&albums).Error; err != nil {
		log.Println("Gagal Mengambil Data Album:", err)
		return
	}

	for _, album := range albums {
		// Hapus gambar dari S3 dan database
		for _, img := range album.AlbumImages {
			if err := utils.DeleteFromS3(img.ImageURL, bucketName); err != nil {
				log.Printf("Gagal hapus file %s: %v", img.ImageURL, err)
				continue
			}
			if err := db.Delete(&img).Error; err != nil {
				log.Printf("Gagal hapus record image dari DB: %v", err)
			}
		}

		// Hapus video dari S3 dan database
		for _, vid := range album.AlbumVideos {
			if err := utils.DeleteFromS3(vid.VideoURL, bucketName); err != nil {
				log.Printf("Gagal hapus file %s: %v", vid.VideoURL, err)
				continue
			}
			if err := db.Delete(&vid).Error; err != nil {
				log.Printf("Gagal hapus record video dari DB: %v", err)
			}
		}

		// Opsional: hapus album itu sendiri
		if err := db.Delete(&album).Error; err != nil {
			log.Printf("Gagal hapus record album dari DB: %v", err)
		}
	}
}
