package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Zackly23/queue-app/models"
	notif "github.com/Zackly23/queue-app/proto/notificationpb"
	"github.com/Zackly23/queue-app/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserDetail struct {
	UserID uuid.UUID `json:"user_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	FullName	 string `json:"full_name"`
	Email        string `json:"email"`
	ProfilePicture string `json:"profile_picture,omitempty"`
}

type AlbumCommentResponse struct {
	ID        uuid.UUID `json:"id"`
	AlbumID   uuid.UUID `json:"album_id"`
	UserID    uuid.UUID `json:"user_id"`
	User      string    `json:"user"` // bisa nama user atau struct kecil
	UserAvatar string	`json:"user_avatar"`
	Comment   string    `json:"comment"`
	CreatedAt string    `json:"created_at"` // dalam format human-readable
}


type AlbumMedia struct {
	AlbumMediaID uuid.UUID      `json:"album_media_id"`
	MediaID      uuid.UUID      `json:"media_id"`
	AlbumID      uuid.UUID      `json:"album_id"`
	Description  string    `json:"description"`
	LikesCount   uint      `json:"likes_count"`
	UserHasLike	bool		`json:"user_has_like"`
	URL          string    `json:"url"`
	Size         float32   `json:"size"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedAtModified    string  `json:"created_at_modified"`
	MediaKind    string    `json:"media_kind"` // "image" or "video"
}

type AlbumDetailRequest struct {
	AlbumID      uuid.UUID  	`json:"album_id"`
	UserDetail	UserDetail    `json:"user_detail"`
	Tags         []string      `json:"tags,omitempty"`         // array string
	Title        string        `json:"title"`
	Description  string        `json:"description,omitempty"`
	LikeCount	int				`json:"like_count"`
	ViewCount	int 			`json:"view_count"`
	ImageCount int `json:"image_count"`
	VideoCount int `json:"video_count"`
	AlbumPrivacy string        `json:"album_privacy"`          // e.g. "public", "private"
	TargetEmail  json.RawMessage `json:"target_email"`
	AlbumImages  []AlbumImageRequest `json:"album_images,omitempty"` // nested images
	AlbumVideos  []AlbumVideoRequest `json:"album_videos,omitempty"`
	CreatedAt 	string		`json:"created_at"`
}

type AlbumImageRequest struct {
	ID          uuid.UUID           `json:"id"`
	ImageURL    string 			`json:"image_url"`
	Description string 			`json:"description,omitempty"`
	LikesCount  uint           `json:"likes_count" gorm:"default:0"`
	Size		float32			`json:"size"`
	Type 		string 			`json:"type"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

type AlbumVideoRequest struct {
	ID          uuid.UUID           `json:"id"`
	VideoURL 	string 			`json:"video_url"`
	Description string 			`json:"description,omitempty"`
	AlbumID     uint           `json:"album_id" gorm:"not null;index"`
	LikesCount  uint           `json:"likes_count"`
	Size		float32			`json:"size"`
	Type        string 			`json:"type"`
	ThumbnailURL string        `json:"thumbnail_url,omitempty"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func updateImage(db *gorm.DB, imageDescription string, albumImageID any) error {
	// Coba konversi ID
	if idStr, ok := albumImageID.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil{
			// Update gambar jika ID valid
			var existingImage models.AlbumImage
			if err := db.First(&existingImage, id).Error; err == nil {
				existingImage.Description = imageDescription
				return db.Save(&existingImage).Error
			}
		}
	}

	return &fiber.Error{Code: 300, Message: "Failed to update"}

}

func updateVideo(db *gorm.DB, videoDescription string, albumVideoId any) error {

	if idStr, ok := albumVideoId.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil {
			var existingVideo models.AlbumVideo
			if err := db.First(&existingVideo, id).Error; err == nil {
				existingVideo.Description = videoDescription
				// ThumbnailURL bisa digenerate kemudian
				return db.Save(&existingVideo).Error
			}
		}
	}

	return &fiber.Error{Code: 300, Message: "Failed to update video"}
	
}

func storeImage(db *gorm.DB, file *multipart.FileHeader, albumID uuid.UUID, imageDescription string, albumImageID any) error {
	s3URL, err := utils.UploadToS3(file, "images/albums/album_" + albumID.String() + "/"+ file.Filename)
	if err != nil {
		return fmt.Errorf("gagal mengupload ke S3: %w", err)
	}

	sizeMB := float32(file.Size) / (1024 * 1024)
	mimeType := file.Header.Get("Content-Type")

	// Coba konversi ID
	if idStr, ok := albumImageID.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil{
			// Update gambar jika ID valid
			var existingImage models.AlbumImage
			if err := db.First(&existingImage, id).Error; err == nil {
				existingImage.ImageURL = s3URL
				existingImage.Size = sizeMB
				existingImage.Type = mimeType
				existingImage.Description = imageDescription
				return db.Save(&existingImage).Error
			}
		}
	}

	// Jika tidak ada ID, buat baru
	image := models.AlbumImage{
		AlbumID:     albumID,
		ImageURL:    s3URL,
		Size:        sizeMB,
		Type:        mimeType,
		Description: imageDescription,
	}

	return db.Create(&image).Error
}

func storeVideo(db *gorm.DB, file *multipart.FileHeader, albumID uuid.UUID, videoDescription string, albumVideoId any) error {
	s3URL, err := utils.UploadToS3(file, "videos/albums/album_"+  albumID.String() + "/" + file.Filename)
	if err != nil {
		return fmt.Errorf("gagal mengupload ke S3: %w", err)
	}

	//Generate Thumbnail
	thumnailVideo := "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/images/default/default_video_thumb.png"

	sizeMB := float32(file.Size) / (1024 * 1024)
	mimeType := file.Header.Get("Content-Type")

	if idStr, ok := albumVideoId.(string); ok {
		if id, err := uuid.Parse(idStr); err == nil {
			var existingVideo models.AlbumVideo
			if err := db.First(&existingVideo, id).Error; err == nil {
				existingVideo.VideoURL = s3URL
				existingVideo.Size = sizeMB
				existingVideo.Type = mimeType
				existingVideo.Description = videoDescription
				existingVideo.ThumbnailURL = thumnailVideo

				// ambil ThumbnailURL
				return db.Save(&existingVideo).Error
			}
		}
	}

	// Buat video baru jika ID tidak valid
	video := models.AlbumVideo{
		AlbumID:      albumID,
		VideoURL:     s3URL,
		Size:         sizeMB,
		Type:         mimeType,
		Description:  videoDescription,
		ThumbnailURL: thumnailVideo,
	}

	return db.Create(&video).Error
}


