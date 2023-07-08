package main

import (
	"fmt"
	"log"
	"net/http"

	echojwt "github.com/labstack/echo-jwt/v4"
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
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:    jwtsecret,
		SigningMethod: "HS256",
		TokenLookup:   "header:Token",
	}))
	api.POST("/mail", app.handleMails)
	api.POST("/keyword", app.handleKeyword)
	api.POST("/keywordmail", app.handleMailsFromKeyword)

	auth := e.Group("auth")
	auth.POST("/register", app.handleRegister)
	auth.POST("/login", app.handleLogin)

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

type HandleRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (app *Application) handleRegister(c echo.Context) error {
	request := new(HandleRegisterRequest)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleRegister(request); err != nil {
		return err
	}

	user := new(User)
	user.SetUsername(request.Username)

	if err := user.SetPassword(request.Password); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Insert(); err != nil {
		return internalError(err)
	}

	return c.JSON(http.StatusOK, "good")
}

type handleLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (app *Application) handleLogin(c echo.Context) error {
	request := new(handleLoginRequest)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleLogin(request); err != nil {
		return err
	}

	// pull user from db
	user, err := getUser(request.Username)
	if err != nil {
		return internalError(err)
	}

	if err := user.IsPassword(request.Password); err != nil {
		return badRequest(fmt.Errorf("%s", protocol.ErrInvalidCredentials))
	}

	// got user right, need to generate token jwt
	token, err := user.generateJWT()
	if err != nil {
		fmt.Println("err:", err)
		return internalError(fmt.Errorf("err generating jwt"))
	}

	return c.JSON(http.StatusOK, token)
}
