package board

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// fixture is a block with one intent in a lane (two lanes of one run), the
// head and readyAfter more READY intents, and parked rows under Later.
func fixture(readyAfter, parked int) *statusblock.Block {
	b := &statusblock.Block{
		Now: []statusblock.Row{
			{ID: "itd-2610031214560142", SpecID: "spc-2610031844142274", Bucket: "planned",
				Title: "The board shows a product thinker what waits on them and what comes next, in a few plain lines",
				Lane:  &statusblock.Lane{Run: "run-2610040000000001", Lane: "lane-1", Stage: "implement", Branch: "build/run-2610040000000001-lane-1", InFlight: true}},
			{ID: "itd-2610031214560142", SpecID: "spc-2610031844142274", Bucket: "planned",
				Title: "The board shows a product thinker what waits on them and what comes next, in a few plain lines",
				Lane:  &statusblock.Lane{Run: "run-2610040000000001", Lane: "lane-2", Stage: "worktree"}},
			{ID: "itd-2610031215002409", SpecID: "spc-2610031844158884", Bucket: "planned", NextUp: true,
				Title: "A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session"},
		},
		Order: statusblock.OrderPick,
	}
	titles := []string{
		"When abcd sets up a project it asks the owner whether abcd keeps its records there",
		"Connecting a model service works out of the box",
	}
	for i := 0; i < readyAfter; i++ {
		t := fmt.Sprintf("Ready intent number %d", i+1)
		if i < len(titles) {
			t = titles[i]
		}
		b.Next = append(b.Next, statusblock.Row{ID: fmt.Sprintf("itd-%d", 100+i), SpecID: fmt.Sprintf("spc-%d", 100+i), Bucket: "planned", Title: t})
	}
	for i := 0; i < parked; i++ {
		b.Later = append(b.Later, statusblock.Row{ID: fmt.Sprintf("itd-%d", 500+i), Bucket: "drafts", Title: "parked"})
	}
	return b
}

// fullRows are the facilitator's labelled rows today's board carries.
func fullRows() []Row {
	return []Row{
		{Label: "git repo", Text: "yes"},
		{Label: "record", Text: "yes"},
		{Label: "work tiers", Text: "development, work, work.local"},
		{Label: "presence", Text: "abcd-managed · abcd · main · itd 4 · iss 0"},
		{Label: "peers", Text: "2 records differing here across 1 live peer — abcd peers"},
		{Label: "inbox", Text: "1 report waiting — `abcd inbox`"},
		{Label: "oracle", Text: "routing table accepted; * marks the row that applies", Items: []string{"intent-auditor  bundled=frontier*"}},
		{Label: "reviews", Text: "1 review folder, 0 past 20 commits since the pin on main",
			Items: []string{"3      abc1234   2026-09-01-spc-7-plan", "receipts: 1 release receipt — the release it gated is 0 commits behind main; a receipt is not re-run, and --json lists each"}},
	}
}

func check(t *testing.T, name string, got []string, want string) {
	t.Helper()
	if g := strings.Join(got, "\n") + "\n"; g != want {
		t.Errorf("%s:\n--- got\n%s--- want\n%s", name, g, want)
	}
}

