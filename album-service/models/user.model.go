package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	FirstName      string          `json:"first_name" gorm:"not null"`
	LastName       string          `json:"last_name" gorm:"not null"`
	UserName       string          `json:"user_name" gorm:"type:varchar(100)"`
	Email          string          `json:"email" gorm:"uniqueIndex;not null"`
	Password       string          `json:"password" gorm:"not null"`
	Phone          *string         `json:"phone,omitempty" gorm:"uniqueIndex"` // perbaikan: hilangkan spasi berlebih
	Bio            string          `json:"bio,omitempty" gorm:"type:varchar(255)"`
	TagPreference  pq.StringArray  `json:"tag_preferences,omitempty" gorm:"type:text[]"`
	Address        string          `json:"address,omitempty" gorm:"type:varchar(255)"`
	JobTitle       string          `json:"job_title,omitempty" gorm:"type:varchar(100)"`
	Country       string          `json:"country,omitempty" gorm:"type:varchar(100)"`
	City           string          `json:"city,omitempty" gorm:"type:varchar(100)"`
	State          string          `json:"state,omitempty" gorm:"type:varchar(50)"`
	ZipCode        string          `json:"zip_code,omitempty" gorm:"type:varchar(20)"`
	CompanyName    string          `json:"company_name,omitempty" gorm:"type:varchar(100)"`
	SocialMedia    json.RawMessage `json:"social_media,omitempty" gorm:"type:jsonb"`
	Subscription   string          `json:"subscription,omitempty" gorm:"type:varchar(50)"`
	Status         string          `json:"status,omitempty" gorm:"type:varchar(50);default:active"`
	DeactivateUntil time.Time      `json:"deactivate_until,omitempty" gorm:"type:timestamp"`
	ProfilePicture string          `json:"profile_picture,omitempty" gorm:"type:varchar(255)"`
	AccountConfig  AccountConfig   `json:"account_config,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"index"`
}

type AccountConfig struct {
	ID                  uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	UserID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	IsTwoFactorEnabled  bool 		   `json:"is_two_factor_enabled" gorm:"default:false"`
	TwoFactorAuthMethod string         `json:"two_factor_auth_method" gorm:"type:varchar(50)"`
	TwoFactorAuthDevice string         `json:"two_factor_auth_device" gorm:"type:varchar(100)"`
	SecretTOTP			string		   `json:"secret_totp" gorm:"type:varchar(255)"`
	CreatedAt           time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}


type Permission struct {
	ID   uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Type string    `json:"type" gorm:"type:varchar(50)"` //active, deactivate, delete
	Name string    `json:"name" gorm:"type:varchar(100)"` //open profile
}
