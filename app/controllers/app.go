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
import "github.com/PuerkitoBio/goquery"

type URLData struct {
    URL        string
    Body       string
    StatusCode int
    ContentLength int
    MetaDesc string
    MetaRobots string
    ContentType string
    NumLinks int
}

type ImageLink struct {
	Src string
	BaseUrl string
	AltText string
	Width string
	Height string
}

type PageLink struct {
	HREF string
	AnchorText string
	Follow bool
}

var NUM_GOROUTINES = 20
var messages = make(chan string, 256)
var to_scrape = make(chan string, 256)
var basehostname string
var scrape_results = make(map[string]URLData)
var image_links = make(map[string][]ImageLink)
var page_links = make(map[string][]PageLink)
var wg sync.WaitGroup
var robots *robotstxt.RobotsData

func parse_html(url string, n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		href := ""
		anchor_text := ""
		if n.FirstChild != nil {
			anchor_text = strings.TrimSpace(n.FirstChild.Data)
	    }
		for _, element := range n.Attr {
			if element.Key == "href" {
				wg.Add(1)
				href = element.Val
				to_scrape <- element.Val
			}
		}

		//fmt.Printf("ADDING A PAGE LINK: %s\n", href)
	    page_links[href] = append(page_links[href], PageLink{url, anchor_text, false})

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

func spider() { 
	fmt.Printf("starting goroutine and waiting")
	for url := range to_scrape {
		fmt.Printf("got url in goroutine %s\n", url)
		if ! strings.HasPrefix(url, "http") {
	    	if ! strings.HasPrefix(url, "/") {
	    		url = "/" + url
	    	}
	    	url = "http://" + basehostname + url
	    }

	    _, present := scrape_results[url]
	    if (present) {
	    	wg.Done()
	    	continue
	    } else {
	    	//fmt.Printf("ENTERING A NEW URLDATA FOR: %s\n", url)
	    	scrape_results[url] = URLData{}
	    }

	    messages <- "spidering " + url + "<br />"

	    url_host, _ := urlhelpers.Parse(url)
	    if ! robots.TestAgent(url_host.Path, "spideryBot") {
	    	wg.Done()
	    	continue
	    }
	    response, err := http.Get(url)
	    if err != nil {
	    	scrape_results[url] = URLData{url, "", -1, 0, "", "", "", 0}
	        fmt.Printf("ERROR: %s\n", err)
	    } else {
	    	data,_ := ioutil.ReadAll(response.Body)
	    	doc, _ := html.Parse(bytes.NewReader(data))
	    	//scrape_results[url] = doc.Data
	       	scrape_results[url] = URLData{url, doc.Data, response.StatusCode, len(data), "", "", response.Header.Get("Content-Type"), 0}

	    	url_host, _ := urlhelpers.Parse(url)
		    if (url_host.Host != basehostname) {
		    	wg.Done()
		    	continue
		    }

		    parse_images(url, data)
	    	parse_html(url, doc)
	    }
	    wg.Done()
	}
	fmt.Printf("spider is closing...")
}

func parse_images(url string, data []byte) {
	reader := bytes.NewReader(data)
	doc, _ := goquery.NewDocumentFromReader(reader)
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
	    src, _ := s.Attr("src")
	    alt, _ := s.Attr("alt")
	    height, _ := s.Attr("height")
	    width, _ := s.Attr("width")
	    wg.Add(1)
	    to_scrape <- src

	    fmt.Printf("IMAGE %d: %s\n", i, src)
	    image_links[src] = append(image_links[src], ImageLink{src, url, alt, height, width})
  	})
}

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) StartSpider(url string, obey_robots string) revel.Result {
	// Start up  goroutines as spider workers
	for i := 0; i < NUM_GOROUTINES; i++ {
        go spider()
    }

	scrape_results = make(map[string]URLData)
	image_links = make(map[string][]ImageLink)
	page_links = make(map[string][]PageLink)

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
	to_scrape <- url
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
	for k, v := range scrape_results { 
		fmt.Printf("scrape_results: v: %s: %s\n", k, v)
		res := v
		res.NumLinks = len(page_links[res.URL])
		container = append(container, res)
	}
	PageName := "all"
	return c.Render(container, PageName)
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
	PageName := "internal"
	c.RenderArgs["PageName"] = PageName
	c.RenderArgs["container"] = container
	return c.RenderTemplate("App/view.html")
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
	c.RenderArgs["PageName"] = "external"
	c.RenderArgs["container"] = container
	return c.RenderTemplate("App/view.html")
}

type ImgData struct {
    URL        string
    StatusCode int
    ContentLength int
    NumLinks int
    NumMissingAlts int
}

func (c App) ViewImages() revel.Result {
	container := []ImgData{}
	for _, v := range scrape_results { 
    	if (strings.Contains(v.ContentType, "image")) {
    		res := v
    		alt_count := 0
    		for _, item := range image_links[res.URL] {
    			if len(item.AltText) <= 0 {
    				alt_count = alt_count + 1
    			}
    		}
		    container = append(container, ImgData{res.URL, res.StatusCode, res.ContentLength, len(image_links[res.URL]), alt_count})
     	}		
	}
	return c.Render(container)
}

func (c App) ImageDetails(url string) revel.Result {
	container := image_links[url]
	return c.Render(container)
}

func (c App) PageDetails(url string) revel.Result {
	container := page_links[url]
	return c.Render(container)
}

func (c App) Summary() revel.Result {
	num_pages := len(scrape_results)
	num_404s := 0
	num_200s := 0
	num_301s := 0
	num_external := 0
	num_internal := 0
	for k, v := range scrape_results {
		if v.StatusCode == 404 {
			num_404s = num_404s + 1
		} else if v.StatusCode == 200 {
			num_200s = num_200s + 1
		} else if v.StatusCode == 301 {
			num_301s = num_301s + 1
		}
		url_host, _ := urlhelpers.Parse(k)
    	if (url_host.Host == basehostname) {
    		num_internal = num_internal +1
     	} else {
     		num_external = num_external + 1
     	}
	}
	return c.Render(num_pages, num_404s, num_external, num_internal, num_200s, num_301s)
}


