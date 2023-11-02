package handlers

import (
	"errors"
	"fmt"
	"image_service/crawler"
	"image_service/db"
	"image_service/types"
	"image_service/util"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func GetImage(c *fiber.Ctx) error {
	txid := uuid.New()
	log.Printf("%s | %s\n", util.GetFunctionName(GetImage), txid.String())
	movie_name := c.Params("movie_name")
	// Get image path from DB
	database, err := db.GetInstance()
	if err != nil {
		log.Printf("Failed to connect to DB\n%s\n", err.Error())
		err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
		return c.Status(fiber.StatusInternalServerError).SendString(err_string)
	}
	query_string := `
		SELECT image_path
		FROM movies_images_vw
		WHERE movie_name = ?;
	`
	row := database.QueryRow(query_string, movie_name)
	var image types.Image
	err = row.Scan(&image.ImagePath)
	if err != nil {
		log.Printf("Database Error:\n%s\n", err.Error())
		err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
		return c.Status(fiber.StatusInternalServerError).SendString(err_string)
	}

	_, err = os.Stat(image.ImagePath)
	if err != nil {
		log.Printf("File not on disk:\n%s\n", err.Error())
		err_string := fmt.Sprintf("File not on disk: %s\n", txid.String())
		return c.Status(fiber.StatusNotFound).SendString(err_string)
	}

	c.Response().Header.Set("Content-Type", "application/octet-stream")
	return c.Status(fiber.StatusOK).SendFile(image.ImagePath)
}

func FetchImage(config types.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		txid := uuid.New()
		log.Printf("%s | %s\n", util.GetFunctionName(FetchImage), txid.String())
		// if security.ValidateJWT(c) != nil {
		// 	return c.Status(fiber.StatusUnauthorized).SendString(fmt.Sprintf("Unauthorized: %s\n", txid.String()))
		// }
		// TODO in config see if we can remove ./ from the image.path
		var movie_data types.Image
		err := c.BodyParser(&movie_data)
		if err != nil {
			log.Printf("Failed to parse movie data\n%s\n", err.Error())
			return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to parse movie data: %s\n", txid.String()))
		}

		// First search imdb and find the url for the requested movie
		movie_title := strings.Replace(movie_data.MovieTitle, " ", "%20", -1)
		main_url := fmt.Sprintf("https://www.imdb.com/find/?q=%s&ref_=nv_sr_sm", movie_title)
		response, err := crawler.Request(config, main_url, fasthttp.MethodGet)
		if err != nil {
			err_str := "Failed to fetch imbd.com\n%s\n"
			log.Printf(err_str, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, txid.String()))
		}
		// search for the route in the html body
		params := crawler.MatcherParams{
			Haystack: string(response.Body()),
			Needle:   movie_data.MovieTitle,
			AttrKey:  "href",
			Atom:     atom.A.String(),
			NodeType: html.ElementNode,
		}
		title_route, err := crawler.Find(params, crawler.ImdbFindMovieUrl)
		if err != nil {
			err_str := "Failed to find url for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, txid.String()))
		}

		// Go to title page and find the url for the image page
		title_page_url := fmt.Sprintf("https://www.imdb.com%s", title_route)
		response, err = crawler.Request(config, title_page_url, fasthttp.MethodGet)
		if err != nil {
			err_str := "Failed to fetch the title page for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}
		// left, right, found, remove everything past the title's base path i.e. /title/12345.../
		left, _, _ := strings.Cut(title_route, "?")
		// search for the url in the html body
		image_poster_params := crawler.MatcherParams{
			Haystack: string(response.Body()),
			Needle:   fmt.Sprintf("%smediaviewer", left),
			AttrKey:  "href",
			Atom:     atom.A.String(),
			NodeType: html.ElementNode,
		}
		image_page_route, err := crawler.Find(image_poster_params, crawler.ImdbFindImagePosterUrl)
		if err != nil {
			err_str := "Failed to find image page for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}

		// // Go to image page and find the link to the actual image
		image_page_url := fmt.Sprintf("https://www.imdb.com%s", image_page_route)
		response, err = crawler.Request(config, image_page_url, fasthttp.MethodGet)
		if err != nil {
			err_str := "Failed to fetch the image page for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}
		// Remove <noscript> tags as they break parsing of img tags in golang.org/x/net/html
		hdata := strings.Replace(string(response.Body()), "<noscript>", "", -1)
		hdata = strings.Replace(hdata, "<noscript data-n-css=\"\">", "", -1)
		hdata = strings.Replace(hdata, "</noscript>", "", -1)
		// search for the url in the html body
		image_params := crawler.MatcherParams{
			Haystack: string(hdata),
			Needle:   "https://m.media-amazon.com/images",
			AttrKey:  "src",
			Atom:     atom.Img.String(),
			NodeType: html.ElementNode,
		}
		image_url, err := crawler.Find(image_params, crawler.ImdbFindImageUrl)
		if err != nil {
			err_str := "Failed to find image url for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}

		// Now that we have the actual image url we can save the image
		// Check to see if folder for the series exists
		path := fmt.Sprintf("%s%s/", config.Images.Path, movie_data.SeriesName)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				log.Println(err)
				err_str := fmt.Sprintf("Could not create folder for %s: %s\n", movie_data.MovieTitle, txid.String())
				return c.Status(fiber.StatusInternalServerError).SendString(err_str)
			}
		}
		// Download the image
		response, err = crawler.Request(config, image_url, fasthttp.MethodGet)
		if err != nil {
			err_str := "Failed to download the image for %s: \n%s\n"
			log.Printf(err_str, movie_data.MovieTitle, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}
		path_and_filename := fmt.Sprintf("%s/%s%s", path, movie_data.MovieName, config.Images.Type)
		// Write file to disk
		err = ioutil.WriteFile(path_and_filename, response.Body(), 0644)
		if err != nil {
			err_str := "Failed to write image to disk: %s%s\n%s\n"
			log.Printf(err_str, path, movie_data.MovieName, err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		}
		// // Alternate way to write to disk...not sure if worth it
		// file, err := os.Create(fmt.Sprintf("%s/%s%s", path, movie_data.MovieName, config.Images.Type))
		// if err != nil {
		// 	err_str := "Failed to create file on disk: %s%s\n%s\n"
		// 	log.Printf(err_str, path, movie_data.MovieName, err.Error())
		// 	return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		// }
		// defer file.Close()
		// img, _, err := image.Decode(bytes.NewReader(response.Body()))
		// if err != nil {
		// 	log.Fatalln(err)
		// }

		// opts := jpeg.Options{
		// 	Quality: 10,
		// }

		// err = jpeg.Encode(file, img, &opts)
		// if err != nil {
		// 	err_str := "Failed to write image to disk: %s%s\n%s\n"
		// 	log.Printf(err_str, path, movie_data.MovieName, err.Error())
		// 	return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf(err_str, movie_data.MovieTitle, txid.String()))
		// }
		// file.Close()

		// Insert image path into DB
		err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
		database, err := db.GetInstance()
		if err != nil {
			log.Printf("Failed to connect to DB\n%s\n", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString(err_string)
		}
		image_id := uuid.New()
		query := `
			INSERT INTO images (image_id, image_path, image_created_on)
			VALUES (UUID_TO_BIN(?), ?, CURDATE());
		`
		result, err := database.Exec(query, image_id, path_and_filename)
		if err != nil {
			log.Printf("Failed to insert record into images:\n%s\n", err.Error())
			return c.Status(fiber.StatusServiceUnavailable).SendString(err.Error())
		}
		_, err = result.LastInsertId()
		if err != nil {
			log.Printf("Failed retrieve inserted id\n%s\n", err.Error())
			return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
		}

		// Insert movie image relation into DB
		movie_image_id := uuid.New()
		query = `
			INSERT INTO movies_images (movie_image_id, movie_name, image_id)
			VALUES (UUID_TO_BIN(?), ?, UUID_TO_BIN(?));
		`
		result, err = database.Exec(query, movie_image_id, movie_data.MovieName, image_id)
		if err != nil {
			log.Printf("Failed to insert record into movies images:\n%s\n", err.Error())
			return c.Status(fiber.StatusServiceUnavailable).SendString(err.Error())
		}
		_, err = result.LastInsertId()
		if err != nil {
			log.Printf("Failed retrieve inserted id\n%s\n", err.Error())
			return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
		}

		json := &fiber.Map{
			"id":             txid,
			"movie_image_id": movie_image_id,
			"movie_name":     movie_data.MovieName,
			"image_id":       image_id,
			"image_path":     path_and_filename,
		}

		return c.Status(fiber.StatusOK).JSON(json)
	}
}
