package site

// The closed page set (itd-2609061543533170, spc-2609212141407459 scope 3).
//
// Every repository abcd renders gets the same pages abcd's own site has: the
// landing page, the record explorer, one page per record, the relationship
// graph, the timeline, the glossary and the status page. The composition
// manifest's `pages` block switches pages OFF; it cannot switch anything on,
// because the set is closed, and a key outside it is refused at load by the
// manifest's unknown-field rule like any other typo.
//
// Two pages carry the site rather than sit in it, so their switches are
// refused rather than honoured:
//
//   - the landing page is the site's root; a site with no root is not a site;
//   - the record pages ARE the explorer's substance, linked from every other
//     explorer page, so they follow the explorer's switch and cannot be turned
//     off beneath it.
//
// Switching the explorer off takes every explorer page with it (the record
// pages, the graph, the timeline, the glossary and the status page), and the
// header's link to it. Each other switch removes its page, its navigation entry
// and every link the renderer would have drawn to it.

// PageSwitches is the `pages` block. A nil field is on.
type PageSwitches struct {
	Landing     *bool `json:"landing,omitempty"`
	Explorer    *bool `json:"explorer,omitempty"`
	RecordPages *bool `json:"record_pages,omitempty"`
	Graph       *bool `json:"graph,omitempty"`
	Timeline    *bool `json:"timeline,omitempty"`
	Glossary    *bool `json:"glossary,omitempty"`
	Status      *bool `json:"status,omitempty"`
}

// PageNames is the closed page set, in the order the manifest documents it.
var PageNames = []string{"landing", "explorer", "record_pages", "graph", "timeline", "glossary", "status"}

// on reports a switch: absent is on.
func on(b *bool) bool { return b == nil || *b }

// pageSet is the resolved switches the renderer consults.
type pageSet struct {
	explorer, graph, timeline, glossary, status bool
}

// resolve folds the explorer switch over the pages beneath it.
func (p PageSwitches) resolve() pageSet {
	ex := on(p.Explorer)
	return pageSet{
		explorer: ex,
		graph:    ex && on(p.Graph),
		timeline: ex && on(p.Timeline),
		glossary: ex && on(p.Glossary),
		status:   ex && on(p.Status),
	}
}

// validate refuses the two switches the set cannot honour.
func (p PageSwitches) validate(bad func(string, ...any) error) error {
	if !on(p.Landing) {
		return bad("pages.landing is false, but the landing page is the site's root and cannot be switched off")
	}
	if on(p.Explorer) && !on(p.RecordPages) {
		return bad("pages.record_pages is false while the explorer is on; the record pages are what every explorer page links to, so they follow pages.explorer")
	}
	if !on(p.Explorer) {
		for _, f := range []struct {
			key string
			v   *bool
		}{{"record_pages", p.RecordPages}, {"graph", p.Graph}, {"timeline", p.Timeline}, {"glossary", p.Glossary}, {"status", p.Status}} {
			if f.v != nil && *f.v {
				return bad("pages.%s is true while pages.explorer is false; it is an explorer page and goes with the explorer", f.key)
			}
		}
	}
	return nil
}