func deleteVideo(db *gorm.DB, albumVideoId uuid.UUID) error {
	var video models.AlbumVideo

	// Cari video berdasarkan ID
	err := db.First(&video, "id = ?", albumVideoId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("video with ID %s not found", albumVideoId)
		}
		return fmt.Errorf("failed to query video: %w", err)
	}

	fmt.Println(albumVideoId)

	// Hapus data video di database
	if err := db.Delete(&video).Error; err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}
	// Hapus file dari S3
	if err := utils.DeleteFromS3(video.VideoURL, "s3-pixovaulty"); err != nil {
		fmt.Printf("⚠️ Failed to delete S3 video: %v\n", err)
	}

	return nil
}

func deleteImage(db *gorm.DB, albumImageId uuid.UUID) error {
	var image models.AlbumImage

	fmt.Println("image id : ", albumImageId)

	err := db.First(&image, "id = ?", albumImageId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Println("❌ GORM: record not found")
			return fmt.Errorf("image with ID %s not found", albumImageId)
		}
		fmt.Println("❌ GORM Error lain:", err)
		return fmt.Errorf("failed to query image: %w", err)
	}

	fmt.Println("✅ Berhasil ambil image:", image)

	// Hapus image record
	if err := db.Delete(&image).Error; err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if image.ImageURL != "" {
		fmt.Println("🔄 File yang akan dihapus:", image.ImageURL)
		// Hapus file dari S3
		if err := utils.DeleteFromS3(image.ImageURL, "s3-pixovaulty"); err != nil {
			fmt.Printf("⚠️ Failed to delete S3 video: %v\n", err)
		}
	
	}

	return nil
}


func storeTags(form *multipart.Form, db *gorm.DB, album models.Album) error {
	tags := form.Value["tags"]
	// fmt.Println("tags : ", tags)
	for _, tag := range tags {
		var tagModel models.AlbumTag
		// Try to find the tag, if not found, create it
		if err := db.Where("tag_name = ?", tag).First(&tagModel).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				tagModel = models.AlbumTag{TagName: tag}
				if errCreate := db.Create(&tagModel).Error; errCreate != nil {
					return fmt.Errorf("failed to create tag: %w", err)
				}
			} else {
				return fmt.Errorf("failed to query tag: %w", err)
			}
		}

		//if tag has associated do nothing
		if err := db.Model(&album).Association("Tags").Append(&tagModel); err != nil {
			return fmt.Errorf("failed to associate tag: %w", err)
		}
	}

	return nil
}

func StoreAlbums(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	userID, err := utils.GetUserID(ctx)

	fmt.Println("userID : x ", userID)
	

	if err != nil {
		return ctx.SendStatus(200)
	}

	var user models.User

	if errAcc := db.Where("id = ?", userID).First(&user).Error; errAcc != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Akun Tidak Ditemukan"})
	}


	// Validasi form input
	title := ctx.FormValue("title")
	if title == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Judul album wajib diisi",
		})
	}

	description := ctx.FormValue("description")
	albumPrivacy := ctx.FormValue("album_privacy")
	
	//get target email
	form, err := ctx.MultipartForm()

	var targetEmailJSON []string
	if albumPrivacy == "restricted" {
		targetEmailJSON = form.Value["target_emails"]
	}

	var targetEmailRaw json.RawMessage
	if len(targetEmailJSON) > 0 {
		if marshaled, errMars := json.Marshal(targetEmailJSON); errMars == nil {
			targetEmailRaw = marshaled
		}
	}
	
	

	// Simpan Album
	album := models.Album{
		UserID:       userID,
		Title:        title,
		Description:  description,
		AlbumPrivacy: albumPrivacy,
		TargetEmail:  targetEmailRaw,
		CreatedAt:    time.Now(),
	}

	if err := db.Create(&album).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan album",
			"error":   err.Error(),
		})
	}

	
	//Store Tags
	if err := storeTags(form, db, album); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan tags",
			"error":   err.Error(),
		})
	}
	

	// Upload file (images & videos)
	// if err != nil {
	// 	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"message": "Gagal membaca file upload",
	// 		"error":   err.Error(),
	// 	})
	// }

	// Upload file (images & videos)
	images := form.File["album_images"]
	imageDescriptions := form.Value["image_descriptions"]

	fmt.Println("images ", images)
	for index, file := range images {
		fmt.Println("Image : ", file)
		fmt.Println("index image : ", index)
		imageDescription := imageDescriptions[index]

		fmt.Println("Deskripsi : ", imageDescription)
		if err := storeImage(db, file, album.ID, imageDescription, nil); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan gambar",
				"error":   err.Error(),
			})
		}
	}

	videos := form.File["album_videos"]
	videoDescriptions := form.Value["video_descriptions"]
	for index, file := range videos {
		videoDescription := videoDescriptions[index]
		if err := storeVideo(db, file, album.ID, videoDescription, nil); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan video",
				"error":   err.Error(),
			})
		}
	}

	sharedBy := user.UserName
	if sharedBy == "" {
		sharedBy = user.FirstName + " " + user.LastName
	}

	//get random image or thumbnail video
			// Get random cover image from album images
	var albumFull models.Album
	if errAlbum := db.Preload("AlbumImages").Preload("AlbumVideos").Where("id = ?", album.ID).First(&albumFull).Error; errAlbum != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal Mengambil Album",
			"error":   err.Error(),
		})
	}

	coverImage := ""
	if len(albumFull.AlbumImages) > 0 {
		randomIdx := time.Now().UnixNano() % int64(len(albumFull.AlbumImages))
		coverImage = albumFull.AlbumImages[randomIdx].ImageURL
	} else {
		randomIdx := time.Now().UnixNano() % int64(len(albumFull.AlbumVideos))
		coverImage = albumFull.AlbumVideos[randomIdx].ThumbnailURL
	}


	albumFull.CoverImage = coverImage

	if err := db.Save(&albumFull).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal update album"})
	}

	// Send notifications in background
	go func(album models.Album, user models.User, emails []string) {
		var wg sync.WaitGroup

		for _, email := range emails {
			wg.Add(1)
			go func(album models.Album, user models.User, email string) {
				defer wg.Done()

				ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
					To:      email,
					Subject: "Anda Telah Ditambahkan ke Album",
					Type:    "album-invitation",
					Name:    email,
					Body:    fmt.Sprintf("Klik link berikut untuk melihat album: http://localhost:5173/album-share?email=%s", email),
					Metadata: map[string]string{
						"album_title": album.Title,
						"album_link" : fmt.Sprintf("http://localhost:5173/albums/%s/details", album.ID),
						"platform_name": "PixoVaulty",
						"platform_url": "www.pixovaulty.com",
						"shared_by": sharedBy,
					},
				})
				if err != nil {
					log.Printf("Gagal mengirim notifikasi ke %s: %v", email, err)
				}
			}(album, user, email)
		}

		wg.Wait()
	}(album, user, targetEmailJSON)


	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Album berhasil disimpan",
		"album":   album,
	})
}

