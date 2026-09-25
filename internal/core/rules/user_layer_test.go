//go:build unix

package rules

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// userHome sets HOME to a fresh directory and returns it.
func userHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// writeUserRules writes body as ~/.abcd/rules.json under home, owner-only
// writable (the shape a hand-edited file ordinarily has).
func writeUserRules(t *testing.T, home, body string) string {
	t.Helper()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// treeEntries lists every path under root, relative, sorted.
func treeEntries(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// AC1: absence is the default and costs nothing — the loaded set is exactly
// what it was before the user layer existed, and nothing is created.
func TestUserLayerAbsentChangesNothing(t *testing.T) {
	for _, withDir := range []bool{false, true} {
		home := userHome(t)
		if withDir {
			if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		before := treeEntries(t, home)
		repo := t.TempDir()
		rs, err := Load(repo)
		if err != nil {
			t.Fatalf("Load with no user file: %v", err)
		}
		if !reflect.DeepEqual(rs, Defaults()) {
			t.Fatalf("no user file, no repo file: the set must be the bundled defaults exactly")
		}
		writeRepoRules(t, repo, `{"schema_version":1,"domains":{"PII":{"rules":["repo pii"]}}}`)
		rs, err = Load(repo)
		if err != nil {
			t.Fatal(err)
		}
		want := Merge(Defaults(), RuleSet{SchemaVersion: 1, Domains: map[string]Domain{"PII": {Rules: []string{"repo pii"}}}})
		if !reflect.DeepEqual(rs, want) {
			t.Fatalf("no user file: the repo merge must be what it always was")
		}
		if after := treeEntries(t, home); !reflect.DeepEqual(before, after) {
			t.Fatalf("Load created something in the user scope: before %v, after %v", before, after)
		}
	}
}

// AC1, the unresolvable home: with HOME unset there is no user scope to read.
func TestUserLayerNoHomeIsAbsent(t *testing.T) {
	t.Setenv("HOME", "")
	rs, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("an unset HOME must read as no user layer: %v", err)
	}
	if !reflect.DeepEqual(rs, Defaults()) {
		t.Fatal("an unset HOME must leave the defaults untouched")
	}
}

// AC2: a user-scope field replaces the bundled one, and says so.
func TestUserLayerOverridesBundled(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{"PII":{"rules":["house pii rule"]}}}`)
	rs, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, ok := rs.Lookup("PII")
	if !ok || !reflect.DeepEqual(d.Rules, []string{"house pii rule"}) {
		t.Fatalf("user rules not injected in place of the bundled ones: %+v", d)
	}
	if d.Source != SourceUser {
		t.Fatalf("PII source = %q, want %q", d.Source, SourceUser)
	}
	if !reflect.DeepEqual(d.Recall, Defaults().Domains["PII"].Recall) {
		t.Fatal("a field the user layer does not set must inherit the bundled value")
	}
	out := Render(rs.Match("redact the api key in this token"))
	if !strings.Contains(out, "## PII (user override)\n- house pii rule\n") {
		t.Fatalf("the injected block does not carry the user rule under a marked heading:\n%s", out)
	}
}

// AC3: the repo wins a field both layers set; a field only the user set
// survives the repo override of another field.
func TestRepoLayerOverridesUser(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{
		"PII":{"rules":["user pii"]},
		"ROADMAP":{"rules":["user roadmap"]}}}`)
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"domains":{
		"PII":{"rules":["repo pii"]},
		"ROADMAP":{"recall":["milestone"]}}}`)
	rs, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := rs.Domains["PII"].Rules; !reflect.DeepEqual(got, []string{"repo pii"}) {
		t.Fatalf("PII rules = %v, want the repo's", got)
	}
	rm := rs.Domains["ROADMAP"]
	if !reflect.DeepEqual(rm.Rules, []string{"user roadmap"}) || !reflect.DeepEqual(rm.Recall, []string{"milestone"}) {
		t.Fatalf("per-field layering lost a field: %+v", rm)
	}
	for _, name := range []string{"PII", "ROADMAP"} {
		if d, _ := rs.Lookup(name); d.Source != SourceRepo {
			t.Errorf("%s source = %q, want %q (the repo named it last)", name, d.Source, SourceRepo)
		}
	}
}

// AC4: a custom domain declared once in the user scope recall-matches in a repo
// that never mentions it, with or without a repo file.
func TestUserCustomDomainInjectsInAnyRepo(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{
		"HOUSE":{"recall":["widget"],"rules":["widgets are named in the singular"]}}}`)
	bare := t.TempDir()
	withFile := t.TempDir()
	writeRepoRules(t, withFile, `{"schema_version":1,"domains":{"ROADMAP":{"state":"dormant"}}}`)
	for _, repo := range []string{bare, withFile} {
		rs, err := Load(repo)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		got := rs.Match("rename the widget")
		if len(got) != 1 || got[0].Name != "HOUSE" || got[0].Source != SourceUser {
			t.Fatalf("the user custom domain did not recall-match: %+v", got)
		}
		res := Inject(rs, "rename the widget", SessionState{}, 0)
		if !strings.Contains(res.Text, "## HOUSE (user override)\n- widgets are named in the singular\n") {
			t.Fatalf("injected text lacks the user domain:\n%s", res.Text)
		}
		if l := res.Labels(); len(l) != 1 || l[0] != "HOUSE (user override)" {
			t.Fatalf("Labels() = %v, want [HOUSE (user override)]", l)
		}
	}
}

