package protocol

import (
	"github.com/google/uuid"
)

func GenerateId() int32 {
	return int32(uuid.New().ID())
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
