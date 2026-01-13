package controllers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/gieart87/gotoko/app/models"
	"github.com/gieart87/gotoko/database/seeders"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/urfave/cli"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	DB        *gorm.DB
	Router    *mux.Router
	AppConfig *AppConfig
}

type AppConfig struct {
	AppName string
	AppEnv  string
	AppPort string
	AppURL  string
}

type DBConfig struct {
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
	DBDriver   string
}

type PageLink struct {
	Page          int32  // huruf besar → bisa diakses template
	Url           string // huruf besar
	IsCurrentPage bool
}

// PERBAIKAN: Mengganti PrevPage dan NextPage menjadi string URL.
type PaginationLinks struct {
	CurrentPage int32
	NextPage    string // Ganti dari int32 ke string (akan diisi URL)
	PrevPage    string // Ganti dari int32 ke string (akan diisi URL)
	TotalRows   int32
	TotalPages  int32
	Links       []PageLink
}

type PaginationParams struct {
	Path        string
	TotalRows   int32
	PerPage     int32
	CurrentPage int32
}

var (
	store = sessions.NewCookieStore([]byte("secret-key"))
	sessionUser = "user-session"
)


func (server *Server) Initialize(appConfig AppConfig, dbConfig DBConfig) {
	fmt.Println("Welcome to " + appConfig.AppName)

	server.initializeDB(dbConfig)
	server.initializeAppConfig(appConfig)
	server.initializeRoutes()
}

func (server *Server) Run(addr string) {
	fmt.Printf("Listening to Port %s \n", addr)
	log.Fatal(http.ListenAndServe(addr, server.Router))
}

func (server *Server) initializeDB(dbConfig DBConfig) {
	var err error

	if dbConfig.DBDriver == "mysql" {
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConfig.DBUser,
			dbConfig.DBPassword,
			dbConfig.DBHost,
			dbConfig.DBPort,
			dbConfig.DBName,
		)
		server.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	} else {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta client_encoding=UTF8",
			dbConfig.DBHost,
			dbConfig.DBUser,
			dbConfig.DBPassword,
			dbConfig.DBName,
			dbConfig.DBPort,
		)
		server.DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		panic("Failed on connecting to database server")
	}
}

func (server *Server) initializeAppConfig(appconfig AppConfig) {
	server.AppConfig = &appconfig
}

func (server *Server) dbMigrate() {
	for _, model := range models.RegisterModels() {
		err := server.DB.Debug().AutoMigrate(model.Model)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Database migrated successfully.")
}

func (server *Server) InitCommands(config AppConfig, dbConfig DBConfig) {
	server.initializeDB(dbConfig)

	cmdApp := cli.NewApp()
	cmdApp.Commands = []cli.Command{
		{
			Name: "db:migrate",
			Action: func(c *cli.Context) error {
				server.dbMigrate()
				return nil
			},
		},
		{
			Name: "db:seed",
			Action: func(c *cli.Context) error {
				err := seeders.DBSeed(server.DB)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
	}

	err := cmdApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

/* ================================
   PAGINATION PERFEK (FINAL)
   ================================ */

// PERBAIKAN: Menghasilkan URL string untuk PrevPage dan NextPage.
func GetPaginationLinks(config *AppConfig, params PaginationParams) (PaginationLinks, error) {
	totalPages := int32(math.Ceil(float64(params.TotalRows) / float64(params.PerPage)))

	if totalPages < 1 {
		totalPages = 1
	}

	currentPage := params.CurrentPage
	if currentPage < 1 {
		currentPage = 1
	}
	if currentPage > totalPages {
		currentPage = totalPages
	}

	var links []PageLink
	for i := int32(1); i <= totalPages; i++ {
		links = append(links, PageLink{
			Page:          i,
			Url:           fmt.Sprintf("/%s?page=%d", params.Path, i),
			IsCurrentPage: i == currentPage,
		})
	}

	// Hitung nomor halaman sebelumnya dan selanjutnya
	prevPageNum := currentPage - 1
	if prevPageNum < 1 {
		prevPageNum = 1
	}

	nextPageNum := currentPage + 1
	if nextPageNum > totalPages {
		nextPageNum = totalPages
	}

	// Buat URL dari nomor halaman yang sudah dihitung
	prevPageURL := fmt.Sprintf("/%s?page=%d", params.Path, prevPageNum)
	nextPageURL := fmt.Sprintf("/%s?page=%d", params.Path, nextPageNum)

	return PaginationLinks{
		CurrentPage: currentPage,
		PrevPage:    prevPageURL, // Menggunakan URL string
		NextPage:    nextPageURL, // Menggunakan URL string
		TotalRows:   params.TotalRows,
		TotalPages:  totalPages,
		Links:       links,
	}, nil
}

func (server *Server) GetProvinces() ([]models.Province, error) {
	var provinces []models.Province

	// Data statis untuk demo
	provinces = []models.Province{
		{ID: "1", Name: "DKI Jakarta"},
		{ID: "2", Name: "Jawa Barat"},
		{ID: "3", Name: "Jawa Tengah"},
		{ID: "4", Name: "Jawa Timur"},
		{ID: "5", Name: "Banten"},
		{ID: "6", Name: "Sumatera Utara"},
		// Tambahkan lebih banyak jika perlu
	}

	return provinces, nil
}