// TestProductViewGolden is A1 in the core: at 80 columns and Mono, the label,
// one building line per intent, three next lines (the head first) and the
// count line, with no record id, lane, target, command word or owed answer.
func TestProductViewGolden(t *testing.T) {
	f := Frame{View: Product, Form: Text, Width: 80, Rung: Mono}
	check(t, "three ready, 40 parked", Render(Input{Status: fixture(2, 40)}, f), `view for the product thinker
● building: The board shows a product thinker what waits on them and what comes…
○ next: A 'what next?' menu under the board offers the next step, in a Terminal…
○ next: When abcd sets up a project it asks the owner whether abcd keeps its re…
○ next: Connecting a model service works out of the box
• no more ready, 40 parked
`)
	check(t, "fourteen ready, 40 parked", Render(Input{Status: fixture(13, 40)}, f), `view for the product thinker
● building: The board shows a product thinker what waits on them and what comes…
○ next: A 'what next?' menu under the board offers the next step, in a Terminal…
○ next: When abcd sets up a project it asks the owner whether abcd keeps its re…
○ next: Connecting a model service works out of the box
• 11 more ready, 40 parked
`)
	check(t, "nothing", Render(Input{Status: &statusblock.Block{}}, f), `view for the product thinker
● building: nothing right now
○ next: nothing is ready
• no more ready, nothing parked
`)
	handle := regexp.MustCompile(`\b(itd|iss|spc|adr|run|lane)-[0-9]`)
	command := regexp.MustCompile(`\babcd (build|capture|intent|implement|ahoy|peers|inbox|mode|spec|drain|lint|decide|update)\b`)
	for _, l := range Render(Input{Status: fixture(13, 40)}, f) {
		if handle.MatchString(l) || command.MatchString(l) || strings.Contains(l, "waiting on") || strings.Contains(l, "target") {
			t.Errorf("the product thinker's view carries a handle, a command or an owed answer: %q", l)
		}
	}
}

// TestFacilitatorViewGolden is A5 in the core: the label, then today's full
// board, every row present, each Now and Next row with its spec, and the lane
// whose branch exists and whose spec is open marked in flight.
func TestFacilitatorViewGolden(t *testing.T) {
	f := Frame{View: Facilitator, Form: Text, Width: 80, Rung: Mono}
	check(t, "facilitator", Render(Input{Dir: "~/code/abcd", Status: fixture(2, 40), Rows: fullRows()}, f), `view for the facilitator
abcd — ~/code/abcd
  git repo:   yes
  record:     yes
  work tiers: development, work, work.local
  presence:   abcd-managed · abcd · main · itd 4 · iss 0
  peers:      2 records differing here across 1 live peer — abcd peers
  inbox:      1 report waiting — `+"`abcd inbox`"+`
  oracle:     routing table accepted; * marks the row that applies
    intent-auditor  bundled=frontier*
  reviews:    1 review folder, 0 past 20 commits since the pin on main
    3      abc1234   2026-09-01-spc-7-plan
    receipts: 1 release receipt — the release it gated is 0 commits behind main;
        a receipt is not re-run, and --json lists each
  status:     Now 3 · Next 2 · Later 40
    Now:
      itd-2610031214560142  spc-2610031844142274  The board shows a product
          thinker what waits on them and what comes next, in a few plain lines
          [lane-1: implement (run-2610040000000001); in flight]
      itd-2610031214560142  spc-2610031844142274  The board shows a product
          thinker what waits on them and what comes next, in a few plain lines
          [lane-2: worktree (run-2610040000000001)]
      itd-2610031215002409  spc-2610031844158884  A 'what next?' menu under the
          board offers the next step, in a Terminal and in a Claude Code session
          [next up]
    Next:
      itd-100  spc-100  When abcd sets up a project it asks the owner whether
          abcd keeps its records there
      itd-101  spc-101  Connecting a model service works out of the box
    Later: 40 intents
`)
}

