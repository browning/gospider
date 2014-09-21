package controllers

import "github.com/revel/revel"
import "fmt"
import "net/http"
import urlhelpers "net/url"
import "code.google.com/p/go.net/html"

var messages = make(chan string)
var basehostname string
var scrape_results = make(map[string]string)

func parse_html(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, element := range n.Attr {
			if element.Key == "href" {
				fmt.Printf("LINK: %s\n", element.Val)

				go spider(element.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parse_html(c)
	}
}

func spider(url string) {
    fmt.Printf("running spider func in a goroutine")
    

    _, present := scrape_results[url]
    fmt.Printf("\npresent %r\n", present)
    if (present) {
    	return
    } else {
    	scrape_results[url] = ""
    }

    messages <- "spidering " + url + "<br />"

    url_host, _ := urlhelpers.Parse(url)
    if (url_host.Host != basehostname) {
    	return
    }

    response, err := http.Get(url)
    if err != nil {
        fmt.Printf("%s", err)
    } else {
    	doc, _ := html.Parse(response.Body)
    	//scrape_results[url] = doc.Data
    	fmt.Printf("%s", doc)
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
	parsed_url, _ := urlhelpers.Parse(url)
	basehostname = parsed_url.Host
	go spider(url)
	return c.Render()
}

func (c App) SpiderStatus() revel.Result {
	status := <- messages
	return c.RenderText(status)
}
