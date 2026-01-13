package controllers

import (
	"fmt"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/models"
	"github.com/gorilla/mux"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/shopspring/decimal"
	"github.com/unrolled/render"
	"gorm.io/gorm"
)

type CheckoutRequest struct {
	Cart            *models.Cart
	ShippingFee     *ShippingFee
	ShippingAddress *ShippingAddress
}

type ShippingFee struct {
	Courier     string
	PackageName string
	Fee         float64
}

type ShippingAddress struct {
	FirstName  string
	LastName   string
	CityID     string
	ProvinceID string
	Address1   string
	Address2   string
	Phone      string
	Email      string
	PostCode   string
}

func (server *Server) Checkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/carts?error=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	user := auth.CurrentUser(server.DB, w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, _ := auth.GetSessionUser(r)
	courier, _ := session.Values["checkout_courier"].(string)
	if courier == "" {
		http.Redirect(w, r, "/carts?error=Shipping+belum+dipilih", http.StatusSeeOther)
		return
	}

	cartID := auth.GetCartID(w, r)
	cart, err := GetShoppingCart(server.DB, cartID)
	if err != nil {
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
		return
	}

	province, _ := session.Values["checkout_province"].(string)
	city, _ := session.Values["checkout_city"].(string)
	cost, _ := session.Values["checkout_shipping_cost"].(int)

	checkoutReq := &CheckoutRequest{
		Cart: cart,
		ShippingFee: &ShippingFee{
			Courier:     courier,
			PackageName: "REG",
			Fee:         float64(cost),
		},
		ShippingAddress: &ShippingAddress{
			FirstName:  r.FormValue("first_name"),
			LastName:   r.FormValue("last_name"),
			Address1:   r.FormValue("address1"),
			Address2:   r.FormValue("address2"),
			Phone:      r.FormValue("phone"),
			Email:      r.FormValue("email"),
			PostCode:   r.FormValue("post_code"),
			CityID:     city,
			ProvinceID: province,
		},
	}

	order, err := server.SaveOrder(user, checkoutReq)
	if err != nil {
		log.Println("❌ SaveOrder error:", err)
		http.Redirect(w, r, "/carts?error=Checkout+gagal", http.StatusSeeOther)
		return
	}

	ClearCart(server.DB, cartID)
	http.Redirect(w, r, "/orders/"+order.ID, http.StatusSeeOther)
}

func (server *Server) ShowOrder(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	vars := mux.Vars(r)
	if vars["id"] == "" {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	var order models.Order
	if err := server.DB.
		Preload("OrderCustomer").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("id = ?", vars["id"]).
		First(&order).Error; err != nil {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	render.HTML(w, http.StatusOK, "show_order", map[string]interface{}{
		"order":   &order,
		"user":    auth.CurrentUser(server.DB, w, r),
		"success": r.URL.Query().Get("success"),
	})
}

func (server *Server) SaveOrder(user *models.User, r *CheckoutRequest) (*models.Order, error) {
	orderID := uuid.New().String()
	shippingCost := decimal.NewFromFloat(r.ShippingFee.Fee)

	grandTotal := r.Cart.BaseTotalPrice.
		Add(r.Cart.TaxAmount).
		Sub(r.Cart.DiscountAmount).
		Add(shippingCost)

	// Gunakan Transaction agar jika stok kurang, order tidak tersimpan
	tx := server.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Buat & Simpan Order Utama
	order := models.Order{
		ID:                  orderID,
		UserID:              user.ID,
		Status:              0,
		OrderDate:           time.Now(),
		PaymentDue:          time.Now().AddDate(0, 0, 7),
		PaymentStatus:       consts.OrderPaymentStatusUnpaid,
		BaseTotalPrice:      r.Cart.BaseTotalPrice,
		TaxAmount:           r.Cart.TaxAmount,
		TaxPercent:          r.Cart.TaxPercent,
		DiscountAmount:      r.Cart.DiscountAmount,
		DiscountPercent:     r.Cart.DiscountPercent,
		ShippingCost:        shippingCost,
		GrandTotal:          grandTotal,
		ShippingCourier:     r.ShippingFee.Courier,
		ShippingServiceName: r.ShippingFee.PackageName,
		PaymentToken:        sql.NullString{String: "", Valid: false},
	}

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. Simpan Order Customer
	orderCustomer := models.OrderCustomer{
		OrderID:      order.ID,
		UserID:       user.ID,
		FirstName:    r.ShippingAddress.FirstName,
		LastName:     r.ShippingAddress.LastName,
		Address1:     r.ShippingAddress.Address1,
		Address2:     r.ShippingAddress.Address2,
		Phone:        r.ShippingAddress.Phone,
		Email:        r.ShippingAddress.Email,
		CityName:     r.ShippingAddress.CityID,
		ProvinceName: r.ShippingAddress.ProvinceID,
		PostCode:     r.ShippingAddress.PostCode,
	}
	if err := tx.Create(&orderCustomer).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 3. Simpan Order Items & KURANGI STOK
	for _, cartItem := range r.Cart.CartItems {
		item := models.OrderItem{
			OrderID:         orderID,
			ProductID:       cartItem.ProductID,
			Qty:             cartItem.Qty,
			BasePrice:       cartItem.BasePrice,
			BaseTotal:       cartItem.BaseTotal,
			TaxAmount:       cartItem.TaxAmount,
			TaxPercent:      cartItem.TaxPercent,
			DiscountAmount:  cartItem.DiscountAmount,
			DiscountPercent: cartItem.DiscountPercent,
			SubTotal:        cartItem.SubTotal,
			Name:            cartItem.Product.Name,
		}

		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// LOGIKA PENGURANGAN STOK:
		// Kurangi stok di tabel 'products' berdasarkan ProductID
		// Kita tambahkan pengecekan agar stok tidak menjadi negatif
		result := tx.Model(&models.Product{}).
			Where("id = ? AND stock >= ?", cartItem.ProductID, cartItem.Qty).
			Update("stock", gorm.Expr("stock - ?", cartItem.Qty))

		if result.Error != nil {
			tx.Rollback()
			return nil, result.Error
		}

		if result.RowsAffected == 0 {
			tx.Rollback()
			return nil, fmt.Errorf("stok produk %s tidak mencukupi", cartItem.Product.Name)
		}
	}

	// 4. Buat payment URL (Midtrans)
	paymentURL, err := server.createPaymentURL(user, grandTotal, order.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update payment token
	order.PaymentToken = sql.NullString{String: paymentURL, Valid: true}
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Selesaikan Transaksi
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &order, nil
}

func (server *Server) createPaymentURL(
	user *models.User,
	grandTotal decimal.Decimal,
	orderID string,
) (string, error) {
	midtrans.ServerKey = os.Getenv("API_MIDTRANS_SERVER_KEY")
	midtrans.Environment = midtrans.Sandbox

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID, // ✅ UUID
			GrossAmt: grandTotal.IntPart(),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.FirstName,
			LName: user.LastName,
			Email: user.Email,
		},
	}

	resp, err := snap.CreateTransaction(req)
	if err != nil {
		return "", err
	}

	return resp.RedirectURL, nil
}