package source

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The fixtures below are invented: no real source, author or project is named.
const (
	confTitle  = "Quiet Harbour Working Notes"
	confAlias  = "harbourwatch"
	confAuthor = "Okonkwo"
	pubTitle   = "An Open Survey of Ledger Formats"
)

var fixedNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

// hermetic isolates every test from the real home, the real corpus and the real
// git identity: HOME is a temp dir (so the default corpus is a temp path), and the
// corpus commits carry a fixture identity.
func hermetic(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	gittest.Env(t)
	t.Setenv("GIT_AUTHOR_NAME", "Alice Example")
	t.Setenv("GIT_AUTHOR_EMAIL", "alice@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Alice Example")
	t.Setenv("GIT_COMMITTER_EMAIL", "alice@example.com")
	return home
}

// newCorpus initialises a corpus under the temp home and returns its path.
func newCorpus(t *testing.T) string {
	t.Helper()
	home := hermetic(t)
	dir := abcdhome.Path(home, "sources")
	if _, err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return dir
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func addConfidential(t *testing.T, corpus string, banAuthors bool) {
	t.Helper()
	src := t.TempDir()
	_, err := Add(AddRequest{
		Corpus: corpus, Key: "conf2026a", Title: confTitle, Type: "report", Class: ClassConfidential,
		Authors:  []Name{{Family: confAuthor, Given: "Adaeze"}},
		Keywords: []string{"harbours", "ledgers"}, Aliases: []string{confAlias}, BanAuthors: banAuthors,
		Original: writeFile(t, src, "notes.pdf", "%PDF-1.4 binary-ish"),
		Text:     writeFile(t, src, "notes.txt", "extracted body of the notes\n"),
	})
	if err != nil {
		t.Fatalf("Add confidential: %v", err)
	}
}

func addPublic(t *testing.T, corpus string) {
	t.Helper()
	src := t.TempDir()
	_, err := Add(AddRequest{
		Corpus: corpus, Key: "survey2026ledger", Title: pubTitle, Type: "article-journal", Class: ClassPublic,
		Original: writeFile(t, src, "survey.md", "# survey\nbody\n"),
	})
	if err != nil {
		t.Fatalf("Add public: %v", err)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func readEntries(t *testing.T, corpus string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(corpus, "sources.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("sources.json is not a CSL-JSON array: %v", err)
	}
	return out
}

// TestAddConfidentialLandsUnderItsClassFolder is AC1: the entry (with its custom
// block) lands in sources.json, the document and its extracted text land under
// confidential/<key>/, nothing lands under public/, and the corpus commits it.
func TestAddConfidentialLandsUnderItsClassFolder(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)

	entries := readEntries(t, corpus)
	if len(entries) != 1 || entries[0]["id"] != "conf2026a" || entries[0]["title"] != confTitle {
		t.Fatalf("entries = %v", entries)
	}
	custom, _ := entries[0]["custom"].(map[string]any)
	if custom["confidential"] != true || custom["permission_status"] != "no-public-citation" {
		t.Fatalf("custom block = %v", custom)
	}
	if _, ok := custom["keywords"]; !ok {
		t.Error("custom block has no keywords")
	}
	if _, ok := custom["aliases"]; !ok {
		t.Error("custom block has no aliases")
	}
	folder := filepath.Join(corpus, "confidential", "conf2026a")
	if b, err := os.ReadFile(filepath.Join(folder, "original.pdf")); err != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("original not stored: %v", err)
	}
	text, err := os.ReadFile(filepath.Join(folder, "text.md"))
	if err != nil || !strings.Contains(string(text), "extracted body") || !strings.Contains(string(text), "key: conf2026a") {
		t.Fatalf("text.md = %q %v", text, err)
	}
	if _, err := os.Stat(filepath.Join(corpus, "public", "conf2026a")); !os.IsNotExist(err) {
		t.Fatal("a confidential source landed under public/")
	}
	if st := git(t, corpus, "status", "--porcelain"); st != "" {
		t.Fatalf("corpus not committed:\n%s", st)
	}
	if log := git(t, corpus, "log", "--oneline"); !strings.Contains(log, "conf2026a") {
		t.Fatalf("no commit names the key:\n%s", log)
	}
}

// TestAddRefusals: every refusal writes nothing, and none quotes a confidential
// title or alias.
func TestAddRefusals(t *testing.T) {
	corpus := newCorpus(t)
	src := t.TempDir()
	pdf := writeFile(t, src, "x.pdf", "%PDF")
	base := AddRequest{Corpus: corpus, Key: "conf2026b", Title: confTitle, Class: ClassConfidential, Original: pdf,
		Text: writeFile(t, src, "x.txt", "t")}
	for _, tc := range []struct {
		name string
		mut  func(r *AddRequest)
		want error
	}{
		{"no class", func(r *AddRequest) { r.Class = "" }, ErrInvalidEntry},
		{"confidential yet citable", func(r *AddRequest) { r.Permission = PermissionCitable }, ErrInvalidEntry},
		{"unknown permission", func(r *AddRequest) { r.Permission = "whatever" }, ErrInvalidEntry},
		{"key names an alias", func(r *AddRequest) { r.Key = "harbourwatch2026"; r.Aliases = []string{confAlias} }, ErrIdentifyingKey},
		{"key names the author", func(r *AddRequest) { r.Key = "okonkwo2026"; r.Authors = []Name{{Family: confAuthor}} }, ErrIdentifyingKey},
		{"bad key", func(r *AddRequest) { r.Key = "../x" }, ErrInvalidEntry},
		{"binary original with no text", func(r *AddRequest) { r.Text = "" }, ErrInvalidEntry},
		{"fragment alias", func(r *AddRequest) { r.Aliases = []string{"hb"} }, ErrInvalidEntry},
		{"empty title", func(r *AddRequest) { r.Title = " " }, ErrInvalidEntry},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.mut(&r)
			_, err := Add(r)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			for _, leak := range []string{confTitle, confAlias} {
				if strings.Contains(err.Error(), leak) {
					t.Fatalf("refusal quotes %q: %v", leak, err)
				}
			}
			if len(readEntries(t, corpus)) != 0 {
				t.Fatal("a refused add wrote an entry")
			}
		})
	}
	addConfidential(t, corpus, false)
	r := base
	r.Key = "conf2026a"
	if _, err := Add(r); !errors.Is(err, ErrDuplicateSource) {
		t.Fatalf("duplicate key: %v", err)
	}
}

