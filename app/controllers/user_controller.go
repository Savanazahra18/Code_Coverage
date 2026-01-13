package controllers

import (
	"fmt"
	"net/http"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/models"
	"github.com/google/uuid"
	"github.com/unrolled/render"
	"github.com/gieart87/gotoko/app/consts"
)

func (server *Server) Login(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	errorMsg := r.URL.Query().Get("error")

	_ = render.HTML(w, http.StatusOK, "login", map[string]interface{}{
		"Error": errorMsg,
	})
}

func (server *Server) DoLogin(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    email := r.FormValue("email")
    password := r.FormValue("password")

    if email == "" || password == "" {
        http.Redirect(w, r, "/login?error=Email dan password wajib diisi", http.StatusSeeOther)
        return
    }

    userModel := models.User{}
    user, err := userModel.FindByEmail(server.DB, email)
    if err != nil || user == nil || !auth.ComparePassword(password, user.Password) {
        http.Redirect(w, r, "/login?error=email+atau+password+salah", http.StatusSeeOther)
        return
    }

   session, err := auth.GetSessionUser(r)
	if err != nil {
    fmt.Println("Gagal mengambil sesi lama:", err)
	}


	session.Values["user_id"] = user.ID


	if err := session.Save(r, w); err != nil {
    http.Redirect(w, r, "/login?error=Gagal menyimpan sesi baru", http.StatusSeeOther)
    return
}
   
    if user.Role.Name == consts.RoleAdmin || user.Role.Name == consts.RoleOperator {
        http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
        return
    }

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (server *Server) Register(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	errorMsg := r.URL.Query().Get("error")

	_ = render.HTML(w, http.StatusOK, "register", map[string]interface{}{
		"Error": errorMsg,
	})
}

func (server *Server) DoRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if firstName == "" || lastName == "" || email == "" || password == "" {
		http.Redirect(w, r, "/register?error=First name, last name, email and password are required!", http.StatusSeeOther)
		return
	}

	userModel := models.User{}
	existUser, _ := userModel.FindByEmail(server.DB, email)
	if existUser != nil {
		http.Redirect(w, r, "/register?error=Sorry, email already registered!", http.StatusSeeOther)
		return
	}

	hashedPassword, err := auth.MakePassword(password)
	if err != nil {
		http.Redirect(w, r, "/register?error=Terjadi kesalahan saat membuat password", http.StatusSeeOther)
		return
	}

	params := &models.User{
		ID:        uuid.New().String(),
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Password:  hashedPassword,
	}

	user, err := userModel.CreateUser(server.DB, params)
	if err != nil {
		http.Redirect(w, r, "/register?error=Registration failed", http.StatusSeeOther)
		return
	}

	session, err := auth.GetSessionUser(r)
	if err != nil {
		http.Redirect(w, r, "/?error=Gagal membuat sesi", http.StatusSeeOther)
		return
	}

	session.Values["user_id"] = user.ID
	if err := session.Save(r, w); err != nil {
		http.Redirect(w, r, "/?error=Gagal menyimpan sesi", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (server *Server) Logout(w http.ResponseWriter, r *http.Request) {
    session, err := auth.GetSessionUser(r)
    if err != nil {
        // Jika sesi tidak ditemukan, tetap lempar ke halaman utama atau login
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // Hapus data user dari session
    delete(session.Values, "user_id")
    
    // Memberi tahu browser untuk menghapus cookie dengan set MaxAge ke -1
    session.Options.MaxAge = -1 
    _ = session.Save(r, w)

    // Redirect ke halaman login agar admin bisa masuk kembali jika mau
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}