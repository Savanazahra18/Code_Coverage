package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Product struct {
	ID               string `gorm:"size:36;not null;uniqueIndex;primary_key"`
	ParentID         string `gorm:"size:36;index"`
	User             User
	UserID           string `gorm:"size:36;index"`
	ProductImages    []ProductImage
    Categories       []Category `gorm:"many2many:product_categories;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Sku              string          `gorm:"size:100;index"`
	Name             string          `gorm:"size:255"`
	Slug             string          `gorm:"size:255"`
	Price            decimal.Decimal `gorm:"type:decimal(16,2);"`
	Stock            int
	Weight           decimal.Decimal `gorm:"type:decimal(10,2);"`
	ShortDescription string          `gorm:"type:text"`
	Description      string          `gorm:"type:text"`
	Status           int             `gorm:"default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt
}

func (p *Product) GetProducts(db *gorm.DB, perPage int, page int, categorySlug string) (*[]Product, int64, error) {
    var products []Product
    var count int64

    // 1. Mulai query dasar tanpa Join dulu untuk menghitung total
    query := db.Debug().Model(&Product{})

    // 2. Jika ada filter kategori, Join HANYA untuk filtering
    if categorySlug != "" {
        query = query.Joins("JOIN product_categories ON product_categories.product_id = products.id").
                      Joins("JOIN categories ON categories.id = product_categories.category_id").
                      Where("categories.slug = ?", categorySlug)
    }

    // Hitung total data (Count)
    if err := query.Count(&count).Error; err != nil {
        return nil, 0, err
    }

    // 3. AMBIL DATA DENGAN PRELOAD TERPISAH
    // Kita panggil Preload di sini agar GORM melakukan query kedua untuk mengambil relasi
    offset := (page - 1) * perPage
    err := query.Preload("Categories"). // Ini akan membaca tabel image_553461.png
                Preload("ProductImages").
                Order("products.created_at desc").
                Limit(perPage).
                Offset(offset).
                Find(&products).Error

    return &products, count, err
}
func (p *Product) FindBySlug(db *gorm.DB, slug string) (*Product, error) {
	var product Product

	// Tambahkan Preload("Categories") untuk halaman detail produk
	err := db.Debug().
		Preload("ProductImages").
		Preload("Categories").
		Model(&Product{}).
		Where("slug = ?", slug).
		First(&product).Error

	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p *Product) FindByID(db *gorm.DB, productID string) (*Product, error) {
	var err error
	var product Product

	err = db.Debug().Preload("ProductImages").Model(&Product{}).Where("id =?", productID).First(&product).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p *Product) SearchByKeywords(db *gorm.DB, keywords []string) ([]Product, error) {
    var products []Product

    if len(keywords) == 0 {
        return products, nil
    }

    // Ambil kata kunci pertama dari AI (biasanya yang paling akurat)
    mainKeyword := "%" + keywords[0] + "%"

    // Gunakan ILIKE (PostgreSQL) atau LIKE (MySQL) 
    // agar 'cetaphil' bisa menemukan 'Cetaphil'
    err := db.Debug().Preload("ProductImages").
        Where("LOWER(name) LIKE LOWER(?)", mainKeyword).
        Find(&products).Error

    return products, err
}

func (p *Product) StandardSearch(db *gorm.DB, query string) ([]Product, error) {
    var products []Product
    
    keyword := "%" + query + "%"
    
    err := db.Debug().Preload("ProductImages").
        Where("LOWER(name) LIKE LOWER(?)", keyword).
        Or("LOWER(description) LIKE LOWER(?)", keyword).
        Find(&products).Error
        
    return products, err
}