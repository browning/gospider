package controllers

import "github.com/revel/revel"
import "fmt"

var messages = make(chan string)

func spider(url string) {
    fmt.Printf("running spider func in a goroutine")
    messages <- "doing go stuff"
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
