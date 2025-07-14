package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Zackly23/queue-app/models"
	"github.com/Zackly23/queue-app/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserUpdateRequest struct {
	ID 			  uint 			  `json:"id"`
	FullName      string          `json:"full_name" validate:"required,min=2,max=100"`
	UserName      string          `json:"user_name" validate:"required,alphanum,min=3,max=30"`
	Email         string          `json:"email" validate:"required,email"`
	PhoneNumber   string          `json:"phone_number" validate:"required,e164"` // Format E.164 (misalnya: +628123456789)
	Bio           string          `json:"bio,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	TagPreference pq.StringArray   `json:"tag_preferences,omitempty" gorm:"type:text[]" validate:"dive,max=50"` // maksimal 50 karakter per tag
	Address       string          `json:"address,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	JobTitle      string          `json:"job_title,omitempty" gorm:"type:varchar(100)" validate:"max=100"`
	State         string          `json:"state,omitempty" gorm:"type:varchar(50)" validate:"max=50"`
	ZipCode       string          `json:"zip_code,omitempty" gorm:"type:varchar(20)" validate:"max=20"`
	Country       string          `json:"country,omitempty"`
	City          string          `json:"city,omitempty" gorm:"type:varchar(100)" validate:"max=100"`
	CompanyName   string          `json:"company_name,omitempty" gorm:"type:varchar(100)" validate:"max=100"`
	SocialMedia   json.RawMessage `json:"social_media,omitempty" gorm:"type:jsonb" validate:"omitempty,json"`
}

type UserStatResponse struct {
	MediaCount int `json:"media_count"`
	FollowersCount int `json:"followers_count"`
	FollowingCount int `json:"following_count"`
	StorageUsed string `json:"storage_used"`
	StorageCapacity string `json:"storage_capacity"`
	StoragePercentage float64 `json:"storage_percentage"`
}

var validate = validator.New()

func GetUserData(ctx *fiber.Ctx, db *gorm.DB) error {

	userLoginID, errUserLoginID := utils.GetUserID(ctx)
	if errUserLoginID != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	
	userId := ctx.Params("userId")
	var user models.User

	userID, errParse := uuid.Parse(userId)

	if errParse != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	areYouFollowThisUser := false
	var existingFollow models.Following

	if err := db.Where("user_id = ? AND following_id = ?", userLoginID, userID).First(&existingFollow).Error; err == nil {
		areYouFollowThisUser = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Jika error bukan karena data tidak ditemukan, maka return error
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memeriksa status follow",
		})
	}


	if err := db.Preload("AccountConfig").Preload("Subscription").First(&user, userID).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Hitung the user stats endpoint to get user statistics
	var albums []models.Album
	var userStats UserStatResponse

	// Get All Album related to the user
	if err := db.Preload("AlbumImages").Preload("AlbumVideos").Where("user_id = ?", user.ID).Find(&albums).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user albums",
		})
	}

	// fmt.Println("ada ga :" ,albums)

	// Loop through albums
	var storageUsed float32
	storageUsed = 0.00
	for _, album := range albums {

		// fmt.Println(album.Title)
		// Count media
		userStats.MediaCount += len(album.AlbumImages) + len(album.AlbumVideos)

		// Sum image sizes
		var sumSizeImage float32
		for _, image := range album.AlbumImages {
			sumSizeImage += image.Size
		}

		// Sum video sizes
		var sumSizeVideo float32
		for _, video := range album.AlbumVideos {
			sumSizeVideo += video.Size
		}

		storageUsed += sumSizeImage + sumSizeVideo
	}

	if storageUsed < 1024 {
		userStats.StorageUsed = fmt.Sprintf("%.2f MB", storageUsed)
	} else {
		storageUsedGB := storageUsed / 1024
		userStats.StorageUsed = fmt.Sprintf("%.2f GB", storageUsedGB)
	}

	// Set storage capacity (example: 10GB)
	fmt.Println("user sub : ", user.Subscription)
	fmt.Println("user storage : ", user.Subscription.StorageCapacity )

	storageCapacity := int(user.Subscription.StorageCapacity)

	userStats.StorageCapacity = fmt.Sprintf("%.2f GB", float64(storageCapacity))

	// Calculate storage percentage
	if storageCapacity > 0 {
		userStats.StoragePercentage = math.Round(float64(storageUsed) / float64(storageCapacity)) / 1024.0
	}


	response := UserLoginResponse{
		ID:             user.ID,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		UserName:       user.UserName,
		Country: 	 user.Country,
		City:           user.City,
		State:          user.State,
		ZipCode:        user.ZipCode,
		CompanyName:    user.CompanyName,
		Bio: 		  	user.Bio,
		Email:          user.Email,
		ProfilePicture: user.ProfilePicture,
		Address:        user.Address,
		Phone:          user.Phone,
		JobTitle:       user.JobTitle,
		SocialMedia:    user.SocialMedia,
		Subscription:  user.Subscription.SubscriptionType,
		TagPreference: user.TagPreference,
		AreYouFollowingUser: areYouFollowThisUser,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"media_count" : userStats,
		"user":    response,
		"message": "User data retrieved successfully",
	})
}

func GetUserConfiguration(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve User Id",
		})
	}
	
	var accountConfig models.AccountConfig

	if err := db.Where("user_id = ?", userID).First(&accountConfig).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve account config",
		})
	}

	// fmt.Println("accont confug : ", accountConfig)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil Mengambil Data Configuration",
		"account_config": accountConfig,
	})
}

