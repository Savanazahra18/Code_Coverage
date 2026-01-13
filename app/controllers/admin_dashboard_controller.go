package controllers

import (
    "fmt"
    "time"
	"net/http"
    "strconv"
    "os"
    "io"
    "bytes"       
    "encoding/json"

    "path/filepath"
    "github.com/gosimple/slug"
    "github.com/google/uuid"
    "github.com/gorilla/mux"
    "github.com/shopspring/decimal"
    "github.com/gieart87/gotoko/app/models"
	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/unrolled/render"
    "gorm.io/gorm"
)

type ProductAI struct {
    Name string `json:"name"`
    Desc string `json:"desc"`
}

// Fungsi pembantu untuk inisialisasi render agar tidak ditulis berulang kali
func adminRender() *render.Render {
	return render.New(render.Options{
		Layout:     "admin_layout", // Menggunakan admin_layout.html sebagai induk
		Extensions: []string{".html", ".tmpl"},
		Directory:  "templates", // Pastikan ini mengarah ke folder templates Anda
	})
}

func (server *Server) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.CurrentUser(server.DB, w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Gunakan path lengkap sesuai struktur folder: "pages/admin_dashboard"
	_ = adminRender().HTML(w, http.StatusOK, "pages/admin_dashboard", map[string]interface{}{
		"user": user,
	})
}

func (server *Server) AdminProducts(w http.ResponseWriter, r *http.Request) {
    var products []models.Product
    
    err := server.DB.Debug().
        Preload("Categories").    
        Preload("ProductImages"). 
        Order("created_at desc"). 
        Find(&products).Error

    if err != nil {
        fmt.Println("Error query produk:", err)
    }

    _ = adminRender().HTML(w, http.StatusOK, "pages/admin_product", map[string]interface{}{
        "products": products,
    })
}
// Handler untuk halaman Orders
func (server *Server) AdminOrders(w http.ResponseWriter, r *http.Request) {
	user := auth.CurrentUser(server.DB, w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = adminRender().HTML(w, http.StatusOK, "pages/show_order", map[string]interface{}{
		"user": user,
	})
}

// Handler untuk halaman Customers
func (server *Server) AdminCustomers(w http.ResponseWriter, r *http.Request) {
	user := auth.CurrentUser(server.DB, w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = adminRender().HTML(w, http.StatusOK, "pages/admin_customers", map[string]interface{}{
		"user": user,
	})
}
// Menampilkan halaman form tambah
func (server *Server) CreateProductPage(w http.ResponseWriter, r *http.Request) {
    // 1. Cek User yang sedang login
    user := auth.CurrentUser(server.DB, w, r)
    if user == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // 2. Ambil semua daftar kategori dari database
    var categories []models.Category
    err := server.DB.Order("name asc").Find(&categories).Error
    if err != nil {
        // Jika error ambil kategori, log ke console
        fmt.Println("Gagal mengambil kategori:", err)
    }

    // 3. Kirim data 'categories' ke template
    // Pastikan nama file di folder adalah admin_product_create.html
    _ = adminRender().HTML(w, http.StatusOK, "pages/admin_product_create", map[string]interface{}{
        "user":       user,
        "categories": categories, // Data ini yang akan diloop di HTML
    })
}

func (server *Server) StoreProduct(w http.ResponseWriter, r *http.Request) {
    // 1. Ambil data user yang sedang login
    user := auth.CurrentUser(server.DB, w, r)
    if user == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // 2. Ambil data dari form (Pastikan name di form HTML adalah "category_id")
    name := r.FormValue("name")
    priceStr := r.FormValue("price")
    stockStr := r.FormValue("stock")
    categoryID := r.FormValue("category_id")

    // Konversi tipe data
    price, _ := decimal.NewFromString(priceStr)
    stock, _ := strconv.Atoi(stockStr)
    productID := uuid.New().String() // ID unik

    // 3. LOGIKA KATEGORI: Ambil data kategori dari DB berdasarkan ID yang dipilih
    var categories []models.Category
    if categoryID != "" {
        var category models.Category
        // Cek apakah kategori tersebut ada di database
        if err := server.DB.Where("id = ?", categoryID).First(&category).Error; err == nil {
            categories = append(categories, category)
        }
    }

    // 4. Inisialisasi Product lengkap dengan slice Categories
    newProduct := models.Product{
        ID:         productID,
        UserID:     user.ID,
        Name:       name,
        Price:      price,
        Stock:      stock,
        Slug:       slug.Make(name),
        Status:     1,
        Categories: categories, // Masukkan kategori di sini agar relasi Many-to-Many terbentuk
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }

    // 5. Proses Upload Gambar
    file, handler, err := r.FormFile("image")
    if err == nil {
        defer file.Close()
        _ = os.MkdirAll("public/uploads", os.ModePerm)

        fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)
        dbPath := "uploads/" + fileName
        physicalPath := "public/" + dbPath

        dst, err := os.Create(physicalPath)
        if err == nil {
            defer dst.Close()
            io.Copy(dst, file)

            newProduct.ProductImages = append(newProduct.ProductImages, models.ProductImage{
                ID:        uuid.New().String(),
                ProductID: productID,
                Path:      dbPath,
            })
        }
    }

    // 6. Simpan ke Database
    // GORM akan otomatis mengisi tabel 'products' DAN tabel 'product_categories'
    err = server.DB.Create(&newProduct).Error
    if err != nil {
        fmt.Println("Gagal simpan ke DB:", err)
        http.Error(w, "Gagal menyimpan produk: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // 7. Redirect kembali ke daftar produk
    http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}
// Menampilkan halaman Edit dengan data lama
func (server *Server) EditProductPage(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var product models.Product
   if err := server.DB.Preload("ProductImages").Where("id = ?", id).First(&product).Error; err != nil {
        http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
        return
    }

    user := auth.CurrentUser(server.DB, w, r)
    _ = adminRender().HTML(w, http.StatusOK, "pages/admin_product_edit", map[string]interface{}{
        "user":    user,
        "product": product,
    })
}

func (server *Server) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// ambil data lama (PENTING)
	var old models.Product
	if err := server.DB.Preload("ProductImages").
		Where("id = ?", id).
		First(&old).Error; err != nil {
		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")

	// ===== PRICE AMAN =====
	priceStr := r.FormValue("price")
	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		price = old.Price
	}

	// ===== STOCK AMAN (INI YANG FIX BUG 0) =====
	stockStr := r.FormValue("stock")
	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		stock = old.Stock
	}

	tx := server.DB.Begin()

	// ===== UPDATE PRODUK =====
	tx.Model(&models.Product{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"name":       name,
			"price":      price,
			"stock":      stock,
			"slug":       slug.Make(name),
			"updated_at": time.Now(),
		})

	// ===== HANDLE GAMBAR (OPSIONAL) =====
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// hapus gambar lama
		for _, img := range old.ProductImages {
			if img.Path != "" {
				_ = os.Remove(filepath.Clean("public/" + img.Path))
			}
		}

		// hapus record image lama
		tx.Unscoped().
			Where("product_id = ?", id).
			Delete(&models.ProductImage{})

		// simpan gambar baru
		_ = os.MkdirAll("public/uploads", os.ModePerm)

		fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)
		dbPath := "uploads/" + fileName
		physicalPath := "public/" + dbPath

		dst, err := os.Create(physicalPath)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)

			tx.Create(&models.ProductImage{
				ID:        uuid.New().String(),
				ProductID: id,
				Path:      dbPath,
			})
		}
	}

	tx.Commit()

	// ===== BALIK KE KATALOG =====
	http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}


