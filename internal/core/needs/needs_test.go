package needs

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
)

// TestTheListIsReadByItsGrammar: .abcd/needs.json is one strict document of
// names, reasons and, for Go, the file that declares its version. Anything the
// grammar does not name is refused, so no path, checksum or version of the
// list's own can ride in on an extra field.
func TestTheListIsReadByItsGrammar(t *testing.T) {
	l, err := parseList([]byte(`{"schema_version": 1, "tools": [
		{"name": "go", "reason": "builds and tests the binary", "version_file": "go.mod"},
		{"name": "make", "reason": "runs the preflight gates"},
		{"name": "golangci-lint", "reason": "lints the code (the linter CI runs, too)"}
	]}`))
	if err != nil {
		t.Fatalf("a well-formed list: %v", err)
	}
	if len(l.Tools) != 3 || l.Tools[0] != (Tool{Name: "go", Reason: "builds and tests the binary", VersionFile: "go.mod"}) ||
		l.Tools[1].Name != "make" || l.Tools[2].Name != "golangci-lint" {
		t.Fatalf("list = %+v", l)
	}
	if empty, err := parseList([]byte(`{"schema_version": 1, "tools": []}`)); err != nil || len(empty.Tools) != 0 {
		t.Fatalf("an empty list = %+v, %v; it lists nothing and refuses nothing", empty, err)
	}
	if ListPath != ".abcd/needs.json" {
		t.Fatalf("ListPath = %q", ListPath)
	}

	refused := map[string]string{
		"not json":            `{"schema_version": 1, "tools": [`,
		"trailing content":    `{"schema_version": 1, "tools": []} {}`,
		"no schema version":   `{"tools": []}`,
		"a later schema":      `{"schema_version": 2, "tools": []}`,
		"an unknown field":    `{"schema_version": 1, "tools": [{"name": "go", "reason": "builds", "version_file": "go.mod", "path": "/usr/local/go/bin/go"}]}`,
		"a checksum":          `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "sha256": "00"}]}`,
		"a version":           `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "version": "4.4"}]}`,
		"a top-level extra":   `{"schema_version": 1, "tools": [], "search": ["/opt/tools"]}`,
		"a repeated key":      `{"schema_version": 1, "schema_version": 1, "tools": []}`,
		"a repeated tool":     `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds"}, {"name": "make", "reason": "again"}]}`,
		"no name":             `{"schema_version": 1, "tools": [{"reason": "builds"}]}`,
		"no reason":           `{"schema_version": 1, "tools": [{"name": "make"}]}`,
		"a path for a name":   `{"schema_version": 1, "tools": [{"name": "/usr/bin/make", "reason": "builds"}]}`,
		"a reserved name":     `{"schema_version": 1, "tools": [{"name": "bash", "reason": "runs scripts"}]}`,
		"a path in a reason":  `{"schema_version": 1, "tools": [{"name": "make", "reason": "lives in /opt/make"}]}`,
		"go with no file":     `{"schema_version": 1, "tools": [{"name": "go", "reason": "builds"}]}`,
		"go with another":     `{"schema_version": 1, "tools": [{"name": "go", "reason": "builds", "version_file": ".tool-versions"}]}`,
		"a file for make":     `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "version_file": "go.mod"}]}`,
		"a nested go.mod":     `{"schema_version": 1, "tools": [{"name": "go", "reason": "builds", "version_file": "tools/go.mod"}]}`,
		"tools not an array":  `{"schema_version": 1, "tools": {"go": "builds"}}`,
		"a document too long": `{"schema_version": 1, "tools": [], "x": "` + strings.Repeat("a", MaxListBytes) + `"}`,
	}
	for name, doc := range refused {
		t.Run(name, func(t *testing.T) {
			if l, err := parseList([]byte(doc)); err == nil {
				t.Fatalf("read as %+v; want it refused", l)
			}
		})
	}
	if _, err := parseList([]byte(`{"schema_version": 1, "schema_version": 1, "tools": []}`)); !errors.As(err, new(*jsonstrict.DuplicateKeyError)) {
		t.Fatalf("a repeated key: %v, want the strict decode's refusal", err)
	}
}

