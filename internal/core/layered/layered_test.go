package layered

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture lays a repository root and a home under one temp dir and returns the
// Roots a Load reads through. Files are written at 0o600 so the machine layer's
// declaration guard (owner-only write) admits them.
type fixture struct {
	t     *testing.T
	roots Roots
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	home := filepath.Join(base, "home")
	for _, d := range []string{repo, home} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return &fixture{t: t, roots: Roots{Repo: repo, Home: home}}
}

func (f *fixture) write(abs, body string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) repoFile(file File, body string) string {
	p := filepath.Join(f.roots.Repo, filepath.FromSlash(file.RepoRel))
	f.write(p, body)
	return p
}

func (f *fixture) machineFile(file File, body string) string {
	p := filepath.Join(f.roots.Home, ".abcd", filepath.FromSlash(file.MachineRel))
	f.write(p, body)
	return p
}

func positive(n int) error {
	if n <= 0 {
		return fmt.Errorf("want a positive whole number of minutes")
	}
	return nil
}

// TestPrecedenceFlagRepoMachineBundled is the resolver's whole contract in one
// table: each layer that holds the key beats every layer below it, and the value
// comes back with the layer and the origin that supplied it.
func TestPrecedenceFlagRepoMachineBundled(t *testing.T) {
	cases := []struct {
		name               string
		repo, machine      string
		flag               any
		wantV              int
		wantLayer          Layer
		wantOriginContains string
	}{
		{"nothing configured: bundled", "", "", nil, 120, Bundled, "bundled"},
		{"machine only", "", `{"pace":{"work_minutes":90}}`, nil, 90, Machine, "~/.abcd/config.json"},
		{"repo over machine", `{"pace":{"work_minutes":60}}`, `{"pace":{"work_minutes":90}}`, nil, 60, Repo, ".abcd/config.json"},
		{"flag over repo and machine", `{"pace":{"work_minutes":60}}`, `{"pace":{"work_minutes":90}}`, 30, 30, Flag, "--pace"},
		{"repo file without the key falls to machine", `{"docs":{"target":"agents_md"}}`, `{"pace":{"work_minutes":90}}`, nil, 90, Machine, "~/.abcd/config.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.repo != "" {
				f.repoFile(Config, tc.repo)
			}
			if tc.machine != "" {
				f.machineFile(Config, tc.machine)
			}
			s, err := Load(Config, f.roots)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if tc.flag != nil {
				if err := s.SetFlag("pace.work_minutes", tc.flag, "--pace"); err != nil {
					t.Fatalf("SetFlag: %v", err)
				}
			}
			got, err := Get(s, "pace.work_minutes", 120, positive)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.V != tc.wantV || got.Layer != tc.wantLayer || !strings.Contains(got.Origin, tc.wantOriginContains) {
				t.Fatalf("got %+v, want value %d from %s (%q)", got, tc.wantV, tc.wantLayer, tc.wantOriginContains)
			}
		})
	}
}

// TestMalformedFileRefusesNeverDefaults: a file that is present and cannot be
// read as a configuration is an error from Load, naming the file — never a
// quiet fall to the layer below or to the bundled default.
func TestMalformedFileRefusesNeverDefaults(t *testing.T) {
	cases := []struct {
		name, body, want string
		machine          bool
	}{
		{"repo invalid json", `{"pace": {`, ".abcd/config.json", false},
		{"machine invalid json", `{"pace": `, "~/.abcd/config.json", true},
		{"top level not an object", `[1,2]`, "not a JSON object", false},
		{"duplicate key", `{"pace":{"work_minutes":1,"work_minutes":500}}`, "more than once", false},
		{"duplicate top-level key", `{"pace":{},"pace":{"work_minutes":1}}`, "more than once", true},
		{"case twin of a key", `{"pace":{"work_minutes":1,"Work_Minutes":500}}`, "more than once", false},
		{"case twin of a top-level key", `{"pace":{},"PACE":{"work_minutes":1}}`, "more than once", true},
		{"repeat inside an array element", `{"pace":{"list":[{"k":1,"k":2}]}}`, "more than once", false},
		{"trailing content", `{"pace":{}} {"pace":{}}`, "trailing", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.machine {
				f.machineFile(Config, tc.body)
			} else {
				f.repoFile(Config, tc.body)
			}
			_, err := Load(Config, f.roots)
			if err == nil {
				t.Fatalf("Load admitted a malformed file")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err, tc.want)
			}
		})
	}
}

