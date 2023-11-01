package crawler

import (
	"errors"
	"fmt"
	"image_service/types"
	"log"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/html"
)

func Request(config types.Config, uri string, method string) (*fasthttp.Response, error) {
	log.Printf("Making request METHOD: %s | URI: %s\n", method, uri)
	request := fasthttp.AcquireRequest()
	request.SetRequestURI(uri)
	request.Header.Add("User-Agent", config.App.Client.UserAgent)
	request.Header.SetMethodBytes([]byte(method))
	response := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	err := client.Do(request, response)
	if err != nil {
		log.Printf("Failed to make request to: %s\n%s\n", uri, err.Error())
		return nil, errors.New(strconv.Itoa(response.StatusCode()))
	}
	return response, nil
}

type MatcherParams struct {
	Haystack string
	Needle   string
	AttrKey  string
	Atom     string
	NodeType html.NodeType
}

type MatcherFunc func(node *html.Node, params MatcherParams) (keep bool, exit bool)

func ImdbFindImageUrl(node *html.Node, params MatcherParams) (keep bool, exit bool) {
	if node.Type == params.NodeType && node.Data == params.Atom {
		for _, attr := range node.Attr {
			if attr.Key == params.AttrKey && strings.Contains(attr.Val, params.Needle) {
				keep = true
			}
		}
	}
	return
}

func ImdbFindImagePosterUrl(node *html.Node, params MatcherParams) (keep bool, exit bool) {
	if node.Type == params.NodeType && node.Data == params.Atom {
		for _, attr := range node.Attr {
			if attr.Key == params.AttrKey && strings.Contains(attr.Val, params.Needle) {
				keep = true
			}
		}
	}
	return
}

func ImdbFindMovieUrl(node *html.Node, params MatcherParams) (keep bool, exit bool) {
	if node.Type == params.NodeType &&
		node.Data == params.Atom &&
		node.FirstChild != nil &&
		node.FirstChild.Data == params.Needle {
		keep = true
	}
	return
}

func Find(params MatcherParams, matcher MatcherFunc) (string, error) {
	doc, err := html.Parse(strings.NewReader(params.Haystack))
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to parse html body: %s", err.Error()))
	}
	nodes := traverse(doc, matcher, params)
	if len(nodes) > 0 {
		for _, attr := range nodes[0].Attr {
			if attr.Key == params.AttrKey {
				return attr.Val, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("Failed to find needle: %s", params.Needle))
}

func traverse(doc *html.Node, matcher MatcherFunc, params MatcherParams) (nodes []*html.Node) {
	var keep, exit bool
	var sifter func(*html.Node)
	sifter = func(node *html.Node) {
		keep, exit = matcher(node, params)
		if keep {
			nodes = append(nodes, node)
		}
		if exit {
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			sifter(child)
		}
	}
	sifter(doc)
	return nodes
}
