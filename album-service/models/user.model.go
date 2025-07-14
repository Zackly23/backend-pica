package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type User struct {
	ID               uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	FirstName        string          `json:"first_name" gorm:"not null"`
	LastName         string          `json:"last_name" gorm:"not null"`
	UserName		 string			 `json:"user_name,omitempty"`
	Email            string          `json:"email" gorm:"uniqueIndex;not null"`
	Password         string          `json:"password" gorm:"not null"`
	Phone            *string         `json:"phone,omitempty" gorm:"uniqueIndex"`
	Bio              string          `json:"bio,omitempty" gorm:"type:varchar(255)"`
	TagPreference    pq.StringArray  `json:"tag_preferences,omitempty" gorm:"type:text[]"`
	Address          string          `json:"address,omitempty"`
	JobTitle         string          `json:"job_title,omitempty"`
	Country          string          `json:"country,omitempty"`
	City             string          `json:"city,omitempty"`
	State            string          `json:"state,omitempty"`
	ZipCode          string          `json:"zip_code,omitempty"`
	CompanyName      string          `json:"company_name,omitempty"`
	SocialMedia      json.RawMessage `json:"social_media,omitempty" gorm:"type:jsonb"`
	SubscriptionID   uuid.UUID       `json:"subscription_id"`
	Subscription     Subscription    `json:"subscription" gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Following		[]Following			`gorm:"foreignKey:UserID" json:"following"`
	UserSubscriptions []UserSubscription `gorm:"foreignKey:UserID"`
	Status           string          `json:"status,omitempty" gorm:"type:varchar(50);default:active"`
	DeactivateUntil  time.Time       `json:"deactivate_until,omitempty"`
	ProfilePicture   string          `json:"profile_picture,omitempty"`
	AccountConfig    AccountConfig   `json:"account_config,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"index"`
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

type Subscription struct {
	ID                uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	SubscriptionType  string          `json:"subscription_type" gorm:"type:varchar(50)"`
	StorageCapacity   float64         `json:"storage_capacity"`
	MaximumMediaSize  float64         `json:"maximum_media_size"`
	Features          json.RawMessage `json:"features" gorm:"type:jsonb"`
	CreatedAt         time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt  `json:"deleted_at,omitempty" gorm:"index"`
}

type UserSubscription struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	User           User           `gorm:"foreignKey:UserID" json:"user"`
	SubscriptionID uuid.UUID      `gorm:"type:uuid;not null" json:"subscription_id"`
	Subscription   Subscription   `gorm:"foreignKey:SubscriptionID" json:"subscription"`
	PaymentMethod  string		  `gorm:"type:varchar(100)" json:"payment_method"`
	StartDate      time.Time      `json:"start_date" gorm:"not null"`
	EndDate        time.Time      `json:"end_date" gorm:"not null"`
	Status 		   string		  `json:"status" gorm:"type:varchar(50)"`
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type Following struct {
	UserID      uuid.UUID      `gorm:"type:uuid;not null;primaryKey" json:"user_id"`        // yang follow
	FollowingID uuid.UUID      `gorm:"type:uuid;not null;primaryKey" json:"following_id"`   // yang di-follow
	User        User           `gorm:"foreignKey:UserID" json:"user"`
	Following   User           `gorm:"foreignKey:FollowingID" json:"following"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
