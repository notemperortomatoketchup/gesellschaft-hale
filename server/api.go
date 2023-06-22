package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
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
	Websites []*protocol.Website `json:"websites"`
}

func (app *Application) initAPI() {
	e := echo.New()
	api := e.Group("api")
	api.POST("/mail", app.handleMails)
	api.POST("/keyword", app.handleKeyword)

	if err := e.StartTLS(":443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatal(err)
	}
}

func (app *Application) handleKeyword(c echo.Context) error {
	request := new(HandleKeywordRequest)
	response := new(HandleKeywordResponse)

	reqId := protocol.GenerateId()
	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
	}

	client, ok := app.GetAvailableClient(0)
	if !ok {
		return internalError(protocol.ErrNoBrowserAvailable)
	}

	app.RequestCh <- &protocol.RequestJobWrapper{
		RequestId:  reqId,
		Type:       protocol.MessageType_GET_KEYWORD,
		ClientId:   client.id,
		Keyword:    request.Keyword,
		PagesCount: int32(request.Pages),
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return internalError(err)
	}

	response.Websites = r.GetResult()

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMails(c echo.Context) error {
	request := new(HandleGetMailsRequest)
	response := new(HandleGetMailsResponse)

	reqId := protocol.GenerateId()
	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleMails(request); err != nil {
		return err
	}

	client, ok := app.GetAvailableClient(int32(len(request.Urls)))
	if !ok {
		return internalError(protocol.ErrNoBrowserAvailable)
	}

	app.RequestCh <- &protocol.RequestJobWrapper{
		RequestId: reqId,
		ClientId:  client.id,
		Type:      protocol.MessageType_GET_MAILS,
		Urls:      request.Urls,
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return internalError(err)
	}
	response.Websites = r.GetResult()

	return c.JSON(http.StatusOK, response)
}
