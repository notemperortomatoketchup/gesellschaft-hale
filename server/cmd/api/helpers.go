package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
)

func (app *Application) awaitResults(id uint32) (*protocol.ResponseJobWrapper, error) {
	var result *protocol.ResponseJobWrapper

	// as long as result is nil we shall range.
	for result == nil {
		time.Sleep(time.Second)
		app.results.Range(func(key, value any) bool {
			if key.(uint32) == id {
				result = value.(*protocol.ResponseJobWrapper)
				app.results.Delete(key)
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

func scrapeFilterWebsites(websites []*protocol.Website) ([]*protocol.Website, []*protocol.Website) {
	var toScrape []*protocol.Website
	var scraped []*protocol.Website

	toScrapeCh := make(chan *protocol.Website, 0)
	scrapedCh := make(chan *protocol.Website, 0)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		var wg sync.WaitGroup
		for _, w := range websites {
			wg.Add(1)
			go func(j *protocol.Website) {
				defer wg.Done()
				w, err := models.GetWebsite(j.BaseUrl)
				if err == protocol.ErrWebsiteNotFound {
					toScrapeCh <- j
				} else {
					scrapedCh <- w
				}
			}(w)
		}

		wg.Wait()
		defer cancel()
	}()

mainloop:
	for {
		select {
		case <-ctx.Done():
			break mainloop
		case website := <-scrapedCh:
			scraped = append(scraped, website)
		case website := <-toScrapeCh:
			toScrape = append(toScrape, website)
		}
	}

	return scraped, toScrape
}

func scrapeFilter(urls []string) ([]*protocol.Website, []string) {
	var toScrape []string
	var scraped []*protocol.Website

	ctx, cancel := context.WithCancel(context.Background())
	toScrapeCh := make(chan string, 0)
	scrapedCh := make(chan *protocol.Website, 0)

	var wg sync.WaitGroup
	go func() {
		if urls != nil {
			for _, u := range urls {
				wg.Add(1)
				go func(j string) {
					defer wg.Done()
					w, err := models.GetWebsite(j)
					if err == protocol.ErrWebsiteNotFound {
						toScrapeCh <- j
					} else {
						scrapedCh <- w
					}
				}(u)
			}
		}

		wg.Wait()
		cancel()
	}()

mainloop:
	for {
		select {
		case <-ctx.Done():
			break mainloop
		case website := <-scrapedCh:
			scraped = append(scraped, website)
		case url := <-toScrapeCh:
			toScrape = append(toScrape, url)
		}
	}

	return scraped, toScrape
}

func (app *Application) getMailsFromUrls(urls []string, method int) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()

	var results []*protocol.Website

	if method == METHOD_FAST {
		// reassign urls so that only unscraped are there.
		results, urls = scrapeFilter(urls)
	}

	// if urls is not 0, then even if fast method we still have some to scrape.
	// but if we are in method slow, we just go, no matter what, as above step didn't ran.
	// we merge urls above so taht we don't have to change the code below, and it runs well in both methods.
	if len(urls) != 0 || method == METHOD_SLOW {
		client, ok := app.GetAvailableClient(int32(len(urls)))
		if !ok {
			return nil, internalError(protocol.ErrNoBrowserAvailable)
		}

		app.requestCh <- &protocol.RequestJobWrapper{
			RequestId: reqId,
			ClientId:  client.id,
			Type:      protocol.MessageType_GET_MAILS_URLS,
			Urls:      urls,
		}

		r, err := app.awaitResults(reqId)
		if err != nil {
			return nil, internalError(err)
		}

		if r.Error != "" && strings.Contains(r.Error, "transport") {
			return nil, internalError(fmt.Errorf("internal error, please retry your request"))
		}

		models.SaveWebsites(r.GetResult())
		results = append(results, r.GetResult()...)
	}

	return results, nil
}

func (app *Application) getMailsFromWebsites(websites []*protocol.Website, method int) ([]*protocol.Website, error) {
	var results []*protocol.Website
	reqId := protocol.GenerateId()

	if method == METHOD_FAST {
		results, websites = scrapeFilterWebsites(websites)
	}

	if len(websites) != 0 || method == METHOD_SLOW {
		client, ok := app.GetAvailableClient(int32(len(websites)))
		if !ok {
			return nil, internalError(protocol.ErrNoBrowserAvailable)
		}

		app.requestCh <- &protocol.RequestJobWrapper{
			RequestId: reqId,
			ClientId:  client.id,
			Type:      protocol.MessageType_GET_MAILS_WEBSITES,
			Websites:  websites,
		}

		r, err := app.awaitResults(reqId)
		if err != nil {
			return nil, internalError(err)
		}

		results = append(results, r.GetResult()...)
		models.SaveWebsites(r.GetResult())
	}

	return results, nil
}

func (app *Application) getKeywordResults(kw string, pages int, domain string, save bool) ([]*protocol.Website, error) {
	reqId := protocol.GenerateId()
	client, ok := app.GetAvailableClient(0)
	if !ok {
		return nil, internalError(protocol.ErrNoBrowserAvailable)
	}

	if domain == "" {
		domain = "google.fr"
	}

	app.requestCh <- &protocol.RequestJobWrapper{
		RequestId:    reqId,
		Type:         protocol.MessageType_GET_KEYWORD,
		ClientId:     client.id,
		Keyword:      kw,
		PagesCount:   int32(pages),
		GoogleDomain: domain,
	}

	r, err := app.awaitResults(reqId)
	if err != nil {
		return nil, internalError(err)
	}

	if save {
		models.SaveWebsites(r.GetResult())
	}

	return r.GetResult(), nil
}

func (r *EditUserRequest) Matches(u *models.User) {
	if r.Username != "" {
		u.SetUsername(r.Username)
	}

	if r.NotionParent != "" {
		u.SetNotionParent(r.NotionParent)
	}

	if r.NotionSecret != "" {
		u.SetNotionSecret(r.NotionSecret)
	}

}