func (server *Server) DeleteProduct(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    // Gunakan Transaction untuk memastikan semua terhapus atau tidak sama sekali
    err := server.DB.Transaction(func(tx *gorm.DB) error {
        var product models.Product
        
        // Cari produk berdasarkan ID
        if err := tx.Where("id = ?", id).First(&product).Error; err != nil {
            return err
        }

        // 1. Bersihkan relasi Many-to-Many di product_categories secara paksa
        // Ini mengatasi error FK Constraint
        if err := tx.Model(&product).Association("Categories").Clear(); err != nil {
            return err
        }

        // 2. Hapus data di tabel product_images
        if err := tx.Where("product_id = ?", id).Delete(&models.ProductImage{}).Error; err != nil {
            return err
        }

        // 3. Hapus produk utama
        // Gunakan Unscoped() jika ingin benar-benar hilang dari database (Hard Delete)
        if err := tx.Unscoped().Delete(&product).Error; err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        fmt.Println("Error hapus produk:", err)
        http.Error(w, "Gagal menghapus produk: " + err.Error(), http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}
// 1. Dashboard Navigasi (Halaman dengan 3 Kotak)
func (server *Server) OrderDashboard(w http.ResponseWriter, r *http.Request) {
    var countCustomers, countItems, countOrders int64
    server.DB.Model(&models.OrderCustomer{}).Count(&countCustomers)
    server.DB.Model(&models.OrderItem{}).Count(&countItems)
    server.DB.Model(&models.Order{}).Count(&countOrders)

    data := map[string]interface{}{
        "countCustomers": countCustomers,
        "countItems":     countItems,
        "countOrders":    countOrders,
        "user":           auth.CurrentUser(server.DB, w, r), // Agar nama di sidebar muncul
    }

    // Path: templates/admin/orders_dashboard.html
    _ = adminRender().HTML(w, http.StatusOK, "pages/admin_order_dashboard", data)
}

// 2. Tabel Customer
func (server *Server) ListCustomers(w http.ResponseWriter, r *http.Request) {
    var customers []models.OrderCustomer
    server.DB.Order("created_at desc").Find(&customers)

    data := map[string]interface{}{
        "customers": customers,
        "user":      auth.CurrentUser(server.DB, w, r),
    }

    // Path: templates/admin/customers.html
    _ = adminRender().HTML(w, http.StatusOK, "pages/admin_customers", data)
}

// 3. Tabel Order Items
func (server *Server) ListOrderItems(w http.ResponseWriter, r *http.Request) {
    var items []models.OrderItem
    server.DB.Preload("Product").Order("created_at desc").Find(&items)

    data := map[string]interface{}{
        "items": items,
        "user":  auth.CurrentUser(server.DB, w, r),
    }

    // Path: templates/admin/order_items.html
    _ = adminRender().HTML(w, http.StatusOK, "pages/order_item", data)
}

// 4. Tabel Orders
func (server *Server) ListOrders(w http.ResponseWriter, r *http.Request) {
    var orders []models.Order
    server.DB.Order("created_at desc").Find(&orders)

    data := map[string]interface{}{
        "orders": orders,
        "user":   auth.CurrentUser(server.DB, w, r),
    }

    // Path: templates/admin/orders.html
    _ = adminRender().HTML(w, http.StatusOK, "pages/order", data)
}

func SyncToAI(name string, desc string) {
    url := "http://localhost:8000/add-product"

    payload := ProductAI{
        Name: name,
        Desc: desc,
    }
    
    // Pastikan pakai "json" dari "encoding/json"
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return
    }

    // Pastikan pakai "bytes"
    _, err = http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Println("Gagal konek ke Python:", err)
    }
}