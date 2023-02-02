package main

import (
	"log"
	"movie_service/db"
	"movie_service/handlers"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func main() {
	log.Println("Starting Movie Service...")
	app := fiber.New()
	defer db.GetInstance().Close()

	// Add CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "127.0.0.1",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Add Rate Limiter
	app.Use(limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
	}))

	// Non Authenticated routes
	app.Get("/movies", handlers.GetMovies)
	app.Get("/movie_trackers/:tracker_id", handlers.GetMovieTrackersByID)
	app.Get("/movie_trackers/:movie_name/:username", handlers.GetMovieTrackers)
	app.Get("/ratings", handlers.GetRatings)
	app.Get("/series", handlers.GetSeries)
	app.Get("/timeline", handlers.GetTimeline)
	app.Get("/trackers", handlers.GetTrackers)

	// JWT Authentication routes
	app.Post("/movies/:series", handlers.PostMovie)
	app.Post("/movie_trackers/:username", handlers.PostMovieTrackers)
	app.Post("/series", handlers.PostSeries)
	app.Post("/trackers", handlers.PostTrackers)
	app.Post("/vote", handlers.PostVote)

	log.Fatal(app.ListenTLS(":4321", "./certs/cert.crt", "./keys/key.key"))
}
