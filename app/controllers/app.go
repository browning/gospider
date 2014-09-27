package controllers

import "github.com/revel/revel"
import "fmt"
import "net/http"
import urlhelpers "net/url"
import "code.google.com/p/go.net/html"
import "sync"
import "io/ioutil"
import "bytes"

type URLData struct {
    URL        string
    Body       string
    StatusCode int
    ContentLength int
}

var messages = make(chan string)
var basehostname string
var scrape_results = make(map[string]URLData)
var wg sync.WaitGroup

func parse_html(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, element := range n.Attr {
			if element.Key == "href" {
				wg.Add(1)
				go spider(element.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parse_html(c)
	}
}

func spider(url string) {    
    _, present := scrape_results[url]
    fmt.Printf("\npresent %r\n", present)
    if (present) {
    	wg.Done()
    	return
    } else {
    	scrape_results[url] = URLData{}
    }

    messages <- "spidering " + url + "<br />"

    url_host, _ := urlhelpers.Parse(url)
    if (url_host.Host != basehostname) {
    	wg.Done()
    	return
    }

    response, err := http.Get(url)
    if err != nil {
        fmt.Printf("%s", err)
    } else {
    	data,_ := ioutil.ReadAll(response.Body)
    	doc, _ := html.Parse(bytes.NewReader(data))
    	//scrape_results[url] = doc.Data
    	
    	fmt.Printf("DATA: %s\n", data)
    	scrape_results[url] = URLData{url, doc.Data, response.StatusCode, len(data)}
    	parse_html(doc)
    }
    wg.Done()
}

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) StartSpider(url string) revel.Result {
	parsed_url, _ := urlhelpers.Parse(url)
	basehostname = parsed_url.Host
	wg.Add(1)
	go spider(url)
	return c.Render()
}

func (c App) SpiderStatus() revel.Result {
	status := <- messages
	return c.RenderText(status)
}

func (c App) SpiderDone() revel.Result {
	wg.Wait()
	return c.RenderText("spidering is complete")
}

type StandardResult struct {
    URL        string
    StatusCode int
    ContentLength int
}

func (c App) View() revel.Result {
	container := []StandardResult{}
	for k, v := range scrape_results { 
		res := StandardResult{k, v.StatusCode, v.ContentLength}
		container = append(container, res)
	}
	return c.Render(container)
}

func (c App) ViewInternal() revel.Result {
	container := []StandardResult{}
	for k, v := range scrape_results { 
		url_host, _ := urlhelpers.Parse(k)
    	if (url_host.Host == basehostname) {
    		res := StandardResult{k, v.StatusCode, v.ContentLength}
		    container = append(container, res)
     	}		
	}
	return c.Render(container)
}