// AC5: every shape the repo file is refused on is refused for the user file
// too, loudly, naming the file in tilde form and never the expanded home.
func TestUserLayerRefusalsAreLoud(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, home string)
		want  string
	}{
		{"malformed JSON", func(t *testing.T, home string) { writeUserRules(t, home, `{ not json`) }, "not valid JSON"},
		{"duplicate key", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{"PII":{"rules":["a"]},"PII":{"rules":["b"]}}}`)
		}, "duplicate key"},
		{"invalid state", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{"PII":{"state":"sleepy"}}}`)
		}, "unknown state"},
		{"bad domain name", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{"../x":{"rules":["a"]}}}`)
		}, "must match"},
		{"empty rule body", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{"PII":{"rules":["  "]}}}`)
		}, "empty or whitespace-only"},
		{"bad schema version", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":2,"domains":{}}`)
		}, "schema_version"},
		{"oversize", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{}}`+strings.Repeat(" ", maxRulesFileBytes))
		}, "cap"},
		{"symlinked leaf", func(t *testing.T, home string) {
			if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(home, "elsewhere.json")
			if err := os.WriteFile(target, []byte(`{"schema_version":1}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(home, ".abcd", "rules.json")); err != nil {
				t.Fatal(err)
			}
		}, "not a regular file"},
		{"fifo leaf", func(t *testing.T, home string) {
			if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(filepath.Join(home, ".abcd", "rules.json"), 0o644); err != nil {
				t.Skipf("mkfifo unavailable: %v", err)
			}
		}, "not a regular file"},
		{"symlinked .abcd directory", func(t *testing.T, home string) {
			real := filepath.Join(home, "dotfiles-abcd")
			if err := os.MkdirAll(real, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(real, "rules.json"), []byte(`{"schema_version":1,"domains":{}}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, filepath.Join(home, ".abcd")); err != nil {
				t.Fatal(err)
			}
		}, "~/.abcd is a symlink"},
		{"writable by others", func(t *testing.T, home string) {
			p := writeUserRules(t, home, `{"schema_version":1,"domains":{}}`)
			if err := os.Chmod(p, 0o666); err != nil {
				t.Fatal(err)
			}
		}, "writable by others"},
		{"foreign owner", func(t *testing.T, home string) {
			writeUserRules(t, home, `{"schema_version":1,"domains":{}}`)
			restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil })
			t.Cleanup(restore)
		}, "not owned by this session's uid"},
		{"unreadable scope directory", func(t *testing.T, home string) {
			if os.Getuid() == 0 {
				t.Skip("root reads through a mode-000 directory")
			}
			writeUserRules(t, home, `{"schema_version":1,"domains":{}}`)
			dir := filepath.Join(home, ".abcd")
			if err := os.Chmod(dir, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		}, "could not be read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := userHome(t)
			tc.setup(t, home)
			rs, err := Load(t.TempDir())
			if err == nil {
				t.Fatalf("a %s user file must be refused, not skipped; loaded %d domains", tc.name, len(rs.Domains))
			}
			msg := err.Error()
			if !strings.Contains(msg, "~/.abcd") {
				t.Errorf("the refusal must name the user file in tilde form: %s", msg)
			}
			if !strings.Contains(msg, tc.want) {
				t.Errorf("the refusal must say why (%q): %s", tc.want, msg)
			}
			if strings.Contains(msg, home) {
				t.Errorf("the refusal must not carry the expanded home path: %s", msg)
			}
		})
	}
}

// A symlinked ~/.abcd with no rules.json behind it is the ordinary state of a
// machine whose user scope lives in a dotfiles checkout: nothing is read, so
// nothing is refused, and the injected rules stay what they were (AC1).
func TestUserLayerSymlinkedScopeWithoutFileIsAbsent(t *testing.T) {
	home := userHome(t)
	real := filepath.Join(home, "dotfiles-abcd")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	rs, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("a symlinked ~/.abcd holding no rules.json must read as absent: %v", err)
	}
	if !reflect.DeepEqual(rs, Defaults()) {
		t.Fatal("the defaults must be untouched")
	}
}

// A refused user layer injects nothing: the error is the whole answer, never a
// partial set built from the layers that did load.
func TestUserLayerRefusalIsNotAPartialSet(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{ broken`)
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"domains":{"PII":{"rules":["repo pii"]}}}`)
	rs, err := Load(repo)
	if err == nil {
		t.Fatal("a malformed user layer must fail the load")
	}
	if len(rs.Domains) != 0 || len(rs.Match("redact the api key")) != 0 {
		t.Fatalf("a failed load must carry no domains, got %d", len(rs.Domains))
	}
	if !strings.Contains(err.Error(), UserDisplayPath) {
		t.Fatalf("the error must name the user file: %v", err)
	}
}

// AC6: each layer is labelled where provenance renders.
func TestProvenanceNamesAllThreeLayers(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{"PII":{"rules":["user pii"]},"ROADMAP":{"rules":["user roadmap"]}}}`)
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"domains":{"ROADMAP":{"rules":["repo roadmap"]}}}`)
	rs, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"COMMITTING": SourceBundled, "PII": SourceUser, "ROADMAP": SourceRepo} {
		d, ok := rs.Lookup(name)
		if !ok || d.Source != want {
			t.Errorf("Lookup(%s).Source = %q, want %q", name, d.Source, want)
		}
	}
	out := Render(rs.Active())
	for _, want := range []string{"\n## COMMITTING\n", "\n## PII (user override)\n", "\n## ROADMAP (repo override)\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered set lacks %q", want)
		}
	}
	user := ResolvedDomain{Name: "PII", Source: SourceUser, Domain: Domain{Rules: []string{"x"}}}
	repoD := ResolvedDomain{Name: "PII", Source: SourceRepo, Domain: Domain{Rules: []string{"x"}}}
	if Signature(user) == Signature(repoD) {
		t.Fatal("the layer marker must sit inside the dedup unit")
	}
}

// AC7: the repo's suppression beats a user layer declaring the domain active.
func TestRepoSuppressionBeatsUserActivation(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"disabled":false,"domains":{
		"ROADMAP":{"state":"active"},
		"HOUSE":{"recall":["widget"],"rules":["house rule"]}}}`)

	dormant := t.TempDir()
	writeRepoRules(t, dormant, `{"schema_version":1,"domains":{"ROADMAP":{"state":"dormant"}}}`)
	rs, err := Load(dormant)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range rs.Active() {
		if d.Name == "ROADMAP" {
			t.Fatal("a repo-dormant domain must stay dormant under a user layer declaring it active")
		}
	}

	killed := t.TempDir()
	writeRepoRules(t, killed, `{"schema_version":1,"disabled":true}`)
	rs, err = Load(killed)
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Disabled || rs.Active() != nil || rs.Match("rename the widget") != nil {
		t.Fatal("the repo kill switch must suppress the user layer's domains too")
	}
	if got := rs.KillSwitchSources(); !reflect.DeepEqual(got, []string{SourceRepo}) {
		t.Fatalf("KillSwitchSources() = %v, want [repo]", got)
	}
}

