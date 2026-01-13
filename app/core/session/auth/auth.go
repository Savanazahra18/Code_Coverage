package auth

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gieart87/gotoko/app/models"
	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	store = sessions.NewCookieStore(securecookie.GenerateRandomKey(32), securecookie.GenerateRandomKey(32))
	sessionUser = "user-session"
)

func GetSessionUser(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionUser)
}

func IsLoggedIn(r *http.Request) bool {
	session, err := store.Get(r, sessionUser) // ✅ store, bukan sessionStore
	if err != nil {
		fmt.Println("error login ==>", err)
		return false
	}
	_, ok := session.Values["user_id"]
	return ok
}

func ComparePassword(password string, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

func MakePassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func CurrentUser(db *gorm.DB, w http.ResponseWriter, r *http.Request) *models.User {
	session, err := store.Get(r, sessionUser)
	if err != nil {
		return nil
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok {
		return nil
	}

	user, err := (&models.User{}).FindByID(db, userID)
	if err != nil {
		session.Options.MaxAge = -1
		session.Save(r, w)
		return nil
	}

	return user
}
func GetCartID(w http.ResponseWriter, r *http.Request) string {
	
	session, err := store.Get(r, sessionUser)
	if err != nil {
		log.Printf("Session error: %v", err)
		session = sessions.NewSession(store, sessionUser)
	}

	if session.Values["cart-id"] == nil {
		session.Values["cart-id"] = uuid.NewString()
	}

	_ = session.Save(r, w)
	return session.Values["cart-id"].(string)
	
}