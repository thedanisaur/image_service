package main

import (
	"encoding/json"
	"fmt"

	"image_service/crawler"
	"image_service/db"
	"image_service/handlers"
	"image_service/security"
	"image_service/types"
	"image_service/util"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func AuthorizationMiddleware(config types.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		txid := uuid.New()
		log.Printf("%s | %s\n", util.GetFunctionName(AuthorizationMiddleware), txid.String())

		err := security.ValidateJWT(config)(c)
		if err != nil {
			log.Printf("Failed to Validate JWT\n%s\n", err)
			err_string := fmt.Sprintf("Unauthorized: %s\n", txid.String())
			return c.Status(fiber.StatusInternalServerError).SendString(err_string)
		}
		return c.Next()
	}
}

// TODO this belongs somewhere else
func fetchImages(config types.Config, fetch_interval time.Duration, query_interval time.Duration) {
	for {
		log.Printf("Image Worker - Start fetch")
		database, err := db.GetInstance()
		if err != nil {
			log.Printf("Failed to connect to DB\n%s\n", err.Error())
		}
		movies_query := `
			SELECT m.movie_title
				, m.movie_name
				, s.series_name
			FROM movies m
			INNER JOIN series s
			ON s.series_name = m.series_name
			LEFT JOIN movies_images mi
			ON m.movie_name = mi.movie_name
			WHERE mi.movie_name IS NULL
		`

		movies_rows, err := database.Query(movies_query)
		if err != nil {
			log.Printf("Failed to query movies:\n%s\n", err.Error())
		}

		for movies_rows.Next() {
			var movie types.Image
			err = movies_rows.Scan(&movie.MovieTitle, &movie.MovieName, &movie.SeriesName)
			if err != nil {
				log.Printf("Failed to scan movies_rows:\n%s\n", err.Error())
			}

			_, err := crawler.Request(config.App.Client.UserAgent,
				fmt.Sprintf("%s:%d/%s", "http://localhost", config.App.Host.Port, "images"),
				fasthttp.MethodPost,
				movie,
			)
			if err != nil {
				log.Printf("request failed:\n%s\n", err.Error())
			}

			time.Sleep(fetch_interval)
		}

		time.Sleep(query_interval)
	}
}

func loadConfig(config_path string) (types.Config, error) {
	var config types.Config
	config_file, err := os.Open(config_path)
	if err != nil {
		return config, err
	}
	defer config_file.Close()
	jsonParser := json.NewDecoder(config_file)
	jsonParser.Decode(&config)
	return config, nil
}

func main() {
	log.Println("Starting Image Service...")
	config, err := loadConfig("./config.json")
	if err != nil {
		log.Printf("Error opening config, cannot continue: %s\n", err.Error())
		return
	}
	app := fiber.New()
	database, err := db.GetInstance()
	if err != nil {
		log.Print(err.Error())
	} else {
		defer database.Close()
	}

	// Start Workers
	fetch_interval := time.Duration(config.App.Workers.ImageFetch.FetchInterval) * time.Millisecond
	query_interval := time.Duration(config.App.Workers.ImageFetch.QueryInterval) * time.Millisecond
	go fetchImages(config, fetch_interval, query_interval)

	// Add CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.App.Cors.AllowOrigins, ","),
		AllowHeaders:     strings.Join(config.App.Cors.AllowHeaders, ","),
		AllowCredentials: config.App.Cors.AllowCredentials,
	}))

	// Add Rate Limiter
	var middleware limiter.LimiterHandler
	if config.App.Limiter.LimiterSlidingMiddleware {
		middleware = limiter.SlidingWindow{}
	} else {
		middleware = limiter.FixedWindow{}
	}
	app.Use(limiter.New(limiter.Config{
		Max:                    config.App.Limiter.Max,
		Expiration:             time.Millisecond * time.Duration(config.App.Limiter.Expiration),
		LimiterMiddleware:      middleware,
		SkipSuccessfulRequests: config.App.Limiter.SkipSuccessfulRequests,
	}))

	// Non Authenticated routes
	app.Get("/images/:movie_name", handlers.GetImage(config))

	// JWT Authentication routes
	app.Post("/images", AuthorizationMiddleware(config), handlers.FetchImage(config))

	port := fmt.Sprintf(":%d", config.App.Host.Port)
	if config.App.Host.UseTLS {
		err = app.ListenTLS(port, config.App.Host.CertificatePath, config.App.Host.KeyPath)
	} else {
		log.Println("Warning - not using TLS")
		err = app.Listen(port)
	}
	if err != nil {
		log.Fatal(err.Error())
	}
}
