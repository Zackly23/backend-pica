package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Zackly23/queue-app/models"
	notif "github.com/Zackly23/queue-app/proto/notificationpb"
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	//godot
)



type UserSignUpRequest struct {
	FirstName       string `json:"firstName" validate:"required"`
	LastName        string `json:"lastName" validate:"required"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type ForgetPasswordRequest struct {
	Email	string	`json:"email" validate:"required,email"`
}


// Response yang aman
type UserLoginResponse struct {
	ID           uuid.UUID   `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	ProfilePicture string `json:"profile_picture,omitempty"`
	Address	  string `json:"address,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	JobTitle    string `json:"job_title,omitempty"`
	SocialMedia json.RawMessage `json:"social_media,omitempty"`
	AccountConfig models.AccountConfig `json:"account_config,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
}

func generateToken(user models.User, duration time.Duration) (string, error) {

	//buat claim data
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"name": user.FirstName + " " + user.LastName,
		"exp":     time.Now().Add(duration).Unix(),
		"iat":     time.Now().Unix(),
	}

	//kunci jwt
	jwtSecret := os.Getenv("JWT_SECRET_KEY")

	//metode claim dan signed jwt
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))

	return signedToken, err
}

func Login(ctx *fiber.Ctx, db *gorm.DB) error {
	var req UserLoginRequest
	var user models.User

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := db.Preload("AccountConfig").Where("email = ?", req.Email).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Akun tidak ditemukan",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Password salah",
		})
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")

	if jwtSecret == "" {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "JWT_SECRET_KEY belum diset",
		})
	}

	// Generate token
	accessToken, accessTokenErr := generateToken(user, time.Hour*2)
	refreshToken, refreshTokenErr := generateToken(user, time.Hour*24*7)

	if (accessTokenErr != nil || refreshTokenErr != nil) {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token",
		})
	}

	// Simpan refresh token ke database
	if err := db.Model(&models.PersonalAccessToken{}).Create(&models.PersonalAccessToken{
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		UserID:          user.ID,
		IPAddress:       ctx.IP(),
		AccessTokenExp:  time.Now().Add(time.Hour * 2),
		RefreshTokenExp: time.Now().Add(time.Hour * 24 * 7),
		Revoked:         false,
	}).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan token ke database",
		})
	}

	// Response aman
	res := UserLoginResponse{
		ID:           user.ID,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		ProfilePicture: user.ProfilePicture,
		Address: 	user.Address,
		Phone:       user.Phone,
		JobTitle:    user.JobTitle,
		SocialMedia: user.SocialMedia,
		AccountConfig: user.AccountConfig,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login berhasil",
		"user":    res,
		"access_token" : accessToken,
		"refresh_token" : refreshToken,
	})
}


func SignUp(ctx *fiber.Ctx, db *gorm.DB) error {
	//get request body
	var req UserSignUpRequest

	fmt.Println("Check Request Body:", string(ctx.Body()))

	// Parse body ke struct
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validasi
	if err := validate.Struct(req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Cek apakah email sudah terdaftar
	if err := db.Where("email = ?", req.Email).First(&models.User{}).Error; err == nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email sudah terdaftar",
		})
	}

	// Cek apakah password sesuai dengan konfirmasi password
	if req.Password != req.PasswordConfirm {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password dan konfirmasi password tidak sesuai",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal meng-hash password",
		})
	}

	// Simpan user baru ke database
	user := models.User{
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Password:    string(hashedPassword),
	}

	if err := db.Create(&user).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan user",
		})
	}
	
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User berhasil dibuat",
		"user": fiber.Map{
			"email":       user.Email,
			"first_name":  user.FirstName,
			"last_name":   user.LastName,
		},
		"user_data" : user,
	})
	
}


func Logout(ctx *fiber.Ctx, db *gorm.DB) error {

	fmt.Println("Logout called")

	// Ambil user ID DARI BODY
	userID := ctx.Locals("user_id").(string)

	userId, errParse := uuid.Parse(userID)

	if errParse != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal Parse token",
		})
	}

	fmt.Println("user Id : ", userId)

	// Hapus token dari database
	if err := db.Model(&models.PersonalAccessToken{}).Where("user_id = ?", userId).Delete(&models.PersonalAccessToken{}).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus token",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout berhasil",
	})

}




//return refresh token
func Refresh(ctx *fiber.Ctx, db *gorm.DB) error {
	// Ambil refresh token dari header
	refreshToken := ctx.Get("Authorization")

	if refreshToken == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Refresh token tidak ditemukan",
		})
	}

	// Verifikasi dan parse token
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	token, errParse := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if errParse != nil || !token.Valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token tidak valid",
		})
	}

	userID := token.Claims.(jwt.MapClaims)["user_id"].(uint)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User tidak ditemukan",
		})
	}

	newAccessToken, err := generateToken(user, time.Hour*2)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat access token baru",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": newAccessToken,
	})
}

func ResetPassword(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	var req ForgetPasswordRequest
	var user models.User
	
	if err := ctx.BodyParser(&req); err != nil {
		ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : "response email tidak ada",
		})
	}

	//cek ke database ada ga emailnya
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Email Tidak Ditemukan",
		})
	}

	//buat context background time
	ctxTime, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//kirim sebuah email ke pengguna berdasarkan emailnya
	res, err := client.SendNotification(ctxTime, &notif.NotificationRequest{
		To:      req.Email,
		Subject: "Permintaan Reset Password",
		Type:   "password-reset",
		Name:   user.FirstName + " " + user.LastName,
		Body:    fmt.Sprintf("Klik link berikut untuk reset password: http://localhost:3000/reset-password?email=%s", req.Email),
	})

	if err != nil {
		fmt.Printf("gRPC error: %v", err)
		return ctx.Status(500).JSON(fiber.Map{"error": "Gagal mengirim email"})
	}


	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message" : "Permintaan Reset Password Sudah Dikirimkan ke Email Pengguna " + res.GetMessage(),
	})
}