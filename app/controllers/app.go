package controllers

import "github.com/revel/revel"
import "fmt"
import "net/http"
import "code.google.com/p/go.net/html"

var messages = make(chan string)

func parse_html(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, element := range n.Attr {
			if element.Key == "href" {
				fmt.Printf("LINK: %s\n", element.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parse_html(c)
	}
}

func spider(url string) {
    fmt.Printf("running spider func in a goroutine")
    messages <- "doing go stuff"

    response, err := http.Get(url)
    if err != nil {
        fmt.Printf("%s", err)
    } else {
    	doc, err := html.Parse(response.Body)
    	fmt.Printf("%s", err)
    	parse_html(doc)
    }

}

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) StartSpider(url string) revel.Result {
	fmt.Printf("%s\n", url)
	go spider(url)
	return c.Render()
}

func (c App) SpiderStatus() revel.Result {
	status := <- messages
	return c.RenderText(status)
}