func UpdateAlbum(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	albumID := ctx.Params("albumID")

	// albumId, errParse := uuid.Parse(albumID)

	// if errParse != nil {
		// return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Gagal Parsing Album ID"})
	// }

	var albumRequest models.Album
	if err := db.Where("id = ?", albumID).First(&albumRequest).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album Tidak Ditemukan"})
	}

	if (userID != albumRequest.UserID) {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Album does not belong to user",
		})
	}

	form, formErr := ctx.MultipartForm()
	if formErr != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Form tidak valid"})
	}

	// Update basic info
	albumRequest.Title = ctx.FormValue("title")
	albumRequest.Description = ctx.FormValue("description")
	albumRequest.AlbumPrivacy = ctx.FormValue("album_privacy")
	albumRequest.UpdatedAt = time.Now()

	if albumRequest.AlbumPrivacy == "restricted" {
		if targetEmails, ok := form.Value["target_emails"]; ok {
			if marshaled, err := json.Marshal(targetEmails); err == nil {
				albumRequest.TargetEmail = marshaled
			}
		}
	}

	if err := db.Save(&albumRequest).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal update album"})
	}

	// Upload file (images & videos)
	images := form.File["album_images"]
	imageDescriptions := form.Value["image_descriptions"]
	imageStatuses := form.Value["image_statuses"]

	fmt.Println(imageStatuses)
	// fmt.Println("images ada berapa : ", images)
	albumImageIds := form.Value["album_image_ids"]

	fmt.Println("Jumlah images         :", len(images))
	fmt.Println("Jumlah imageStatuses  :", len(imageStatuses))
	fmt.Println("Jumlah imageDesc      :", len(imageDescriptions))
	fmt.Println("Jumlah albumImageIds  :", len(albumImageIds))
	imageIndex := 0
	for index, imageStatus := range imageStatuses {
		// fmt.Println(file)
		imageDescription := imageDescriptions[index]
		// imageStatus := imageStatuses[index]
		albumImageId := albumImageIds[index]

		fmt.Println("status image : ", imageStatus)
		fmt.Println("index : ", index)

		if (imageStatus == "delete") {
			fmt.Println("status image  delete: ", imageStatus)

			albumImageID, errParse := uuid.Parse(albumImageId)

			if errParse != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Gagal Parse video ID",
				})
			}

			fmt.Println("album id image : ", albumImageID)
			
			if err := deleteImage(db, albumImageID); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error Delete Image",
				})
			}

		} else if (imageStatus == "new") {
			fmt.Println("status image else : ", imageStatus)

			file := images[imageIndex]

			if err := storeImage(db, file, albumRequest.ID, imageDescription, albumImageId); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Gagal menyimpan gambar",
					"error":   err.Error(),
				})
			}

			imageIndex++
			
		} else {
			fmt.Println("status image else : ", imageStatus)

			
			if err := updateImage(db, imageDescription, albumImageId); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Gagal Update gambar",
					"error":   err.Error(),
				})
			}
		}
	}

	videos := form.File["album_videos"]
	videoDescriptions := form.Value["video_descriptions"]
	albumVideoIds := form.Value["album_video_ids"]
	videoStatuses := form.Value["video_statuses"]
	videoIndex := 0


	fmt.Println("Jumlah video         :", len(videos))
	fmt.Println("Jumlah videoStatuses  :", len(videoStatuses))
	fmt.Println("Jumlah videoDescriptions      :", len(videoDescriptions))
	fmt.Println("Jumlah albumVideoIds  :", len(albumVideoIds))
	for index, videoStatus := range videoStatuses {
		videoDescription := videoDescriptions[index]
		// videoStatus := videoStatuses[index]
		albumVideoID := albumVideoIds[index]
		
		if videoStatus == "delete" {

			albumVideoId, errParse := uuid.Parse(albumVideoID)

			if errParse != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Gagal Parse video ID",
				})
			}

			if err := deleteVideo(db, albumVideoId); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Gagal Menghapus video",
					"error":   err.Error(),
				})
			}
		} else if (videoStatus == "new") {

			file := videos[videoIndex]

			if err := storeVideo(db, file, albumRequest.ID, videoDescription, albumVideoID); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Gagal menyimpan video",
					"error":   err.Error(),
				})
			}

			videoIndex++
		} else {
			if err := updateVideo(db, videoDescription, albumVideoID); err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Gagal Update video",
					"error":   err.Error(),
				})
			}
		}
	}

		//Store Tags
	if err := storeTags(form, db, albumRequest); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan tags",
			"error":   err.Error(),
		})
	}

	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message" : "Album Berhasil Diupdate",
	})
}


