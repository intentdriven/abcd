package site

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The install command exists in four committed forms: the universal one-liner in
// README.md, the script in site-src/install.sh.tmpl, and the per-OS one-liners
// under docs/how-to/install.md's macOS and Linux headings. adr-47 makes all four
// surfaces of one repository, so they are one command written four times — and a
// reader who follows any of them must end up with the same bytes in the same
// place.
//
// The per-OS forms are the universal one with OS detection resolved: same release
// URLs, same checksum step, same install destination, `uname -s` replaced by the
// OS it would have printed. This test holds them to exactly that, so a change to
// one of them fails the build rather than shipping a page that quietly disagrees
// with the script beside it.
const (
	checksumsAsset = "checksums.txt"
	installDir     = "$HOME/.local/bin"
	osDetection    = "uname -s"

	verifierGNU = "sha256sum -c"
	verifierBSD = "shasum -a 256 -c"
)

// downloadURL matches a release-asset download, strictly: the scheme, host and
// path are the assertion, and only the asset name after the final slash is free.
var downloadURL = regexp.MustCompile(`https://[^\s"']+/releases/latest/download/[^\s"']*`)

// binaryName matches the shell assignment that names the release asset, in
// either spelling the surfaces use (`b=` in a one-liner, `binary=` in the
// script). The captured platform expression is what must agree.
var binaryName = regexp.MustCompile(`\b(?:b|binary)="abcd-([^"]*)"`)

// manifestLookup matches the grep that pulls this binary's line out of
// checksums.txt. Anchored on the trailing `$` because that anchor is the check:
// without it a manifest that does not list the binary still matches something.
var manifestLookup = regexp.MustCompile(`grep\s+"\s\$[A-Za-z0-9_]+\\?\$"\s+checksums\.txt`)

// installInvocation matches `install -m 0755` only in COMMAND POSITION — at the
// start of a line or after a shell separator. A Contains check would accept
// `sudo install -m 0755`, which is the one thing the install must never be:
// abcd does not escalate privileges, and a command that asks for a password to
// write into the user's own directory is a command that has been tampered with.
// Internal whitespace is free; the anchor is not.
var installInvocation = regexp.MustCompile(`(?m)(?:^|[;&|(]|\bthen\b|\bdo\b|\belse\b)\s*install\s+-m\s+0755\b`)

// privilegeEscalation matches the two escalation commands by word, so a mention
// inside a longer word ("pseudo") does not trip it.
var privilegeEscalation = regexp.MustCompile(`\b(?:sudo|doas)\b`)

// The curl hardening bootstrap already carries (hooks/bootstrap.sh), held here
// so the public install surfaces cannot drift back to the vacuous-verification
// hole GHSA-x4v8-rxvx-8v89 records. `--proto '=https'` keeps the scheme, but it
// does NOT defend the connection: a ~/.curlrc `connect-to`/`resolve` line still
// re-points where curl actually dials while the URL still reads https://github…,
// and an HTTPS_PROXY + CURL_CA_BUNDLE pair still reroutes the fetch through a
// server of the setter's choosing. Because the same poisoned surface serves both
// the binary AND the checksums.txt that "verifies" it, SHA-256 verification is
// vacuous unless the fetch is closed at the source: `-q` as curl's FIRST argument
// (so no .curlrc is read at all) and the proxy/CA/CURL_HOME scrub before any
// fetch. A surface that pins the transport but skips those two is the exact drift
// this holds every form to.

// fetchingCurl matches a curl used to fetch — `curl` followed by a dash option.
// `command -v curl >/dev/null` is `curl` followed by a redirection, not a dash,
// so it is not counted; every match here is a download that must be hardened.
var fetchingCurl = regexp.MustCompile(`curl\s+-`)

// qFirstCurl matches a curl whose FIRST argument is `-q`. Only the first
// position suppresses the .curlrc read: `curl -fsSL -q` still loads it before it
// sees the flag.
var qFirstCurl = regexp.MustCompile(`curl\s+-q\b`)