// TestARefusedListNeverRepeatsItsText: a list the strict decode refuses is
// refused in words abcd wrote, never in the decoder's, which repeat the key
// the repository chose; errors.As still reaches the decoder's own error.
func TestARefusedListNeverRepeatsItsText(t *testing.T) {
	const planted = "see /opt/evil/bin/make"
	for name, doc := range map[string]string{
		"an unknown field": `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "` + planted + `": 1}]}`,
		"a repeated key":   `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "` + planted + `": 1, "` + planted + `": 1}]}`,
		"a top-level key":  `{"schema_version": 1, "tools": [], "` + planted + `": 1}`,
		"a bad value":      `{"schema_version": 1, "tools": [{"name": "make", "reason": "builds"}], "` + planted + `"`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseList([]byte(doc))
			if err == nil {
				t.Fatal("accepted, want it refused")
			}
			if strings.Contains(err.Error(), "/opt/evil") || strings.Contains(err.Error(), "see ") {
				t.Fatalf("the refusal repeats the repository's key: %v", err)
			}
			if !strings.Contains(err.Error(), ListPath) {
				t.Fatalf("the refusal does not name the list: %v", err)
			}
			var le *ListError
			if !errors.As(err, &le) || le.Unwrap() == nil {
				t.Fatalf("err = %v, want a *ListError carrying the decoder's error", err)
			}
		})
	}
	_, err := parseList([]byte(`{"schema_version": 1, "tools": [{"name": "make", "reason": "builds", "` + planted + `": 1, "` + planted + `": 1}]}`))
	var dup *jsonstrict.DuplicateKeyError
	if !errors.As(err, &dup) {
		t.Fatalf("a repeated key: %v, want errors.As to reach the strict decode's refusal", err)
	}
}

// TestANameFollowsTheStrictGrammar: a name is a plain lowercase program name,
// never a path, an option, a hidden file or a word with a space in it.
func TestANameFollowsTheStrictGrammar(t *testing.T) {
	for _, ok := range []string{"go", "make", "golangci-lint", "python3", "g++", "protoc-gen-go", "clang-format", "x", "yq_4", "a.b"} {
		if err := checkName(ok); err != nil {
			t.Errorf("%q: %v, want it accepted", ok, err)
		}
	}
	bad := []string{
		"", "Go", "MAKE", "-rf", ".hidden", "..", ".", "_x", "+x", "a/b", `a\b`, "~", "$HOME", "a b",
		"go\n", "go\x00", "gö", "a;b", "a*", "`x`", "'x'", "\"x\"", strings.Repeat("a", 65),
	}
	for _, name := range bad {
		if err := checkName(name); err == nil {
			t.Errorf("%q accepted, want it refused", name)
		}
	}
	if err := checkName(strings.Repeat("a", 64)); err != nil {
		t.Errorf("a 64-character name: %v, want it accepted", err)
	}
}

// TestReservedNamesAreRefusedAndGoAndMakeAreNot: a shell, git, abcd, node and
// the command each runner starts are refused, since listing one would move a
// registered copy ahead of the one a harness or abcd itself runs on. go and
// make are not reserved: abcd runs them, but a repository may need them.
func TestReservedNamesAreRefusedAndGoAndMakeAreNot(t *testing.T) {
	for _, name := range []string{"sh", "bash", "zsh", "dash", "ksh", "fish", "csh", "tcsh", "pwsh", "busybox", "toybox", "osh", "ysh",
		"git", "abcd", "node", "claude", "opencode"} {
		if !isReserved(name) {
			t.Errorf("%s is not reserved", name)
		}
		if err := checkName(name); err == nil {
			t.Errorf("%s passes checkName, want it refused as reserved", name)
		} else if !strings.Contains(err.Error(), "reserved") {
			t.Errorf("%s: %v, want the refusal to say it is reserved", name, err)
		}
	}
	for _, name := range []string{"go", "make", "gofmt", "git-lfs", "nodejs-tool"} {
		if isReserved(name) {
			t.Errorf("%s is reserved, want it listable", name)
		}
		if err := checkName(name); err != nil {
			t.Errorf("%s: %v, want it accepted", name, err)
		}
	}
}