func GetAlbum(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User

	if errAcc := db.Where("id = ?", userID).First(&user).Error; errAcc != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Akun Tidak Ditemukan"})
	}

	albumID := ctx.Params("albumId")

	albumId, errParse := uuid.Parse(albumID)

	// fmt.Println("Album kok ID : ", albumId)

	if errParse != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Album ID is failed to parse",
		})
	}

	var albumRequest models.Album
	if errAlbum := db.Preload("Tags").Preload("AlbumImages").Preload("AlbumVideos").Preload("User").Where("id = ?", albumId).First(&albumRequest).Error; errAlbum != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album Tidak Ditemukan"})
	}

	// if userID != albumRequest.UserID {
	// 	return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
	// 		"message": "Album does not belong to user",
	// 	})
	// }

	switch albumRequest.AlbumPrivacy {
	case "restricted":
		// Jika user adalah pemilik album, izinkan langsung
		if user.ID == albumRequest.UserID {
			break
		}

		var allowedEmails []string
		if errEmail := json.Unmarshal(albumRequest.TargetEmail, &allowedEmails); errEmail != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal Mendapatkan Target Email",
			})
		}

		// Cek apakah email user ada di daftar allowedEmails
		isAllowed := false
		for _, email := range allowedEmails {
			if email == user.Email {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "User tidak diperbolehkan melihat album",
			})
		}

	case "public":
		// Tidak ada batasan akses
	case "private":
		// Jika user adalah pemilik album, izinkan
		if user.ID == albumRequest.UserID {
			break
		}
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User tidak diperbolehkan melihat album",
		})
	}


		
	var albumMedias []AlbumMedia
	var indexMedia uuid.UUID
	imageCount := 0

	for _, img := range albumRequest.AlbumImages {
	// Check apakah user sudah like media ini
		hasLike := true
		if errLike := db.Where("user_id = ? AND media_id = ?", userID, img.ID).First(&models.MediaLike{}).Error; errLike != nil {
			if errors.Is(errLike, gorm.ErrRecordNotFound) {
				hasLike = false
			} else {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal cek like media"})
			}
		}

		// Ambil key dari URL
		key := strings.TrimPrefix(img.ImageURL, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		signedURL, errURL := utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}

		indexMedia = uuid.New()
		albumMedias = append(albumMedias, AlbumMedia{
			AlbumMediaID: indexMedia,
			MediaID:      img.ID,
			AlbumID:      img.AlbumID,
			Description:  img.Description,
			LikesCount:   img.LikesCount,
			URL:          signedURL,
			Size:         img.Size,
			Type:         img.Type,
			CreatedAt:    img.CreatedAt,
			CreatedAtModified: img.CreatedAt.Format("02 Jan 2006"),
			UserHasLike:  hasLike,
			MediaKind:    "image",
		})

		imageCount++
	}

	videoCount := 0

	for _, vid := range albumRequest.AlbumVideos {
		hasLike := true
		if errLikeMedia := db.Where("user_id = ? AND media_id = ?", userID, vid.ID).First(&models.MediaLike{}).Error; errLikeMedia != nil {
			if errors.Is(errLikeMedia, gorm.ErrRecordNotFound) {
				hasLike = false
			} else {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal cek like media"})
			}
		}

		key := strings.TrimPrefix(vid.VideoURL, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		signedURL, errURL := utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}


		indexMedia = uuid.New()
		albumMedias = append(albumMedias, AlbumMedia{
			AlbumMediaID: indexMedia,
			MediaID:      vid.ID,
			AlbumID:      vid.AlbumID,
			Description:  vid.Description,
			LikesCount:   vid.LikesCount,
			UserHasLike:  hasLike,
			URL:          signedURL,
			Size:         vid.Size,
			Type:         vid.Type,
			CreatedAt:    vid.CreatedAt,
			CreatedAtModified: vid.CreatedAt.Format("02 January 2006"),
			MediaKind:    "video",
		})

		videoCount++
		// likeCounts += int(vid.LikesCount)
	}


	sortBy := ctx.Query("sort_by", "date") // Options: date, title, popular
	orderBy := ctx.Query("order_by", "DESC") // Options: DESC, ASC


	lessFunc := func(i, j int) bool {
		return albumMedias[i].CreatedAt.After(albumMedias[j].CreatedAt) // default: by date desc
	}

	switch sortBy {
	case "date":
		lessFunc = func(i, j int) bool {
			return albumMedias[i].CreatedAt.After(albumMedias[j].CreatedAt)
		}
	case "popular":
		lessFunc = func(i, j int) bool {
			return albumMedias[i].LikesCount > albumMedias[j].LikesCount
		}
	case "title":
		lessFunc = func(i, j int) bool {
			return strings.ToLower(albumMedias[i].Description) < strings.ToLower(albumMedias[j].Description)
		}
	}

	// If orderBy is ASC, reverse the lessFunc logic
	if strings.ToUpper(orderBy) == "ASC" {
		original := lessFunc
		lessFunc = func(i, j int) bool {
			return original(j, i)
		}
	}

	// Apply sorting
	sort.Slice(albumMedias, lessFunc)


	sort.SliceStable(albumMedias, lessFunc)

	// Re-index AlbumMediaID after sorting
	// for i := range albumMedias {
	// 	albumMedias[i].AlbumMediaID = uint(i + 1)
	// }

	albumsTags := albumRequest.Tags


	var albumTagList []string
	for _, tag := range albumsTags {
		albumTagList = append(albumTagList, tag.TagName)
	}

	//sum of user like this album
	var likeCount int64
	if errCount := db.Model(&models.AlbumLike{}).Where("album_id = ?", albumId).Count(&likeCount).Error; errCount != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal Mendapatkan Jumlah Like",
		})
	}

	userDetail := UserDetail{
		UserID: albumRequest.User.ID,
		FirstName: albumRequest.User.FirstName,
		LastName: albumRequest.User.LastName,
		FullName: albumRequest.User.FirstName + " " + albumRequest.User.LastName,
		Email: albumRequest.User.Email,
		ProfilePicture: albumRequest.User.ProfilePicture,
	}

	albumDetail := AlbumDetailRequest{
		AlbumID: albumRequest.ID,
		UserDetail: userDetail,
		Description: albumRequest.Description,
		Tags: albumTagList,
		Title: albumRequest.Title,
		LikeCount: int(albumRequest.LikesCount),
		ViewCount: int(albumRequest.ViewCount),
		ImageCount: imageCount,
		VideoCount: videoCount,
		AlbumPrivacy: albumRequest.AlbumPrivacy,
		TargetEmail: albumRequest.TargetEmail,
		CreatedAt: albumRequest.CreatedAt.Format("02 January 2006"),
	}

	var like models.AlbumLike
	err = db.Where("user_id = ? AND album_id = ?", userID, albumId).First(&like).Error

	wacherHasLike := true
	if errors.Is(err, gorm.ErrRecordNotFound) {
		wacherHasLike = false
	} else if err != nil {
		// Handle error lain (misal DB down, query gagal)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memeriksa like"})
	}

	// lanjut pakai watcherHasLike untuk logika berikutnya
	fmt.Println("Apakah user sudah like?", wacherHasLike)

	if err := db.Model(&albumRequest).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal update view count",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Record successfully retrieved",
		"album":         albumDetail,
		"album_medias":  albumMedias,
		"user_has_like": wacherHasLike,
		"user_login_id": user.ID,
	})
}

