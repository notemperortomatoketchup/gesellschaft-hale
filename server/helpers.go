package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func (app *Application) awaitResults(id uint32) (*protocol.ResponseJobWrapper, error) {
	var result *protocol.ResponseJobWrapper

	// as long as result is nil we shall range.
	for result == nil {
		time.Sleep(time.Second)
		app.Results.Range(func(key, value any) bool {
			if key.(uint32) == id {
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

func (app *Application) getMailsFromUrls(urls []string, method int) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()

	var results []*protocol.Website
	var urlsToScrape []string

	if method == METHOD_FAST {
		ctx, cancel := context.WithCancel(context.Background())
		toScrapeCh := make(chan string, 0)
		scrapedCh := make(chan *protocol.Website, 0)

		var wg sync.WaitGroup
		go func() {
			for _, u := range urls {
				wg.Add(1)
				go func(j string) {
					defer wg.Done()
					w, err := getWebsite(j)
					if err == protocol.ErrWebsiteNotFound {
						toScrapeCh <- j
					} else {
						scrapedCh <- w
					}
				}(u)
			}
			wg.Wait()
			// avoid unnecessary changes below.
			urls = urlsToScrape
			defer cancel()
		}()

	mainloop:
		for {
			select {
			case <-ctx.Done():
				break mainloop
			case website := <-scrapedCh:
				results = append(results, website)
			case url := <-toScrapeCh:
				urlsToScrape = append(urlsToScrape, url)
			}
		}
	}

	// if urls is not 0, then even if fast method we still have some to scrape.
	// but if we are in method slow, we just go, no matter what, as above step didn't ran.
	// we merge urls above so taht we don't have to change the code below, and it runs well in both methods.
	if len(urls) != 0 || method == METHOD_SLOW {
		client, ok := app.GetAvailableClient(int32(len(urls)))
		if !ok {
			return nil, internalError(protocol.ErrNoBrowserAvailable)
		}

		app.RequestCh <- &protocol.RequestJobWrapper{
			RequestId: reqId,
			ClientId:  client.id,
			Type:      protocol.MessageType_GET_MAILS_URLS,
			Urls:      urls,
		}

		r, err := app.awaitResults(reqId)
		if err != nil {
			return nil, internalError(err)
		}

		saveWebsites(r.GetResult())

		results = append(results, r.GetResult()...)
	}

	return results, nil
}

func (app *Application) getMailsFromWebsites(websites []*protocol.Website) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()
	client, ok := app.GetAvailableClient(int32(len(websites)))
	if !ok {
		return nil, internalError(protocol.ErrNoBrowserAvailable)
	}

	app.RequestCh <- &protocol.RequestJobWrapper{
		RequestId: reqId,
		ClientId:  client.id,
		Type:      protocol.MessageType_GET_MAILS_WEBSITES,
		Websites:  websites,
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return nil, internalError(err)
	}

	saveWebsites(r.GetResult())

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

	saveWebsites(r.GetResult())

	return r.GetResult(), nil
}

func getCurrentTime() string {
	now := time.Now()
	postgresTimestamp := now.Format("2006-01-02 15:04:05-07")
	return postgresTimestamp
}

func verifyCampaignOwnership(u *User, campaignID int) error {
	has, err := u.HasCampaign(campaignID)
	if err != nil {
		return internalError(err)
	}
	if !has {
		return internalError(protocol.ErrCampaignUnowned)
	}
	return nil
}

func saveToCampaign(u *User, id int, websites []*protocol.Website) error {
	if id != 0 {
		if err := verifyCampaignOwnership(u, id); err != nil {
			return err
		}

		campaign, err := getCampaign(id)
		if err != nil {
			return internalError(protocol.ErrCampaignNotFound)
		}

		campaign.AddWebsites(websites...)
	}

	return nil
}
