package controllers

import (
	"log"
	"net/http"
	"strconv"
	"errors"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/models"
	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
	"github.com/unrolled/render"
	"gorm.io/gorm"
)




func ClearCart(db *gorm.DB, cartID string) error {
	var cart models.Cart
	return cart.ClearCart(db, cartID)
}

func GetShoppingCart(db *gorm.DB, cartID string) (*models.Cart, error) {
	var cart models.Cart

	err := db.Preload("CartItems.Product.ProductImages").Where("id = ?", cartID).First(&cart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cart.ID = cartID
			cart.BaseTotalPrice = decimal.NewFromInt(0)
			cart.TaxAmount = decimal.NewFromInt(0)
			cart.DiscountAmount = decimal.NewFromInt(0)
			cart.ShippingCost = decimal.NewFromInt(0)
			cart.GrandTotal = decimal.NewFromInt(0)

			if err := db.Create(&cart).Error; err != nil {
				return nil, err
			}
			return &cart, nil
		}
		return nil, err
	}

	return &cart, nil
}

func (server *Server) GetCart(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	user := auth.CurrentUser(server.DB, w, r)
	cartID := auth.GetCartID(w, r)

	cart, err := GetShoppingCart(server.DB, cartID)
	if err != nil {
		log.Printf("❌ Gagal muat keranjang: %v", err)
		http.Error(w, "Gagal memuat keranjang", http.StatusInternalServerError)
		return
	}

	// ✅ Gunakan CartItems yang sudah di-preload
	items := cart.CartItems

	// Provinsi statis
	provinces := []models.Province{
		{ID: "1", Name: "DKI Jakarta"},
		{ID: "2", Name: "Jawa Barat"},
		{ID: "3", Name: "Jawa Tengah"},
		{ID: "4", Name: "Jawa Timur"},
		{ID: "5", Name: "Banten"},
		{ID: "6", Name: "Yogyakarta"},
		{ID: "7", Name: "Bali"},
		{ID: "8", Name: "Sumatera Utara"},
	}

	// Kota per provinsi
	cityMap := map[string][]string{
		"DKI Jakarta":    {"Jakarta Selatan", "Jakarta Pusat", "Jakarta Barat", "Jakarta Timur", "Jakarta Utara"},
		"Jawa Barat":     {"Bandung", "Bekasi", "Bogor", "Depok", "Cimahi"},
		"Banten":         {"Serang", "Tangerang", "Cilegon", "Tangerang Selatan", "Pandeglang"},
		"Jawa Tengah":    {"Semarang", "Solo", "Surakarta", "Magelang", "Salatiga"},
		"Jawa Timur":     {"Surabaya", "Malang", "Batu", "Sidoarjo", "Gresik"},
		"Yogyakarta":     {"Yogyakarta", "Sleman", "Bantul", "Kulon Progo", "Gunung Kidul"},
		"Bali":           {"Denpasar", "Badung", "Tabanan", "Gianyar", "Buleleng"},
		"Sumatera Utara": {"Medan", "Binjai", "Pematangsiantar", "Tebing Tinggi", "Padangsidempuan"},
	}

	// Service pengiriman
	services := []string{"REG", "OKE", "YES"}

	message := r.URL.Query().Get("message")
	errorMsg := r.URL.Query().Get("error")

	_ = render.HTML(w, http.StatusOK, "cart", map[string]interface{}{
		"cart":      cart,
		"items":     items,
		"provinces": provinces,
		"cityMap":   cityMap,
		"services":  services,
		"Message":   message,
		"Error":     errorMsg, 
		"user": user,
		
	})
}