func DeleteAlbum(ctx *fiber.Ctx, db *gorm.DB) error {
	albumID := ctx.Params("albumID")

	albumId, errParser := uuid.Parse(albumID)

	fmt.Println("album id = ? ", albumID)

	if errParser != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error Parsing to UUID",
		})
	}


	userLoginID, errUserLogin := utils.GetUserID(ctx)
	if errUserLogin != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User is unauthorized",
		})
	}

		fmt.Print("albumID : ", albumID, "  user ID ", userLoginID)


	var album models.Album
	if err := db.Preload("AlbumVideos").
		Preload("AlbumImages").
		Preload("Comments").
		Where("id = ? AND user_id = ?", albumId, userLoginID).
		First(&album).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Album tidak ditemukan atau Anda bukan pemilik",
		})
	}

	// Hapus semua gambar terkait
	for _, image := range album.AlbumImages {
		if err := deleteImage(db, image.ID); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Gagal menghapus image %s", image.ID),
			})
		}
	}

	// Hapus semua video terkait
	for _, video := range album.AlbumVideos {
		if err := deleteVideo(db, video.ID); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Gagal menghapus video %s", video.ID),
			})
		}
	}

	// Hapus semua komentar
	for _, comment := range album.Comments {
		if err := db.Delete(&comment).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Gagal menghapus komentar %s", comment.ID),
			})
		}
	}

	// Hapus relasi many-to-many (misal: album_tags)
	if err := db.Model(&album).Association("Tags").Clear(); err != nil {
		// Hanya log jika ada error (jika kamu memang punya relasi Tags)
		log.Println("Gagal menghapus relasi tags:", err)
	}

	// Hapus album-nya
	if err := db.Delete(&album).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus album",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Album berhasil dihapus",
	})
}



func GetAllAlbums(ctx *fiber.Ctx, db *gorm.DB) error {
	// Get query params for sorting, searching, and pagination
	sortBy := ctx.Query("sort_by", "recent") // recent, oldest, popular
	search := ctx.Query("search", "")
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	userID := ctx.Query("user_id")

	fmt.Println("userID query : ", userID)

	userId, errParser := uuid.Parse(userID)

	fmt.Println("QUERY SORT BY : ", sortBy, " PAGE : ", page, " LIMIT : ", limit)

	if errParser != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error Parsing to UUID",
		})
	}

	userLogin, errUserLogin := utils.GetUserID(ctx)

	if errUserLogin != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User is not found",
		})
	}

	var userLoginData models.User 
	if err := db.Where("id = ?" , userLogin).First(&userLoginData).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User is not found in dartabase",
		})
	}

	var albums []models.Album
	
	query := db.Preload("AlbumVideos").Preload("AlbumImages")

	if (userId == userLoginData.ID) {
		query.Where("user_id = ?", userId)
	} else {
		query = query.Where("user_id = ? AND (album_privacy = ? OR (album_privacy = ? AND target_email IS NOT NULL))", userID, "public", "restricted")
	}


	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	switch sortBy {
	case "oldest":
		query = query.Order("updated_at ASC")
	case "popular":
		query = query.Order("likes_count DESC")
	default: // recent
		query = query.Order("updated_at DESC")
	}

	var total int64
	query.Model(&models.Album{}).Count(&total)

	// 🔍 Filter lebih lanjut untuk 'restricted' berdasarkan targetEmail
	// Hanya berlaku jika user yang login BUKAN pemilik album
	if userId != userLoginData.ID {
		filteredAlbums := []models.Album{}
		for _, album := range albums {
			if album.AlbumPrivacy == "public" {
				filteredAlbums = append(filteredAlbums, album)
			} else if album.AlbumPrivacy == "restricted" {
				var allowedEmails []string
				if err := json.Unmarshal(album.TargetEmail, &allowedEmails); err == nil {
					fmt.Println(allowedEmails)
					for _, email := range allowedEmails {
						if email == userLoginData.Email {
							filteredAlbums = append(filteredAlbums, album)
							break
						}
					}
				}
			}
		}

		albums = filteredAlbums
	}
	

	offset := (page - 1) * limit

	if err := query.Offset(offset).Limit(limit).Find(&albums).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Album failed to reload",
		})
	}

	// Add field last_update to albums, based on created_at, in format like "3 days ago", "1 week ago", "1 month ago"
	type AlbumWithLastUpdate struct {
		AlbumID uuid.UUID `json:"album_id"`
		Title string `json:"title"`
		Description string `json:"description"`
		MediaCount int `json:"media_count"`
		ImageCount int `json:"image_count"`
		VideoCount int `json:"video_count"`
		ThumbnailURL string `json:"thumbnail_url"`
		LastUpdate string `json:"last_update"`
	}


	var albumsWithLastUpdate []AlbumWithLastUpdate
	now := time.Now()
	for _, album := range albums {
		diff := now.Sub(album.UpdatedAt)
		var lastUpdate string
		switch {
		case diff < 24*time.Hour:
			hours := int(diff.Hours())
			if hours <= 1 {
				lastUpdate = "just now"
			} else {
				lastUpdate = fmt.Sprintf("%d hours ago", hours)
			}
		case diff < 7*24*time.Hour:
			days := int(diff.Hours() / 24)
			lastUpdate = fmt.Sprintf("%d days ago", days)
		case diff < 30*24*time.Hour:
			weeks := int(diff.Hours() / (24 * 7))
			lastUpdate = fmt.Sprintf("%d weeks ago", weeks)
		default:
			months := int(diff.Hours() / (24 * 30))
			lastUpdate = fmt.Sprintf("%d months ago", months)
		}

		// Get random cover image from album images
		// coverImage := ""
		// if len(album.AlbumImages) > 0 {
		// 	randomIdx := time.Now().UnixNano() % int64(len(album.AlbumImages))
		// 	coverImage = album.AlbumImages[randomIdx].ImageURL
		// } else {
		// 	randomIdx := time.Now().UnixNano() % int64(len(album.AlbumVideos))
		// 	coverImage = album.AlbumVideos[randomIdx].ThumbnailURL
		// }

		key := strings.TrimPrefix(album.CoverImage, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		coverImageSignedURL, errURL :=  utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}

		// album.CoverImage = coverImage

		albumsWithLastUpdate = append(albumsWithLastUpdate, AlbumWithLastUpdate{
			AlbumID: album.ID,
			Title:      album.Title,
			Description: album.Description,
			ThumbnailURL: coverImageSignedURL,
			ImageCount: len(album.AlbumImages),
			VideoCount:  len(album.AlbumVideos),
			MediaCount: len(album.AlbumImages) + len(album.AlbumVideos),
			LastUpdate: lastUpdate,
		})
	}
	

	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":      "All albums successfully retrieved",
		"albums":       albumsWithLastUpdate,
		"total":        total,
		"page":         page,
		"limit":        limit,
		"total_pages":  (total + int64(limit) - 1) / int64(limit),
	})
}

