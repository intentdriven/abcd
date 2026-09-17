package statusline

import (
	"math"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// fullInput is the reference render input: the fixture payload, a repository,
// the record's counts and the quiet badge state. Every render test starts from
// it and removes exactly one thing, so a failure names what was removed.
func fullInput(t *testing.T) Input {
	t.Helper()
	p, err := ParsePayload([]byte(fullPayloadJSON))
	if err != nil {
		t.Fatalf("ParsePayload: %v", err)
	}
	return Input{
		State:   StateManaged,
		Repo:    Repo{Name: "abcd", Branch: "main"},
		Payload: p,
		Counts:  Counts{Intents: 12, Issues: 34},
	}
}

// wantFullRow is ac-11: the nine elements in the order itd-200 fixes, each
// separated the same way.
const wantFullRow = "abcd · abcd · main · Opus · ctx 8% · 5h 24% · 7d 41% · itd 12 · iss 34"

// TestRenderFullRow is ac-11. Nine elements (the spec's "twelve" is a
// miscount of its own list), in the order the intent's commitments fix, with
// one separator between each pair.
func TestRenderFullRow(t *testing.T) {
	row := Render(fullInput(t), Defaults())

	if got := row.Plain(); got != wantFullRow {
		t.Fatalf("row.Plain() =\n  %q\nwant\n  %q", got, wantFullRow)
	}
	wantKeys := []ElementKey{
		KeyPresence, KeyRepo, KeyBranch, KeyModel,
		KeyContext, KeyFiveHour, KeySevenDay, KeyIntents, KeyIssues,
	}
	if len(row.Elements) != len(wantKeys) {
		t.Fatalf("row has %d elements, want %d: %+v", len(row.Elements), len(wantKeys), row.Elements)
	}
	for i, k := range wantKeys {
		if row.Elements[i].Key != k {
			t.Fatalf("element %d is %q, want %q", i, row.Elements[i].Key, k)
		}
	}
	if n := strings.Count(row.Plain(), Separator); n != len(wantKeys)-1 {
		t.Fatalf("row carries %d separators, want %d", n, len(wantKeys)-1)
	}
}

// TestRenderOrderIsFixed pins the order as a value, not as a property of one
// fixture, so a reordering shows up here as well as in the row test.
func TestRenderOrderIsFixed(t *testing.T) {
	want := []ElementKey{
		KeyPresence, KeyRepo, KeyBranch, KeyModel,
		KeyContext, KeyFiveHour, KeySevenDay, KeyIntents, KeyIssues,
	}
	got := Order()
	if len(got) != len(want) {
		t.Fatalf("Order() has %d keys, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Order()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if got[0] != KeyPresence {
		t.Fatal("the badge is not element one")
	}
	// Order() hands back a copy; mutating it must not move the badge.
	got[0] = KeyIssues
	if Order()[0] != KeyPresence {
		t.Fatal("Order() returned the package's own slice")
	}
}

// TestRenderDropsAnAbsentPayloadField is ac-12, walked field by field: an
// absent field drops its element outright — no placeholder, no empty slot, and
// no separator left where the element was.
func TestRenderDropsAnAbsentPayloadField(t *testing.T) {
	cases := []struct {
		name    string
		remove  func(in *Input)
		dropped ElementKey
	}{
		{name: "model", remove: func(in *Input) { in.Payload.Model = "" }, dropped: KeyModel},
		{name: "context", remove: func(in *Input) { in.Payload.ContextPct = nil }, dropped: KeyContext},
		{name: "five hour", remove: func(in *Input) { in.Payload.FiveHourPct = nil }, dropped: KeyFiveHour},
		{name: "seven day", remove: func(in *Input) { in.Payload.SevenDayPct = nil }, dropped: KeySevenDay},
		{name: "branch", remove: func(in *Input) { in.Repo.Branch = "" }, dropped: KeyBranch},
		{name: "repository name", remove: func(in *Input) { in.Repo.Name = "" }, dropped: KeyRepo},
	}
	full := Render(fullInput(t), Defaults())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := fullInput(t)
			tc.remove(&in)
			row := Render(in, Defaults())

			if _, ok := row.Element(tc.dropped); ok {
				t.Fatalf("%q survived its field's absence", tc.dropped)
			}
			if len(row.Elements) != len(full.Elements)-1 {
				t.Fatalf("row has %d elements, want %d", len(row.Elements), len(full.Elements)-1)
			}
			assertNoPlaceholder(t, row)
			// Every other element is untouched and still in order.
			assertOrdered(t, row)
		})
	}
}