// 🔹 Tabel ongkir statis (tanpa database)
var shippingRates = map[string]map[string]map[string]int{
	"JNE": {
		"DKI Jakarta": {
			"Jakarta Selatan":      12000,
			"Jakarta Pusat":        10000,
			"Jakarta Barat":        11000,
			"Jakarta Timur":        11500,
			"Jakarta Utara":        12500,
		},
		"Jawa Barat": {
			"Bandung":              18000,
			"Bekasi":               15000,
			"Bogor":                17000,
			"Depok":                14000,
			"Cimahi":               19000,
		},
		"Banten": {
			"Serang":               20000,
			"Tangerang":            16000,
			"Cilegon":              22000,
			"Tangerang Selatan":    15000,
			"Pandeglang":           24000,
		},
		"Jawa Tengah": {
			"Semarang":             25000,
			"Solo":                 24000,
			"Surakarta":            24000,
			"Magelang":             23000,
			"Salatiga":             24500,
		},
		"Jawa Timur": {
			"Surabaya":             30000,
			"Malang":               29000,
			"Batu":                 29500,
			"Sidoarjo":             28000,
			"Gresik":               28500,
		},
		"Yogyakarta": {
			"Yogyakarta":           26000,
			"Sleman":               25500,
			"Bantul":               25000,
			"Kulon Progo":          27000,
			"Gunung Kidul":         28000,
		},
		"Bali": {
			"Denpasar":             45000,
			"Badung":               44000,
			"Tabanan":              46000,
			"Gianyar":              45500,
			"Buleleng":             48000,
		},
		"Sumatera Utara": {
			"Medan":                40000,
			"Binjai":               39000,
			"Pematangsiantar":      38000,
			"Tebing Tinggi":        38500,
			"Padangsidempuan":      42000,
		},
	},
	"TIKI": {
		"DKI Jakarta": {
			"Jakarta Selatan":      13000,
			"Jakarta Pusat":        11000,
			"Jakarta Barat":        12000,
			"Jakarta Timur":        12500,
			"Jakarta Utara":        13500,
		},
		"Jawa Barat": {
			"Bandung":              19000,
			"Bekasi":               16000,
			"Bogor":                18000,
			"Depok":                15000,
			"Cimahi":               20000,
		},
		"Banten": {
			"Serang":               21000,
			"Tangerang":            17000,
			"Cilegon":              23000,
			"Tangerang Selatan":    16000,
			"Pandeglang":           25000,
		},
		"Jawa Tengah": {
			"Semarang":             26000,
			"Solo":                 25000,
			"Surakarta":            25000,
			"Magelang":             24000,
			"Salatiga":             25500,
		},
		"Jawa Timur": {
			"Surabaya":             31000,
			"Malang":               30000,
			"Batu":                 30500,
			"Sidoarjo":             29000,
			"Gresik":               29500,
		},
		"Yogyakarta": {
			"Yogyakarta":           27000,
			"Sleman":               26500,
			"Bantul":               26000,
			"Kulon Progo":          28000,
			"Gunung Kidul":         29000,
		},
		"Bali": {
			"Denpasar":             47000,
			"Badung":               46000,
			"Tabanan":              48000,
			"Gianyar":              47500,
			"Buleleng":             50000,
		},
		"Sumatera Utara": {
			"Medan":                42000,
			"Binjai":               41000,
			"Pematangsiantar":      40000,
			"Tebing Tinggi":        40500,
			"Padangsidempuan":      44000,
		},
	},
	"POS": {
		"DKI Jakarta": {
			"Jakarta Selatan":      10000,
			"Jakarta Pusat":        8000,
			"Jakarta Barat":        9000,
			"Jakarta Timur":        9500,
			"Jakarta Utara":        10500,
		},
		"Jawa Barat": {
			"Bandung":              17000,
			"Bekasi":               14000,
			"Bogor":                16000,
			"Depok":                13000,
			"Cimahi":               18000,
		},
		"Banten": {
			"Serang":               19000,
			"Tangerang":            15000,
			"Cilegon":              21000,
			"Tangerang Selatan":    14000,
			"Pandeglang":           23000,
		},
		"Jawa Tengah": {
			"Semarang":             24000,
			"Solo":                 23000,
			"Surakarta":            23000,
			"Magelang":             22000,
			"Salatiga":             23500,
		},
		"Jawa Timur": {
			"Surabaya":             29000,
			"Malang":               28000,
			"Batu":                 28500,
			"Sidoarjo":             27000,
			"Gresik":               27500,
		},
		"Yogyakarta": {
			"Yogyakarta":           25000,
			"Sleman":               24500,
			"Bantul":               24000,
			"Kulon Progo":          26000,
			"Gunung Kidul":         27000,
		},
		"Bali": {
			"Denpasar":             43000,
			"Badung":               42000,
			"Tabanan":              44000,
			"Gianyar":              43500,
			"Buleleng":             46000,
		},
		"Sumatera Utara": {
			"Medan":                38000,
			"Binjai":               37000,
			"Pematangsiantar":      36000,
			"Tebing Tinggi":        36500,
			"Padangsidempuan":      40000,
		},
	},
	
}

