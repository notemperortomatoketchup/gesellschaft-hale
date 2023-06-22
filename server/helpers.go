package main

import (
	"errors"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func (app *Application) awaitResults(id int32) (*protocol.ResponseJobWrapper, error) {
	var result *protocol.ResponseJobWrapper

	// as long as result is nil we shall range.
	for result == nil {
		time.Sleep(time.Second)
		app.Results.Range(func(key, value any) bool {
			if key.(int32) == id {
				result = value.(*protocol.ResponseJobWrapper)
				app.Results.Delete(key)
				return false
			}
			return true
		})
	}

	if result.Type == protocol.MessageType_ERROR {
		return nil, errors.New(result.GetError())
	}

	return result, nil
}