// TestRenderSwitchesEachElementOff walks the per-element switches one at a
// time. The badge is the exception and has its own test below.
func TestRenderSwitchesEachElementOff(t *testing.T) {
	full := Render(fullInput(t), Defaults())
	for _, k := range Order() {
		if k == KeyPresence {
			continue
		}
		t.Run(string(k), func(t *testing.T) {
			set := Defaults()
			set.Elements[k] = false
			row := Render(fullInput(t), set)

			if _, ok := row.Element(k); ok {
				t.Fatalf("%q rendered although it is switched off", k)
			}
			if len(row.Elements) != len(full.Elements)-1 {
				t.Fatalf("row has %d elements, want %d", len(row.Elements), len(full.Elements)-1)
			}
			assertNoPlaceholder(t, row)
			assertOrdered(t, row)
		})
	}
}

// TestRenderNeverSwitchesTheBadgeOff: the badge is element one and is not
// switchable (itd-200's commitment). A settings file that names it is ignored,
// not honoured.
func TestRenderNeverSwitchesTheBadgeOff(t *testing.T) {
	set := Defaults()
	set.Elements[KeyPresence] = false
	row := Render(fullInput(t), set)
	if len(row.Elements) == 0 || row.Elements[0].Key != KeyPresence {
		t.Fatalf("the badge was switched off: %+v", row.Elements)
	}
	if !set.Enabled(KeyPresence) {
		t.Fatal("Settings.Enabled reports the badge as switchable")
	}
}

// TestRenderEverySwitchOffStillLeadsWithTheBadge: with every switchable
// element off the row is the badge alone, and nothing else.
func TestRenderEverySwitchOffStillLeadsWithTheBadge(t *testing.T) {
	set := Defaults()
	for _, k := range Order() {
		set.Elements[k] = false
	}
	row := Render(fullInput(t), set)
	if len(row.Elements) != 1 || row.Elements[0].Key != KeyPresence {
		t.Fatalf("row = %+v, want the badge alone", row.Elements)
	}
	if strings.Contains(row.Plain(), Separator) {
		t.Fatalf("a one-element row carries a separator: %q", row.Plain())
	}
}

// TestRenderDisabledContributesNothing is ac-6: the off switch makes the
// render produce no row at all, badge included.
func TestRenderDisabledContributesNothing(t *testing.T) {
	set := Defaults()
	set.Disabled = true
	row := Render(fullInput(t), set)
	if len(row.Elements) != 0 {
		t.Fatalf("a disabled render produced %+v", row.Elements)
	}
	if row.Plain() != "" || row.String() != "" {
		t.Fatalf("a disabled render produced %q / %q", row.Plain(), row.String())
	}
	if !row.Empty() {
		t.Fatal("Row.Empty() is false for a disabled render")
	}
}

// TestRenderBadgeStates is ac-1, ac-2 and ac-3 in one table: each of the three
// states renders its own badge, and the badge is element one in all three.
func TestRenderBadgeStates(t *testing.T) {
	cases := []struct {
		state     State
		wantPlain string
	}{
		{state: StateManaged, wantPlain: "abcd"},
		{state: StateFacilitator, wantPlain: "waiting: facilitator"},
		{state: StateProductThinker, wantPlain: "waiting: product thinker"},
	}
	seen := map[string]State{}
	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			in := fullInput(t)
			in.State = tc.state
			row := Render(in, Defaults())

			badge, ok := row.Element(KeyPresence)
			if !ok {
				t.Fatal("no badge element")
			}
			if row.Elements[0].Key != KeyPresence {
				t.Fatalf("the badge is element %d, not element one", indexOf(row, KeyPresence))
			}
			if badge.Plain != tc.wantPlain {
				t.Fatalf("badge plain = %q, want %q", badge.Plain, tc.wantPlain)
			}
			if prev, dup := seen[badge.Plain]; dup {
				t.Fatalf("%s and %s render the same word %q", prev, tc.state, badge.Plain)
			}
			seen[badge.Plain] = tc.state
		})
	}
	if len(seen) != 3 {
		t.Fatalf("only %d distinct badge words across three states", len(seen))
	}
}

