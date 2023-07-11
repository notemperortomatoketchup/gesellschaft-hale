package protocol

import (
	"github.com/google/uuid"
)

func GenerateId() uint32 {
	return uuid.New().ID()
}

func AppendUnique[T comparable](s []T, el T) []T {
	if isIn := IsExists(s, el); !isIn {
		s = append(s, el)
		return s
	} else {
		return s
	}
}

func IsExists[T comparable](s []T, el T) bool {
	for _, a := range s {
		if a == el {
			return true
		}
	}
	return false
}

func RemoveStrFromSlice(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
