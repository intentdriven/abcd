package ask

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/term"
)

// env is a fixed environment for a getenv seam.
func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

// TestNumberedReaderIsSelected holds the selection of the answer loop's mode
// (spc-2610030911534855, "The numbered fallback"; open question 2 decided):
// arrows by default; numbered by the interview.list setting, by
// ABCD_ACCESSIBLE non-empty, or by ACCESSIBLE non-empty, ABCD_ACCESSIBLE named
// when both are set; and always under TERM=dumb.
func TestNumberedReaderIsSelected(t *testing.T) {
	for _, tc := range []struct {
		name    string
		getenv  func(string) string
		list    string
		want    Mode
		because string
	}{
		{"default", env("TERM", "xterm-256color"), layered.InterviewListArrows, Arrows, ""},
		{"no setting read", env("TERM", "xterm-256color"), "", Arrows, ""},
		{"the setting", env("TERM", "xterm-256color"), layered.InterviewListNumbered, Numbered, "interview.list"},
		{"ABCD_ACCESSIBLE", env("TERM", "xterm", "ABCD_ACCESSIBLE", "1"), layered.InterviewListArrows, Numbered, "ABCD_ACCESSIBLE"},
		{"ACCESSIBLE", env("TERM", "xterm", "ACCESSIBLE", "1"), layered.InterviewListArrows, Numbered, "ACCESSIBLE"},
		{"both, abcd's own named", env("TERM", "xterm", "ACCESSIBLE", "1", "ABCD_ACCESSIBLE", "yes"), "", Numbered, "ABCD_ACCESSIBLE"},
		{"empty is unset", env("TERM", "xterm", "ACCESSIBLE", "", "ABCD_ACCESSIBLE", ""), "", Arrows, ""},
		{"TERM=dumb", env("TERM", "dumb"), layered.InterviewListArrows, Numbered, "TERM=dumb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, why := SelectMode(tc.getenv, tc.list)
			if m != tc.want {
				t.Errorf("mode %v, want %v", m, tc.want)
			}
			if !strings.Contains(why, tc.because) {
				t.Errorf("reason %q does not name %q", why, tc.because)
			}
		})
	}
}

// roots lays a home whose ~/.abcd.noindex/config.json holds machine (none when
// empty), and a repository whose .abcd/config.json holds repo (none when
// empty), for Put to read interview.list from.
func roots(t *testing.T, machine, repo string) layered.Roots {
	t.Helper()
	base := t.TempDir()
	r := layered.Roots{Repo: filepath.Join(base, "repo"), Home: filepath.Join(base, "hm-e8term2")}
	for _, f := range []struct{ path, body string }{
		{abcdhome.Path(r.Home, "config.json"), machine},
		{filepath.Join(r.Repo, ".abcd", "config.json"), repo},
	} {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
			t.Fatal(err)
		}
		if f.body == "" {
			continue
		}
		if err := os.WriteFile(f.path, []byte(f.body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

const numberedSetting = `{"interview": {"list": "numbered"}}`

// pipeIn is a non-terminal input holding s: the numbered reader takes whole
// lines from it, and raw mode is refused on it.
func pipeIn(t *testing.T, s string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(s); err != nil {
		t.Fatal(err)
	}
	w.Close()
	t.Cleanup(func() { r.Close() })
	return r
}

// lineCount counts the lines of out holding sub.
func lineCount(out, sub string) int {
	n := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, sub) {
			n++
		}
	}
	return n
}

// TestDumbTerminalAndRefusedRawFallToNumbered holds the modes no keyboard mode
// is possible in: TERM=dumb, and a raw mode the terminal refuses, each fall to
// the numbered reader with one line saying so, never a failed interview.
func TestDumbTerminalAndRefusedRawFallToNumbered(t *testing.T) {
	for _, tc := range []struct {
		name, term, says string
	}{
		{"TERM=dumb", "dumb", "TERM=dumb"},
		{"raw mode refused", "xterm-256color", "refused the arrow-key list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			tm := Terminal{In: pipeIn(t, "2\n"), Out: &out, Getenv: env("TERM", tc.term), Mode: term.Mono, Roots: roots(t, "", "")}
			got, err := tm.Put(fixture(t, "key-home"))
			if err != nil {
				t.Fatalf("a fallback failed the interview: %v\n%s", err, out.String())
			}
			if len(got) != 1 || got[0].Value != "file" {
				t.Errorf("answers %+v, want key-home = file by its number", got)
			}
			if n := lineCount(out.String(), tc.says); n != 1 {
				t.Errorf("%d lines say %q, want one:\n%s", n, tc.says, out.String())
			}
			if n := lineCount(out.String(), "numbered"); n != 1 {
				t.Errorf("%d lines name the numbered list, want one:\n%s", n, out.String())
			}
			if strings.ContainsRune(out.String(), 0x1b) {
				t.Errorf("the numbered fallback wrote an escape byte:\n%q", out.String())
			}
		})
	}
}

