package site

// A `.list` row stays readable on a phone.
//
// On a wide screen a row is two columns, the id and then the title. On a phone
// the id column (a sixteen-digit timestamp id and its date, about 220 px) leaves
// the title a column a few characters wide, so below the width where the two
// columns stop fitting the row stacks: the id above, the title below at the full
// width of the row. A sideways scroll is not the alternative; the page never
// scrolls sideways (iss-2610040729344770).
//
// The id column wraps between ids and never inside one: a supersession row on
// /record/health/ carries two timestamp ids and an arrow, wider than the row on
// a 360 px phone, so the span gives at its spaces while each id link, the stub
// and the date keep their nowrap.

import (
	"regexp"
	"strconv"
	"testing"
)

// listStackPhone and listStackTablet bound the stacking breakpoint: at or above
// the wider of the two phone widths the overflow audit measures (390 px), and
// below its tablet width (768 px), where a row keeps its two columns. The
// measured value is 520 px, where the title column of the dashboard's latest
// decisions falls below about 240 px.
const (
	listStackPhone  = 390
	listStackTablet = 768
)

var maxWidthPrelude = regexp.MustCompile(`^@media \(max-width:(\d+)px\)$`)

// TestAListRowStacksOnANarrowScreen holds both stylesheets to a wide row of two
// columns at the top level and a single-column row under a max-width query
// between the phone and tablet widths, placed after the rule it overrides.
func TestAListRowStacksOnANarrowScreen(t *testing.T) {
	for name, src := range bothStylesheets(t) {
		c := cascadeOf(src)
		wide, ok := c.top[".list li"]["grid-template-columns"]
		if !ok || wide.value != "auto 1fr" {
			t.Errorf("%s: a wide .list row is grid-template-columns:%q, want auto 1fr", name, wide.value)
		}
		var found bool
		for prelude, rules := range c.media {
			m := maxWidthPrelude.FindStringSubmatch(prelude)
			if m == nil {
				continue
			}
			px, _ := strconv.Atoi(m[1])
			narrow, ok := rules[".list li"]["grid-template-columns"]
			if !ok {
				continue
			}
			found = true
			if narrow.value != "minmax(0,1fr)" {
				t.Errorf("%s: under %s a .list row is grid-template-columns:%q, want minmax(0,1fr)", name, prelude, narrow.value)
			}
			if px < listStackPhone || px >= listStackTablet {
				t.Errorf("%s: a .list row stacks under %s, want a breakpoint in [%d,%d) px", name, prelude, listStackPhone, listStackTablet)
			}
			if narrow.at < wide.at {
				t.Errorf("%s: the stacking rule under %s comes before the two-column rule, which overrides it", name, prelude)
			}
		}
		if !found {
			t.Errorf("%s: no max-width media query stacks a .list row: on a phone the title is a column a few characters wide", name)
		}
	}
}

// TestAListIdWrapsBetweenIdsNotInsideOne holds the id span to normal wrapping
// and every element in it to nowrap, at the top level in both stylesheets.
func TestAListIdWrapsBetweenIdsNotInsideOne(t *testing.T) {
	for name, src := range bothStylesheets(t) {
		c := cascadeOf(src)
		if got := c.top[".list .id"]["white-space"].value; got != "normal" {
			t.Errorf("%s: .list .id is white-space:%q, want normal: two ids and an arrow cannot wrap", name, got)
		}
		if got := c.top[".list .id>*"]["white-space"].value; got != "nowrap" {
			t.Errorf("%s: .list .id>* is white-space:%q, want nowrap: an id breaks inside itself", name, got)
		}
	}
}
