package seeders

import (
	"github.com/gieart87/gotoko/database/fakers"
	"gorm.io/gorm"
)

type Seeder struct {
	Seeder interface{}
}

func RegisterSeeders(db *gorm.DB) []Seeder {
	return []Seeder{
		{Seeder: fakers.UserFaker(db)},
		{Seeder: fakers.ProductFaker(db)},
	}
}

// database/seeders/seeders.go

func DBSeed(db *gorm.DB) error {
	for _, seeder := range RegisterSeeders(db) {
		// Gunakan reflection atau interface untuk panggil Create()
		// Tapi karena struct berbeda, lebih aman pakai switch atau pastikan semua seeder punya method Create()
		
		// Alternatif: langsung panggil Create jika seeder punya method tersebut
		if creator, ok := seeder.Seeder.(interface{ Create() error }); ok {
			if err := creator.Create(); err != nil {
				return err
			}
		}
	}
	return nil
}