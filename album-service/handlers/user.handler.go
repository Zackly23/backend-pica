package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Zackly23/queue-app/models"
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
	ZipCode       string          `json:"zip_code,omitempty" gorm:"type:varchar(20)" validate:"max=20,numeric"`
	CompanyName   string          `json:"company_name,omitempty" gorm:"type:varchar(100)" validate:"max=100"`
	SocialMedia   json.RawMessage `json:"social_media,omitempty" gorm:"type:jsonb" validate:"omitempty,json"`
}

type UserStatResponse struct {
	MediaCount int `json:"media_count"`
	FollowersCount int `json:"followers_count"`
	FollowingCount int `json:"following_count"`
	StorageUsed int `json:"storage_used"`
	StorageCapacity int `json:"storage_capacity"`
	StoragePercentage int `json:"storage_percentage"`
}

var validate = validator.New()

func GetUserData(ctx *fiber.Ctx, db *gorm.DB) error {
	userId := ctx.Params("userId")
	var user models.User

	userID, errParse := uuid.Parse(userId)

	if errParse != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
	})
	}

	if err := db.First(&user, userID).Error; err != nil {
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

	fmt.Println("ada ga :" ,albums)

	// Loop through albums
	for _, album := range albums {

		fmt.Println(album.Title)
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

		userStats.StorageUsed += int(sumSizeImage) + int(sumSizeVideo)
	}

	// Set storage capacity (example: 10GB)
	userStats.StorageCapacity = 10 * 1024 // in MB, adjust as needed

	// Calculate storage percentage
	if userStats.StorageCapacity > 0 {
		userStats.StoragePercentage = int(float64(userStats.StorageUsed) / float64(userStats.StorageCapacity) * 100)
	}

	response := UserLoginResponse{
		ID:             user.ID,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Email:          user.Email,
		ProfilePicture: user.ProfilePicture,
		Address:        user.Address,
		Phone:          user.Phone,
		JobTitle:       user.JobTitle,
		SocialMedia:    user.SocialMedia,
		AccountConfig:  user.AccountConfig,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"media count" : userStats,
		"user":    response,
		"message": "User data retrieved successfully",
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

	// 3. Validasi input
	if err := validate.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

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
	userID := ctx.Params("userId")
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