// TestNumberedReaderPagesAndNarrows holds the numbered mode: whole lines, the
// hint, 20 at a time with n and p paging, typed text narrowing and redrawing
// numbered, a number out of range named, and the full list's numbers kept so a
// number chooses what it names.
func TestNumberedReaderPagesAndNarrows(t *testing.T) {
	a := longList(300)
	choices := a.Questions[0].List.Choices
	var out bytes.Buffer
	in := pipeIn(t, "n\nn\np\nclaude model 4\n999\n\n145\n")
	tm := Terminal{In: in, Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
	got, err := tm.Put(a)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if len(got) != 1 || got[0].Value != choices[144].Value {
		t.Errorf("answers %+v, want number 145 (%q)", got, choices[144].Value)
	}
	s := out.String()
	for _, want := range []string{
		"Type a number, or part of a name to narrow the list, then Enter.",
		" 20. GPT Model 19",
		" 21. GPT Model 20",
		" 41. GPT Model 40",
		"filter: claude model 4",
		"105. Claude Model 4",
		"141. Claude Model 40",
		"no number 999",
		"301. Decide later",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the numbered reader never wrote %q:\n%s", want, s)
		}
	}
	// The first page is 1 to 20; n, n and p draw 21 to 40, 41 to 60, then 21
	// to 40 again.
	if n := strings.Count(s, " 21. GPT Model 20\n"); n != 2 {
		t.Errorf("n, n, p should draw the second page twice; drew it %d times:\n%s", n, s)
	}
	if n := strings.Count(s, " 61. GPT Model 60\n"); n != 0 {
		t.Errorf("a page drew more than 20 names:\n%s", s)
	}
	if strings.ContainsRune(s, 0x1b) {
		t.Errorf("the numbered reader wrote an escape byte under Mono:\n%q", s)
	}
}

