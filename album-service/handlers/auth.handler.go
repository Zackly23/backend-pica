package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Zackly23/queue-app/models"
	notif "github.com/Zackly23/queue-app/proto/notificationpb"
	"github.com/Zackly23/queue-app/utils"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
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
	AgreeTermService bool `json:"agreeTermService" validate:"required"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type ForgetPasswordRequest struct {
	Email	string	`json:"email" validate:"required,email"`
}

type ChangePasswordRequest struct {
	RecentPassword	string	`json:"recent_password" validate:"required"`
	NewPassword     string 	`json:"new_password" validate:"required"`
}

// Response yang aman
type UserLoginResponse struct {
	ID           uuid.UUID   `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	UserName   string `json:"user_name,omitempty"`
	Email        string `json:"email"`
	ProfilePicture string `json:"profile_picture,omitempty"`
	Bio 		 string `json:"bio,omitempty"`
	Status		string `json:"status"`
	DeactivateUntil	time.Time `json:"deactivate_until,omitempty"`
	IsTwoFactorEnabled bool `json:"is_two_factor_enabled,omitempty"`
	AreYouFollowingUser	bool `json:"are_you_following_user"`
	Address	  string `json:"address,omitempty"`
	Country       string          `json:"country,omitempty"`
	City           string          `json:"city,omitempty"`
	State          string          `json:"state,omitempty"`
	ZipCode        string          `json:"zip_code,omitempty"`
	CompanyName    string          `json:"company_name,omitempty" gorm:"type:varchar(100)"`
	Phone       *string `json:"phone,omitempty"`
	TagPreference  pq.StringArray  `json:"tag_preferences,omitempty" gorm:"type:text[]"`
	JobTitle    string `json:"job_title,omitempty"`
	Subscription string `json:"subscription,omitempty"`
	SocialMedia json.RawMessage `json:"social_media,omitempty"`
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
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Akun tidak ditemukan",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Password salah",
		})
	}

	fmt.Println("status : ", user.Status)

	switch user.Status {
	case "deleted":
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Akun Telah Dihapus Sebelumnya",
		})
	case "deactivated":
		user.Status = "active"
		user.DeactivateUntil = time.Time{}
		if err := db.Save(&user).Error; err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal mengupdate status user",
			})
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "JWT_SECRET_KEY belum diset",
		})
	}

	accessToken, accessTokenErr := generateToken(user, time.Hour*2)
	refreshToken, refreshTokenErr := generateToken(user, time.Hour*24*7)

	if accessTokenErr != nil || refreshTokenErr != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token",
		})
	}

	if err := db.Create(&models.PersonalAccessToken{
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

	res := UserLoginResponse{
		ID:              user.ID,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Email:           user.Email,
		ProfilePicture:  user.ProfilePicture,
		Address:         user.Address,
		Phone:           user.Phone,
		JobTitle:        user.JobTitle,
		Status:          user.Status,
		DeactivateUntil: user.DeactivateUntil,
		IsTwoFactorEnabled: user.AccountConfig.IsTwoFactorEnabled,
		SocialMedia:     user.SocialMedia,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	// Simpan accessToken di body, refreshToken di cookie
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		HTTPOnly: true,
		Secure:   true, // Wajib true jika pakai HTTPS
		SameSite: "Lax", // Bisa diatur sesuai kebutuhan
	})

	// response
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":       "Login berhasil",
		"user":          res,
		"access_token":  accessToken, // ini tetap dikirim
		"refresh_token": refreshToken,
	})

}



