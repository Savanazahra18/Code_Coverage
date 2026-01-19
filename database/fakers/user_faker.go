package fakers

import (
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/gieart87/gotoko/app/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UserFaker(db *gorm.DB) *models.User {
	return &models.User{
		ID:            uuid.New().String(),
		FirstName:     faker.FirstName(),
		LastName:      faker.LastName(),
		Email:         faker.Email(),
		Password:      "$2y$10$92IXUNpkjJO0rOQ5byMi.Ye4okoEa3Ro9llC/.og/at2.uheWG/igi",
		RememberToken: "",
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
		DeletedAt:     gorm.DeletedAt{},
	}
}