// TestNumberedReaderRefusesAClosedInput holds that input ending before an
// answer is an error, never a default taken silently.
func TestNumberedReaderRefusesAClosedInput(t *testing.T) {
	var out bytes.Buffer
	tm := Terminal{In: pipeIn(t, "zzz\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
	got, err := tm.Put(fixture(t, "key-home"))
	if err == nil {
		t.Fatalf("answers %+v from an input that closed unanswered", got)
	}
	if !strings.Contains(out.String(), `nothing matches "zzz"`) {
		t.Errorf("a filter nothing matches is not named:\n%s", out.String())
	}
}

// TestPutReadsTheListSetting holds that Put takes interview.list from the
// layered configuration itself: the machine's numbered selects the numbered
// reader with no line said (it was asked for). A setting the reader refuses
// never fails the interview, and never lets a repository decide how the
// person is asked: a fault the repository layer holds (the key set there, a
// misspelt key under interview, a malformed file) is passed over for the
// machine's own setting, with one line naming the refusal; a fault in the
// machine's own file gives the numbered reader, with one line naming it.
func TestPutReadsTheListSetting(t *testing.T) {
	t.Run("the machine's numbered", func(t *testing.T) {
		var out bytes.Buffer
		tm := Terminal{In: pipeIn(t, "1\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
		got, err := tm.Put(fixture(t, "key-home"))
		if err != nil {
			t.Fatalf("%v\n%s", err, out.String())
		}
		if len(got) != 1 || got[0].Value != "keychain" {
			t.Errorf("answers %+v, want keychain by its number", got)
		}
		if !strings.Contains(out.String(), NumberedHint) {
			t.Errorf("the machine's numbered did not select the numbered reader:\n%s", out.String())
		}
		if strings.Contains(out.String(), "abcd:") {
			t.Errorf("a numbered list that was asked for says why:\n%s", out.String())
		}
	})
	for _, tc := range []struct {
		name, machine, repo string
		names               string // what the one refusal line names
		arrows              bool   // the machine's mode, arrows, is kept
	}{
		{"a repository's numbered keeps the machine's arrows", "", numberedSetting, "interview.list", true},
		{"a repository's misspelt key keeps the machine's arrows", "", `{"interview": {"lst": "numbered"}}`, "interview.lst", true},
		{"a repository's malformed file keeps the machine's arrows", "", `{"interview": `, ".abcd/config.json", true},
		{"a repository's arrows keeps the machine's numbered", numberedSetting, `{"interview": {"list": "arrows"}}`, "interview.list", false},
		{"a malformed machine file gives numbered", `{"interview": `, "", "~/.abcd.noindex/config.json", false},
		{"a machine value outside the two gives numbered", `{"interview": {"list": "tabs"}}`, "", "~/.abcd.noindex/config.json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			tm := Terminal{In: pipeIn(t, "1\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, tc.machine, tc.repo)}
			got, err := tm.Put(fixture(t, "key-home"))
			if err != nil {
				t.Fatalf("a refused setting failed the interview: %v\n%s", err, out.String())
			}
			if len(got) != 1 || got[0].Value != "keychain" {
				t.Errorf("answers %+v, want keychain by its number", got)
			}
			s := out.String()
			var said []string
			for _, l := range strings.Split(s, "\n") {
				if strings.HasPrefix(l, "abcd: ") && !strings.Contains(l, "refused the arrow-key list") {
					said = append(said, l)
				}
			}
			if len(said) != 1 || !strings.Contains(said[0], tc.names) {
				t.Errorf("want one line naming the refusal (%q), got %q:\n%s", tc.names, said, s)
			}
			// The input is a pipe, so an arrow-key list is refused raw mode
			// and says so: that line is how the test sees arrows chosen.
			if tried := lineCount(s, "refused the arrow-key list") == 1; tried != tc.arrows {
				t.Errorf("arrow-key list tried %v, want %v (the machine's mode):\n%s", tried, tc.arrows, s)
			}
		})
	}
}

// TestNumberedFilterEchoIsSanitised holds that the numbered reader's echo of
// the typed line passes termsafe.Sanitize, as the arrow-key list's does: a
// line carrying an erase-screen, a cursor-home, a C1 control and a bare
// carriage return reaches the screen as visible '?', never as live control
// bytes that could forge what the person reads.
func TestNumberedFilterEchoIsSanitised(t *testing.T) {
	var out bytes.Buffer
	in := pipeIn(t, "\x1b[2J\x1b[Hforged\u0085\rline\n")
	tm := Terminal{In: in, Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
	_, _ = tm.Put(fixture(t, "key-home"))
	s := out.String()
	if strings.ContainsRune(s, 0x1b) {
		t.Errorf("the echo wrote an escape byte:\n%q", s)
	}
	if strings.ContainsRune(s, 0x85) {
		t.Errorf("the echo wrote a C1 control:\n%q", s)
	}
	if strings.ContainsRune(s, '\r') {
		t.Errorf("the echo wrote a bare carriage return:\n%q", s)
	}
	if !strings.Contains(s, "filter: ?[2J?[Hforged??line") {
		t.Errorf("the echo does not show the typed line made visible:\n%q", s)
	}
}

// TestNumberedLetterNarrowsWhenThereAreNoPages holds "n" and "p" to what the
// screen says of them: they page only a list long enough to have pages, whose
// paging line says a lone n or p pages; on a list with one page they narrow
// like any other text, so no name is out of reach of its first letter.
func TestNumberedLetterNarrowsWhenThereAreNoPages(t *testing.T) {
	var out bytes.Buffer
	tm := Terminal{In: pipeIn(t, "p\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
	_, _ = tm.Put(fixture(t, "key-home"))
	if s := out.String(); !strings.Contains(s, "filter: p") {
		t.Errorf("a lone p on a one-page list did not narrow:\n%s", s)
	}

	out.Reset()
	tm = Terminal{In: pipeIn(t, "n\n"), Out: &out, Getenv: env("TERM", "xterm"), Mode: term.Mono, Roots: roots(t, numberedSetting, "")}
	_, _ = tm.Put(longList(300))
	s := out.String()
	if strings.Contains(s, "filter: n") {
		t.Errorf("a lone n on a paged list narrowed instead of paging:\n%s", s)
	}
	if !strings.Contains(s, "a lone n or p pages") {
		t.Errorf("the paging line does not say a lone n or p pages:\n%s", s)
	}
}
