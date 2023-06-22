package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Engine struct {
	baseLink string
}

type ScrapeKeywordResp struct {
	Data       []*KeywordEntry `json:"organic_results"`
	Pagination PaginationResp  `json:"pagination"`
}

type PaginationResp struct {
	Next string `json:"next"`
}

type KeywordEntry struct {
	Link         string       `json:"link"`
	Title        string       `json:"title"`
	Snippet      string       `json:"snippet"`
	MatchedWords []string     `json:"snippet_highlighted_words"`
	Related      RelatedInfos `json:"about_this_result"`
}

type RelatedInfos struct {
	Language []string `json:"languages"`
	Region   []string `json:"regions"`
}

func (app *Application) startEngine() {
	if app.Client.cfg.engine.baseLink == "" {
		log.Fatal(protocol.ErrNoBaseLink.Error())
	}

	app.Client.engine = &Engine{
		baseLink: app.Client.cfg.engine.baseLink,
	}
}

func (e *Engine) scrapeKeyword(kw string, pages int) ([]*protocol.Website, error) {
	if pages == 0 {
		pages = 1
	}

	kw = strings.ReplaceAll(kw, " ", "+") // for send http
	results := new(Results)

	var seen sync.Map
	var wg sync.WaitGroup

	for i := 0; i < pages; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := new(ScrapeKeywordResp)
			url := e.baseLink + "q=" + kw + "&start=" + fmt.Sprintf("%d", i*10)

			resp, err := http.Get(url)
			if err != nil {
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()

			if err = json.Unmarshal(body, res); err != nil {
				return
			}

			for _, w := range res.Data {
				// putting them as Website back
				base, _ := getBaseUrl(w.Link)
				if _, ok := seen.Load(base); ok {
					continue
				}
				seen.Store(base, true)

				results.Append(&protocol.Website{
					BaseUrl:      base,
					Title:        w.Title,
					Description:  w.Snippet,
					MatchedWords: w.MatchedWords,
					Country: &protocol.CountryInfo{
						Language: w.Related.Language,
						Region:   w.Related.Region,
					},
				})
			}
		}(i)
	}

	wg.Wait()
	return results.Get(), nil
}
