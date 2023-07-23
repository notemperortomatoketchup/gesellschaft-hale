package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-rod/rod"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func stepExtractMetadata(page *rod.Page, w *protocol.Website) {
	rod.Try(func() {
		page.MustNavigate(w.BaseUrl).MustWaitLoad()

		infos, err := page.Info()
		if err != nil {
			fmt.Println("Err extracting infos ->", err)
		}
		w.Title = infos.Title

		found, metadesc, err := page.Has(`meta[name="description"]`)
		if err != nil {
			fmt.Println("Err extract meta description ->", err)
		}

		if found {
			desc, err := metadesc.Attribute("content")
			if err != nil {
				fmt.Println("Err extracting attribute ->", err)
			}
			w.Description = *desc

		}

	})
}

func stepExtractPaths(page *rod.Page, w *protocol.Website, patterns []string) {
	rod.Try(func() {
		page.MustNavigate(w.BaseUrl).MustWaitLoad()

		anchors := page.MustElements("a[href]")
		for _, a := range anchors {
			href := *a.MustAttribute("href")
			// important to do that to avoid issues with main domain having one of hte patterns string like .info
			hrefParsed, _ := url.Parse(href)
			if strings.Contains(hrefParsed.Path, "mailto:") {
				continue
			}

			if has := strContains(hrefParsed.Path, patterns...); has {
				if sublink, err := constructSublink(w.BaseUrl, hrefParsed.Path); err == nil {
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
	socialPatterns := []string{"twitter.com", "facebook.com", "instagram.com", "linkedin.com", "dribbble.com", "behance.net"}
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

			anchors := page.MustElements("a[href]")
			for _, a := range anchors {
				href := *a.MustAttribute("href")
				if has := strContains(href, socialPatterns...); has {
					w.Socials = protocol.AppendUnique(w.Socials, href)
				}
			}
		})
	}
}