// protoPin and protoRedirPin match the HTTPS transport pin in either quoting the
// surfaces use — `'=https'` in the standalone script, bare `=https` inside the
// single-quoted `sh -c '…'` one-liners, which cannot nest a single quote. The
// `\s` after `--proto` keeps protoPin from also matching `--proto-redir`.
var protoPin = regexp.MustCompile(`--proto\s+'?=https'?`)
var protoRedirPin = regexp.MustCompile(`--proto-redir\s+'?=https'?`)

// scrubbedCurlEnv is the environment every surface must unset before it fetches:
// the proxy variables that reroute the connection, the CA overrides that make
// such a route pass TLS, and CURL_HOME, which relocates the .curlrc `-q` alone
// would otherwise still be asked to ignore. This is bootstrap's list, verbatim.
var scrubbedCurlEnv = []string{
	"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy",
	"ALL_PROXY", "all_proxy", "CURL_HOME",
	"CURL_CA_BUNDLE", "SSL_CERT_FILE", "SSL_CERT_DIR",
}

// unsetsVar matches an `unset` statement that lists the named variable. The
// character class stops at a newline or `;`, so a variable unset on one statement
// is not credited to an `unset` on a different one.
func unsetsVar(script, v string) bool {
	return regexp.MustCompile(`\bunset\b[^\n;]*\b` + regexp.QuoteMeta(v) + `\b`).MatchString(script)
}

// TestInstallSurfacesHardenCurl holds all four public install surfaces (and the
// served render) to the curl hardening bootstrap already carries. Without it the
// checksum step every surface performs is defeated by a local curl surface the
// victim need not have set maliciously — direnv, a sourced .env — so the fix is
// not "verify harder" but "close the fetch": -q first, the env scrub, the proto
// pins. See GHSA-x4v8-rxvx-8v89 and hooks/bootstrap.sh.
func TestInstallSurfacesHardenCurl(t *testing.T) {
	for _, s := range loadInstallSurfaces(t) {
		t.Run(s.name, func(t *testing.T) {
			fetches := len(fetchingCurl.FindAllString(s.script, -1))
			if fetches == 0 {
				t.Fatalf("%s makes no curl fetch to harden", s.source)
			}

			if got := len(qFirstCurl.FindAllString(s.script, -1)); got != fetches {
				t.Errorf("%s: %d curl fetch(es), %d lead with -q; -q must be curl's FIRST argument or ~/.curlrc is read and the checksum check is vacuous", s.source, fetches, got)
			}
			if got := len(protoPin.FindAllString(s.script, -1)); got != fetches {
				t.Errorf("%s: %d curl fetch(es), %d carry --proto =https", s.source, fetches, got)
			}
			if got := len(protoRedirPin.FindAllString(s.script, -1)); got != fetches {
				t.Errorf("%s: %d curl fetch(es), %d carry --proto-redir =https", s.source, fetches, got)
			}
			for _, v := range scrubbedCurlEnv {
				if !unsetsVar(s.script, v) {
					t.Errorf("%s does not unset %s before fetching; a poisoned proxy/CA/CURL_HOME surface reroutes both the binary and the checksums.txt that verifies it", s.source, v)
				}
			}
		})
	}
}

// archMapping is one `uname -m` value and the release architecture it must map
// to. The release matrix names amd64 and arm64; a machine that reports the
// kernel spellings and is not translated downloads an asset that does not exist.
type archMapping struct{ from, to string }

var (
	amd64FromX8664   = archMapping{"x86_64", "amd64"}
	arm64FromAarch64 = archMapping{"aarch64", "arm64"}
)

// installSurface is one committed form of the install command.
type installSurface struct {
	name      string        // what to call it in a failure message
	source    string        // where it lives, repo-relative
	script    string        // the command text itself
	osToken   string        // the OS the form is specialised to: "$os" when it detects
	verifiers []string      // the checksum tools this form may use
	archMap   []archMapping // the `uname -m` translations this form must carry
}

func TestInstallSurfacesAgree(t *testing.T) {
	surfaces := loadInstallSurfaces(t)

	for _, s := range surfaces {
		t.Run(s.name, func(t *testing.T) {
			assertDownloads(t, s)
			assertChecksumStep(t, s)
			assertDestination(t, s)
			assertNoPrivilegeEscalation(t, s)
			assertOSResolution(t, s)
			assertArchMapping(t, s)
		})
	}
}