func SignUp(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
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
			"error": "Email sudah terdaftar Sebelumnya",
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

	// Get Subscription ID where SubscriptionType = type-1
	var subscription models.Subscription
	if err := db.First(&subscription, "subscription_type = ?", "Basic").Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal Mengambil Data Subscription",
		})
	}

	defaultAvatar := "https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/images/default/default_avatar.png"


	// Simpan user baru ke database
	user := models.User{
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Password:    string(hashedPassword),
		Status: "active",
		SubscriptionID: subscription.ID,
		AgreeTermService: req.AgreeTermService,
		ProfilePicture: defaultAvatar,
		
	}

	if err := db.Create(&user).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan user",
		})
	}

	fmt.Println("User created with ID:", user.ID)
	// Simpan konfigurasi akun default
	

	// Buat record baru untuk riwayat subscription user
	newUserSubscription := models.UserSubscription{
		// ID:             uuid.New(),
		UserID:         user.ID,
		SubscriptionID: subscription.ID,
		StartDate: time.Now(),
		EndDate: time.Now().AddDate(0,0,30),
		PaymentMethod: "Free Method",
		Status: "Free Tier",
		Amount: 0.00,
	}

	if err := db.Create(&newUserSubscription).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal Menyimpan Riwayat Subscription",
		})
	}


	accountConfigs := models.AccountConfig{
		UserID:              user.ID,
		IsTwoFactorEnabled: false,
		TwoFactorAuthMethod: "",
		TwoFactorAuthDevice: "",
	}

	if err := db.Create(&accountConfigs).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan konfigurasi akun",
		})
	}

	// 🔄 Kirim notifikasi lewat gRPC di background
	go func(user models.User, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Pendaftaran Akun Anda Berhasil",
			Type:    "account-signup",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Akun Anda berhasil didaftarkan. Silakan login untuk mulai menggunakan aplikasi.",
		})

		if err != nil {
			log.Printf("Gagal mengirim notifikasi ke %s: %v", user.Email, err)
		}
	}(user, client)


	go func(user models.User, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Subscription Free Tier",
			Type:    "subscription",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Akun Anda berhasil didaftarkan. Silakan login untuk mulai menggunakan aplikasi.",
			Metadata: map[string]string{
				"expired_date": time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
			},

		})

		if err != nil {
			log.Printf("Gagal mengirim notifikasi ke %s: %v", user.Email, err)
		}
	}(user, client)
	
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
		Subject: "Permintaan Change Password",
		Type:   "password-reset",
		Name:   user.FirstName + " " + user.LastName,
		Body:    fmt.Sprintf("Klik link berikut untuk reset password: http://localhost:3000/change-password?email=%s", req.Email),
	})

	if err != nil {
		fmt.Printf("gRPC error: %v", err)
		return ctx.Status(500).JSON(fiber.Map{"error": "Gagal mengirim email"})
	}


	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message" : "Permintaan Reset Password Sudah Dikirimkan ke Email Pengguna " + res.GetMessage(),
	})
}

func ChangePassword(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	var req ChangePasswordRequest
	var user models.User

	userID, errID := utils.GetUserID(ctx)

	fmt.Println("user id : ", userID)

	if errID != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : "Error mendapatkan user ID",
		})
	}
	
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : "response email tidak ada",
		})
	}

	//cek ke database ada ga emailnya
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Akun Tidak Ditemukan",
		})

	}



	// Check if recent password matches the current password in DB
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.RecentPassword)); err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Password lama salah",
		})
	}

	// Hash the new password
	hashedPassword, errHash := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if errHash != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal meng-hash password baru",
		})
	}

	// Update the user's password in the database
	if err := db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui password",
		})
	}


	//kirim sebuah email ke pengguna berdasarkan emailnya
	fmt.Println("user s : ", user)

	fmt.Println("email : ", user.Email)

	go func (user models.User, client notif.NotificationServiceClient)  {
		ctxTime, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxTime, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Permintaan Perubahan Password",
			Type:   "password-reset",
			Name:   user.FirstName + " " + user.LastName,
			Body:    fmt.Sprintf("Klik link berikut untuk reset password: http://localhost:3000/reset-password?email=%s", user.Email),
		})

		if err != nil {
			log.Printf("Gagal mengirim notifikasi ke %s: %v", user.Email, err)
			fmt.Println("Gagal GRPC ")
		}

	} (user, client)


	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message" : "Permintaan Reset Password Sudah Dikirimkan ke Email Pengguna ",
	})
}