// TestMarkdownFormGolden is A3's form: a list, never a table, no escape byte,
// titles whole, in both views.
func TestMarkdownFormGolden(t *testing.T) {
	in := Input{Dir: "~/code/abcd", Status: fixture(2, 40), Rows: fullRows()[:2]}
	product := Render(in, Frame{View: Product, Form: Markdown, Width: 80, Rung: TrueColor})
	check(t, "product markdown", product, `view for the product thinker

- ● building: The board shows a product thinker what waits on them and what comes next, in a few plain lines
- ○ next: A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session
- ○ next: When abcd sets up a project it asks the owner whether abcd keeps its records there
- ○ next: Connecting a model service works out of the box
- • no more ready, 40 parked
`)
	facil := Render(in, Frame{View: Facilitator, Form: Markdown, Width: 80, Rung: TrueColor, ASCII: true})
	check(t, "facilitator markdown", facil, `view for the facilitator

- abcd — ~/code/abcd
  - git repo:   yes
  - record:     yes
  - status:     Now 3 · Next 2 · Later 40
    - Now:
      - itd-2610031214560142  spc-2610031844142274  The board shows a product thinker what waits on them and what comes next, in a few plain lines  [lane-1: implement (run-2610040000000001); in flight]
      - itd-2610031214560142  spc-2610031844142274  The board shows a product thinker what waits on them and what comes next, in a few plain lines  [lane-2: worktree (run-2610040000000001)]
      - itd-2610031215002409  spc-2610031844158884  A 'what next?' menu under the board offers the next step, in a Terminal and in a Claude Code session  [next up]
    - Next:
      - itd-100  spc-100  When abcd sets up a project it asks the owner whether abcd keeps its records there
      - itd-101  spc-101  Connecting a model service works out of the box
    - Later: 40 intents
`)
	for _, l := range append(product, facil...) {
		if strings.ContainsRune(l, '\x1b') || strings.HasPrefix(strings.TrimSpace(l), "|") {
			t.Errorf("the markdown form carries an escape byte or a table row: %q", l)
		}
	}
}

// TestFenceSafeMarkdownTitles: every markdown line begins with the label, a
// blank, or a list item, so a title of backticks and tildes can never open or
// close the page's fence, and a title cannot add a line.
func TestFenceSafeMarkdownTitles(t *testing.T) {
	b := fixture(2, 0)
	b.Now[2].Title = "```\n~~~ close the fence ``` and ~~~"
	b.Next[0].Title = "~~~"
	for _, v := range []View{Product, Facilitator} {
		rows := []Row{{Label: "git repo", Text: "```"}, {Label: "presence", Text: "x\n```", Items: []string{"y\n~~~ an item"}}}
		// The lines as a reader sees them: an element carrying a newline is
		// two lines on the page.
		lines := strings.Split(strings.Join(Render(Input{Dir: "```", Status: b, Rows: rows}, Frame{View: v, Form: Markdown}), "\n"), "\n")
		if !strings.Contains(strings.Join(lines, "\n"), "close the fence") {
			t.Fatalf("view %d drew no title to hold the rule over:\n%s", v, strings.Join(lines, "\n"))
		}
		for i, l := range lines {
			trimmed := strings.TrimLeft(l, " ")
			if i == 0 || l == "" || strings.HasPrefix(trimmed, "- ") {
				continue
			}
			t.Errorf("view %d line %d %q is neither the label, a blank nor a list item", v, i, l)
		}
	}
}

// TestLabelPaintsOnlyAtTrueColor is decision 10: the label is painted in the
// role's badge pair at TrueColor, is its words alone at Ansi256 and Ansi16
// (no 4-bit stand-in), and is never painted at Mono or in markdown.
func TestLabelPaintsOnlyAtTrueColor(t *testing.T) {
	in := Input{Status: fixture(2, 1)}
	for _, tc := range []struct {
		v     View
		state statusline.State
	}{{Product, statusline.StateProductThinker}, {Facilitator, statusline.StateFacilitator}} {
		if got, want := Render(in, Frame{View: tc.v, Width: 80, Rung: TrueColor})[0], statusline.PaintRole(tc.state, label(tc.v)); got != want {
			t.Errorf("view %d at TrueColor: label %q, want the role pair %q", tc.v, got, want)
		}
		for _, r := range []Rung{Ansi256, Ansi16, Mono} {
			if got := Render(in, Frame{View: tc.v, Width: 80, Rung: r})[0]; got != label(tc.v) {
				t.Errorf("view %d at rung %d: label %q, want its words unpainted", tc.v, r, got)
			}
		}
		if got := Render(in, Frame{View: tc.v, Form: Markdown, Rung: TrueColor})[0]; got != label(tc.v) {
			t.Errorf("view %d in markdown: label %q, want its words unpainted", tc.v, got)
		}
	}
}

