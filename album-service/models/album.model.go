package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlbumTag struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	TagName   string         `gorm:"not null;uniqueIndex" json:"tag_name"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type AlbumImage struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	AlbumID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"album_id"`
	Album       Album          `gorm:"foreignKey:AlbumID" json:"album,omitempty"` // optional
	ImageURL    string         `gorm:"not null;type:varchar(255)" json:"image_url"`
	Description string         `json:"description,omitempty"`
	LikesCount  uint           `gorm:"default:0" json:"likes_count"`
	Size        float32        `json:"size"`
	Type        string         `json:"type"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type AlbumVideo struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	AlbumID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"album_id"`
	Album        Album          `gorm:"foreignKey:AlbumID" json:"album,omitempty"` // optional
	VideoURL     string         `gorm:"not null;type:varchar(255)" json:"video_url"`
	Description  string         `json:"description,omitempty"`
	LikesCount   uint           `gorm:"default:0" json:"likes_count"`
	Size         float32        `json:"size"`
	Type         string         `json:"type"`
	ThumbnailURL string         `gorm:"type:varchar(255)" json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Album struct {
	ID           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	User      	 User      `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Tags         []AlbumTag      `gorm:"many2many:album_album_tags" json:"tags,omitempty"`
	Title        string          `gorm:"not null" json:"title"`
	Description  string          `json:"description,omitempty"`
	CoverImage   string          `gorm:"type:varchar(255)" json:"cover_image,omitempty"`
	AlbumPrivacy string          `json:"album_privacy"`
	AlbumImages  []AlbumImage    `gorm:"foreignKey:AlbumID" json:"album_images,omitempty"`
	AlbumVideos  []AlbumVideo    `gorm:"foreignKey:AlbumID" json:"album_videos,omitempty"`
	Comments	 []AlbumComment  `gorm:"foreignKey:AlbumID" json:"album_comments,omitempty"`
	TargetEmail  json.RawMessage `gorm:"type:jsonb" json:"target_email,omitempty"`
	ViewCount    uint           	`gorm:"default:0" json:"view_count"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`
}

type TempMedia struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	MediaURL  string         `gorm:"not null;type:varchar(255)" json:"media_url"`
	LikeCount int            `gorm:"default:0" json:"like_count"`
	ExpiredAt time.Time      `json:"expired_at"`
	IPAddress string         `json:"ip_address"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}


type AlbumLike struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	AlbumID   uuid.UUID `json:"album_id" gorm:"type:uuid;not null;index"`
	Album     Album     `gorm:"foreignKey:AlbumID;references:ID" json:"album,omitempty"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	User      User      `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}


type MediaLike struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	MediaID   uuid.UUID      `json:"media_id" gorm:"type:uuid;not null;index:idx_user_media,unique"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index:idx_user_media,unique"`
	User      User           `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type AlbumComment struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	AlbumID   uuid.UUID      `json:"album_id" gorm:"type:uuid;not null;index"`
	Album     Album          `gorm:"foreignKey:AlbumID;references:ID" json:"album,omitempty"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User      User           `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Comment   string         `json:"comment" gorm:"type:text;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
