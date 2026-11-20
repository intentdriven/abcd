package site

// A composed site whose record is built elsewhere (spc-2610142146250176 step
// 3): `"record": {"elsewhere": {"address": …, "forward": …, "narrow": …}}`
// names the address and holds the two cut-over switches step 4 turns on, each
// on the person's word and each on its own. Both off, the site builds exactly
// what it built before. Forward on, each of the record's addresses here
// answers with a 301 to the same path at the address. Narrow on, the build
// writes none of the record's pages and no record.json, and every link into
// the record names the address; the contributors and references pages stay,
// since the record's own site, the project route, builds neither
// (dcn-2611201438402486).

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// elsewhereAddress is the fixture's record address.
const elsewhereAddress = "https://example.github.io/widgets/"

// declareRecordElsewhere names the fixture's record at elsewhereAddress, with
// the two switches as given.
func declareRecordElsewhere(t *testing.T, f *fixture, forward, narrow bool) {
	t.Helper()
	repointManifest(t, f, `"record": {"issue_ledger": true},`,
		fmt.Sprintf(`"record": {"issue_ledger": true, "elsewhere": {"address": %q, "forward": %v, "narrow": %v}},`,
			elsewhereAddress, forward, narrow))
}

// builtTree is every file a build wrote, by output path.
func builtTree(t *testing.T, out string, res Result) map[string]string {
	t.Helper()
	tree := map[string]string{}
	for _, name := range res.Files {
		tree[name] = outFile(t, out, name)
	}
	return tree
}

// Declaring the address with both switches off changes nothing the site
// serves: the same files, byte for byte. This is what lets step 3 merge, and
// a release follow it, with nothing public moving.
func TestARecordAddressWithBothSwitchesOffBuildsTheSameSite(t *testing.T) {
	before := newFixture(t)
	beforeOut := t.TempDir()
	want := builtTree(t, beforeOut, buildFixture(t, before, beforeOut))

	f := newFixture(t)
	declareRecordElsewhere(t, f, false, false)
	out := t.TempDir()
	got := builtTree(t, out, buildFixture(t, f, out))

	if len(got) != len(want) {
		t.Errorf("the build wrote %d files, and %d before the address was declared", len(got), len(want))
	}
	for name, body := range want {
		if got[name] != body {
			t.Errorf("%s differs once the record's address is declared with both switches off", name)
		}
	}
}

// hostRedirect is one rule of the docs host's redirect map.
type hostRedirect struct {
	line             int
	from, to, status string
}

// readHostRedirects parses a redirect map as the host reads it.
func readHostRedirects(data string) []hostRedirect {
	var out []hostRedirect
	sc := bufio.NewScanner(strings.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		r := hostRedirect{line: n, from: fields[0], to: fields[1], status: "302"}
		if len(fields) > 2 {
			r.status = fields[2]
		}
		out = append(out, r)
	}
	return out
}

// follow is where the host sends path p, and with what status: the top-most
// rule whose source matches wins, a trailing `*` matches the rest of the path
// (nothing included) and `:splat` in the destination is what it matched. ok
// is false when no rule matches and the host serves p itself. The host keeps
// the query string over a redirect by itself (checked against the docs
// host's live redirects on 2026-11-20: /reference/?q=1 answers 301 to
// /docs/reference/?q=1), so a rule never names one.
func follow(rules []hostRedirect, p string) (to, status string, ok bool) {
	for _, r := range rules {
		if prefix, dynamic := strings.CutSuffix(r.from, "*"); dynamic {
			if rest, match := strings.CutPrefix(p, prefix); match {
				return strings.ReplaceAll(r.to, ":splat", rest), r.status, true
			}
			continue
		}
		if r.from == p {
			return r.to, r.status, true
		}
	}
	return "", "", false
}

func TestTheForwardSwitchForwardsTheRecordsAddressesToTheSamePathThere(t *testing.T) {
	f := newFixture(t)
	declareRecordElsewhere(t, f, true, false)
	out := t.TempDir()
	buildFixture(t, f, out)
	rules := readHostRedirects(outFile(t, out, redirectsOutRel))

	// Every address the record is served at, by kind: the bare directory
	// without its slash, the explorer's home, its pages, one record's page,
	// and the export.
	for _, p := range []string{
		"/record", "/record/", "/record/graph/", "/record/foundations/",
		"/record/glossary/core/phase/", "/record/intent/itd-1/", "/record.json",
	} {
		want := strings.TrimSuffix(elsewhereAddress, "/") + p
		to, status, ok := follow(rules, p)
		switch {
		case !ok:
			t.Errorf("%s does not forward: no rule in the built %s matches it", p, redirectsOutRel)
		case to != want || status != "301":
			t.Errorf("%s forwards with %s to %s, want 301 to %s", p, status, to, want)
		}
	}
	// The documentation site's own addresses stay, the contributors and
	// references pages among them, and the committed map still answers.
	for _, p := range []string{"/", "/contributors/", "/references/", "/docs/", "/install.sh", "/records/", "/recordings"} {
		if to, _, ok := follow(rules, p); ok {
			t.Errorf("%s forwards to %s, but it is the documentation site's own address", p, to)
		}
	}
	if to, status, ok := follow(rules, "/old/"); !ok || to != "/docs/old/" || status != "301" {
		t.Errorf("the committed rule for /old/ no longer answers: %s %s %v", to, status, ok)
	}
	// The host applies a static rule only ahead of every rule with a splat.
	dynamic := 0
	for _, r := range rules {
		if strings.ContainsAny(r.from, "*:") {
			if dynamic == 0 {
				dynamic = r.line
			}
			continue
		}
		if dynamic != 0 {
			t.Errorf("line %d is a static rule after the rule with a splat on line %d", r.line, dynamic)
		}
	}
	// Forwarding alone narrows nothing: the record is still built here, and
	// shadowed by the forwards until the narrow switch is turned on too.
	if outFile(t, out, "record/index.html") == "" || !strings.Contains(outFile(t, out, "index.html"), `href="/record/"`) {
		t.Error("the forward switch alone changed what the site builds")
	}
}

