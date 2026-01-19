package seeders

import (
	"gorm.io/gorm"
	"github.com/gieart87/gotoko/app/models"
)

func SeedProvinces(db *gorm.DB) error {
	provinces := []models.Province{
		{Name: "DKI Jakarta"},
		{Name: "Jawa Barat"},
		{Name: "Jawa Tengah"},
		{Name: "Jawa Timur"},
		{Name: "Banten"},
	}

	for _, p := range provinces {
		db.Create(&p)
	}

	return nil
}