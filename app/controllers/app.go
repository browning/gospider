package controllers

import "github.com/revel/revel"
import "fmt"

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) StartSpider(url string) revel.Result {
	fmt.Printf("%s\n", url)
	return c.Render()
}
