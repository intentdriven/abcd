package decisions_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/gittest"
)

// liveHistory is the set whose mentions of the retired log stay word for word
// (spc-2610110931381316, "What is left as history"): each is a verbatim record
// of a past state. The decision records are the log's own entries, migrated
// verbatim (A8).
var liveHistory = []string{
	"CHANGELOG.md",
	".abcd/work/reviews/",
	".abcd/development/research/data/",
	"internal/core/guard/testdata/corpus/repo-mined.txt",
	decisions.RelDir + "/",
}

// liveSubjects are the records whose subject is the log itself: they talk about
// it rather than cite an entry of it, so rewriting them would change what they
// say.
var liveSubjects = []string{
	"itd-2609151138388536", "adr-2610110925465593", "adr-2609151138420062",
	"spc-2610110931381316", "iss-2609100507439414", "iss-2608271804494867",
	"iss-2608291814575169",
}

// liveSurfaces are the paths a reader takes as the repository's current
// behaviour or conventions: code, the front doors, the user docs, the root
// prose, CI and the scripts, the build, the attributes, the configurations, the
// brief and the principles.
var liveSurfaces = []string{
	"cmd/", "internal/", "evals/", "commands/", "agents/", "skills/", "hooks/",
	"docs/", ".github/", ".githooks/", "scripts/", "Makefile", ".gitattributes", ".gitignore",
	".abcd/README.md", ".abcd/work/CONTEXT.md",
	".abcd/development/brief/", ".abcd/development/principles/",
}

var (
	// markdownLinkToLogRe is a Markdown link whose target is the log.
	markdownLinkToLogRe = regexp.MustCompile(`\]\([^)\s]*DECISIONS\.md[^)\s]*\)`)
	// datedCitationRes are the shapes that cite one dated entry of the log: the
	// log followed by a line number or a date, or a date followed closely by
	// the log.
	datedCitationRes = []*regexp.Regexp{
		regexp.MustCompile(`DECISIONS(?:\.md)?` + "`?" + `:[0-9]+`),
		regexp.MustCompile(`DECISIONS(?:\.md)?` + "`?" + `[,;]?\s*\(?(?:on |under |of |entry of |entries of )?20[0-9]{2}-[0-9]{2}-[0-9]{2}`),
		regexp.MustCompile(`20[0-9]{2}-[0-9]{2}-[0-9]{2}[^.\n]{0,60}\bDECISIONS\b`),
	}
	// liveGateRe names the retired decisions-append gate or its make target.
	liveGateRe = regexp.MustCompile(`decisions-append|lint-decisions`)
)

// TestNoLiveDecisionsLogPath holds abcd's tree to the retirement of its single
// decision log (itd-2609151138388536, criteria A3 and A7): the log is gone; no
// live surface names its path; no file outside the history set links to it or
// cites a dated entry of it, since each such citation now points at the entry's
// own record; and no live surface names the decisions-append gate or its make
// target. The decisions package itself is exempt (it holds FrozenLog, a managed
// repository's frozen history, read-only), and so is a line that says "retired"
// or "frozen", the per-line allow-word convention record-lint's banned tokens
// use.
func TestNoLiveDecisionsLogPath(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(decisions.FrozenLog))); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s exists in abcd's own tree (lstat err=%v); its entries are records under %s/ now",
			decisions.FrozenLog, err, decisions.RelDir)
	}

	cmd := exec.Command("git", "-C", root, "ls-files", "-z")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable in %s: %v", root, err)
	}
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if rel == "" || inHistory(rel) || strings.HasPrefix(rel, "internal/core/decisions/") {
			continue
		}
		if rel == decisions.FrozenLog {
			t.Errorf("git tracks %s; abcd's own decision log is retired", rel)
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // deleted in the working tree, not yet staged
			}
			t.Fatalf("read %s: %v", rel, err)
		}
		if bytes.IndexByte(body, 0) >= 0 {
			continue // a binary file
		}
		live := onLiveSurface(rel)
		for i, line := range strings.Split(string(body), "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "retired") || strings.Contains(lower, "frozen") {
				continue
			}
			at := rel + ":" + strconv.Itoa(i+1)
			if live && strings.Contains(line, decisions.FrozenLog) {
				t.Errorf("%s names %s as a live path: %s", at, decisions.FrozenLog, clip(line))
			}
			if markdownLinkToLogRe.MatchString(line) {
				t.Errorf("%s links to the retired log: %s", at, clip(line))
			}
			for _, re := range datedCitationRes {
				if re.MatchString(line) {
					t.Errorf("%s cites a dated entry of the retired log; cite the entry's own record: %s", at, clip(line))
					break
				}
			}
			if live && liveGateRe.MatchString(line) {
				t.Errorf("%s names the retired decisions-append gate as live: %s", at, clip(line))
			}
		}
	}
}

func inHistory(rel string) bool {
	for _, p := range liveHistory {
		if rel == p || strings.HasPrefix(rel, p) {
			return true
		}
	}
	base := path.Base(rel)
	for _, id := range liveSubjects {
		if strings.HasPrefix(base, id+"-") {
			return true
		}
	}
	return false
}

func onLiveSurface(rel string) bool {
	if !strings.Contains(rel, "/") && strings.HasSuffix(rel, ".md") {
		return true // the root prose files
	}
	if strings.HasPrefix(rel, ".abcd/") && !strings.Contains(strings.TrimPrefix(rel, ".abcd/"), "/") && strings.HasSuffix(rel, ".json") {
		return true // the .abcd/*.json configurations
	}
	for _, p := range liveSurfaces {
		if rel == p || strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
