package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type HandleGetMailsRequest struct {
	Urls []string `json:"urls"`
}

type HandleGetMailsResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type HandleKeywordRequest struct {
	Keyword string `json:"keyword"`
	Pages   int    `json:"pages"`
}

type HandleKeywordResponse struct {
	Websites []*protocol.Website `json:"data"`
}

func (app *Application) initAPI() {
	e := echo.New()
	e.Use(middleware.CORS())
	api := e.Group("api")

	api.POST("/mail", app.handleMails)
	api.POST("/keyword", app.handleKeyword)
	api.POST("/keywordmail", app.handleMailsFromKeyword)

	if err := e.StartTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatal(err)
	}
}

func (app *Application) handleKeyword(c echo.Context) error {
	request := new(HandleKeywordRequest)
	response := new(HandleKeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
	}

	results, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMails(c echo.Context) error {
	request := new(HandleGetMailsRequest)
	response := new(HandleGetMailsResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleMails(request); err != nil {
		return err
	}

	results, err := app.getMailsFromUrls(request.Urls)
	if err != nil {
		return err
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMailsFromKeyword(c echo.Context) error {
	request := new(HandleKeywordRequest)
	response := new(HandleKeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
	}

	scraped, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	results, err := app.getMailsFromWebsites(scraped)
	if err != nil {
		return err
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}
