package handlers

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"sort"
	"strconv"
	"time"

	"github.com/Zackly23/queue-app/models"
	"github.com/Zackly23/queue-app/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserDetail struct {
	UserID uuid.UUID `json:"user_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	ProfilePicture string `json:"profile_picture,omitempty"`
}

type AlbumMedia struct {
	AlbumMediaID uuid.UUID      `json:"album_media_id"`
	MediaID      uuid.UUID      `json:"media_id"`
	AlbumID      uuid.UUID      `json:"album_id"`
	Description  string    `json:"description"`
	LikesCount   uint      `json:"likes_count"`
	URL          string    `json:"url"`
	Size         float32   `json:"size"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
	MediaKind    string    `json:"media_kind"` // "image" or "video"
}

type AlbumDetailRequest struct {
	UserID       uuid.UUID          `json:"user_id"`
	UserDetail	UserDetail    `json:"user_detail"`
	Tags         []string      `json:"tags,omitempty"`         // array string
	Title        string        `json:"title"`
	Description  string        `json:"description,omitempty"`
	AlbumPrivacy string        `json:"album_privacy"`          // e.g. "public", "private"
	TargetEmail  json.RawMessage `json:"target_email"`
	AlbumImages  []AlbumImageRequest `json:"album_images,omitempty"` // nested images
	AlbumVideos  []AlbumVideoRequest `json:"album_videos,omitempty`
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

func storeImage(db *gorm.DB, file *multipart.FileHeader, albumID uuid.UUID, imageDescription string, albumImageID any) error {
	dst := fmt.Sprintf("./storages/images/%s", file.Filename)
	if err := utils.SaveMultipartFile(file, dst); err != nil {
		return fmt.Errorf("gagal menyimpan file gambar: %w", err)
	}

	sizeMB := float32(file.Size) / (1024 * 1024)
	mimeType := file.Header.Get("Content-Type")

	// Coba konversi ID
	if idStr, ok := albumImageID.(string); ok {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil && id > 0 {
			// Update gambar jika ID valid
			var existingImage models.AlbumImage
			if err := db.First(&existingImage, id).Error; err == nil {
				existingImage.ImageURL = dst
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
		ImageURL:    dst,
		Size:        sizeMB,
		Type:        mimeType,
		Description: imageDescription,
	}

	return db.Create(&image).Error
}

func storeVideo(db *gorm.DB, file *multipart.FileHeader, albumID uuid.UUID, videoDescription string, albumVideoId any) error {
	dst := fmt.Sprintf("./storages/videos/%s", file.Filename)
	if err := utils.SaveMultipartFile(file, dst); err != nil {
		return fmt.Errorf("gagal menyimpan file video: %w", err)
	}

	sizeMB := float32(file.Size) / (1024 * 1024)
	mimeType := file.Header.Get("Content-Type")

	if idStr, ok := albumVideoId.(string); ok {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil && id > 0 {
			var existingVideo models.AlbumVideo
			if err := db.First(&existingVideo, id).Error; err == nil {
				existingVideo.VideoURL = dst
				existingVideo.Size = sizeMB
				existingVideo.Type = mimeType
				existingVideo.Description = videoDescription
				// ThumbnailURL bisa digenerate kemudian
				return db.Save(&existingVideo).Error
			}
		}
	}

	// Buat video baru jika ID tidak valid
	video := models.AlbumVideo{
		AlbumID:      albumID,
		VideoURL:     dst,
		Size:         sizeMB,
		Type:         mimeType,
		Description:  videoDescription,
		ThumbnailURL: "none",
	}

	return db.Create(&video).Error
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

func StoreAlbums(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)

	fmt.Println("userID : x ", userID)

	if err != nil {
		return ctx.SendStatus(200)
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
		if marshaled, err := json.Marshal(targetEmailJSON); err == nil {
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
	for index, file := range images {
		// fmt.Println("Image : ", file)
		// fmt.Println("index image : ", index)
		imageDescription := imageDescriptions[index]
		// fmt.Println("Deskripsi : ", imageDescription)
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
	albumImageIds := form.Value["album_image_ids"]
	for index, file := range images {
		imageDescription := imageDescriptions[index]
		albumImageId := albumImageIds[index] //convert to uint

		if err := storeImage(db, file, albumRequest.ID, imageDescription, albumImageId); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan gambar",
				"error":   err.Error(),
			})
		}
	}

	videos := form.File["album_videos"]
	videoDescriptions := form.Value["video_descriptions"]
	albumVideoIds := form.Value["album_video_ids"]
	for index, file := range videos {
		videoDescription := videoDescriptions[index]
		albumVideoId := albumVideoIds[index]
		if err := storeVideo(db, file, albumRequest.ID, videoDescription, albumVideoId); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan video",
				"error":   err.Error(),
			})
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

	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Akun Tidak Ditemukan"})
	}

	albumID := ctx.Params("albumId")

	albumId, errParse := uuid.Parse(albumID)

	if errParse != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Album ID is failed to parse",
		})
	}

	var albumRequest models.Album
	if err := db.Preload("Tags").Preload("AlbumImages").Preload("AlbumVideos").Where("id = ?", albumId).First(&albumRequest).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album Tidak Ditemukan"})
	}

	if userID != albumRequest.UserID {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Album does not belong to user",
		})
	}

	var albumMedias []AlbumMedia
	var indexMedia uuid.UUID

	for _, img := range albumRequest.AlbumImages {
		indexMedia = uuid.New()
		albumMedias = append(albumMedias, AlbumMedia{
			AlbumMediaID: indexMedia,
			MediaID:      img.ID,
			AlbumID:      img.AlbumID,
			Description:  img.Description,
			LikesCount:   img.LikesCount,
			URL:          img.ImageURL,
			Size:         img.Size,
			Type:         img.Type,
			CreatedAt:    img.CreatedAt,
			MediaKind:    "image",
		})
		
	}

	for _, vid := range albumRequest.AlbumVideos {
		indexMedia = uuid.New()
		albumMedias = append(albumMedias, AlbumMedia{
			AlbumMediaID: indexMedia,
			MediaID:      vid.ID,
			AlbumID:      vid.AlbumID,
			Description:  vid.Description,
			LikesCount:   vid.LikesCount,
			URL:          vid.VideoURL,
			Size:         vid.Size,
			Type:         vid.Type,
			CreatedAt:    vid.CreatedAt,
			MediaKind:    "video",
		})
		
	}

	sortBy := ctx.Query("sort_by", "recent") // Options: recent, oldest, popular
	orderBy := ctx.Query("order_by", "DESC") // Options: DESC, ASC

	lessFunc := func(i, j int) bool { return false }

	switch sortBy {
	case "oldest":
		lessFunc = func(i, j int) bool {
			return albumMedias[i].CreatedAt.Before(albumMedias[j].CreatedAt)
		}
	case "popular":
		lessFunc = func(i, j int) bool {
			return albumMedias[i].LikesCount > albumMedias[j].LikesCount
		}
	default: // recent
		lessFunc = func(i, j int) bool {
			return albumMedias[i].CreatedAt.After(albumMedias[j].CreatedAt)
		}
	}

	// If orderBy is ASC, reverse the lessFunc logic
	if orderBy == "ASC" {
		origLessFunc := lessFunc
		lessFunc = func(i, j int) bool {
			return origLessFunc(j, i)
		}
	}

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

	userDetail := UserDetail{
		UserID: user.ID,
		FirstName: user.FirstName,
		LastName: user.LastName,
		Email: user.Email,
		ProfilePicture: user.ProfilePicture,
	}

	albumDetail := AlbumDetailRequest{
		UserID: albumRequest.UserID,
		UserDetail: userDetail,
		Description: albumRequest.Description,
		Tags: albumTagList,
		Title: albumRequest.Title,
		AlbumPrivacy: albumRequest.AlbumPrivacy,
		TargetEmail: albumRequest.TargetEmail,
	}


	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Record successfully retrieved",
		"album":         albumDetail,
		"album_medias":  albumMedias,
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

	userID := ctx.Params("userID")

	userId, errParser := uuid.Parse(userID)

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
		coverImage := ""
		if len(album.AlbumImages) > 0 {
			randomIdx := time.Now().UnixNano() % int64(len(album.AlbumImages))
			coverImage = album.AlbumImages[randomIdx].ImageURL
		}
		// album.CoverImage = coverImage

		albumsWithLastUpdate = append(albumsWithLastUpdate, AlbumWithLastUpdate{
			AlbumID: album.ID,
			Title:      album.Title,
			Description: album.Description,
			ThumbnailURL: coverImage,
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

		imageLatestList = append(imageLatestList, ImageLatest{
			AlbumID: album.ID,
			Description:  desc,
			LikeCount:    likecount,
			ThumbnailURL: coverImage,
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

func UpdateTargetEmail(ctx *fiber.Ctx, db *gorm.DB) error {
	albumID := ctx.Params("albumId")
	albumId, errParse := uuid.Parse(albumID)

	if errParse != nil {
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Failed to Parse Album ID",
		})	
	}

	userId, err := utils.GetUserID(ctx)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed retrived User ID",
		})	
	}

	var album models.Album
	if err := db.Where("id = ?", albumId).Where("album_privacy = ?", "restricted").Find(&album).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to retrieved album",
		})
	}

	if userId != album.UserID {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User is not belong to this Album",
		})
	}

	form, formErr := ctx.MultipartForm()

	if formErr != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed retrived to from form",
		})
	}



	if targetEmails, ok := form.Value["target_emails"]; ok {
		if marshaled, err := json.Marshal(targetEmails); err == nil {
			album.TargetEmail = marshaled
		}
	}

	if err := db.Save(&album).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal update album"})
	}
	

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Target Email Berhasil Diupdate",
	})	
	
}