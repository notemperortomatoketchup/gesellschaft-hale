package main

import (
	"fmt"
	"reflect"
	"regexp"

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

func assertNotEmptySlice[T comparable](fieldname string, fieldvalue []T) error {
	if len(fieldvalue) == 0 {
		return badRequest(fmt.Errorf("%s: %s", fieldname, protocol.ErrEmpty.Error()))
	}
	return nil
}

func assertNotEmptyString(fieldname string, fieldvalue string) error {
	if fieldvalue == "" {
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
	if err := assertNotEmptyString("keyword", r.Keyword); err != nil {
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