// TestRenderUnknownStateFallsBackToManaged: the badge state arrives from a
// stored file written by two writers, so an unreadable value must render the
// quiet state rather than an empty or an invented badge.
func TestRenderUnknownStateFallsBackToManaged(t *testing.T) {
	in := fullInput(t)
	in.State = State("nonsense")
	row := Render(in, Defaults())
	badge, ok := row.Element(KeyPresence)
	if !ok {
		t.Fatal("no badge element")
	}
	if badge.Plain != "abcd" {
		t.Fatalf("badge plain = %q, want the managed word", badge.Plain)
	}
}

// TestBadgeMeaningSurvivesColourRemoval is the accessibility commitment made
// structural: strip every ANSI sequence from the rendered row and the three
// states are still told apart, because the word — not the colour — carries the
// meaning ("Word and inverse video carry the meaning; colour only reinforces
// and is never the sole signal").
func TestBadgeMeaningSurvivesColourRemoval(t *testing.T) {
	uncoloured := map[string]State{}
	for _, st := range mode.States() {
		in := fullInput(t)
		in.State = st
		row := Render(in, Defaults())

		stripped := stripANSI(row.String())
		if stripped == "" {
			t.Fatalf("%s: the row is nothing but escapes", st)
		}
		badge, _ := row.Element(KeyPresence)
		if !strings.Contains(stripped, badge.Plain) {
			t.Fatalf("%s: stripping colour from the rendered row gave %q, which does not carry the badge word %q", st, stripped, badge.Plain)
		}
		if !strings.Contains(stripANSI(badge.Rendered), badge.Plain) {
			t.Fatalf("%s: the badge's word does not survive colour removal: %q", st, badge.Rendered)
		}
		if prev, dup := uncoloured[stripped]; dup {
			t.Fatalf("%s and %s are indistinguishable without colour", prev, st)
		}
		uncoloured[stripped] = st
	}
}

// TestRenderedBadgeCarriesItsColourPair: colour reinforces, so it must
// actually be emitted, and it must be the pair the settings hold.
func TestRenderedBadgeCarriesItsColourPair(t *testing.T) {
	set := Defaults()
	row := Render(fullInput(t), set)
	badge, _ := row.Element(KeyPresence)
	fg, err := ParseColor(set.Presence.Foreground)
	if err != nil {
		t.Fatal(err)
	}
	bg, err := ParseColor(set.Presence.Background)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"38;2;240;192;82",
		"48;2;68;68;68",
		reset,
	} {
		if !strings.Contains(badge.Rendered, want) {
			t.Fatalf("badge.Rendered = %q, missing %q", badge.Rendered, want)
		}
	}
	if fg != (RGB{0xf0, 0xc0, 0x52}) || bg != (RGB{0x44, 0x44, 0x44}) {
		t.Fatalf("the shipped default pair moved: %v on %v", fg, bg)
	}
	// A configured pair reaches the render.
	set.Presence = Pair{Foreground: "#ffffff", Background: "#000000"}
	row = Render(fullInput(t), set)
	badge, _ = row.Element(KeyPresence)
	if !strings.Contains(badge.Rendered, "38;2;255;255;255") || !strings.Contains(badge.Rendered, "48;2;0;0;0") {
		t.Fatalf("the configured pair did not reach the render: %q", badge.Rendered)
	}
}

// TestRoleBadgesIgnoreTheConfiguredPair: the presence pair is a setting and
// the role pairs are fixed constants (iss-168: "recognition without reading
// only works if it looks the same in every repository").
func TestRoleBadgesIgnoreTheConfiguredPair(t *testing.T) {
	set := Defaults()
	set.Presence = Pair{Foreground: "#ffffff", Background: "#000000"}
	for _, st := range []State{StateFacilitator, StateProductThinker} {
		in := fullInput(t)
		in.State = st
		row := Render(in, set)
		badge, _ := row.Element(KeyPresence)
		if strings.Contains(badge.Rendered, "48;2;0;0;0") {
			t.Fatalf("%s took the configured presence background: %q", st, badge.Rendered)
		}
		pair := fixedPair(st)
		bg, err := ParseColor(pair.Background)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(badge.Rendered, bg.sgr(48)) {
			t.Fatalf("%s did not render its fixed background: %q", st, badge.Rendered)
		}
	}
}