func GetLatestImage(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)

	// fmt.Println("INI : ", userID)

	if err != nil {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}

	var albums []models.Album

	if err := db.Preload("AlbumImages").Where("user_id = ?", userID).Order("updated_at DESC").Limit(4).Find(&albums).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Album Tidak Ditemukan",
		})
	}
	
	type ImageLatest struct {
		AlbumID uuid.UUID `json:"album_id"`
		Description  string `json:"description"`
		LikeCount    int    `json:"like_count"`
		ThumbnailURL string `json:"thumbnail_url"`
	}

	var imageLatestList []ImageLatest
	for _, album := range albums {
		coverImage := ""
		likecount := 0
		desc := ""
		if len(album.AlbumImages) > 0 {
			randomIdx := time.Now().UnixNano() % int64(len(album.AlbumImages))
			coverImage = album.AlbumImages[randomIdx].ImageURL
			desc = album.AlbumImages[randomIdx].Description
			likecount = int(album.AlbumImages[randomIdx].LikesCount)
		}
		
		key := strings.TrimPrefix(coverImage, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		coverImageSignedURL, errURL :=  utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}

		imageLatestList = append(imageLatestList, ImageLatest{
			AlbumID: album.ID,
			Description:  desc,
			LikeCount:    likecount,
			ThumbnailURL: coverImageSignedURL,
		})

	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "All albums successfully retrieved",
		"album":   imageLatestList,
		// "albums": albums,
	})
}

func UploadTemporary(ctx *fiber.Ctx, db *gorm.DB) error {

	var tempMedia models.TempMedia
	//ambil image
	media, mediaErr := ctx.FormFile("media_temp")

	if mediaErr != nil {
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "All albums successfully retrieved",
		})
	}

	dst := fmt.Sprintf("./storages/images/temp/%s", media.Filename)
	if err := ctx.SaveFile(media, dst); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed Storing Media",
		})
	}

	ipAddress := ctx.IP()
	
	tempMedia = models.TempMedia{
		MediaURL:  fmt.Sprintf("/static/images/temp/%s", media.Filename),
		LikeCount: 0,
		IPAddress: ipAddress,
		ExpiredAt: time.Now().Add(10 * time.Minute),
	}

	if err := db.Create(&tempMedia).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"message": "Failed to Store Temporary",
		})	
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Media Temporary Berhasil Disimpan",
	})	
	
}

func UploadMediaAlbum(ctx *fiber.Ctx, db *gorm.DB) error {
	albumIDParam := ctx.Query("album_id")
	albumID, errParse := uuid.Parse(albumIDParam)
	if errParse != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "album_id tidak valid",
		})
	}

	form, errForm := ctx.MultipartForm()
	if errForm != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membaca form multipart",
		})
	}

	// Upload file (images & videos)
	images := form.File["album_images"]
	imageDescriptions := form.Value["image_descriptions"]

	if len(imageDescriptions) != len(images) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Jumlah deskripsi gambar tidak sesuai jumlah file",
		})
	}

	for index, file := range images {
		imageDescription := imageDescriptions[index]
		if err := storeImage(db, file, albumID, imageDescription, nil); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan gambar",
				"error":   err.Error(),
			})
		}
	}

	videos := form.File["album_videos"]
	videoDescriptions := form.Value["video_descriptions"]

	if len(videoDescriptions) != len(videos) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Jumlah deskripsi video tidak sesuai jumlah file",
		})
	}

	for index, file := range videos {
		fmt.Println("ada file ? ", file)
		videoDescription := videoDescriptions[index]
		fmt.Println("ada deskripsi ? ", videoDescription)
		if err := storeVideo(db, file, albumID, videoDescription, nil); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan video",
				"error":   err.Error(),
			})
		}
	}

	if err := db.Model(&models.Album{}).
		Where("id = ?", albumID).
		Update("updated_at", time.Now()).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate waktu album",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Media berhasil disimpan",
	})
}