// TestLedgerAppendsAndNeverEdits is AC2: each record is a new line carrying the
// spec's fields with cited_publicly false, and a correction is another new line —
// the earlier bytes are untouched.
func TestLedgerAppendsAndNeverEdits(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	req := AppendRequest{Corpus: corpus, Repo: "0123456789ab", DecisionRef: "adr-41", Claim: "the ledger is append-only",
		SourceKey: "conf2026a", Locator: "§2", Influence: "supports"}
	first, err := Append(req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Line != 1 || first.Path != "ledger/0123456789ab.jsonl" {
		t.Fatalf("first = %+v", first)
	}
	path := filepath.Join(corpus, "ledger", "0123456789ab.jsonl")
	before, _ := os.ReadFile(path)

	req.Claim = "the ledger is append-only, corrected"
	req.Corrects = 1
	second, err := Append(req)
	if err != nil || second.Line != 2 {
		t.Fatalf("second = %+v %v", second, err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.HasPrefix(after, before) {
		t.Fatal("a correction rewrote an earlier line")
	}
	lines := strings.Split(strings.TrimRight(string(after), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("ledger has %d lines", len(lines))
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"ts", "repo", "decision_ref", "claim", "source_key", "locator", "influence", "cited_publicly"} {
		if _, ok := rec[f]; !ok {
			t.Errorf("ledger line lacks %q", f)
		}
	}
	if rec["cited_publicly"] != false {
		t.Errorf("cited_publicly = %v", rec["cited_publicly"])
	}
	if st := git(t, corpus, "status", "--porcelain"); st != "" {
		t.Fatalf("ledger append not committed:\n%s", st)
	}

	for _, bad := range []func(r *AppendRequest){
		func(r *AppendRequest) { r.Influence = "vibes" },
		func(r *AppendRequest) { r.SourceKey = "nosuchkey" },
		func(r *AppendRequest) { r.Claim = "" },
		func(r *AppendRequest) { r.Corrects = 9 },
		func(r *AppendRequest) { r.Repo = "../x" },
	} {
		r := req
		bad(&r)
		if _, err := Append(r); err == nil {
			t.Errorf("a bad append was accepted: %+v", r)
		}
	}
}

// TestFlipNeedsBothGates is AC5: a flip on a line whose source lacks permission is
// refused naming the failing gate and appends nothing; with permission present the
// flip succeeds as a NEW line, and a second flip of the same line is refused.
func TestFlipNeedsBothGates(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	addPublic(t, corpus)
	mk := func(key string) int {
		res, err := Append(AppendRequest{Corpus: corpus, Repo: "0123456789ab", DecisionRef: "d", Claim: "c", SourceKey: key, Influence: "method"})
		if err != nil {
			t.Fatal(err)
		}
		return res.Line
	}
	confLine := mk("conf2026a")
	pubLine := mk("survey2026ledger")

	_, err := Flip(corpus, "0123456789ab", confLine, fixedNow)
	if !errors.Is(err, ErrCitationRefused) || !strings.Contains(err.Error(), "permission_status") {
		t.Fatalf("flip on an unpermitted source: %v", err)
	}
	lines, _ := List(corpus, "0123456789ab")
	if len(lines) != 2 {
		t.Fatalf("a refused flip appended: %d lines", len(lines))
	}

	res, err := Flip(corpus, "0123456789ab", pubLine, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if res.Line != 3 || !res.Record.CitedPublicly || res.Record.Flips != pubLine {
		t.Fatalf("flip result = %+v", res)
	}
	if _, err := Flip(corpus, "0123456789ab", pubLine, fixedNow); !errors.Is(err, ErrCitationRefused) {
		t.Fatalf("second flip: %v", err)
	}
	if _, err := Flip(corpus, "0123456789ab", 3, fixedNow); !errors.Is(err, ErrCitationRefused) {
		t.Fatalf("flipping a flip line: %v", err)
	}
}

// TestProjectionTitlesAliasesAlwaysAuthorsOnlyOnOptIn is AC3's projection half.
func TestProjectionTitlesAliasesAlwaysAuthorsOnlyOnOptIn(t *testing.T) {
	for _, ban := range []bool{false, true} {
		corpus := newCorpus(t)
		addConfidential(t, corpus, ban)
		addPublic(t, corpus)
		c, err := loadCorpus(corpus)
		if err != nil {
			t.Fatal(err)
		}
		pats, err := c.Projection()
		if err != nil {
			t.Fatal(err)
		}
		keys := map[string]bool{}
		for _, p := range pats {
			keys[p.Key] = true
		}
		if !keys["sources/conf2026a/title"] || !keys["sources/conf2026a/alias-1"] {
			t.Errorf("ban=%v: projection keys %v", ban, keys)
		}
		if keys["sources/conf2026a/author-1"] != ban {
			t.Errorf("ban=%v: author key present=%v", ban, keys["sources/conf2026a/author-1"])
		}
		for k := range keys {
			if strings.Contains(k, "survey2026ledger") {
				t.Errorf("a public source was projected: %s", k)
			}
		}
	}
}

// guardRepo is a throwaway repository with the committed pre-commit guard installed
// and the local tier gitignored, as a managed repo has it.
func guardRepo(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not in a checkout: the committed guard cannot be found")
	}
	hook, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), ".githooks", "pre-commit"))
	if err != nil {
		t.Skipf("guard not found: %v", err)
	}
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.name", "Alice Example")
	git(t, repo, "config", "user.email", "alice@example.com")
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), hook, 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestSyncBanlistFeedsTheGuard is AC3 end to end: the sync writes the generated
// block into the repo's untracked private store, and the committed guard then
// refuses a commit carrying the confidential title (named by key only) while a
// commit naming the public source passes.
func TestSyncBanlistFeedsTheGuard(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	addPublic(t, corpus)
	repo := guardRepo(t)

	res, err := SyncBanlist(corpus, repo, SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Sources != 1 || res.Block.Entries != 2 {
		t.Fatalf("sync = %+v", res)
	}

	if err := os.WriteFile(filepath.Join(repo, "ok.md"), []byte("we follow "+pubTitle+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "ok.md", ".gitignore")
	cmd := exec.Command("git", "-C", repo, "commit", "-q", "-m", "ok")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("a public citation was refused: %v\n%s", err, out)
	}

	if err := os.WriteFile(filepath.Join(repo, "leak.md"), []byte("per the "+strings.ToLower(confTitle)+", we\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "leak.md")
	cmd = exec.Command("git", "-C", repo, "commit", "-q", "-m", "leak")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the guard let a confidential title through:\n%s", out)
	}
	if !strings.Contains(string(out), "sources/conf2026a/title") {
		t.Errorf("the refusal does not name the key:\n%s", out)
	}
	if strings.Contains(strings.ToLower(string(out)), "harbour") {
		t.Errorf("the refusal leaks the title:\n%s", out)
	}
}

// TestCiteCheckReportsByKeyOnly is AC4: offending sources are reported by key and
// position, and nothing in the report — text or JSON — carries the matched string.
func TestCiteCheckReportsByKeyOnly(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, true)
	addPublic(t, corpus)
	text := "intro\nAs " + confTitle + " argues, and " + pubTitle + " agrees.\nAsk Adaeze " + confAuthor + ".\n"
	rep, err := CiteCheck(corpus, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Clean() {
		t.Fatal("cite-check called the text clean")
	}
	keys := map[string]bool{}
	for _, f := range rep.Findings {
		keys[f.Source] = true
	}
	if len(keys) != 1 || !keys["conf2026a"] {
		t.Fatalf("findings name %v", keys)
	}
	blob, _ := json.Marshal(rep)
	for _, leak := range []string{"Harbour", "harbour", confAuthor, "Adaeze", "Ledger Formats"} {
		if strings.Contains(string(blob), leak) {
			t.Fatalf("the report carries %q: %s", leak, blob)
		}
	}
	clean, err := CiteCheck(corpus, []byte("nothing to see; "+pubTitle+"\n"))
	if err != nil || !clean.Clean() {
		t.Fatalf("clean text: %+v %v", clean, err)
	}
}

// TestDeclassifyDropsTheBanAndOpensTheFlip is AC7: the visible move to public/ is
// what the next sync reads, so the key's strings leave the block, and the source's
// ledger lines become flippable once declassification has set its permission.
func TestDeclassifyDropsTheBanAndOpensTheFlip(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	repo := guardRepo(t)
	line, err := Append(AppendRequest{Corpus: corpus, Repo: "0123456789ab", DecisionRef: "d", Claim: "c", SourceKey: "conf2026a", Influence: "background"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SyncBanlist(corpus, repo, SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(repo, ".abcd", ".work.local", "private-names.txt")
	if b, _ := os.ReadFile(store); !strings.Contains(string(b), "sources/conf2026a/title") {
		t.Fatal("block does not carry the confidential key before declassification")
	}

	if _, err := Declassify(corpus, "conf2026a", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(corpus, "public", "conf2026a", "text.md")); err != nil {
		t.Fatalf("folder not moved to public/: %v", err)
	}
	if st := git(t, corpus, "status", "--porcelain"); st != "" {
		t.Fatalf("declassification not committed:\n%s", st)
	}
	if _, err := SyncBanlist(corpus, repo, SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(store); strings.Contains(string(b), "sources/conf2026a") {
		t.Fatalf("the declassified key survived the refresh:\n%s", b)
	}
	if _, err := Flip(corpus, "0123456789ab", line.Line, fixedNow); err != nil {
		t.Fatalf("flip after declassification: %v", err)
	}
}

// TestAMismatchedClassRefusesTheSync: a folder moved by hand without its entry is a
// corpus whose classes disagree. The sync writes nothing and names the key, so the
// existing block keeps banning — the safe direction.
func TestAMismatchedClassRefusesTheSync(t *testing.T) {
	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	repo := guardRepo(t)
	if _, err := SyncBanlist(corpus, repo, SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(repo, ".abcd", ".work.local", "private-names.txt")
	before, _ := os.ReadFile(store)
	if err := os.MkdirAll(filepath.Join(corpus, "public"), 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, corpus, "mv", "confidential/conf2026a", "public/conf2026a")

	_, err := SyncBanlist(corpus, repo, SyncOptions{})
	if !errors.Is(err, ErrClassMismatch) || !strings.Contains(err.Error(), "conf2026a") {
		t.Fatalf("sync over a mismatch: %v", err)
	}
	after, _ := os.ReadFile(store)
	if !bytes.Equal(before, after) {
		t.Fatal("a refused sync rewrote the store")
	}
	if _, err := CiteCheck(corpus, []byte("x")); !errors.Is(err, ErrClassMismatch) {
		t.Fatalf("cite-check over a mismatch: %v", err)
	}
}

// TestNoCorpusIsNamedByEveryStep is AC6's core half: every corpus-dependent step
// reports the one sentinel a front door turns into its loud notice, and none of
// them creates the corpus.
func TestNoCorpusIsNamedByEveryStep(t *testing.T) {
	home := hermetic(t)
	dir := abcdhome.Path(home, "sources")
	repo := t.TempDir()
	steps := map[string]func() error{
		"add": func() error {
			_, err := Add(AddRequest{Corpus: dir, Key: "k2026x", Title: "Some Title", Class: ClassPublic})
			return err
		},
		"ledger": func() error {
			_, err := Append(AppendRequest{Corpus: dir, Repo: "r", DecisionRef: "d", Claim: "c", SourceKey: "k", Influence: "method"})
			return err
		},
		"flip":        func() error { _, err := Flip(dir, "r", 1, fixedNow); return err },
		"sync":        func() error { _, err := SyncBanlist(dir, repo, SyncOptions{}); return err },
		"cite-check":  func() error { _, err := CiteCheck(dir, []byte("x")); return err },
		"declassify":  func() error { _, err := Declassify(dir, "k", ""); return err },
		"list ledger": func() error { _, err := List(dir, "r"); return err },
	}
	for name, step := range steps {
		if err := step(); !errors.Is(err, ErrNoCorpus) {
			t.Errorf("%s: err = %v, want ErrNoCorpus", name, err)
		}
	}
	st, err := Status(dir)
	if err != nil || st.Present {
		t.Fatalf("status over no corpus: %+v %v", st, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("a step created the corpus")
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))); !os.IsNotExist(err) {
		t.Fatal("a sync with no corpus wrote the private store")
	}
}

// TestInitRefusesInsideAnotherRepository: the corpus never lives inside a working
// tree, where one `git add -A` would carry documents into a repository (adr-41).
func TestInitRefusesInsideAnotherRepository(t *testing.T) {
	hermetic(t)
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	if _, err := Init(filepath.Join(repo, "corpus")); !errors.Is(err, ErrCorpusInvalid) {
		t.Fatalf("init inside a repo: %v", err)
	}
}

// TestTheRepositoryGuardRefreshesWithItsOwnBuild is AC3 through this repository's
// real guard, end to end (iss-2609252007414882): a clone whose hooks path is this
// checkout's .githooks builds ./cmd/abcd from the checkout and runs ITS
// `source sync-banlist --refresh`, so a source added to the corpus after the last
// by-hand sync is banned on the very next commit, and the build's one-line count
// is relayed. No installed abcd is involved: the only abcd this test can reach is
// the one the hook builds.
func TestTheRepositoryGuardRefreshesWithItsOwnBuild(t *testing.T) {
	for _, tool := range []string{"bash", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s unavailable", tool)
		}
	}
	// The Go caches are read BEFORE the HOME moves: the hook builds abcd, and a
	// HOME-relative default cache would build it from cold.
	goEnv, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH").Output()
	if err != nil {
		t.Skipf("go env: %v", err)
	}
	vals := strings.Split(strings.TrimSpace(string(goEnv)), "\n")
	if len(vals) != 3 {
		t.Fatalf("go env returned %d values", len(vals))
	}
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not in a checkout: the committed guard cannot be found")
	}
	hooks := filepath.Join(strings.TrimSpace(string(top)), ".githooks")

	corpus := newCorpus(t)
	addConfidential(t, corpus, false)
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.name", "Alice Example")
	git(t, repo, "config", "user.email", "alice@example.com")
	writeFile(t, repo, ".gitignore", ".abcd/.work.local/\n")
	if _, err := SyncBanlist(corpus, repo, SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "config", "core.hooksPath", hooks)

	src := t.TempDir()
	if _, err := Add(AddRequest{
		Corpus: corpus, Key: "conf2026b", Title: "Tidewater Staffing Memo", Type: "report", Class: ClassConfidential,
		Original: writeFile(t, src, "memo.md", "# memo\nbody\n"),
	}); err != nil {
		t.Fatal(err)
	}

	writeFile(t, repo, "leak.md", "per the tidewater staffing memo, we\n")
	git(t, repo, "add", "leak.md", ".gitignore")
	cmd := exec.Command("git", "-C", repo, "commit", "-q", "-m", "docs: a note")
	cmd.Env = append(os.Environ(), "GOCACHE="+vals[0], "GOMODCACHE="+vals[1], "GOPATH="+vals[2], "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the guard let a source added after the last sync through:\n%s", out)
	}
	if !strings.Contains(string(out), "pre-commit: abcd source sync-banlist — 2 confidential sources") {
		t.Errorf("the refresh's count line was not relayed:\n%s", out)
	}
	if !strings.Contains(string(out), "sources/conf2026b/title") {
		t.Errorf("the refusal does not name the new source's key:\n%s", out)
	}
	if strings.Contains(strings.ToLower(string(out)), "tidewater") {
		t.Errorf("the guard's output leaks the title:\n%s", out)
	}
}
