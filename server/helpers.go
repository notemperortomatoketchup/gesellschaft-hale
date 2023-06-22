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

func (app *Application) getMailsFromUrls(urls []string) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()
	client, ok := app.GetAvailableClient(int32(len(urls)))
	if !ok {
		return nil, internalError(protocol.ErrNoBrowserAvailable)
	}

	app.RequestCh <- &protocol.RequestJobWrapper{
		RequestId: reqId,
		ClientId:  client.id,
		Type:      protocol.MessageType_GET_MAILS,
		Urls:      urls,
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return nil, internalError(err)
	}

	return r.GetResult(), nil
}

func (app *Application) getKeywordResults(kw string, pages int) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()
	client, ok := app.GetAvailableClient(0)
	if !ok {
		return nil, internalError(protocol.ErrNoBrowserAvailable)
	}

	app.RequestCh <- &protocol.RequestJobWrapper{
		RequestId:  reqId,
		Type:       protocol.MessageType_GET_KEYWORD,
		ClientId:   client.id,
		Keyword:    kw,
		PagesCount: int32(pages),
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return nil, internalError(err)
	}

	return r.GetResult(), nil
}