func UpdateTargetEmail(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	albumID := ctx.Params("albumId")

	albumUUID, err := uuid.Parse(albumID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Album ID tidak valid",
		})
	}

	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil User ID",
		})
	}

	var album models.Album
	if errALbum := db.Where("id = ?", albumUUID).
		Where("album_privacy = ?", "restricted").
		First(&album).Error; errALbum != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Album tidak ditemukan",
		})
	}

	if album.UserID != userID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User tidak memiliki akses ke album ini",
		})
	}

	// Ambil data email baru dari body
	type NewTargetEmail struct {
		Email string `json:"email"`
	}
	var newEmail NewTargetEmail
	if errBodyParser := ctx.BodyParser(&newEmail); errBodyParser != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Gagal membaca permintaan email",
		})
	}

	// Decode daftar email saat ini dari album
	var emailList []string
	if len(album.TargetEmail) > 0 {
		if errUnMarshal := json.Unmarshal(album.TargetEmail, &emailList); errUnMarshal != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal membaca daftar email sebelumnya",
			})
		}
	}

	// Tambahkan email baru jika belum ada
	emailList = append(emailList, newEmail.Email)
	updatedJSON, err := json.Marshal(emailList)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memproses data email",
		})
	}

	album.TargetEmail = updatedJSON
	if err := db.Save(&album).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan perubahan email",
		})
	}

	// 🔄 Kirim notifikasi lewat gRPC di background
	go func(album models.Album, email string, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      email,
			Subject: "Anda Telah Ditambahkan ke Album",
			Type:    "album-invitation",
			Name:    email,
			Body:    "Anda telah diberi akses ke album. Silakan buka aplikasi untuk melihatnya.",
			Metadata: map[string]string{
				"album_name": album.Title,
				"album_url" : fmt.Sprintf("http://localhost:5173/albums/=%s/details", album.ID),
				"platform_name": "PixoVaulty",
				"platform_url": "www.pixovaulty.com",
			},
		})

		if err != nil {
			log.Printf("Gagal mengirim notifikasi ke %s: %v", email, err)
		}
	}(album, newEmail.Email, client)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Target Email berhasil diperbarui dan notifikasi dikirim",
	})
}




func ClickLikeMedia(ctx *fiber.Ctx, db *gorm.DB) error {
	type likeRequest struct {
		MediaID   string `json:"media_id"`
		UserID    string `json:"user_id"`
		MediaType string `json:"media_type"` // "image" or "video"
	}

	var req likeRequest

	// Parse body
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Gagal membaca permintaan",
		})
	}

	mediaID, err := uuid.Parse(req.MediaID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Media ID tidak valid",
		})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID tidak valid",
		})
	}

	// Cek apakah sudah pernah like
	var existing models.MediaLike
	err = db.Where("user_id = ? AND media_id = ?", userID, mediaID).First(&existing).Error

	if err == nil {
		// Sudah like → Unlike
		if req.MediaType == "image" {
			if err := db.Model(&models.AlbumImage{}).
				Where("id = ?", mediaID).
				Update("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like image"})
			}


		} else if req.MediaType == "video" {
			if err := db.Model(&models.AlbumVideo{}).
				Where("id = ?", mediaID).
				Update("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like video"})
			}
		} else {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipe media tidak valid"})
		}

		// Hapus record like
		db.Unscoped().Delete(&existing)


		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Unlike berhasil",
		})
	}

	// Belum like → Tambah
	if req.MediaType == "image" {
		if err := db.Model(&models.AlbumImage{}).
			Where("id = ?", mediaID).
			Update("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like image"})
		}
	} else if req.MediaType == "video" {
		if err := db.Model(&models.AlbumVideo{}).
			Where("id = ?", mediaID).
			Update("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like video"})
		}
	} else {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipe media tidak valid"})
	}

	newLike := models.MediaLike{
		ID:      uuid.New(),
		UserID:  userID,
		MediaID: mediaID,
	}

	if err := db.Create(&newLike).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan data like",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Like berhasil",
	})
}


func ClickLikeAlbum(ctx *fiber.Ctx, db *gorm.DB) error {
	type likeRequest struct {
		AlbumID string `json:"album_id"`
		UserID  string `json:"user_id"`
	}

	var req likeRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Gagal membaca data permintaan",
		})
	}

	fmt.Println("req : ", req)

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID tidak valid",
		})
	}

	fmt.Println("album id req : ", req.AlbumID)

	albumID, err := uuid.Parse(req.AlbumID)

	fmt.Println("album id : ", albumID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Album ID tidak valid",
		})
	}

	// Cek apakah album ada
	var album models.Album
	if errAlbum := db.Preload("AlbumImages").Preload("AlbumVideos").First(&album, albumID).Error; errAlbum != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Album tidak ditemukan",
		})
	}

	// Cek apakah user sudah like
	var existingLike models.AlbumLike
	err = db.Where("user_id = ? AND album_id = ?", userID, albumID).First(&existingLike).Error

	// Sudah like → unlike
	if err == nil {
		// for _, image := range album.AlbumImages {
		// 	if err := db.Model(&image).Update("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
		// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like image"})
		// 	}

		// 	if err := db.Where("user_id = ? AND media_id = ?", userID, image.ID).Unscoped().Delete(&models.MediaLike{}).Error; err != nil {
		// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus like image"})
		// 	}
		// }

		// for _, video := range album.AlbumVideos {
		// 	if err := db.Model(&video).Update("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
		// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like video"})
		// 	}

		// 	if err := db.Where("user_id = ? AND media_id = ?", userID, video.ID).Unscoped().Delete(&models.MediaLike{}).Error; err != nil {
		// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus like video"})
		// 	}
		// }

		// Hapus record like album
		db.Delete(&existingLike)

		//kurangi Like di album
		if err := db.Model(&album).Update("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like album"})
		}

		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Unlike berhasil"})
	}

		// Belum like → like

	// for _, image := range album.AlbumImages {
	// 	if err := db.Model(&image).Update("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
	// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like image"})
	// 	}

	// 	mediaLike := models.MediaLike{
	// 		UserID:  userID,
	// 		MediaID: image.ID,
	// 	}

	// 	var existingMediaLike models.MediaLike
	// 	if err := db.Where("user_id = ? AND media_id = ?", userID, image.ID).First(&existingMediaLike).Error; err != nil {
	// 		if errors.Is(err, gorm.ErrRecordNotFound) {
	// 			// Hanya create kalau belum ada
	// 			if err := db.Create(&mediaLike).Error; err != nil {
	// 				return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like image"})
	// 			}
	// 		} else {
	// 			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal cek existing like media"})
	// 		}
	// 	}

	// }

	// for _, video := range album.AlbumVideos {
	// 	if err := db.Model(&video).Update("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
	// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like video"})
	// 	}

	// mediaLike := models.MediaLike{
	// 	UserID:  userID,
	// 	MediaID: video.ID,
	// }

	// if err := db.Create(&mediaLike).Error; err != nil {
	// 	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah like video"})
	// }