// assertDownloads checks that the form fetches exactly two things — the platform
// binary and the manifest that verifies it — from one release-download prefix,
// the same prefix every other form uses. Two origins would make the check
// vacuous: whoever serves the manifest can serve the payload it "verifies".
func assertDownloads(t *testing.T, s installSurface) {
	t.Helper()

	want := releaseDownloadPrefix(t)
	prefixes := map[string]bool{}
	gotChecksums, gotBinary := false, false
	for _, u := range downloadURL.FindAllString(s.script, -1) {
		cut := strings.LastIndex(u, "/")
		prefix, asset := u[:cut+1], u[cut+1:]
		prefixes[prefix] = true
		switch {
		case asset == checksumsAsset:
			gotChecksums = true
		case strings.HasPrefix(asset, "$"):
			gotBinary = true
		default:
			t.Errorf("%s downloads a fixed asset %q; the binary is per-platform", s.source, asset)
		}
	}

	if len(prefixes) != 1 {
		t.Fatalf("%s downloads from %d release prefixes, want exactly 1: %v", s.source, len(prefixes), sortedKeys(prefixes))
	}
	if !prefixes[want] {
		t.Errorf("%s downloads from %v, want %q", s.source, sortedKeys(prefixes), want)
	}
	if !gotBinary {
		t.Errorf("%s never downloads the platform binary from %s", s.source, want)
	}
	if !gotChecksums {
		t.Errorf("%s never downloads %s from %s", s.source, checksumsAsset, want)
	}
}

// assertChecksumStep checks that the form looks its binary up in the manifest and
// verifies the digest with the tool its platform has. A form that skipped either
// half would install an unverified download.
func assertChecksumStep(t *testing.T, s installSurface) {
	t.Helper()

	if !manifestLookup.MatchString(collapse(s.script)) {
		t.Errorf("%s has no anchored %s lookup for its binary", s.source, checksumsAsset)
	}

	var got []string
	for _, v := range []string{verifierGNU, verifierBSD} {
		if strings.Contains(collapse(s.script), v) {
			got = append(got, v)
		}
	}
	if !equalStrings(got, s.verifiers) {
		t.Errorf("%s verifies with %v, want %v", s.source, got, s.verifiers)
	}
}

// assertDestination checks that the form installs the same way into the same
// single-user directory, with the install itself in command position.
func assertDestination(t *testing.T, s installSurface) {
	t.Helper()

	if !installInvocation.MatchString(s.script) {
		t.Errorf("%s has no `install -m 0755` in command position", s.source)
	}
	if !strings.Contains(s.script, installDir) {
		t.Errorf("%s does not install into %q", s.source, installDir)
	}
}

// assertNoPrivilegeEscalation checks the invariant every form shares: abcd never
// asks for administrator rights. ~/.local/bin is the user's own directory, so a
// surface that reaches for sudo is either wrong or hostile.
func assertNoPrivilegeEscalation(t *testing.T, s installSurface) {
	t.Helper()

	if found := privilegeEscalation.FindAllString(s.script, -1); len(found) > 0 {
		t.Errorf("%s escalates privileges (%v); abcd installs into a directory the user already owns", s.source, found)
	}
}

// assertOSResolution checks the one difference the forms are allowed: a per-OS
// form names its OS where the universal form detects it, and nothing else moves.
func assertOSResolution(t *testing.T, s installSurface) {
	t.Helper()

	detects := strings.Contains(s.script, osDetection)
	if want := s.osToken == "$os"; detects != want {
		t.Errorf("%s detects the OS with %q: %v, want %v", s.source, osDetection, detects, want)
	}

	m := binaryName.FindAllStringSubmatch(s.script, -1)
	if len(m) != 1 {
		t.Fatalf("%s names the release binary %d times, want exactly 1", s.source, len(m))
	}
	platform := m[0][1]
	if !strings.HasPrefix(platform, s.osToken+"-") {
		t.Fatalf("%s builds the binary name from %q, want it to start with %q", s.source, platform, s.osToken)
	}
	if rest := strings.TrimPrefix(platform, s.osToken+"-"); rest != "$arch" {
		t.Errorf("%s resolves more than the OS: binary name is abcd-%s, want the architecture left as $arch", s.source, platform)
	}
}

