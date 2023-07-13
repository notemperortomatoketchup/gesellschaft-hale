package main

import (
	"regexp"

	"gopkg.in/go-playground/validator.v9"
)

var validate *validator.Validate
var urlRegex = `\bhttps?:\/\/(?:www\.)?[a-zA-Z0-9-]+(?:\.[a-zA-Z]{2,})(?:\/\S*)?\b`

type ErrorResponse struct {
	FailedField string `json:"field"`
	Hint        string `json:"hint"`
}

func (app *Application) StartValidator() {
	validate = validator.New()
	validate.RegisterValidation("urls", validateUrl)
}

func validateUrl(fl validator.FieldLevel) bool {
	urls := fl.Field().Interface().([]string)
	regex := regexp.MustCompile(urlRegex)

	for _, u := range urls {
		if ok := regex.MatchString(u); !ok {
			return false
		}
	}
	return true
}

func ValidateStruct(v any) []*ErrorResponse {
	var errors []*ErrorResponse

	err := validate.Struct(v)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			msg := makeValidatonErrorMsg(err)
			errors = append(errors, &msg)
		}
	}

	return errors
}

func makeValidatonErrorMsg(err validator.FieldError) ErrorResponse {
	var response ErrorResponse

	response.FailedField = err.Field()

	switch err.Tag() {
	case "urls":
		response.Hint = "one of the urls has wrong format"
	case "oneof":
		response.Hint = "must be within the following values"
	default:
		response.Hint = err.Tag()
	}

	if err.Param() != "" {
		response.Hint += " -> " + err.Param()
	}

	return response
}