// TestEveryPrefixOfTheRowBeginsWithTheBadge is ac-10 checked directly. A
// narrow host truncates from the right and hands the render no width, so the
// only truncation guarantee available is order: cut the row at any point and
// what is left is still the badge, or a prefix of it.
func TestEveryPrefixOfTheRowBeginsWithTheBadge(t *testing.T) {
	settings := []struct {
		name string
		set  Settings
	}{
		{name: "defaults", set: Defaults()},
		{name: "counts off", set: withOff(KeyIntents, KeyIssues)},
		{name: "repo and branch off", set: withOff(KeyRepo, KeyBranch)},
		{name: "everything switchable off", set: withOff(Order()...)},
	}
	for _, st := range mode.States() {
		for _, sc := range settings {
			t.Run(string(st)+"/"+sc.name, func(t *testing.T) {
				in := fullInput(t)
				in.State = st
				row := Render(in, sc.set)

				badge, ok := row.Element(KeyPresence)
				if !ok {
					t.Fatal("no badge element")
				}
				plain := row.Plain()
				r := []rune(plain)
				word := []rune(badge.Plain)
				for n := 0; n <= len(r); n++ {
					prefix := string(r[:n])
					if n <= len(word) {
						if !strings.HasPrefix(badge.Plain, prefix) {
							t.Fatalf("the %d-rune prefix %q is not a prefix of the badge %q", n, prefix, badge.Plain)
						}
						continue
					}
					if !strings.HasPrefix(prefix, badge.Plain) {
						t.Fatalf("the %d-rune prefix %q does not begin with the badge %q", n, prefix, badge.Plain)
					}
				}
				// The rendered form leads with the badge too, escapes included.
				if !strings.HasPrefix(row.String(), badge.Rendered) {
					t.Fatalf("the rendered row does not open with the badge: %q", row.String())
				}
			})
		}
	}
}

// TestRenderCountsAreAlwaysRendered: the counts come from the record rather
// than the payload, so zero is a fact about the record, not an absence, and it
// renders. A caller with no record switches the elements off instead.
func TestRenderCountsAreAlwaysRendered(t *testing.T) {
	in := fullInput(t)
	in.Counts = Counts{}
	row := Render(in, Defaults())
	for k, want := range map[ElementKey]string{KeyIntents: "itd 0", KeyIssues: "iss 0"} {
		el, ok := row.Element(k)
		if !ok {
			t.Fatalf("%q dropped at a zero count", k)
		}
		if el.Plain != want {
			t.Fatalf("%q = %q, want %q", k, el.Plain, want)
		}
	}
}

// TestRenderPercentageRounding pins how a fractional usage percentage reaches
// the row: rounded to a whole percent, never truncated, and never clamped —
// a rate-limit window can read above 100 and the row says so.
func TestRenderPercentageRounding(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{in: 0, want: "ctx 0%"},
		{in: 8, want: "ctx 8%"},
		{in: 23.4, want: "ctx 23%"},
		{in: 23.5, want: "ctx 24%"},
		{in: 99.9, want: "ctx 100%"},
		{in: 104.2, want: "ctx 104%"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			in := fullInput(t)
			v := tc.in
			in.Payload.ContextPct = &v
			row := Render(in, Defaults())
			el, ok := row.Element(KeyContext)
			if !ok {
				t.Fatal("the context element dropped")
			}
			if el.Plain != tc.want {
				t.Fatalf("context = %q, want %q", el.Plain, tc.want)
			}
		})
	}
}