// assertArchMapping checks that the form translates the kernel's architecture
// spellings into the ones the release matrix publishes. The OS-detecting forms
// carry both translations; the macOS form carries only x86_64, because Apple
// silicon already reports arm64 and aarch64 never appears there.
func assertArchMapping(t *testing.T, s installSurface) {
	t.Helper()

	for _, m := range s.archMap {
		mapping := regexp.MustCompile(regexp.QuoteMeta(m.from) + `\s*\)\s*arch=` + regexp.QuoteMeta(m.to) + `\b`)
		if !mapping.MatchString(s.script) {
			t.Errorf("%s does not map %s to %s; that machine would download an asset the release does not publish", s.source, m.from, m.to)
		}
	}
}

func loadInstallSurfaces(t *testing.T) []installSurface {
	t.Helper()

	readmePath := filepath.Join(repoRoot(), "README.md")
	templatePath := filepath.Join(repoRoot(), "site-src", "install.sh.tmpl")
	guidePath := filepath.Join(repoRoot(), "docs", "how-to", "install.md")

	guide := readFile(t, guidePath)
	bothArches := []archMapping{amd64FromX8664, arm64FromAarch64}

	surfaces := []installSurface{
		{
			name:      "readme-one-liner",
			source:    "README.md",
			script:    onlyInstallBlock(t, "README.md", fencedBlocks(readFile(t, readmePath))),
			osToken:   "$os",
			verifiers: []string{verifierGNU, verifierBSD},
			archMap:   bothArches,
		},
		{
			name:      "install-sh-template",
			source:    "site-src/install.sh.tmpl",
			script:    readFile(t, templatePath),
			osToken:   "$os",
			verifiers: []string{verifierGNU, verifierBSD},
			archMap:   bothArches,
		},
		{
			name:      "how-to-macos",
			source:    "docs/how-to/install.md (### macOS)",
			script:    onlyInstallBlock(t, "docs/how-to/install.md (### macOS)", blocksUnder(fencedBlocks(guide), "macOS")),
			osToken:   "darwin",
			verifiers: []string{verifierBSD},
			archMap:   []archMapping{amd64FromX8664},
		},
		{
			name:      "how-to-linux",
			source:    "docs/how-to/install.md (### Linux)",
			script:    onlyInstallBlock(t, "docs/how-to/install.md (### Linux)", blocksUnder(fencedBlocks(guide), "Linux")),
			osToken:   "linux",
			verifiers: []string{verifierGNU},
			archMap:   bothArches,
		},
	}

	// The FIFTH form: the script this repository SERVES at /install.sh. It is the
	// only form a reader of abcdev.app ever executes — the template is what was
	// reviewed, this is what is downloaded — so holding it to the same rules is
	// what makes the render's "copy plus a comment" claim checkable rather than
	// asserted.
	//
	// It is RENDERED here rather than read off the disk. `abcd site build` writes
	// it into a gitignored directory, and a test that read that file would make
	// the suite's answer depend on whether somebody had happened to run a build:
	// skipped in every clean checkout, including CI, and able to fail red over a
	// stale artifact that `git status` does not show and no edit to the tree can
	// fix. So the surface goes through the same function the build calls, and
	// `TestBuildRendersTheInstallScript` closes the loop by proving that what
	// `Build` writes to install.sh is exactly what that function returns.
	surfaces = append(surfaces, installSurface{
		name:      "emitted-install-sh",
		source:    "the rendered " + installScriptName,
		script:    string(renderInstallScript([]byte(readFile(t, templatePath)), fixtureStamp)),
		osToken:   "$os",
		verifiers: []string{verifierGNU, verifierBSD},
		archMap:   bothArches,
	})
	return surfaces
}

func repoRoot() string { return filepath.Join("..", "..", "..") }

