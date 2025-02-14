package main

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var validSchemes = map[string]struct{}{
	"":      {},
	"http":  {},
	"https": {},
}

var atomToAttributes = map[atom.Atom][]string{
	atom.A:      {"href"},
	atom.Frame:  {"src"},
	atom.Iframe: {"src"},
	atom.Img:    {"src"},
	atom.Link:   {"href"},
	atom.Script: {"src"},
	atom.Source: {"src", "srcset"},
	atom.Track:  {"src"},
}

var imageDescriptorPattern = regexp.MustCompile(" [^ ]*$")

type linkFinder struct {
	excludedPatterns []*regexp.Regexp
	includedPatterns []*regexp.Regexp
}

func newLinkFinder(rs []*regexp.Regexp, includePatterns []*regexp.Regexp) linkFinder {
	return linkFinder{excludedPatterns: rs, includedPatterns: includePatterns}
}

func (f linkFinder) Find(n *html.Node, base *url.URL) map[string]error {
	ls := map[string]error{}

	for _, n := range scrape.FindAllNested(n, func(n *html.Node) bool {
		_, ok := atomToAttributes[n.DataAtom]
		return ok
	}) {
		for _, a := range atomToAttributes[n.DataAtom] {
			ss := f.parseLinks(n, a)

			for _, s := range ss {
				s := strings.TrimSpace(s)

				if s == "" || f.isLinkExcluded(s) {
					continue
				}

				// only use include pattern when not empty
				if len(f.includedPatterns) > 0 && !f.isLinkIncluded(s) {
					continue
				}

				u, err := url.Parse(s)
				if err != nil {
					ls[s] = err
					continue
				} else if _, ok := validSchemes[u.Scheme]; ok {
					ls[base.ResolveReference(u).String()] = nil
				}
			}
		}
	}

	return ls
}

func (linkFinder) parseLinks(n *html.Node, a string) []string {
	s := scrape.Attr(n, a)
	ss := []string{}

	if a == "srcset" {
		for _, s := range strings.Split(s, ",") {
			ss = append(ss, imageDescriptorPattern.ReplaceAllString(strings.TrimSpace(s), ""))
		}
	} else {
		ss = append(ss, s)
	}

	return ss
}

func (f linkFinder) isLinkExcluded(u string) bool {
	return f.isMatch(u, f.excludedPatterns)
}

func (f linkFinder) isLinkIncluded(u string) bool {
	return f.isMatch(u, f.includedPatterns)
}

func (f linkFinder) isMatch(u string, patterns []*regexp.Regexp) bool {
	for _, r := range patterns {
		if r.MatchString(u) {
			return true
		}
	}

	return false
}
