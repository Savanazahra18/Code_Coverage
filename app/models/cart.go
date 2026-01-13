package models

import (
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Cart struct {
	ID              string `gorm:"size:36;not null;uniqueIndex;primaryKey"`
	CartItems       []CartItem
	BaseTotalPrice  decimal.Decimal `gorm:"type:decimal(16,2)"`
	TaxAmount       decimal.Decimal `gorm:"type:decimal(16,2)"`
	TaxPercent      decimal.Decimal `gorm:"type:decimal(10,2)"`
	DiscountAmount  decimal.Decimal `gorm:"type:decimal(16,2)"`
	DiscountPercent decimal.Decimal `gorm:"type:decimal(10,2)"`
	ShippingCost    decimal.Decimal `gorm:"type:decimal(16,2)"`
	GrandTotal      decimal.Decimal `gorm:"type:decimal(16,2)"`
}


func (c *Cart) GetCart(db *gorm.DB, cartID string) (*Cart, error) {
	var err error
	var cart Cart

	err = db.Debug().
		Preload("CartItems").
		Preload("CartItems.Product").
		Model(Cart{}).
		Where("id = ?", cartID).
		First(&cart).Error
	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (c *Cart) CreateCart(db *gorm.DB, cartID string) (*Cart, error) {
	cart := &Cart{
		ID:              cartID,
		BaseTotalPrice:  decimal.NewFromInt(0),
		TaxAmount:       decimal.NewFromInt(0),
		TaxPercent:      decimal.NewFromInt(11),
		DiscountAmount:  decimal.NewFromInt(0),
		DiscountPercent: decimal.NewFromInt(0),
		GrandTotal:      decimal.NewFromInt(0),
	}

	err := db.Debug().Create(cart).Error
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (c *Cart) CalculateCart(db *gorm.DB, cartID string) (*Cart, error) {
    // 1. Selalu ambil data terbaru dari database untuk memastikan sinkronisasi
    var items []CartItem
    if err := db.Where("cart_id = ?", cartID).Find(&items).Error; err != nil {
        return nil, err
    }

    baseTotal := decimal.Zero
    taxTotal := decimal.Zero
    discountTotal := decimal.Zero

    // 2. Hitung total dari data item terbaru
    for _, item := range items {
        baseTotal = baseTotal.Add(item.BaseTotal)
        taxTotal = taxTotal.Add(item.TaxAmount)
        discountTotal = discountTotal.Add(item.DiscountAmount)
    }

    // 3. Ambil nilai shipping cost saat ini dari struct atau DB
    grandTotal := baseTotal.Add(taxTotal).Sub(discountTotal).Add(c.ShippingCost)

    // 4. Update nilai ke objek struct
    c.BaseTotalPrice = baseTotal
    c.TaxAmount = taxTotal
    c.DiscountAmount = discountTotal
    c.GrandTotal = grandTotal

    // 5. Simpan ke database menggunakan map untuk akurasi GORM
    err := db.Model(&Cart{}).Where("id = ?", cartID).Updates(map[string]interface{}{
        "base_total_price": baseTotal,
        "tax_amount":       taxTotal,
        "discount_amount":  discountTotal,
        "shipping_cost":    c.ShippingCost,
        "grand_total":      grandTotal,
    }).Error

    if err != nil {
        return nil, err
    }

    return c, nil
}

func (c *Cart) AddItem(db *gorm.DB, inputItem CartItem) (*CartItem, error) {
	// Validasi produk
	var product Product
	if err := db.Where("id = ?", inputItem.ProductID).First(&product).Error; err != nil {
		return nil, err
	}

	// Cek apakah item sudah ada di cart
	var existingItem CartItem
	result := db.Where("cart_id = ? AND product_id = ?", c.ID, inputItem.ProductID).First(&existingItem)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Item belum ada → buat baru

		qty := inputItem.Qty
		if qty <= 0 {
			return nil, errors.New("quantity must be greater than zero")
		}

		// Hitung BaseTotal: Price * Qty
		baseTotal := product.Price.Mul(decimal.NewFromInt(int64(qty)))

		// Tax: 11% dari BaseTotal
		taxPercent := decimal.NewFromFloat(0.11)
		taxAmount := baseTotal.Mul(taxPercent)

		// SubTotal = BaseTotal + TaxAmount
		subTotal := baseTotal.Add(taxAmount)

		newItem := CartItem{
			ID:              uuid.NewString(),
			CartID:          c.ID,
			ProductID:       product.ID,
			Qty:             qty,
			BasePrice:       product.Price,           // harga per item
			BaseTotal:       baseTotal,               // total harga = price * qty
			TaxPercent:      taxPercent,              // 0.11
			TaxAmount:       taxAmount,               // baseTotal * taxPercent
			DiscountPercent: decimal.NewFromInt(0),
			DiscountAmount:  decimal.NewFromInt(0),
			SubTotal:        subTotal,                // baseTotal + taxAmount
		}

		if err := db.Create(&newItem).Error; err != nil {
			return nil, err
		}

		// Update total cart
		c.CalculateCart(db, c.ID)

		return &newItem, nil

	} else if result.Error != nil {
		return nil, result.Error
	}

	// Item sudah ada → update qty
	existingItem.Qty += inputItem.Qty

	if existingItem.Qty <= 0 {
		// Opsional: hapus item jika qty <= 0
		db.Delete(&existingItem)
		c.CalculateCart(db, c.ID)
		return &existingItem, nil
	}

	// Hitung ulang nilai item
	baseTotal := product.Price.Mul(decimal.NewFromInt(int64(existingItem.Qty)))
	taxPercent := decimal.NewFromFloat(0.11)
	taxAmount := baseTotal.Mul(taxPercent)
	subTotal := baseTotal.Add(taxAmount)

	existingItem.BaseTotal = baseTotal
	existingItem.TaxAmount = taxAmount
	existingItem.SubTotal = subTotal

	if err := db.Save(&existingItem).Error; err != nil {
		return nil, err
	}

	// Update total cart
	c.CalculateCart(db, c.ID)

	return &existingItem, nil
}

func (c *Cart) GetItems(db *gorm.DB, cartID string) ([]CartItem, error) {
	var items []CartItem

	err := db.Debug().Preload("Product").Model(&CartItem{}).
	Where("cart_id = ?", cartID).
	Order("created_at desc").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (c *Cart) UpdateItemQty(db *gorm.DB, itemID string, qty int) (*CartItem, error) {
    var exisItem CartItem

    // 1. Ambil item
    if err := db.Where("id = ?", itemID).First(&exisItem).Error; err != nil {
        return nil, err
    }

    // 2. Ambil product untuk mendapatkan harga terbaru
    var product Product
    if err := db.Where("id = ?", exisItem.ProductID).First(&product).Error; err != nil {
        return nil, err
    }

    // 3. Hitung ulang nilai item
    taxPercent := decimal.NewFromFloat(0.11)
    baseTotal := product.Price.Mul(decimal.NewFromInt(int64(qty)))
    taxAmount := baseTotal.Mul(taxPercent)
    subTotal := baseTotal.Add(taxAmount)

    // 4. Update value existing item
    exisItem.Qty = qty
    exisItem.BaseTotal = baseTotal
    exisItem.TaxAmount = taxAmount
    exisItem.SubTotal = subTotal

    // 5. Simpan perubahan ke tabel cart_items
    if err := db.Save(&exisItem).Error; err != nil {
        return nil, err
    }

    // ✅ KUNCI PERBAIKAN: Update tabel 'carts' agar total bawah berubah
    c.CalculateCart(db, exisItem.CartID)

    return &exisItem, nil
}

func (c *Cart) RemoveItemByID(db *gorm.DB, itemID string) error {
	var err error
	var item CartItem

	err = db.Debug().Model(&CartItem{}).Where("id = ?", itemID).First(&item).Error
	if err != nil {
		return err
	}

	err = db.Debug().Delete(&item).Error
	if err != nil {
		return err
	}	

	return nil
}

func GetOrCreateCartByUser(db *gorm.DB, userID string) (*Cart, error) {
	var cart Cart

	err := db.
		Preload("CartItems").
		Where("user_id = ?", userID).
		First(&cart).Error

	// Jika cart belum ada → buat baru
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cart = Cart{
			ID:              uuid.NewString(),
			BaseTotalPrice:  decimal.Zero,
			TaxAmount:       decimal.Zero,
			TaxPercent:      decimal.NewFromInt(11),
			DiscountAmount:  decimal.Zero,
			DiscountPercent: decimal.Zero,
			ShippingCost:    decimal.Zero,
			GrandTotal:      decimal.Zero,
		}

		if err := db.Create(&cart).Error; err != nil {
			return nil, err
		}

		return &cart, nil
	}

	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (c *Cart) ClearCart(db *gorm.DB, cartID string) error {
	err := db.Debug().Where("cart_id = ?", cartID).Delete(&CartItem{}).Error
	if err != nil {
		return err
	}

	err = db.Debug().Where("id = ?", cartID).Delete(&Cart{}).Error
	if err != nil {
		return err
	}

	return nil
}