// releaseDownloadPrefix derives the release-download base from go.mod's module
// path rather than hardcoding it. go.mod is the one place this repository's
// identity is already load-bearing for the build, so a rename lands there first
// and these surfaces fail WITH the rename instead of after it — the same
// derivation rule internal/core/launch's install-guide detector applies to the
// marketplace slug.
func releaseDownloadPrefix(t *testing.T) string {
	t.Helper()

	gomod := readFile(t, filepath.Join(repoRoot(), "go.mod"))
	for _, line := range strings.Split(gomod, "\n") {
		path, ok := strings.CutPrefix(strings.TrimSpace(line), "module ")
		if !ok {
			continue
		}
		path = strings.TrimSpace(path)
		if strings.Count(path, "/") != 2 {
			t.Fatalf("module path %q is not host/owner/repo", path)
		}
		return "https://" + path + "/releases/latest/download/"
	}
	t.Fatal("go.mod declares no module path")
	return ""
}

// fencedBlock is a fenced code block and the H3 it sits under, if any.
type fencedBlock struct {
	heading string
	body    string
}

// fencedBlocks splits Markdown into its fenced code blocks. Fences are matched on
// the trimmed line so indentation inside a list does not hide one. An H1 or H2
// CLEARS the current H3: without that, blocks in a later section inherit the last
// H3 seen and a block can be attributed to a heading it does not sit under.
func fencedBlocks(md string) []fencedBlock {
	var out []fencedBlock
	var body []string
	inFence, heading := false, ""

	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			if inFence {
				out = append(out, fencedBlock{heading: heading, body: strings.Join(body, "\n")})
				body = nil
			}
			inFence = !inFence
		case inFence:
			body = append(body, line)
		case strings.HasPrefix(trimmed, "### "):
			heading = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
		case strings.HasPrefix(trimmed, "## "), strings.HasPrefix(trimmed, "# "):
			heading = ""
		}
	}
	return out
}

// blocksUnder keeps the blocks sitting under one H3.
func blocksUnder(blocks []fencedBlock, heading string) []fencedBlock {
	var out []fencedBlock
	for _, b := range blocks {
		if b.heading == heading {
			out = append(out, b)
		}
	}
	return out
}

