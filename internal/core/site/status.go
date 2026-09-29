package site

// The Now / Next / Later block at the head of the Status page — the record
// health page, `/record/health/`, which the `pages.status` switch governs
// (itd-2609061543533170 decision 7) — for itd-2609212103568351.
//
// The block is statusblock.Read's, the one read the bare `abcd` board renders
// too: the build calls it with the lane reader its caller hands in (the front
// door passes the implement loop's; a nil one reads as an absent state file),
// so the page and the board cannot disagree about what is Now, Next or Later.
// Every word the block adds is an id, a title, a lane state from the state file,
// a readiness check's name, or an interface label from `site-src/ui.json`.

import (
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// statusSection renders the block as three panels, Now, Next and Later, each
// noted with its count. It is empty when the page carries no block.
func (e *explorer) statusSection() string {
	if e.status == nil {
		return ""
	}
	ui := e.c.ui.Status
	var b strings.Builder
	b.WriteString(`<div class="dash reading status-block">`)
	for _, list := range []struct {
		heading string
		rows    []statusblock.Row
	}{{ui.Now, e.status.Now}, {ui.Next, e.status.Next}, {ui.Later, e.status.Later}} {
		b.WriteString(panel("c4", list.heading, strconv.Itoa(len(list.rows)), e.statusRows(list.rows)))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// statusRows is one list: each row's id (linked to its record page where it has
// one), its title, and what places it there.
func (e *explorer) statusRows(rows []statusblock.Row) string {
	ui := e.c.ui
	if len(rows) == 0 {
		return `<div class="health"><div class="hsum">` + escapeText(ui.Status.None) + `</div></div>`
	}
	var b strings.Builder
	b.WriteString(`<ul class="list">`)
	for _, r := range rows {
		id := escapeText(r.ID)
		if route := e.routeOf(r.ID); route != "" {
			id = `<a href="/` + escapeAttr(route) + `">` + id + `</a>`
		}
		b.WriteString(`<li><span class="id">` + id + `</span><span>` + escapeText(r.Title) + `</span>`)
		if tag := e.statusTag(r); tag != "" {
			b.WriteString(`<span class="s">` + tag + `</span>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// statusTag is what places a row, as escaped HTML: the lane and its next step,
// the next-up mark, the gating checks a refused intent fails, or the draft mark.
func (e *explorer) statusTag(r statusblock.Row) string {
	ui := e.c.ui.Status
	switch {
	case r.Lane != nil:
		parts := []string{}
		for _, p := range []string{r.Lane.Lane, r.Lane.Step, r.Lane.Awaiting} {
			if p != "" {
				parts = append(parts, escapeText(p))
			}
		}
		return strings.Join(parts, " · ")
	case r.NextUp:
		return `<b>` + escapeText(ui.NextUp) + `</b>`
	case len(r.Failing) > 0:
		return escapeText(ui.Fails) + ` ` + escapeText(strings.Join(r.Failing, ", "))
	case r.Bucket == intent.BucketDrafts:
		return escapeText(ui.Draft)
	}
	return ""
}
