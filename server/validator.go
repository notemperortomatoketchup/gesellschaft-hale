package main

import (
	"fmt"
	"reflect"
	"regexp"

	passwordvalidator "github.com/wagslane/go-password-validator"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func assertType(fieldname string, fieldvalue any, expected string) error {
	got := reflect.TypeOf(fieldvalue).String()

	if got != expected {
		return badRequest(
			fmt.Errorf(
				"%s: %s (expected: %s, got: %s)",
				fieldname,
				protocol.ErrNotValidType.Error(),
				expected,
				got,
			),
		)
	}

	return nil
}

func assertNotEmpty(fieldname string, fieldvalue any) error {
	typeStr := reflect.TypeOf(fieldvalue).String()
	// we type insert so that we can use == nil, len..

	switch typeStr {
	case "string":
		if fieldvalue == "" {
			return badRequest(fmt.Errorf("%s: %s", fieldname, protocol.ErrEmpty.Error()))
		}
	case "*uint":
		if fieldvalue.(*uint) == nil {
			return badRequest(fmt.Errorf("%s: %s", fieldname, protocol.ErrEmpty.Error()))
		}
	}

	return nil
}

func assertNotEmptySlice[T comparable](fieldname string, fieldvalue []T) error {
	if len(fieldvalue) == 0 {
		return badRequest(fmt.Errorf("%s: %s", fieldname, protocol.ErrEmpty.Error()))
	}
	return nil
}

func assertRangeInt(fieldname string, fieldvalue int, min, max int) error {
	if fieldvalue >= min && fieldvalue <= max {
		return nil
	}
	return badRequest(
		fmt.Errorf("%s: %s (%d-%d)", fieldname, protocol.ErrNotInRange.Error(), min, max),
	)
}

func assertRangeStr(fieldname string, fieldvalue string, min, max int) error {
	if len(fieldvalue) >= min && len(fieldvalue) <= max {
		return nil
	}

	return badRequest(
		fmt.Errorf("%s: %s (%d-%d)", fieldname, protocol.ErrNotInRange.Error(), min, max),
	)
}

func assertNoDuplicate[T comparable](fieldname string, fieldvalue []T) error {
	seen := make(map[interface{}]bool)
	for _, v := range fieldvalue {
		if _, ok := seen[v]; ok {
			return badRequest(
				fmt.Errorf("%s: %s (%s)", fieldname, protocol.ErrNoDuplicate.Error(), fmt.Sprint(v)),
			)
		}
		seen[v] = true
	}

	return nil
}

func assertValidPassword(fieldname, password string) error {
	if err := passwordvalidator.Validate(password, 60); err != nil {
		return badRequest(fmt.Errorf("%s: %s", fieldname, err.Error()))
	}

	return nil
}

func assertValidUrls(fieldname string, urls ...string) error {
	re := regexp.MustCompile(`\bhttps?:\/\/(?:www\.)?[a-zA-Z0-9-]+(?:\.[a-zA-Z]{2,})(?:\/\S*)?\b`)
	var invalids []string
	for _, u := range urls {
		if ok := re.MatchString(u); !ok {
			invalids = protocol.AppendUnique(invalids, u)
		}
	}

	if len(invalids) != 0 {
		return badRequest(
			fmt.Errorf("%s: %s (%v)", fieldname, protocol.ErrNotValidUrls.Error(), invalids),
		)
	}
	return nil
}

func validateHandleMails(r *HandleGetMailsRequest) error {
	if err := assertNotEmptySlice("urls", r.Urls); err != nil {
		return err
	}

	if err := assertValidUrls("urls", r.Urls...); err != nil {
		return err
	}

	return nil
}

func validateHandleKeyword(r *HandleKeywordRequest) error {
	if err := assertNotEmpty("keyword", r.Keyword); err != nil {
		return err
	}

	if err := assertType("keyword", r.Keyword, "string"); err != nil {
		return err
	}

	if err := assertType("pages", r.Pages, "int"); err != nil {
		return err
	}

	if err := assertRangeInt("pages", r.Pages, 1, 30); err != nil {
		return err
	}

	return nil
}

func validateHandleRegister(r *HandleRegisterRequest) error {
	if err := assertType("username", r.Username, "string"); err != nil {
		return err
	}
	if err := assertType("password", r.Password, "string"); err != nil {
		return err
	}

	if err := assertRangeStr("username", r.Username, 1, 32); err != nil {
		return err
	}

	if err := assertValidPassword("password", r.Password); err != nil {
		return err
	}

	return nil
}

func validateHandleLogin(r *handleLoginRequest) error {
	if err := assertType("username", r.Username, "string"); err != nil {
		return err
	}

	if err := assertRangeStr("username", r.Username, 1, 32); err != nil {
		return err
	}
	return nil
}

func validateHandleChangePassword(r *handleChangePasswordRequest) error {
	if err := assertType("old_password", r.OldPassword, "string"); err != nil {
		return err
	}

	if err := assertType("new_password", r.NewPassword, "string"); err != nil {
		return err
	}

	if err := assertNotEmpty("old_password", r.OldPassword); err != nil {
		return err
	}

	if err := assertNotEmpty("new_password", r.OldPassword); err != nil {
		return err
	}

	if err := assertValidPassword("new_password", r.NewPassword); err != nil {
		return err
	}

	return nil
}

func validateHandleCreateCampaign(r *handleCreateCampaignRequest) error {
	if err := assertType("title", r.Title, "string"); err != nil {
		return err
	}

	if err := assertNotEmpty("title", r.Title); err != nil {
		return err
	}

	if err := assertRangeStr("title", r.Title, 3, 128); err != nil {
		return err
	}

	return nil
}

func validateHandleGetListsCampaign(r *CampaignOpts) error {
	if err := assertType("id", r.ID, "int"); err != nil {
		return err
	}

	return nil
}

// generic handler that only needs an id to work with, was getting many
func validateHandleID(r *handleIDRequest) error {
	if err := assertType("id", r.ID, "*uint"); err != nil {
		return err
	}

	if err := assertNotEmpty("test", r.ID); err != nil {
		return err
	}

	return nil
}
