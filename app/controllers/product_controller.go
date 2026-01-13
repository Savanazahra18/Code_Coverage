package controllers

import (
	"net/http"
	"strconv"


	"github.com/gieart87/gotoko/app/utils"
	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/models"
	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

func (server *Server) getRenderer() *render.Render {
	return render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})
}

func (server *Server) Products(w http.ResponseWriter, r *http.Request) {
    render := render.New(render.Options{
        Layout:     "layout",
        Extensions: []string{".html", ".tmpl"},
    })

    q := r.URL.Query()
    categorySlug := q.Get("category")

    page, _ := strconv.Atoi(q.Get("page"))
    if page <= 0 { page = 1 }
    perPage := 9

    productModel := models.Product{}
    
    // GetProducts di model harus sudah menggunakan .Preload("ProductImages")
    products, totalRows, err := productModel.GetProducts(server.DB, perPage, page, categorySlug)
    if err != nil {
        http.Error(w, "Gagal memuat produk", http.StatusInternalServerError)
        return
    }

    pagination, _ := GetPaginationLinks(server.AppConfig, PaginationParams{
        Path:        "products",
        TotalRows:   int32(totalRows),
        PerPage:     int32(perPage),
        CurrentPage: int32(page),
    })

    user := auth.CurrentUser(server.DB, w, r)

    _ = render.HTML(w, http.StatusOK, "products", map[string]interface{}{
        "products":   products, // Slice ini sekarang membawa data .Stock
        "pagination": pagination,
        "user":       user,
        "category":   categorySlug, 
    })
}

func (server *Server) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout: "layout",
		Extensions: []string{".html", ".tmpl"},
	})
	vars := mux.Vars(r)

	if vars["slug"] == "" {
		return
	}

	productModel := models.Product{}
	product, err := productModel.FindBySlug(server.DB, vars["slug"])
	if err != nil {
		return
	}

	user := auth.CurrentUser(server.DB, w, r)
	_ = render.HTML(w, http.StatusOK, "product", map[string]interface{}{
		"product": product,
		"user":    user,
	})
}

func (server *Server) SearchProducts(w http.ResponseWriter, r *http.Request) {
    // Ganti server.getRenderer() dengan ini agar tidak merah:
    render := render.New(render.Options{
        Layout:     "layout",
        Extensions: []string{".html", ".tmpl"},
    })
	
	// 1. Ambil keyword dari URL (?q=obat+pusing)
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	// 2. Tanya ke AI Python (FastAPI) via Utils
	smartKeywords, err := utils.GetSmartSearch(query)
	
	var products []models.Product
	productModel := models.Product{}

	// 3. Logika Hasil: Jika AI memberikan saran, cari berdasarkan saran AI tersebut
	if err == nil && len(smartKeywords) > 0 {
		// Kita ambil kata kunci pertama hasil perbaikan AI untuk cari di DB
		products, _ = productModel.SearchByKeywords(server.DB, smartKeywords)
	} else {
		// Jika AI gagal/tidak ada hasil, lakukan pencarian LIKE standar di DB
		products, _ = productModel.StandardSearch(server.DB, query)
	}

	user := auth.CurrentUser(server.DB, w, r)

	// 4. Kirim ke template search_results.html
	_ = render.HTML(w, http.StatusOK, "search_results", map[string]interface{}{
		"products": products,
		"keyword":  query,
		"user":     user,
		"ai_suggestions": smartKeywords, // Tampilkan di UI jika ingin "Maksud anda: ..."
	})
}