func (server *Server) CalculateShipping(w http.ResponseWriter, r *http.Request) {
	cartID := auth.GetCartID(w, r) // ✅

	courier := r.FormValue("courier")
	province := r.FormValue("province")
	city := r.FormValue("city")

	if courier == "" || province == "" || city == "" {
		http.Redirect(w, r, "/carts?error=Data+ongkir+tidak+lengkap", http.StatusSeeOther)
		return
	}

	cost := 0
	if provMap, ok := shippingRates[courier]; ok {
		if cityMap, ok := provMap[province]; ok {
			if c, exists := cityMap[city]; exists {
				cost = c
			}
		}
	}

	if cost == 0 {
		http.Redirect(w, r, "/carts?error=Ongkir+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	// Simpan ke session
	session, err := auth.GetSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/carts?error=Gagal+simpan+ongkir", http.StatusSeeOther)
		return
	}
	session.Values["checkout_courier"] = courier
	session.Values["checkout_province"] = province
	session.Values["checkout_city"] = city
	session.Values["checkout_shipping_cost"] = cost
	session.Save(r, w)

	// Update cart
	cart, err := GetShoppingCart(server.DB, cartID)
	if err != nil {
		http.Redirect(w, r, "/carts?error=Gagal+update+ongkir", http.StatusSeeOther)
		return
	}

	cart.ShippingCost = decimal.NewFromInt(int64(cost))
	cart.CalculateCart(server.DB, cartID)

	http.Redirect(w, r, "/carts?message=Ongkir+berhasil+dihitung", http.StatusSeeOther)
}
// ... sisa fungsi (AddItemToCart, UpdateCart, RemoveItemByID) tetap sama
func (server *Server) AddItemToCart(w http.ResponseWriter, r *http.Request) {
	productID := r.FormValue("product_id")
	qtyStr := r.FormValue("qty")

	if productID == "" || qtyStr == "" {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	productModel := models.Product{}
	product, err := productModel.FindByID(server.DB, productID)
	if err != nil {
		log.Printf("⚠ Produk tidak ditemukan: %s", productID)
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	if qty > product.Stock {
	http.Redirect(w, r, "/products/"+product.Slug+"?error=Stok+tidak+mencukupi!", http.StatusSeeOther)
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
		return
	}

	cartID := auth.GetCartID(w, r)
	cart, err := GetShoppingCart(server.DB, cartID)
	if err != nil {
		log.Printf("⚠ Gagal buat keranjang: %v", err)
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
		return
	}

	_, err = cart.AddItem(server.DB, models.CartItem{
		ProductID: productID,
		Qty:       qty,
	})
	if err != nil {
		log.Printf("⚠ Gagal tambah ke keranjang: %v", err)
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
		return
	}

	// Redirect dengan pesan di URL
http.Redirect(w, r, "/carts?message=Item+berhasil+ditambahkan+ke+keranjang!", http.StatusSeeOther)
}

func (server *Server) UpdateCart(w http.ResponseWriter, r *http.Request) {
    cartID := auth.GetCartID(w, r)
    cart, err := GetShoppingCart(server.DB, cartID)
    if err != nil {
        http.Redirect(w, r, "/carts", http.StatusSeeOther)
        return
    }

    for _, item := range cart.CartItems {
        qtyStr := r.FormValue(item.ID)
        if qtyStr == "" { continue }
        
        qty, _ := strconv.Atoi(qtyStr)
        if qty <= 0 {
            cart.RemoveItemByID(server.DB, item.ID)
        } else {
            // Update qty di tabel cart_items
            cart.UpdateItemQty(server.DB, item.ID, qty)
        }
    }

    // ✅ GARIS MERAH HILANG: Karena CalculateCart mengembalikan (*Cart, error)
    // Panggil fungsi ini untuk memperbarui tabel 'carts'
    _, err = cart.CalculateCart(server.DB, cartID) 
    if err != nil {
        log.Printf("Gagal hitung ulang: %v", err)
    }

    http.Redirect(w, r, "/carts?message=Keranjang+diperbarui", http.StatusSeeOther)
}

func (server *Server) RemoveItemByID(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    itemID := vars["id"]

    if itemID == "" {
        http.Redirect(w, r, "/carts", http.StatusSeeOther)
        return
    }

    cartID := auth.GetCartID(w, r)
    cart, err := GetShoppingCart(server.DB, cartID)
    if err != nil {
        log.Printf("⚠ Gagal muat keranjang saat hapus: %v", err)
        http.Redirect(w, r, "/carts", http.StatusSeeOther)
        return
    }

    err = cart.RemoveItemByID(server.DB, itemID)
    if err != nil {
        log.Printf("⚠ Gagal hapus item %s: %v", itemID, err)
    }

    // ✅ TAMBAHKAN INI: Supaya total harga langsung sinkron setelah hapus barang
    cart.CalculateCart(server.DB, cartID)

    http.Redirect(w, r, "/carts?message=Item+berhasil+dihapus", http.StatusSeeOther)
}