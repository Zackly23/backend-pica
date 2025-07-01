package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PersonalAccessToken struct {
	ID           uuid.UUID   `gorm:"type:uuid;default:uuid_generate_v4()"`
	AccessToken  string `gorm:"uniqueIndex;not null;size:512" json:"access_token"`
	RefreshToken string `gorm:"uniqueIndex;not null;size:512" json:"refresh_token"`
	UserID uuid.UUID `gorm:"not null;type:uuid" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"user"`
	IPAddress    string `gorm:"not null" json:"ip_address"`
	AccessTokenExp time.Time `gorm:"not null" json:"access_token_exp"`
	RefreshTokenExp time.Time `gorm:"not null" json:"refresh_token_exp"`
	Revoked      bool        `gorm:"default:false" json:"revoked"`
	RevokedAt    *time.Time `gorm:"default:null" json:"revoked_at,omitempty"`
	CreatedAt    time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

}


