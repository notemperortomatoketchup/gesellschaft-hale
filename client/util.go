package main

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func isValidMail(m string) bool {
	re := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	return re.MatchString(m)
}

func strContains(str string, patterns ...string) bool {
	for _, pattern := range patterns {
		if has := strings.Contains(str, pattern); has {
			return true
		}
	}
	return false
}

func constructSublink(baseURL, sublink string) (string, error) {
	// in case the string starts with / or #, we unite the link with the base url to get
	// full path

	// cut all but the path of the sublink
	u, _ := url.Parse(sublink)
	sublink = u.Path

	if strings.HasPrefix(sublink, "/") || strings.HasPrefix(sublink, "#") {
		return baseURL + sublink, nil
	}

	// in case the sublink starts with the base url, the link is already complete.
	if strings.HasPrefix(sublink, baseURL) {
		return sublink, nil
	}

	// encountered some site doing that, just putting href="info", so nothing before it.
	// we place this if after the one with base url, because it would trigger it too.
	// Important to check in this order.
	if strings.HasPrefix(sublink, "") && !strings.Contains(sublink, "http") {
		return baseURL + "/" + sublink, nil

	}
	return "", protocol.ErrConstructPath
}
func getBaseUrl(u string) (string, error) {
	complete, err := url.Parse(u)
	if err != nil {
		return u, err
	}

	complete.Path = ""
	complete.Fragment = ""
	complete.RawQuery = ""

	return complete.String(), nil
}

func extractEmailsFromBody(bodyHTML string) []string {
	// Use regular expression to extract email addresses
	re := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	matches := re.FindAllString(bodyHTML, -1)

	patterns := []string{".png", ".jpg", ".jpeg", ".svg", ".webp", ".pdf"}

	var finalMatches []string
	for _, match := range matches {
		if has := strContains(match, patterns...); has {
			continue
		}

		finalMatches = append(finalMatches, match)
	}

	return finalMatches
}

// Normalizers make the strings respect a specific format, making it easier to compare
// them and to modify them without breaking changes. Especially since a lot of websites have differents practices
func normalizeMailTo(s string) string {
	if strings.Contains(s, "mailto:") {
		s = normalizeString(s)
		s = strings.TrimSpace(strings.Split(strings.TrimPrefix(s, "mailto:"), "?")[0])
		return s
	}

	return s
}

func normalizeString(s string) string {
	// removing invisible characters common in scrapping
	// as well as lowering & trimming it for comparaison purposes
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "%20", " ")
	s = strings.TrimSpace(s)

	return s
}