func DeactivateAccount(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	userID, errID := utils.GetUserID(ctx)

	fmt.Println("user id : ", userID)

	if errID != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : "Error mendapatkan user ID",
		})
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Akun tidak ditemukan",
		})
	}

	user.Status = "deactivated"

	user.DeactivateUntil = time.Now().AddDate(0, 0, 30) // add 30 days

	if err := db.Save(&user).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Gagal Menyimpan Status",
		})
	}

	// Hapus token dari database
	if err := db.Model(&models.PersonalAccessToken{}).Where("user_id = ?", userID).Delete(&models.PersonalAccessToken{}).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus token",
		})
	}

	// 🔄 Kirim notifikasi lewat gRPC di background
	go func(user models.User, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Deaktifasi Akun Berhasil",
			Type:    "deactivate-account",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Successfully Deactivate Your Account",
		})

		if err != nil {
			log.Printf("Failed to send notification to %s: %v", user.Email, err)
		}
	}(user, client)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message" : "Permintaan Deactivate Akun Sudah Ditindak Lanjuti ",
	})
}

func DeleteAccount(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	userID, errID := utils.GetUserID(ctx)

	fmt.Println("user id : ", userID)

	if errID != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error" : "Error mendapatkan user ID",
		})
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Akun tidak ditemukan",
		})
	}

		// Hapus token dari database
	if err := db.Model(&models.PersonalAccessToken{}).Where("user_id = ?", userID).Delete(&models.PersonalAccessToken{}).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus token",
		})
	}

	user.Status = "deleted"

	if err := db.Save(&user).Error; err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Gagal Menyimpan Status",
		})
	}

	//hapus semua media AlbumImages dan AlbumVideos

	//hapus Riwayat UserSubscription

	//hapus riwayat Following

	//hapus AccountConfig


	// 🔄 Kirim notifikasi lewat gRPC di background
	go func(user models.User, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Delete Account",
			Type:    "delete-account",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Two-factor authentication (TFA) has been successfully enabled for your account.",
		})

		if err != nil {
			log.Printf("Failed to send notification to %s: %v", user.Email, err)
		}
	}(user, client)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message" : "Permintaan Deactivate Akun Sudah Ditindak Lanjuti ",
	})
}

func GenerateTOTP(ctx *fiber.Ctx, db *gorm.DB) error {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	if errAcc := db.Preload("AccountConfig").Where("id = ?", userID).First(&user).Error; errAcc != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User tidak ditemukan"})
	}

	// Jangan regenerate kalau sudah punya secret
	if (user.AccountConfig.SecretTOTP != "" && user.AccountConfig.IsTwoFactorEnabled) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "TOTP sudah diinisialisasi",
		})
	}

	// Generate new secret
	// secret := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", 
	// 	os.Getenv("APP_NAME"), user.Email, otp.RandomSecret(10), os.Getenv("APP_NAME")))

		// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      os.Getenv("APP_NAME"),
		AccountName: user.Email,
	})
	
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate TOTP"})
	}

	// Ambil secret-nya langsung
	secret := key.Secret()

	// Simpan ke DB
	if errAcc := db.Model(&models.AccountConfig{}).
		Where("user_id = ?", userID).
		Update("secret_totp", secret).Error; errAcc != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan secret ke database"})
	}

	// Generate QR image
	img, err := key.Image(200, 200)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate QR"})
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal encode QR"})
	}
	qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Kirim secret yang sudah pasti disimpan
	return ctx.JSON(fiber.Map{
		"secret":    secret,
		"otp_url":   key.URL(),
		"qr_base64": qrBase64,
	})

}