// TestSchemaVersionIsRequiredWhereTheFileDeclaresOne covers the routing file's
// shape: a missing or foreign schema_version refuses.
func TestSchemaVersionIsRequiredWhereTheFileDeclaresOne(t *testing.T) {
	for _, body := range []string{`{"agents":{}}`, `{"schema_version":2,"agents":{}}`, `{"schema_version":"1"}`} {
		f := newFixture(t)
		f.repoFile(OracleRouting, body)
		_, err := Load(OracleRouting, f.roots)
		if err == nil || !strings.Contains(err.Error(), "schema_version") {
			t.Fatalf("body %s: err = %v, want a schema_version refusal", body, err)
		}
	}
	f := newFixture(t)
	f.repoFile(OracleRouting, `{"schema_version":1,"agents":{}}`)
	if _, err := Load(OracleRouting, f.roots); err != nil {
		t.Fatalf("a well-formed routing file refused: %v", err)
	}
}

// TestGuardedReads: the repo layer is read inside an os.Root (a symlinked leaf
// or ancestor refuses) and the machine layer through the home-declaration guard
// (writable by others refuses). Each refusal is an error, never an absent layer.
func TestGuardedReads(t *testing.T) {
	t.Run("repo symlinked leaf", func(t *testing.T) {
		f := newFixture(t)
		target := filepath.Join(t.TempDir(), "elsewhere.json")
		f.write(target, `{"pace":{"work_minutes":5}}`)
		leaf := filepath.Join(f.roots.Repo, ".abcd", "config.json")
		if err := os.MkdirAll(filepath.Dir(leaf), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, leaf); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(Config, f.roots); err == nil {
			t.Fatal("a symlinked repo file was read")
		}
	})
	t.Run("repo symlinked ancestor", func(t *testing.T) {
		f := newFixture(t)
		outside := t.TempDir()
		f.write(filepath.Join(outside, "config.json"), `{"pace":{"work_minutes":5}}`)
		if err := os.Symlink(outside, filepath.Join(f.roots.Repo, ".abcd")); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(Config, f.roots); err == nil {
			t.Fatal("a repo file behind a symlinked ancestor was read")
		}
	})
	t.Run("machine writable by others", func(t *testing.T) {
		f := newFixture(t)
		p := f.machineFile(Config, `{"pace":{"work_minutes":5}}`)
		if err := os.Chmod(p, 0o666); err != nil {
			t.Fatal(err)
		}
		_, err := Load(Config, f.roots)
		if err == nil || !strings.Contains(err.Error(), "~/.abcd/config.json") {
			t.Fatalf("err = %v, want a refusal naming the machine file", err)
		}
	})
	t.Run("no home", func(t *testing.T) {
		f := newFixture(t)
		f.roots.Home = ""
		if _, err := Load(Config, f.roots); err == nil {
			t.Fatal("an unresolved home silently dropped the machine layer")
		}
	})
	t.Run("no repo is an absent layer", func(t *testing.T) {
		f := newFixture(t)
		f.machineFile(Config, `{"pace":{"work_minutes":7}}`)
		f.roots.Repo = ""
		s, err := Load(Config, f.roots)
		if err != nil {
			t.Fatalf("Load outside a checkout: %v", err)
		}
		got, err := Get(s, "pace.work_minutes", 120, positive)
		if err != nil || got.V != 7 || got.Layer != Machine {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
}

// TestUnknownKeyInAClaimedNamespaceRefuses: a consumer claims its namespace and
// every key in it; a key nobody reads is refused by name and origin, so a typo
// never lets the default apply unannounced.
func TestUnknownKeyInAClaimedNamespaceRefuses(t *testing.T) {
	f := newFixture(t)
	f.repoFile(Config, `{"docs":{"target":"agents_md"},"pace":{"work_minuts":90}}`)
	s, err := Load(Config, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Claim("pace", "work_minutes", "pause_minutes", "sub_agents")
	if err == nil {
		t.Fatal("an unknown key under a claimed namespace was admitted")
	}
	for _, want := range []string{"pace.work_minuts", ".abcd/config.json", "work_minutes"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
	// A namespace nobody claimed is another reader's business.
	if err := s.Claim("match", "threshold"); err != nil {
		t.Fatalf("claiming an absent namespace: %v", err)
	}
}

func TestClaimRefusesANamespaceThatIsNotAnObject(t *testing.T) {
	f := newFixture(t)
	f.machineFile(Config, `{"pace":90}`)
	s, err := Load(Config, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim("pace", "work_minutes"); err == nil || !strings.Contains(err.Error(), "object") {
		t.Fatalf("err = %v, want a not-an-object refusal", err)
	}
}

// TestClaimWildcard serves roles.<role>.runner: the role name is open, the keys
// under each role are closed.
func TestClaimWildcard(t *testing.T) {
	f := newFixture(t)
	f.repoFile(Config, `{"roles":{"implementer":{"runner":"host"},"validator":{"runnr":"claude"}}}`)
	s, err := Load(Config, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Claim("roles.*", "runner")
	if err == nil || !strings.Contains(err.Error(), "roles.validator.runnr") {
		t.Fatalf("err = %v, want a refusal naming roles.validator.runnr", err)
	}
}

// TestTopLevelClaim is the owned-file shape: the routing file admits exactly its
// own top-level keys.
func TestTopLevelClaim(t *testing.T) {
	f := newFixture(t)
	f.repoFile(OracleRouting, `{"schema_version":1,"agents":{},"agnets":{}}`)
	s, err := Load(OracleRouting, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim("", "schema_version", "agents"); err == nil || !strings.Contains(err.Error(), "agnets") {
		t.Fatalf("err = %v, want a refusal naming agnets", err)
	}
}

// TestWrongTypeOrRangeRefusesNamingTheValue: the winning layer's value must
// decode and pass the check; if it does not, Get refuses naming the key, the
// value and where it came from — it never moves on to a lower layer.
func TestWrongTypeOrRangeRefusesNamingTheValue(t *testing.T) {
	cases := []struct{ body, want string }{
		{`{"pace":{"work_minutes":"ninety"}}`, `"ninety"`},
		{`{"pace":{"work_minutes":90.5}}`, "90.5"},
		{`{"pace":{"work_minutes":0}}`, "positive whole number"},
		{`{"pace":{"work_minutes":null}}`, "null"},
	}
	for _, tc := range cases {
		f := newFixture(t)
		f.repoFile(Config, tc.body)
		f.machineFile(Config, `{"pace":{"work_minutes":90}}`)
		s, err := Load(Config, f.roots)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Get(s, "pace.work_minutes", 120, positive)
		if err == nil {
			t.Fatalf("%s: Get admitted it as %+v", tc.body, got)
		}
		for _, want := range []string{tc.want, "pace.work_minutes", ".abcd/config.json"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: error %q does not name %q", tc.body, err, want)
			}
		}
	}
}

// TestIntermediateNotAnObjectRefuses: `"pace": 90` cannot hold pace.work_minutes;
// reading through it is a malformed configuration, not an absent key.
func TestIntermediateNotAnObjectRefuses(t *testing.T) {
	f := newFixture(t)
	f.repoFile(Config, `{"pace":90}`)
	s, err := Load(Config, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Get(s, "pace.work_minutes", 120, positive); err == nil {
		t.Fatal("a key read through a scalar was treated as absent")
	}
}

// TestLookupListsEveryLayerInPrecedenceOrder is what a board renders: each
// layer holding the key, highest first, so the winner is the first entry.
func TestLookupListsEveryLayerInPrecedenceOrder(t *testing.T) {
	f := newFixture(t)
	f.repoFile(OracleRouting, `{"schema_version":1,"agents":{"scribe":{"tier":"local"}}}`)
	f.machineFile(OracleRouting, `{"schema_version":1,"agents":{"scribe":{"tier":"economy"}}}`)
	s, err := Load(OracleRouting, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetFlag("agents.scribe", map[string]any{"tier": "frontier"}, "--route scribe=frontier"); err != nil {
		t.Fatal(err)
	}
	found, err := s.Lookup("agents.scribe")
	if err != nil {
		t.Fatal(err)
	}
	var layers []Layer
	for _, fd := range found {
		layers = append(layers, fd.Layer)
	}
	if fmt.Sprint(layers) != fmt.Sprint([]Layer{Flag, Repo, Machine}) {
		t.Fatalf("layers = %v", layers)
	}
	if found[0].Origin != "--route scribe=frontier" || string(found[1].Raw) != `{"tier":"local"}` {
		t.Fatalf("found = %+v", found)
	}
	members, err := s.Members(Machine, "agents")
	if err != nil || fmt.Sprint(members) != "[scribe]" {
		t.Fatalf("Members = %v, %v", members, err)
	}
}

func TestSetFlagRefusesAMalformedKey(t *testing.T) {
	f := newFixture(t)
	s, err := Load(Config, f.roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", ".pace", "pace.", "pace..x"} {
		if err := s.SetFlag(key, 1, "--x"); err == nil {
			t.Fatalf("SetFlag(%q) admitted", key)
		}
	}
}

func TestLayerNames(t *testing.T) {
	want := map[Layer]string{None: "none", Bundled: "bundled", Machine: "machine", Repo: "repo", Flag: "flag"}
	for l, s := range want {
		if l.String() != s {
			t.Fatalf("%d.String() = %q, want %q", l, l.String(), s)
		}
	}
}
