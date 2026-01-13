package controllers

import "github.com/gorilla/sessions"

var (
	sessionStore *sessions.CookieStore
	sessionName  = "user-session"
)

func SetSessionStore(store *sessions.CookieStore, name string) {
	sessionStore = store
	sessionName = name
}
