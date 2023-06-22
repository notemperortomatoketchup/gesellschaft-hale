package main

import (
	"strings"

	"github.com/go-rod/rod"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func stepExtractPaths(page *rod.Page, w *protocol.Website, patterns []string) {
	rod.Try(func() {
		page.MustNavigate(w.BaseUrl).MustWaitLoad()
		anchors := page.MustElements("a[href]")
		for _, a := range anchors {
			href := *a.MustAttribute("href")

			if strings.Contains(href, "mailto:") {
				continue
			}

			if has := strContains(href, patterns...); has {
				if sublink, err := constructSublink(w.BaseUrl, href); err == nil {
					w.Paths = protocol.AppendUnique(w.Paths, sublink)
				}
			}
		}
	})
}

func stepExtractMails(page *rod.Page, w *protocol.Website) {
	// add base url to the list of to extract, without adding it to the paths.
	paths := w.Paths
	paths = append(paths, w.BaseUrl)

	for _, path := range paths {
		rod.Try(func() {
			page.MustNavigate(path).MustWaitLoad()
			body := page.MustElement("body").MustHTML()
			mails := extractEmailsFromBody(body)

			for _, mail := range mails {
				if strings.Contains(mail, "mailto:") {
					mail = normalizeMailTo(mail)
				} else {
					mail = normalizeString(mail)
				}
				w.Mails = protocol.AppendUnique(w.Mails, mail)
			}
		})
	}
}