func VerifyTOTP(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	type TOTPVerifyRequest struct {
		Code string `json:"code"`
	}

	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	if errUser := db.Where("id = ?", userID).First(&user).Error; errUser != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User Not Found"})

	}

	var req TOTPVerifyRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input tidak valid"})
	}

	var accountConfig models.AccountConfig
	if err := db.Where("user_id = ?", userID).First(&accountConfig).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data konfigurasi tidak ditemukan"})
	}

	secret := strings.TrimSpace(accountConfig.SecretTOTP)
	code := strings.TrimSpace(req.Code)

	if secret == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Secret belum tersedia"})
	}

	// Debug opsional
	// codeExpected, _ := totp.GenerateCode(secret, time.Now())
	// fmt.Println("Expected code:", codeExpected)

	codeNow, _ := totp.GenerateCode(secret, time.Now())

	fmt.Println("code : ", code, " code now : ", codeNow)

	valid, errValid := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      3,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})

	if errValid != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kode OTP Tidak Valid"})
	}

	if !valid {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kode OTP salah"})
	}

	
	// Setel status 2FA aktif
	accountConfig.IsTwoFactorEnabled = true
	accountConfig.TwoFactorAuthMethod = "totp"
	if err := db.Save(&accountConfig).Error; err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengaktifkan 2FA"})

	}
	// 🔄 Kirim notifikasi lewat gRPC di background
	go func(user models.User, client notif.NotificationServiceClient) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Two-Factor Authentication Activated",
			Type:    "two-factor-auth",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Two-factor authentication (TFA) has been successfully enabled for your account.",
		})

		if err != nil {
			log.Printf("Failed to send notification to %s: %v", user.Email, err)
		}
	}(user, client)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "TOTP berhasil diverifikasi"})
}

func VerifyTFA(ctx *fiber.Ctx, db *gorm.DB, client notif.NotificationServiceClient) error {
	type TOTPVerifyRequest struct {
		Code string `json:"code"`
	}

	userID, err := utils.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var user models.User
	if errUser := db.Where("id = ?", userID).First(&user).Error; errUser != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User Not Found"})

	}

	var req TOTPVerifyRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input tidak valid"})
	}

	var accountConfig models.AccountConfig
	if err := db.Where("user_id = ?", userID).First(&accountConfig).Error; err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data konfigurasi tidak ditemukan"})
	}

	secret := strings.TrimSpace(accountConfig.SecretTOTP)
	code := strings.TrimSpace(req.Code)

	if secret == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Secret belum tersedia"})
	}

	// Debug opsional
	// codeExpected, _ := totp.GenerateCode(secret, time.Now())
	// fmt.Println("Expected code:", codeExpected)

	codeNow, _ := totp.GenerateCode(secret, time.Now())

	fmt.Println("code : ", code, " code now : ", codeNow)

	valid, errValid := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      3,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})

	if errValid != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kode OTP Tidak Valid"})
	}

	if !valid {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kode OTP salah"})
	}

	// Setel status 2FA aktif
	// accountConfig.IsTwoFactorEnabled = true
	// accountConfig.TwoFactorAuthMethod = "totp"
	// if err := db.Save(&accountConfig).Error; err != nil {
	// 	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengaktifkan 2FA"})
	// }

	ip := ctx.IP()
	loginTime := time.Now().Format(time.RFC3339)

	go func(user models.User, client notif.NotificationServiceClient, ip, loginTime string) {
		ctxNotif, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := client.SendNotification(ctxNotif, &notif.NotificationRequest{
			To:      user.Email,
			Subject: "Login Two-Factor Authentication",
			Type:    "two-factor-login",
			Name:    user.FirstName + " " + user.LastName,
			Body:    "Anda berhasil login menggunakan two-factor authentication (TFA). Jika ini bukan Anda, segera amankan akun Anda.",
			Metadata: map[string]string{
				"login_time": loginTime,
				"ip_address": ip,
				"location":   "-",
			},
		})

		if err != nil {
			log.Printf("Failed to send notification to %s: %v", user.Email, err)
		}
	}(user, client, ip, loginTime)


	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "TOTP berhasil diverifikasi"})
}
