package controllers

import "github.com/revel/revel"
import "fmt"
import "net/http"
import urlhelpers "net/url"
import "code.google.com/p/go.net/html"
import "sync"
import "io/ioutil"
import "bytes"
import "strings"
import "github.com/temoto/robotstxt.go"

type URLData struct {
    URL        string
    Body       string
    StatusCode int
    ContentLength int
    MetaDesc string
    MetaRobots string
}

var messages = make(chan string)
var basehostname string
var scrape_results = make(map[string]URLData)
var wg sync.WaitGroup
var robots *robotstxt.RobotsData

func parse_html(url string, n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, element := range n.Attr {
			if element.Key == "href" {
				wg.Add(1)
				go spider(element.Val)
			}
		}
	}

	if n.Type == html.ElementNode && n.Data == "meta" {
		for _, element := range n.Attr {
			if element.Key == "name" && element.Val == "description" {
				for _, e2 := range n.Attr {
					if e2.Key == "content" {
						x := scrape_results[url]
						x.MetaDesc = e2.Val
						scrape_results[url] = x
					}
				}
			}
			if element.Key == "name" && element.Val == "robots" {
				for _, e2 := range n.Attr {
					if e2.Key == "content" {
						x := scrape_results[url]
						x.MetaDesc = e2.Val
						scrape_results[url] = x
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parse_html(url, c)
	}
}

func spider(url string) { 
	if ! strings.HasPrefix(url, "http") {
    	if ! strings.HasPrefix(url, "/") {
    		url = "/" + url
    	}
    	url = "http://" + basehostname + url
    }

    _, present := scrape_results[url]
    if (present) {
    	wg.Done()
    	return
    } else {
    	scrape_results[url] = URLData{}
    }

    messages <- "spidering " + url + "<br />"

    url_host, _ := urlhelpers.Parse(url)
    if ! robots.TestAgent(url_host.Path, "spideryBot") {
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
       	scrape_results[url] = URLData{url, doc.Data, response.StatusCode, len(data), "", ""}

    	url_host, _ := urlhelpers.Parse(url)
	    if (url_host.Host != basehostname) {
	    	wg.Done()
	    	return
	    }

    	parse_html(url, doc)
    }
    wg.Done()
}

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) StartSpider(url string, obey_robots string) revel.Result {
	if obey_robots == "obey_robots" {
		resp, err := http.Get(url + "/robots.txt")
		// parse robots.txt here and store 
		robots, err = robotstxt.FromResponse(resp)
		resp.Body.Close()
		if err != nil {
    		fmt.Print("Error parsing robots.txt:", err.Error())
		}
	}

    if ! strings.HasPrefix(url, "http") {
    	if ! strings.HasPrefix(url, "/") {
    		url = "/" + url
    	}
    	url = "http://" + basehostname + url
    }

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

func (c App) View() revel.Result {
	container := []URLData{}
	for _, v := range scrape_results { 
		res := v
		container = append(container, res)
	}
	return c.Render(container)
}

func (c App) ViewInternal() revel.Result {
	container := []URLData{}
	for k, v := range scrape_results { 
		url_host, _ := urlhelpers.Parse(k)
    	if (url_host.Host == basehostname) {
    		res := v
		    container = append(container, res)
     	}		
	}
	return c.Render(container)
}

func (c App) ViewExternal() revel.Result {
	container := []URLData{}
	for k, v := range scrape_results { 
		url_host, _ := urlhelpers.Parse(k)
    	if (url_host.Host != basehostname) {
    		res := v
		    container = append(container, res)
     	}		
	}
	return c.Render(container)
}