// TestBoardWithoutColourKeepsWordsAndSymbols is A6 in the core: at Mono no
// escape byte is drawn and each state's word stands beside its symbol; with
// no UTF-8 locale the plain-text symbols and ellipsis replace them.
func TestBoardWithoutColourKeepsWordsAndSymbols(t *testing.T) {
	in := Input{Status: fixture(2, 3)}
	mono := Render(in, Frame{Width: 80, Rung: Mono})
	painted := Render(in, Frame{Width: 80, Rung: Ansi16})
	strip := regexp.MustCompile("\x1b\\[[0-9;]*m")
	for i := range mono {
		if strings.ContainsRune(mono[i], '\x1b') {
			t.Errorf("a Mono line carries an escape byte: %q", mono[i])
		}
		if got := strip.ReplaceAllString(painted[i], ""); got != mono[i] {
			t.Errorf("line %d painted, unpainted, is %q; want the Mono line %q", i, got, mono[i])
		}
	}
	joined := strings.Join(mono, "\n")
	for _, want := range []string{"● building:", "○ next:", "• "} {
		if !strings.Contains(joined, want) {
			t.Errorf("the Mono board lacks %q:\n%s", want, joined)
		}
	}
	ascii := strings.Join(Render(in, Frame{Width: 80, Rung: Mono, ASCII: true}), "\n")
	for _, want := range []string{"* building:", "o next:", "- no more ready, 3 parked", "..."} {
		if !strings.Contains(ascii, want) {
			t.Errorf("the plain-text board lacks %q:\n%s", want, ascii)
		}
	}
	if strings.ContainsAny(ascii, "●○•…") {
		t.Errorf("the plain-text board carries a UTF-8 symbol:\n%s", ascii)
	}
}

// TestNarrowWindowsKeepEveryLineInside: below the width a product line needs,
// every line, the label and the count included, still fits the window, and a
// title too long for the room keeps its ellipsis rather than vanishing.
func TestNarrowWindowsKeepEveryLineInside(t *testing.T) {
	strip := regexp.MustCompile("\x1b\\[[0-9;]*m")
	for _, w := range []int{12, 20, 30} {
		for _, r := range []Rung{Mono, TrueColor} {
			lines := Render(Input{Status: fixture(2, 40)}, Frame{View: Product, Width: w, Rung: r})
			for _, l := range lines {
				if n := textwidth.Columns(strip.ReplaceAllString(l, "")); n > w {
					t.Errorf("width %d rung %d: a line takes %d columns: %q", w, r, n, l)
				}
			}
		}
	}
	if got := textwidth.Fit("a title", 1, "…"); got != "…" {
		t.Errorf("Fit with room for the ellipsis alone = %q, want the ellipsis", got)
	}
}

// TestBoardFitsTheWindow is A7: a 200-character title and a title of East
// Asian wide glyphs, drawn at 80 columns in both views and both colour
// extremes, leave no line wider than 80 once escapes are stripped.
func TestBoardFitsTheWindow(t *testing.T) {
	b := fixture(2, 1)
	b.Now[0].Title = strings.Repeat("a long title word ", 12)[:200]
	b.Next[0].Title = strings.Repeat("広い文字の題名", 12)
	strip := regexp.MustCompile("\x1b\\[[0-9;]*m")
	for _, v := range []View{Product, Facilitator} {
		for _, r := range []Rung{Mono, TrueColor} {
			lines := Render(Input{Dir: "~/code/abcd", Status: b, Rows: fullRows()}, Frame{View: v, Width: 80, Rung: r})
			if !strings.Contains(strings.Join(lines, "\n"), "広い文字") {
				t.Fatalf("view %d rung %d drew no wide title to measure:\n%s", v, r, strings.Join(lines, "\n"))
			}
			for _, l := range lines {
				if n := textwidth.Columns(strip.ReplaceAllString(l, "")); n > 80 {
					t.Errorf("view %d rung %d: a line takes %d columns: %q", v, r, n, l)
				}
			}
		}
	}
}