// }

	newLike := models.AlbumLike{
		ID:      uuid.New(),
		AlbumID: albumID,
		UserID:  userID,
	}

	if err := db.Create(&newLike).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan data like"})
	}

	if err := db.Model(&album).Update("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengurangi like video"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Like berhasil",
	})
}


func GetAlbumComments(ctx *fiber.Ctx, db *gorm.DB) error {
	albumIDParam := ctx.Query("album_id")
	fmt.Println("album_id : ", albumIDParam)
	if albumIDParam == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album ID harus disertakan"})
	}

	albumID, err := uuid.Parse(albumIDParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album ID tidak valid"})
	}

	var comments []models.AlbumComment
	if err := db.Preload("User").Where("album_id = ?", albumID).Order("created_at desc").Find(&comments).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil komentar"})
	}



	var response []AlbumCommentResponse
	for _, comment := range comments {
		key := strings.TrimPrefix(comment.User.ProfilePicture, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		avatarSignedURL, errURL :=  utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}

		response = append(response, AlbumCommentResponse{
			ID:        comment.ID,
			AlbumID:   comment.AlbumID,
			UserID:    comment.UserID,
			User:      fmt.Sprintf("%s %s", comment.User.FirstName, comment.User.LastName),
			UserAvatar:	avatarSignedURL,
			Comment:   comment.Comment,
			CreatedAt: comment.CreatedAt.Format("02 Jan 2006 15:04"),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Komen Berhasil Diambil",
		"album_id": albumID,
		"comments": response,
	})
}

func PostAlbumComment(ctx *fiber.Ctx, db *gorm.DB) error {
	type requestBody struct {
		AlbumID string `json:"album_id"`
		UserID  string `json:"user_id"`
		Comment string `json:"comment"`
	}

	var req requestBody
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format permintaan tidak valid"})
	}

	albumID, err := uuid.Parse(req.AlbumID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album ID tidak valid"})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID tidak valid"})
	}

	// Cek apakah album ada
	var album models.Album
	if err := db.First(&album, "id = ?", albumID).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Album tidak ditemukan"})
	}

	// Cek apakah user ada
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User tidak ditemukan"})
	}

	// Simpan komentar
	comment := models.AlbumComment{
		ID:      uuid.New(),
		AlbumID: albumID,
		UserID:  userID,
		Comment: req.Comment,
	}

	if err := db.Create(&comment).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan komentar"})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Komentar berhasil ditambahkan",
		"data":    comment,
	})
}


func GetAlbumFollower(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, errUserID := utils.GetUserID(ctx)
	if errUserID != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var userLogin models.User
	if err := db.Preload("Following").Where("id = ?", userID).First(&userLogin).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var userIDFollowing []uuid.UUID
	for _, follow := range userLogin.Following {
		userIDFollowing = append(userIDFollowing, follow.FollowingID)
	}

	if len(userIDFollowing) == 0 {
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"albums": []interface{}{},
		})
	}

	var albums []models.Album
	if err := db.Preload("AlbumImages").
		Preload("AlbumVideos").Preload("User").
		Where("user_id IN ?", userIDFollowing).
		Where("album_privacy != ?", "private").
		Order("updated_at DESC").
		Find(&albums).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve albums",
		})
	}

	type AlbumWithLastUpdate struct {
		AlbumID      uuid.UUID `json:"album_id"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		MediaCount   int       `json:"media_count"`
		ImageCount   int       `json:"image_count"`
		VideoCount   int       `json:"video_count"`
		ThumbnailURL string    `json:"thumbnail_url"`
		LastUpdate   string    `json:"last_update"`
		UserDetail	UserDetail	`json:"user_detail,omitempty"`
	}

	var albumsWithLastUpdate []AlbumWithLastUpdate
	now := time.Now()

	for _, album := range albums {
		diff := now.Sub(album.UpdatedAt)
		var lastUpdate string
		switch {
		case diff < time.Hour:
			lastUpdate = "just now"
		case diff < 24*time.Hour:
			lastUpdate = fmt.Sprintf("%d hours ago", int(diff.Hours()))
		case diff < 7*24*time.Hour:
			lastUpdate = fmt.Sprintf("%d days ago", int(diff.Hours()/24))
		case diff < 30*24*time.Hour:
			lastUpdate = fmt.Sprintf("%d weeks ago", int(diff.Hours()/(24*7)))
		default:
			lastUpdate = fmt.Sprintf("%d months ago", int(diff.Hours()/(24*30)))
		}

		// Ambil cover image random
		coverImage := ""
		if len(album.AlbumImages) > 0 {
			randomIdx := time.Now().UnixNano() % int64(len(album.AlbumImages))
			coverImage = album.AlbumImages[randomIdx].ImageURL
		} else {
			randomIdx := time.Now().UnixNano() % int64(len(album.AlbumVideos))
			coverImage = album.AlbumVideos[randomIdx].VideoURL
		}

		if album.AlbumPrivacy == "restricted" {
			var allowedEmails []string
			if err := json.Unmarshal(album.TargetEmail, &allowedEmails); err != nil {
				continue // jika gagal unmarshal, skip
			}

			isAllowed := false
			for _, email := range allowedEmails {
				if email == userLogin.Email {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				continue // skip album ini karena user tidak termasuk
			}
		}

		key := strings.TrimPrefix(coverImage, "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/")
		coverImageSignedURL, errURL :=  utils.GeneratePresignedURL("s3-pixovaulty", key)
		if errURL != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate presigned URL"})
		}


		albumsWithLastUpdate = append(albumsWithLastUpdate, AlbumWithLastUpdate{
			AlbumID:      album.ID,
			Title:        album.Title,
			Description:  album.Description,
			MediaCount:   len(album.AlbumImages) + len(album.AlbumVideos),
			ImageCount:   len(album.AlbumImages),
			VideoCount:   len(album.AlbumVideos),
			ThumbnailURL: coverImageSignedURL,
			LastUpdate:   lastUpdate,
			UserDetail: UserDetail{
				UserID: album.User.ID,
				FirstName: album.User.FirstName,
				LastName: album.User.LastName,
				FullName: album.User.FirstName + " " + album.User.LastName,

			},
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"albums": albumsWithLastUpdate,
	})
}

