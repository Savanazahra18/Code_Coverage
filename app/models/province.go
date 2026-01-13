package models

type Province struct {
	ID   string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name string `json:"name"`
}