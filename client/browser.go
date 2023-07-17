package main

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Browser struct {
	id       int
	instance *rod.Browser
	timeout  time.Duration
	results  Results
	active   bool
	pages    rod.PagePool
	queue    *Queue
}

type BrowserAction func(b *Browser, w *protocol.Website) error

func (app *Application) newBrowser(id int, timeout time.Duration) *Browser {
	l := launcher.New()
	// fix for docker
	l.Append("--disable-dev-shm-usage")
	l.Set("ignore-certificate-errors")
	l.Set("ignore-certificate-errors-spki-list")
	l.Set("ignore-ssl-errors")

	if app.Client.cfg.browser.noSandbox {
		l = l.NoSandbox(true)
	}

	path, _ := launcher.LookPath()
	u := l.Bin(path).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()

	router := browser.HijackRequests()

	// ignore images, fonts and css files, useless to scrape.
	router.MustAdd("*", func(ctx *rod.Hijack) {
		if ctx.Request.Type() == proto.NetworkResourceTypeMedia ||
			ctx.Request.Type() == proto.NetworkResourceTypeFetch ||
			ctx.Request.Type() == proto.NetworkResourceTypeWebSocket ||
			ctx.Request.Type() == proto.NetworkResourceTypeImage ||
			ctx.Request.Type() == proto.NetworkResourceTypeFont ||
			ctx.Request.Type() == proto.NetworkResourceTypeStylesheet {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()

	return &Browser{
		id:       id,
		instance: browser,
		timeout:  timeout,
		results:  Results{},
		queue:    app.newQueue(),
	}
}

func (b *Browser) cleanup() {
	opens, err := b.instance.Pages()
	if err != nil {
		fmt.Println("err getting pages ->", err)
	}

	for _, page := range opens {
		page.MustClose()
	}

	fmt.Println("cleaned browser, got ->", len(opens), "opened pages")
	b.active = false
	b.results = Results{}
}

func (b *Browser) createPage() (*rod.Page, error) {
	page, err := b.instance.MustIncognito().Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}

	return page, nil
}

func (app *Application) currentCapacity() int32 {
	return int32(app.Client.cfg.queue.maxTasks) * app.Client.pool.stats.idleCount.Load()
}