// The spec's recorded consequence: stickiness runs both ways, so a user-scope
// kill switch silences every repo on the machine and no repo re-enables it.
func TestUserKillSwitchIsSticky(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"disabled":true}`)
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"disabled":false}`)
	rs, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Disabled {
		t.Fatal("a repo must not re-enable a user-scope kill switch")
	}
	if got := rs.KillSwitchSources(); !reflect.DeepEqual(got, []string{SourceUser}) {
		t.Fatalf("KillSwitchSources() = %v, want [user]", got)
	}
}

// A ruleless user domain is skipped with a note naming the user file, exactly
// as a ruleless repo domain is skipped with one naming the repo file.
func TestUserRulelessDomainIsSkippedWithANote(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{"HOUSE":{"recall":["widget"]}}}`)
	rs, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("a ruleless user domain must not fail the load: %v", err)
	}
	if _, ok := rs.Domains["HOUSE"]; ok {
		t.Fatal("the ruleless user domain must be dropped")
	}
	notes := strings.Join(rs.Notes(), "\n")
	if !strings.Contains(notes, "HOUSE") || !strings.Contains(notes, UserDisplayPath) {
		t.Fatalf("the note must name the domain and the user file:\n%s", notes)
	}
}

// The trust guard is the home-declaration guard: nothing looser.
func TestUserLayerReadsThroughTheDeclarationGuard(t *testing.T) {
	home := userHome(t)
	p := writeUserRules(t, home, `{"schema_version":1,"domains":{}}`)
	if err := os.Chmod(p, 0o620); err != nil {
		t.Fatal(err)
	}
	_, err := Load(t.TempDir())
	if err == nil || !errors.Is(err, fsutil.ErrDeclarationWritable) {
		t.Fatalf("a group-writable user file must be refused through fsutil.ReadDeclaration: %v", err)
	}
}