func TestTheNarrowSwitchLeavesTheContributorsAndReferencesPages(t *testing.T) {
	before := newFixture(t)
	beforeOut := t.TempDir()
	buildFixture(t, before, beforeOut)

	f := newFixture(t)
	declareRecordElsewhere(t, f, false, true)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	kept := map[string]bool{}
	for _, name := range res.Files {
		if name == "record.json" || strings.HasPrefix(name, "record/") {
			t.Errorf("the build wrote %s, but the record is built at %s", name, elsewhereAddress)
		}
		kept[name] = true
	}
	for _, page := range []string{"index.html", "contributors/index.html", "references/index.html"} {
		if !kept[page] {
			t.Fatalf("the build no longer writes %s", page)
		}
		// Each is the page it was, but that a link into the record now names
		// the record's address rather than this site's own path.
		want := strings.ReplaceAll(outFile(t, beforeOut, page), `href="/record`, `href="`+elsewhereAddress+`record`)
		if got := outFile(t, out, page); got != want {
			t.Errorf("%s changed beyond its links into the record:\n%s", page, strings.Join(diffLines(want, got), "\n"))
		}
	}
	// Narrowing alone forwards nothing: the redirect map is the committed one.
	if got := outFile(t, out, redirectsOutRel); got != outFile(t, beforeOut, redirectsOutRel) {
		t.Errorf("the narrow switch alone changed the redirect map:\n%s", got)
	}

	// The gate measures no ratchet where no record export is written, and
	// finds no link to a page that is not there.
	chk, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	for _, fd := range chk.Findings {
		if fd.Check == CheckBaseline || fd.Check == CheckLinkTargets {
			t.Errorf("the gate fails the narrowed render: %+v", fd)
		}
	}
}

func TestARecordAddressIsAnHTTPSDirectory(t *testing.T) {
	for _, addr := range []string{
		"http://example.github.io/widgets/",
		"https://example.github.io/widgets",
		"https://example.github.io/widgets/?x=1",
		"https://example.github.io/widgets/#top",
		"https:///widgets/",
		"https://someone@example.github.io/widgets/",
		"",
	} {
		t.Run(addr, func(t *testing.T) {
			f := newFixture(t)
			repointManifest(t, f, `"record": {"issue_ledger": true},`,
				`"record": {"issue_ledger": true, "elsewhere": {"address": "`+addr+`", "forward": false, "narrow": false}},`)
			if _, err := LoadManifest(f.Root()); !errors.Is(err, ErrManifestInvalid) || !strings.Contains(err.Error(), "record.elsewhere.address") {
				t.Fatalf("got %v, want a refusal naming record.elsewhere.address", err)
			}
		})
	}
	f := newFixture(t)
	declareRecordElsewhere(t, f, true, true)
	if m, err := LoadManifest(f.Root()); err != nil || m.Record.Elsewhere == nil || m.Record.Elsewhere.Address != elsewhereAddress {
		t.Fatalf("a well-formed record address does not load: %+v, %v", m.Record, err)
	}
	// A project site builds its record where it is served, so its settings
	// never name another address for it.
	f.write(ProjectSettingsRelPath, strings.Replace(projectSettingsJSON, `"schema_version": 1,`,
		`"schema_version": 1, "record": {"elsewhere": {"address": "`+elsewhereAddress+`", "forward": false, "narrow": false}},`, 1))
	if _, err := loadRoute(f.Root(), true); !errors.Is(err, ErrManifestInvalid) || !strings.Contains(err.Error(), "record.elsewhere") {
		t.Fatalf("a project site's settings naming a record address: %v", err)
	}
}

// abcd's documentation site names its record at the Pages address the
// project route builds it onto (spc-2610142146250176 step 3). The two
// switches are step 4's, turned on in the cut-over's order on the person's
// word, so this test holds the address and leaves them alone.
func TestAbcdsDocumentationSiteNamesItsRecordAtItsPagesAddress(t *testing.T) {
	m, err := LoadManifest(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	if m.Record.Elsewhere == nil || m.Record.Elsewhere.Address != abcdRecordAddress {
		t.Fatalf("%s names its record elsewhere as %+v, want at %s", ManifestRelPath, m.Record.Elsewhere, abcdRecordAddress)
	}
}