// TestRenderDropsAnImplausiblePercentage: this package may not truncate, so no
// single element may be allowed to decide the row's width. A percentage far
// outside anything the three windows can report is dropped the way an absent
// field is — and so is a NEGATIVE one, which no usage window can report and
// which would otherwise render as "-7%" or "-0%".
func TestRenderDropsAnImplausiblePercentage(t *testing.T) {
	for _, v := range []float64{1e300, -1e300, math.NaN(), math.Inf(1), math.Inf(-1), maxPercent + 1, -7, math.Copysign(0, -1)} {
		in := fullInput(t)
		x := v
		in.Payload.ContextPct = &x
		row := Render(in, Defaults())
		if el, ok := row.Element(KeyContext); ok {
			t.Fatalf("a context percentage of %v rendered as %q", v, el.Plain)
		}
		assertNoPlaceholder(t, row)
	}
	// The boundary itself is plausible and renders.
	in := fullInput(t)
	x := float64(maxPercent)
	in.Payload.ContextPct = &x
	if _, ok := Render(in, Defaults()).Element(KeyContext); !ok {
		t.Fatalf("a context percentage of %v was dropped at the boundary", maxPercent)
	}
}

// TestRenderIsPure: the same input renders the same row, and the render does
// not reach back into its inputs.
func TestRenderIsPure(t *testing.T) {
	in := fullInput(t)
	set := Defaults()
	first := Render(in, set)
	second := Render(in, set)
	if first.Plain() != second.Plain() || first.String() != second.String() {
		t.Fatal("two renders of one input disagree")
	}
	if in.Repo.Name != "abcd" || in.Repo.Branch != "main" || in.Payload.Model != "Opus" {
		t.Fatalf("Render mutated its input: %+v", in)
	}
	if *in.Payload.ContextPct != 8 {
		t.Fatal("Render mutated the payload behind its pointer")
	}
}

// TestRenderSanitizesRepositoryText: the repository name and the branch are
// attacker-influenceable strings (a branch name is whatever was pushed), so
// they go through the same sanitiser the model name does.
func TestRenderSanitizesRepositoryText(t *testing.T) {
	in := fullInput(t)
	in.Repo = Repo{Name: "re\x1b[31mpo", Branch: "ma\x1b[0min"}
	row := Render(in, Defaults())
	for _, k := range []ElementKey{KeyRepo, KeyBranch} {
		el, ok := row.Element(k)
		if !ok {
			t.Fatalf("%q dropped", k)
		}
		if strings.ContainsRune(el.Plain, 0x1b) || strings.ContainsRune(el.Rendered, 0x1b) {
			t.Fatalf("%q carries a raw escape: %q", k, el.Plain)
		}
	}
}

// --- helpers -------------------------------------------------------------

func withOff(keys ...ElementKey) Settings {
	set := Defaults()
	for _, k := range keys {
		set.Elements[k] = false
	}
	return set
}

func indexOf(row Row, k ElementKey) int {
	for i, el := range row.Elements {
		if el.Key == k {
			return i
		}
	}
	return -1
}

func assertOrdered(t *testing.T, row Row) {
	t.Helper()
	pos := map[ElementKey]int{}
	for i, k := range Order() {
		pos[k] = i
	}
	last := -1
	for _, el := range row.Elements {
		p, ok := pos[el.Key]
		if !ok {
			t.Fatalf("row carries an element outside the fixed order: %q", el.Key)
		}
		if p <= last {
			t.Fatalf("element %q is out of order", el.Key)
		}
		last = p
	}
}

// assertNoPlaceholder is ac-12's negative half: a dropped element leaves no
// empty text, no repeated separator and no trailing or leading one.
func assertNoPlaceholder(t *testing.T, row Row) {
	t.Helper()
	for _, el := range row.Elements {
		if strings.TrimSpace(el.Plain) == "" {
			t.Fatalf("element %q rendered an empty plain form", el.Key)
		}
	}
	plain := row.Plain()
	if strings.Contains(plain, Separator+Separator) {
		t.Fatalf("row carries a doubled separator: %q", plain)
	}
	if strings.HasPrefix(plain, Separator) || strings.HasSuffix(plain, Separator) {
		t.Fatalf("row carries a dangling separator: %q", plain)
	}
	for _, bad := range []string{"-", "n/a", "?%", "%%"} {
		for _, el := range row.Elements {
			if el.Plain == bad {
				t.Fatalf("element %q rendered the placeholder %q", el.Key, bad)
			}
		}
	}
}

// stripANSI removes every CSI sequence, which is what a colour-blind reading
// of the row amounts to.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
