package middlewares

import (
	"net/http"

	"github.com/gieart87/gotoko/app/core/session/auth"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(r) {
		http.Redirect(w, r, "/login?error=Anda+Perlu+Login", http.StatusSeeOther)
		return
	}
		next.ServeHTTP(w, r)
	})
}