// onlyInstallBlock returns the single block that downloads from the release, and
// fails when a surface grew a second one — two install commands on one page is
// the drift this test exists to catch.
func onlyInstallBlock(t *testing.T, source string, blocks []fencedBlock) string {
	t.Helper()

	var found []string
	for _, b := range blocks {
		if downloadURL.MatchString(b.body) {
			found = append(found, b.body)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s holds %d release-download command blocks, want exactly 1", source, len(found))
	}
	return found[0]
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// collapse folds every run of whitespace to one space, so a command split over
// lines in the script reads the same as the one-liner it agrees with.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestLinkedAssetsMatchTheReleaseMatrix earns the manifest's `release.assets`
// claim.
//
// The hero links four binaries and a checksums manifest by NAME. The manifest
// says those names come from the release workflow, and the build deliberately
// does not read that workflow — a page that parses CI configuration to decide
// what to link is a worse dependency than a constant. So the claim is held here
// instead: the linked names must be exactly the `abcd-<os>-<arch>` matrix the
// four committed install surfaces themselves derive.
//
// A platform added to the release matrix with no link on the page, or a link to
// a platform nothing publishes, fails this test — which is the whole content of
// the claim.
func TestLinkedAssetsMatchTheReleaseMatrix(t *testing.T) {
	surfaces := loadInstallSurfaces(t)

	// The OSes the per-OS surfaces resolve to, and the architectures their
	// mappings produce. Both are read off the surfaces rather than restated.
	oses := map[string]bool{}
	arches := map[string]bool{}
	for _, s := range surfaces {
		if s.osToken != "$os" {
			oses[s.osToken] = true
		}
		for _, m := range s.archMap {
			arches[m.to] = true
		}
	}
	// Apple silicon reports arm64 already, so no surface maps to it; it is the
	// passthrough the aarch64 mapping proves is a published architecture.
	if !arches["arm64"] || !arches["amd64"] {
		t.Fatalf("the install surfaces resolve architectures %v; expected at least amd64 and arm64", keysOf(arches))
	}
	if len(oses) == 0 {
		t.Fatal("no per-OS install surface names an operating system")
	}

	want := map[string]bool{}
	for os := range oses {
		for arch := range arches {
			want["abcd-"+os+"-"+arch] = true
		}
	}

	got := map[string]bool{}
	for _, a := range LinkedBinaryAssets {
		got[a] = true
	}
	for a := range want {
		if !got[a] {
			t.Errorf("the release matrix publishes %s but the page links no such asset", a)
		}
	}
	for a := range got {
		if !want[a] {
			t.Errorf("the page links %s, which the install surfaces' release matrix does not publish", a)
		}
	}

	// The one fixed-name asset must be the same file every install form verifies
	// against — a page offering a different manifest would verify nothing.
	if AssetChecksums != checksumsAsset {
		t.Errorf("the page links %q as the checksum manifest; the install surfaces use %q",
			AssetChecksums, checksumsAsset)
	}
}

// keysOf renders a set for an error message, in a fixed order.
func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// symlinkedHomeGuard is the test every install form that records the owned
// PATH copy in ~/.abcd.noindex/path-entry runs before it fetches anything.
var symlinkedHomeGuard = `[ ! -L "$HOME/` + abcdhome.Rel() + `" ]`

// pathEntryRecord is the file a form writes to record the owned PATH copy, and
// oldHome the folder abcd's home was called before the rename. A form that
// created the old folder would leave every later abcd command stopped
// (spc-2610031309233367, "The stop").
var (
	pathEntryRecord = `"$HOME/` + abcdhome.Rel("path-entry") + `"`
	oldHome         = `"$HOME/.abcd"`
)

// writesPathEntry reports whether a form records the owned PATH copy, under
// either spelling of the home, so a form that still names the old folder is
// judged rather than skipped.
func writesPathEntry(script string) bool {
	return strings.Contains(script, "/path-entry")
}

// TestInstallSurfacesRecordInTheNoindexHome: a form that records the owned PATH
// copy writes the record under ~/.abcd.noindex and never names the old
// ~/.abcd. The binary stops while ~/.abcd stands, so a one-liner that created
// it would install a binary whose own --version, its last step, refuses to run.
func TestInstallSurfacesRecordInTheNoindexHome(t *testing.T) {
	seen := 0
	for _, s := range loadInstallSurfaces(t) {
		if !writesPathEntry(s.script) {
			continue
		}
		seen++
		t.Run(s.name, func(t *testing.T) {
			if !strings.Contains(s.script, "> "+pathEntryRecord) {
				t.Errorf("%s does not write its record to %s", s.source, pathEntryRecord)
			}
			if strings.Contains(s.script, oldHome) || strings.Contains(s.script, `"$HOME/.abcd/`) {
				t.Errorf("%s names the old folder %s; abcd stops while it stands", s.source, oldHome)
			}
		})
	}
	if seen == 0 {
		t.Fatal("no install form records the owned PATH copy; the README and the install guide's one-liners each do")
	}
}

// TestInstallSurfacesRefuseASymlinkedAbcdHome: a form that writes
// ~/.abcd.noindex/path-entry refuses a ~/.abcd.noindex that is a symlink before it downloads
// anything. The hooks, ahoy and the bootstrap refuse a record behind the link
// (iss-2609281017573862), so a form that wrote one through it would report a
// finished install whose record every reader ignores — and put that record in
// whatever the link points at, a dotfiles checkout typically.
func TestInstallSurfacesRefuseASymlinkedAbcdHome(t *testing.T) {
	for _, s := range loadInstallSurfaces(t) {
		if !writesPathEntry(s.script) {
			continue
		}
		t.Run(s.name, func(t *testing.T) {
			guard := strings.Index(s.script, symlinkedHomeGuard)
			fetch := strings.Index(s.script, "curl ")
			if guard < 0 || (fetch >= 0 && guard > fetch) {
				t.Errorf("%s writes ~/.abcd.noindex/path-entry but does not test %s before its first fetch", s.source, symlinkedHomeGuard)
			}
		})
	}
}