// TestEachToolHasAClosedSetOfVersionFiles: Go's version comes from go.mod or
// go.work at the repository's root and nowhere else, and a tool abcd checks no
// version for names no file at all.
func TestEachToolHasAClosedSetOfVersionFiles(t *testing.T) {
	if got := versionFiles("go"); len(got) != 2 || got[0] != "go.mod" || got[1] != "go.work" {
		t.Fatalf("versionFiles(go) = %q, want [go.mod go.work]", got)
	}
	got := versionFiles("go")
	got[0] = "changed"
	if versionFiles("go")[0] != "go.mod" {
		t.Fatal("versionFiles hands out the closed set itself; a caller changed it")
	}
	for _, f := range []string{"go.mod", "go.work"} {
		if err := checkVersionFile("go", f); err != nil {
			t.Errorf("go %s: %v", f, err)
		}
	}
	for _, f := range []string{"", ".tool-versions", ".go-version", "mise.toml", "tools/go.mod", "./go.mod", "/go.mod", "go.sum", "GO.MOD"} {
		if err := checkVersionFile("go", f); err == nil {
			t.Errorf("go %q accepted, want it refused", f)
		}
	}
	if len(versionFiles("make")) != 0 {
		t.Fatalf("versionFiles(make) = %q, want none", versionFiles("make"))
	}
	if err := checkVersionFile("make", ""); err != nil {
		t.Errorf("make with no version file: %v", err)
	}
	for _, f := range []string{"go.mod", "Makefile", ".tool-versions"} {
		if err := checkVersionFile("make", f); err == nil {
			t.Errorf("make %q accepted, want it refused: abcd checks no version of make", f)
		}
	}
}

// TestAReasonHasALimitAndAnAllowlist: a reason is shown as data, so it may
// describe a place in words but cannot carry a path to type: no path
// separator, ~, $, backtick, quote or control character, and no more than
// MaxReasonRunes characters.
func TestAReasonHasALimitAndAnAllowlist(t *testing.T) {
	for _, ok := range []string{
		"builds and tests the binary",
		"runs the preflight gates (fmt, vet, race)",
		"lints: the linter CI runs, too",
		"généré par protoc; 100% needed!",
		"the copy in the tools directory of the home folder",
		strings.Repeat("a", MaxReasonRunes),
		strings.Repeat("é", MaxReasonRunes),
	} {
		if err := checkReason(ok); err != nil {
			t.Errorf("%q: %v, want it accepted", ok, err)
		}
	}
	for _, bad := range []string{
		"",
		strings.Repeat("a", MaxReasonRunes+1),
		"lives in /opt/make",
		`lives in C:\tools`,
		"in ~ somewhere",
		"see $HOME",
		"run `make`",
		`the "real" one`,
		"the 'real' one",
		"the \u2018real\u2019 one",
		"the \u201creal\u201d one",
		"two\nlines",
		"a\ttab",
		"bell\a",
		"escape \x1b[31m",
		"right-to-left \u202e override",
		"zero\u200bwidth",
		"fullwidth \uff0f slash",
		"division \u2215 slash",
		"invalid \xff utf-8",
		"e\u0301 decomposed",
		"\u092c\u093f\u0932\u094d\u0921",
		"long solidus a\u0338 overlay",
		"short solidus a\u0337 overlay",
		"tilde a\u0334 overlay",
		"long stroke a\u0336 overlay",
		"combining long vertical a\u20d2 overlay",
	} {
		if err := checkReason(bad); err == nil {
			t.Errorf("%q accepted, want it refused", bad)
		}
	}
}
