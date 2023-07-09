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

type handleLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type HandleRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type handleChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (app *Application) initAPI() {
	e := echo.New()
	e.Use(middleware.CORS())

	api := e.Group("api")
	if app.UseJWT {
		api.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey:    jwtsecret,
			SigningMethod: "HS256",
			TokenLookup:   "header:Token",
		}))
	}
	api.POST("/mail", app.handleMails)
	api.POST("/keyword", app.handleKeyword)
	api.POST("/keywordmail", app.handleMailsFromKeyword)

	auth := e.Group("auth")
	auth.POST("/register", app.handleRegister)
	auth.POST("/login", app.handleLogin)

	user := e.Group("user")
	if app.UseJWT {
		user.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey:    jwtsecret,
			SigningMethod: "HS256",
			TokenLookup:   "header:Token",
		}))
	}
	user.POST("/changepassword", app.handleChangePassword)

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

	fmt.Println("urls:", request.Urls)
	results, err := app.getMailsFromUrls(request.Urls)
	if err != nil {
		return err
	}

	fmt.Println("results:", results)

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
		return internalError(fmt.Errorf("err generating jwt"))
	}

	return c.JSON(http.StatusOK, token)
}

func (app *Application) handleChangePassword(c echo.Context) error {
	request := new(handleChangePasswordRequest)

	if err := bind(c, request); err != nil {
		return err
	}

	user, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	if err := validateHandleChangePassword(request); err != nil {
		return err
	}

	if err := user.IsPassword(request.OldPassword); err != nil {
		return badRequest(err)
	}

	if err := user.SetPassword(request.NewPassword); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Update(); err != nil {
		return internalError(err)
	}

	return c.JSON(http.StatusOK, "good")
}
