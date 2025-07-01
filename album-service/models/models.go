package models

func GetModels() []interface{} {
	return []interface{}{
		&User{},
		&PersonalAccessToken{},
		&AccountConfig{},
		&AlbumTag{},
		&Album{},
		&AlbumImage{},
		&AlbumVideo{},
		&TempMedia{},
	}
}