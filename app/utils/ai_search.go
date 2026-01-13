package utils // Sesuaikan dengan nama foldernya

import (
	"bytes"
	"encoding/json"
	"net/http"
	"github.com/gieart87/gotoko/app/models"
	"gorm.io/gorm"
)

// GetSmartSearch mengirim keyword ke server Python FastAPI
func GetSmartSearch(keyword string) ([]string, error) {
	jsonData, _ := json.Marshal(map[string]string{
		"keyword": keyword,
	})

	// Ganti localhost:8000 jika port Python kamu berbeda
	resp, err := http.Post("http://localhost:8000/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []string `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Results, nil
}

func SyncAI(db *gorm.DB) error {
	var products []models.Product
	// Mengambil semua produk dari database
	if err := db.Find(&products).Error; err != nil {
		return err
	}

	// Struktur data yang sesuai dengan Pydantic di Python
	type ProductItem struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	
	type SyncPayload struct {
		Products []ProductItem `json:"products"`
	}

	var payload SyncPayload
	for _, p := range products {
		payload.Products = append(payload.Products, ProductItem{
			Name: p.Name,
			Desc: p.Description,
		})
	}

	// Konversi ke JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Kirim ke endpoint /sync milik Python
	resp, err := http.Post("http://localhost:8000/sync", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}