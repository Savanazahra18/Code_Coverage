package app

import (
	"net/http"
	"flag"
	"log"
	"os"

	"github.com/gieart87/gotoko/app/controllers"
	"github.com/joho/godotenv"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Run() {
	server := controllers.Server{}
	appConfig := controllers.AppConfig{}
	dbConfig := controllers.DBConfig{}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	appConfig.AppName = getEnv("APP_NAME", "GoObat")
	appConfig.AppEnv = getEnv("APP_ENV", "development")
	appConfig.AppPort = getEnv("APP_PORT", "9000")
	appConfig.AppURL = getEnv("APP_URL", "http://localhost:9000")

	dbConfig.DBHost = getEnv("DB_HOST", "localhost")
	dbConfig.DBUser = getEnv("DB_USER", "postgres")
	dbConfig.DBPassword = getEnv("DB_PASSWORD", "nurfanis123")
	dbConfig.DBName = getEnv("DB_NAME", "indokoding_goobatdb")
	dbConfig.DBPort = getEnv("DB_PORT", "5433")
	dbConfig.DBDriver = getEnv("DB_DRIVER", "postgres")

	flag.Parse()
	arg := flag.Arg(0)

	if arg != "" {
		server.InitCommands(appConfig, dbConfig)
		return
	}

	
	server.Initialize(appConfig, dbConfig)


	server.Router.PathPrefix("/uploads/").
		Handler(http.StripPrefix("/uploads/",
			http.FileServer(http.Dir("public/uploads"))))

	server.Router.PathPrefix("/assets/").
		Handler(http.StripPrefix("/assets/",
			http.FileServer(http.Dir("public/assets"))))

	server.Run(":" + appConfig.AppPort)
}
