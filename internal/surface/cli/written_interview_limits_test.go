package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
)

// limitsRoots is a repository and a home, each with the config.json given
// ("" for none).
func limitsRoots(t *testing.T, repo, machine string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	write := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if repo != "" {
		write(filepath.Join(r.Repo, ".abcd", "config.json"), repo)
	}
	if machine != "" {
		write(abcdhome.Path(r.Home, "config.json"), machine)
	}
	return r
}

// TestWrittenInterviewTakesTheResolvedLimits: the AI-written interviews'
// front door hands the interview the limits in force, refuses a refused value
// before anything runs naming the key, the value, the bounds and the file,
// and fails open and loud on a layer it cannot read (itd-2610301914350541
// decisions 7 and 8).
func TestWrittenInterviewTakesTheResolvedLimits(t *testing.T) {
	var errOut bytes.Buffer
	l, err := questionLimitsFor(limitsRoots(t, `{"question": {"rows": 40}}`, `{"question": {"label_words": 4}}`), &errOut)
	if err != nil || l.Rows != 40 || l.LabelWords != 4 || errOut.Len() != 0 {
		t.Errorf("in force: limits %v, err %v, stderr %q", l.Configurable(), err, errOut.String())
	}

	_, err = questionLimitsFor(limitsRoots(t, `{"question": {"rows": 100}}`, ""), &errOut)
	if err == nil || err.Error() != ".abcd/config.json (repo layer): question.rows is 100: outside 24 to 60" {
		t.Errorf("refused: err = %v", err)
	}

	r := limitsRoots(t, "", `{"pace": {"work_minutes": 5}}`)
	if err := os.Chmod(abcdhome.Path(r.Home, "config.json"), 0o666); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	l, err = questionLimitsFor(r, &errOut)
	if err != nil || !reflect.DeepEqual(l, question.Default) {
		t.Errorf("unread: limits %v, err %v; want the defaults and no error", l.Configurable(), err)
	}
	line := errOut.String()
	if !strings.HasPrefix(line, "abcd guard: NOT CHECKED — ") || !strings.Contains(line, abcdhome.Display("config.json")) ||
		!strings.HasSuffix(line, "The question runs UNGATED.\n") {
		t.Errorf("unread: stderr %q, want the gate's fail-open line naming the machine file", line)
	}
}
