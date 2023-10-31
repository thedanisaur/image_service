package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"image_service/db"
	"image_service/types"
	"image_service/util"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func GetImages(c *fiber.Ctx) error {
	txid := uuid.New()
	log.Printf("%s | %s\n", util.GetFunctionName(GetImages), txid.String())
	err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
	database, err := db.GetInstance()
	if err != nil {
		log.Printf("Failed to connect to DB\n%s\n", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString(err_string)
	}
	query := `
		SELECT *
		FROM images
	`
	// TODO this will fail, so for now we just won't name the vars.
	// rows, err := database.Query(query)
	_, err = database.Query(query)
	if err != nil {
		log.Printf("Failed to query images:\n%s\n", err.Error())
		return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
	}

	// var images []types.Image
	// i := 0
	// for rows.Next() {
	// 	var image types.Image
	// 	err = rows.Scan(&image.ID,
	// 		&image.Text,
	// 		&image.Count,
	// 		&image.CreatedOn,
	// 		&image.UpdatedOn,
	// 		&image.CreatedBy)
	// 	if err != nil {
	// 		log.Printf("Failed to scan row:\n%s\n", err.Error())
	// 		return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
	// 	}
	// 	images = append(images, image)
	// 	i = i + 1
	// }

	// err = rows.Err()
	// if err != nil {
	// 	log.Printf("Failed after row scan:\n%s\n", err.Error())
	// 	return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
	// }

	// return c.Status(fiber.StatusOK).JSON(images)
	response := fiber.Map{
		"txid": txid.String(),
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

func GetImage(c *fiber.Ctx) error {
	txid := uuid.New()
	log.Printf("%s | %s\n", util.GetFunctionName(GetImage), txid.String())
	image_id := c.Params("id")
	database, err := db.GetInstance()
	if err != nil {
		log.Printf("Failed to connect to DB\n%s\n", err.Error())
		err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
		return c.Status(fiber.StatusInternalServerError).SendString(err_string)
	}
	query_string := `
		SELECT *
		FROM images
		WHERE id = ?
	`
	row := database.QueryRow(query_string, image_id)
	var image types.Image
	err = row.Scan(&image.ID)
	if err != nil {
		log.Printf("Database Error:\n%s\n", err.Error())
		err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
		return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
	}

	return c.Status(fiber.StatusOK).JSON(image)
}

func FetchImage(c *fiber.Ctx) error {
	txid := uuid.New()
	log.Printf("%s | %s\n", util.GetFunctionName(FetchImage), txid.String())
	// if security.ValidateJWT(c) != nil {
	// 	return c.Status(fiber.StatusUnauthorized).SendString(fmt.Sprintf("Unauthorized: %s\n", txid.String()))
	// }
	var movie_data types.Image
	err := c.BodyParser(&movie_data)
	if err != nil {
		log.Printf("Failed to parse movie data\n%s\n", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to parse movie data: %s\n", txid.String()))
	}

	// TODO fix this to allow variable searches
	movie_name := strings.Replace(movie_data.MovieName, " ", "%20", -1)
	url := fmt.Sprintf("https://www.imdb.com/find/?q=%s&ref_=nv_sr_sm", movie_name)
	log.Printf("URL: %s\n", url)
	// TODO do this properly with fiber
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36")
	response, err := client.Do(req)
	// response, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch imdb body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to fetch imdb body: %s\n", txid.String()))
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to read body: %s\n", txid.String()))
	}
	response.Body.Close()

	title_url, err := findMovieURL(string(body), movie_data.MovieName)
	if err != nil {
		log.Printf("Failed to find image\n%s\n", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to find image: %s\n", txid.String()))
	}

	// Go to title page
	title_page_url := fmt.Sprintf("https://imdb.com%s", title_url)
	log.Printf("Title Page Url: %s\n", title_page_url)
	req, _ = http.NewRequest("GET", title_page_url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36")
	client = &http.Client{}
	response, err = client.Do(req)
	// response, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch imdb body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to fetch imdb body: %s\n", txid.String()))
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to read body: %s\n", txid.String()))
	}
	response.Body.Close()

	// left, right, found
	left, _, _ := strings.Cut(title_url, "?")
	image_page_url, err := findImagePosterUrl(string(body), fmt.Sprintf("%smediaviewer", left))
	if err != nil {
		log.Printf("Failed to find image\n%s\n", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to find image: %s\n", txid.String()))
	}

	// Go to image page
	image_page_url = fmt.Sprintf("https://imdb.com%s", image_page_url)
	log.Printf("Image page Url: %s\n", image_page_url)
	req, _ = http.NewRequest("GET", image_page_url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36")
	client = &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch imdb body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to fetch imdb body: %s\n", txid.String()))
	}
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read body\n%s\n", err.Error())
		// TODO this isn't a bad request, but vs code is being rude
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to read body: %s\n", txid.String()))
	}
	response.Body.Close()
	// Remove <noscript> tags as they break parsing of img tags in golang.org/x/net/html
	hdata := strings.Replace(string(body), "<noscript>", "", -1)
	hdata = strings.Replace(hdata, "<noscript data-n-css=\"\">", "", -1)
	hdata = strings.Replace(hdata, "</noscript>", "", -1)

	image_url, err := findImageUrl(string(hdata), "https://m.media-amazon.com/images")
	if err != nil {
		log.Printf("Failed to find image\n%s\n", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Failed to find image: %s\n", txid.String()))
	}
	log.Printf("Image Url: %s\n", image_url)

	// TODO save image
	// url := ""
	// // don't worry about errors
	// response, e := http.Get(url)
	// if e != nil {
	// 	log.Fatal(e)
	// }
	// defer response.Body.Close()

	// //open a file for writing
	// file, err := os.Create("./asdf.jpg")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer file.Close()

	// // Use io.Copy to just dump the response body to the file. This supports huge files
	// _, err = io.Copy(file, response.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Success!")

	// Insert image into DB
	// err_string := fmt.Sprintf("Database Error: %s\n", txid.String())
	// database, err := db.GetInstance()
	// if err != nil {
	// 	log.Printf("Failed to connect to DB\n%s\n", err.Error())
	// 	return c.Status(fiber.StatusInternalServerError).SendString(err_string)
	// }
	// query := `
	// 	INSERT INTO trackers
	// 	(
	// 		tracker_id
	// 		, tracker_text
	// 		, tracker_created_on
	// 		, tracker_updated_on
	// 		, person_id
	// 	)
	// 	SELECT  UUID_TO_BIN(UUID())
	// 			, ?
	// 			, CURDATE()
	// 			, CURDATE()
	// 			, person_id
	// 	FROM people
	// 	WHERE person_username = LOWER(?);
	// `
	// result, err := database.Exec(query, tracker.Text, tracker.CreatedBy)
	// if err != nil {
	// 	log.Printf("Failed to insert record into trackers:\n%s\n", err.Error())
	// 	return c.Status(fiber.StatusServiceUnavailable).SendString(err.Error())
	// }
	// id, err := result.LastInsertId()
	// if err != nil {
	// 	log.Printf("Failed retrieve inserted id\n%s\n", err.Error())
	// 	return c.Status(fiber.StatusServiceUnavailable).SendString(err_string)
	// }

	json := &fiber.Map{
		"id":         txid,
		"movie_name": movie_data.MovieName,
	}

	return c.Status(fiber.StatusOK).JSON(json)
}

func findImageUrl(body string, search_text string) (string, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to parse html body: %s", err.Error()))
	}

	matcher := func(node *html.Node) (keep bool, exit bool) {
		if node.Type == html.ElementNode && node.Data == atom.Img.String() {
			for _, attr := range node.Attr {
				if attr.Key == "src" && strings.Contains(attr.Val, search_text) {
					keep = true
				}
			}
			// I could exit early, but for now let's not.
			// exit = true
		}
		return
	}

	nodes := traverseNode(doc, matcher)
	// [drd] leaving this here in case I want to look at the page
	// for i, node := range nodes {
	// 	fmt.Println(i, renderNode(node))
	// }
	if len(nodes) > 0 {
		for _, attr := range nodes[0].Attr {
			if attr.Key == "src" {
				return attr.Val, nil
			}
		}
	}
	return "", errors.New("Image URL Not Found")
}

func findImagePosterUrl(body string, search_text string) (string, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to parse html body: %s", err.Error()))
	}

	matcher := func(node *html.Node) (keep bool, exit bool) {
		if node.Type == html.ElementNode && node.Data == atom.A.String() {
			for _, attr := range node.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, search_text) {
					keep = true
				}
			}
			// I could exit early, but for now let's not.
			// exit = true
		}
		return
	}

	nodes := traverseNode(doc, matcher)
	// [drd] leaving this here in case I want to look at the page
	// for i, node := range nodes {
	// 	fmt.Println(i, renderNode(node))
	// }
	if len(nodes) > 0 {
		for _, attr := range nodes[0].Attr {
			if attr.Key == "href" {
				return attr.Val, nil
			}
		}
	}
	return "", errors.New("Image Poster URL Not Found")
}

func findMovieURL(body string, search_text string) (string, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to parse html body: %s", err.Error()))
	}

	matcher := func(node *html.Node) (keep bool, exit bool) {
		if node.Type == html.ElementNode &&
			node.Data == atom.A.String() &&
			node.FirstChild != nil &&
			node.FirstChild.Data == search_text {
			keep = true
			// I could exit early, but for now let's not.
			// exit = true
		}
		return
	}

	nodes := traverseNode(doc, matcher)
	// [drd] leaving this here in case I want to look at the page
	// for i, node := range nodes {
	// 	fmt.Println(i, renderNode(node))
	// }
	if len(nodes) > 0 {
		for _, attr := range nodes[0].Attr {
			if attr.Key == "href" {
				return attr.Val, nil
			}
		}
	}
	return "", errors.New("Movie Title URL Not Found")
}

// traverse the nodes collecting the nodes that match the given function
func traverseNode(doc *html.Node, matcher func(node *html.Node) (bool, bool)) (nodes []*html.Node) {
	var keep, exit bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		keep, exit = matcher(n)
		if keep {
			nodes = append(nodes, n)
		}
		if exit {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nodes
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}
