package controllers

import (
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/models"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/shopspring/decimal"
)

func (server *Server) Midtrans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload models.MidtransNotification
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("❌ JSON Decode Error: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	log.Printf("📥 Midtrans Webhook: OrderID=%s, Status=%s, Fraud=%s",
		payload.OrderID, payload.TransactionStatus, payload.FraudStatus)

	// ✅ VALIDASI SIGNATURE
	if err := validateSignatureKey(&payload); err != nil {
		log.Printf("❌ Invalid Signature: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// 🔑 CARI ORDER BERDASARKAN ID
	var order models.Order
	if err := server.DB.
		Preload("OrderCustomer").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Preload("User").
		Where("id = ?", payload.OrderID).
		First(&order).Error; err != nil {

		log.Printf("❌ Order ID %s not found in database", payload.OrderID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	log.Printf("✅ Order Found: ID=%s, Code=%s, PaymentStatus=%s",
		order.ID, order.Code, order.PaymentStatus)

	// Jika sudah paid, abaikan
	if order.IsPaid() {
		log.Println("⚠️ Order already paid")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// ✅ SIMPAN PAYMENT RECORD (langsung pakai GORM)
	amount, _ := decimal.NewFromString(payload.GrossAmount)
	rawPayload, _ := json.Marshal(payload)
	raw := json.RawMessage(rawPayload)

	payment := models.Payment{
		OrderID:           order.ID,
		Amount:            amount,
		TransactionID:     payload.TransactionID,
		TransactionStatus: payload.TransactionStatus,
		PaymentType:       payload.PaymentType,
		PayLoad:           &raw,
	}

	// 🔥 PERBAIKAN UTAMA: GUNAKAN db.Create LANG Langsung
	if err := server.DB.Create(&payment).Error; err != nil {
		log.Printf("❌ Failed to save payment: %v", err)
		// Tapi tetap kirim OK ke Midtrans
	}

	// Cek apakah pembayaran sukses
	if isPaymentSuccess(&payload) {
		log.Println("🎉 Payment SUCCESS — updating order status...")

		// Update status order
		if err := order.MarkAsPaid(server.DB); err != nil {
			log.Printf("❌ Failed to mark order as paid: %v", err)
		} else {
			log.Printf("✅ Order %s successfully marked as PAID!", order.ID)
		}
	} else {
		log.Printf("⚠️ Payment not successful: Status=%s, Fraud=%s",
			payload.TransactionStatus, payload.FraudStatus)
	}

	// ✅ RESPON WAJIB: 200 OK + "OK"
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ===================================================
// HELPER FUNCTIONS
// ===================================================

func isPaymentSuccess(p *models.MidtransNotification) bool {
	switch p.PaymentType {
	case string(snap.PaymentTypeCreditCard):
		return p.TransactionStatus == consts.PaymentStatusCapture &&
			p.FraudStatus == consts.FraudStatusAccept
	default:
		return p.TransactionStatus == consts.PaymentStatusSettlement &&
			p.FraudStatus == consts.FraudStatusAccept // ✅ tambahkan ini!
	}
}

func validateSignatureKey(p *models.MidtransNotification) error {
	if os.Getenv("APP_ENV") == "development" {
		return nil
	}

	payload := p.OrderID + p.StatusCode + p.GrossAmount + os.Getenv("API_MIDTRANS_SERVER_KEY")
	hash := sha512.Sum512([]byte(payload))
	signature := fmt.Sprintf("%x", hash)

	if signature != p.SignatureKey {
		return errors.New("invalid signature key")
	}

	return nil
}