func UpdateUserData(ctx *fiber.Ctx, db *gorm.DB) error {
	userID := ctx.Params("userId")

	userId, errParser := uuid.Parse(userID)

	if errParser != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error Parsing to UUID",
		})
	}

	// 1. Ambil data user dari DB menggunakan model GORM
	var existingUser models.User
	if err := db.First(&existingUser, userId).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// 2. Parsing body request ke struct input
	var req UserUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// fmt.Println("ini body nya : ", req)

	// 3. Validasi input
	if err := validate.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// fmt.Println("Ini udah di validate : ", req.JobTitle)

	// 4. Mapping field-field dari request ke user model
	existingUser.FirstName = strings.SplitN(req.FullName, " ", 2)[0]
	if parts := strings.SplitN(req.FullName, " ", 2); len(parts) > 1 {
		existingUser.LastName = parts[1]
	}
	existingUser.Email = req.Email
	if req.PhoneNumber != "" {
		existingUser.Phone = &req.PhoneNumber
	}

	// fmt.Println("preference : ", req.TagPreference)

	existingUser.Bio = req.Bio
	existingUser.TagPreference = req.TagPreference
	existingUser.Address = req.Address
	existingUser.JobTitle = req.JobTitle
	existingUser.State = req.State
	existingUser.ZipCode = req.ZipCode
	existingUser.CompanyName = req.CompanyName
	existingUser.SocialMedia = req.SocialMedia
	existingUser.UserName = req.UserName
	existingUser.Country = req.Country
	existingUser.City = req.City

	// 5. Simpan perubahan ke DB
	if err := db.Save(&existingUser).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user : " + err.Error(),
		})
	}

	// 6. Kembalikan respons sukses
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User updated successfully",
		"user":    existingUser,
	})
}

func UpdateProfilePicture(ctx *fiber.Ctx, db *gorm.DB) error {
	var user models.User
	// userID := ctx.Params("userId")
	userID, errID := utils.GetUserID(ctx)

	if errID != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieced User ID",
		})
	}

	form, errForm := ctx.FormFile("profile_picture")

	if errForm != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded with key 'profile_picture'. Please ensure the form field name matches.",
		})
	}

	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Save the uploaded file to a directory (e.g., "./uploads/profile_pictures/")
	savePath := fmt.Sprintf("./storages/images/profile/%s", form.Filename)
	if err := ctx.SaveFile(form, savePath); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save profile picture",
		})
	}

	// Set the profile picture URL (assuming static files are served from /static/profile_pictures/)
	user.ProfilePicture = fmt.Sprintf("/static/profile_pictures/%s", form.Filename)

	if err := db.Save(&user).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile picture",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Profile picture updated successfully",
		"user":    user,
	})
}

func FollowUser(ctx *fiber.Ctx, db *gorm.DB) error {
	userLoginID, errUserLoginID := utils.GetUserID(ctx)
	if errUserLoginID != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	userToFollowID := ctx.Query("user_to_follow")
	userToFollowParseID, errParse := uuid.Parse(userToFollowID)
	if errParse != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user_to_follow ID",
		})
	}

	// Cek apakah user sedang follow user lain
	var existingFollow models.Following
	err := db.
		Where("user_id = ? AND following_id = ?", userLoginID, userToFollowParseID).
		First(&existingFollow).Error

	if err == nil {
		// Sudah follow, maka lakukan unfollow (hapus)
		if errFollow := db.Unscoped().Delete(&existingFollow).Error; errFollow != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal melakukan unfollow",
			})
		}
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Berhasil unfollow",
			"is_user_now_follow": false,
		})
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Belum follow, maka buat follow baru
		newFollow := models.Following{
			UserID:      userLoginID,
			FollowingID: userToFollowParseID,
			CreatedAt:   time.Now(),
		}
		if err := db.Create(&newFollow).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal melakukan follow",
			})
		}
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Berhasil follow",
			"is_user_now_follow": true,
		})
	}

	// Jika error selain not found
	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Gagal memproses follow",
	})
}

func ChangeSubscription(ctx *fiber.Ctx, db *gorm.DB) error {
	// Define request body struct
	type SubscriptionBodyReq struct {
		TypeID        uint   `json:"type_id"`
		PaymentMethod string `json:"payment_method"`
	}

	var req SubscriptionBodyReq
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user ID from context
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Find user
	var user models.User
	if err := db.Preload("Subscription").Preload("UserSubscriptions").First(&user, "id = ?", userID).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Find new subscription type
	var newSubType models.Subscription
	if err := db.First(&newSubType, req.TypeID).Error; err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Subscription type not found",
		})
	}

	// Change current UserSubscription status to Expired
	if len(user.UserSubscriptions) > 0 {
		for i := range user.UserSubscriptions {
			us := &user.UserSubscriptions[i]
			if us.Status == "Active" {
				us.Status = "Expired"
				us.EndDate = time.Now()
				if err := db.Save(us).Error; err != nil {
					return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Failed to expire current subscription",
					})
				}
			}
		}
	}

	// Create new UserSubscription
	newUserSub := models.UserSubscription{
		UserID:         user.ID,
		SubscriptionID: newSubType.ID,
		Status:         "Active",
		PaymentMethod:  req.PaymentMethod,
		StartDate:      time.Now(),
		EndDate:      time.Now().AddDate(0, 1, 0), // Example: 1 month from now
	}
	if err := db.Create(&newUserSub).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create new subscription",
		})
	}

	// Update user's Subscription reference
	user.SubscriptionID = newSubType.ID
	if err := db.Save(&user).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user subscription",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Subscription updated successfully",
		"subscription": newSubType,
	})
}