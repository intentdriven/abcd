package site

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The installer exists in two committed forms: the script in
// site-src/install.sh.tmpl, which the release publishes as install.sh and
// abcdev.app serves, and that script as the site build renders it. The README
// and docs/how-to/install.md document one short command that pipes the
// release's install.sh into sh (iss-2610240827243654), and the guide adds a
// hardened form of it, so the logic lives in one script and the documented
// forms are held to fetching it
// (TestDocumentedInstallCommandFetchesTheReleasedInstaller).
//
// The forms below must end up with the same bytes in the same place: same
// release URLs, same checksum step, same install destination. This test holds
// them to exactly that, so a change to one fails the build rather than shipping
// a page that quietly disagrees with the script beside it.
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

	templatePath := filepath.Join(repoRoot(), "site-src", "install.sh.tmpl")
	bothArches := []archMapping{amd64FromX8664, arm64FromAarch64}

	surfaces := []installSurface{
		{
			name:      "install-sh-template",
			source:    "site-src/install.sh.tmpl",
			script:    readFile(t, templatePath),
			osToken:   "$os",
			verifiers: []string{verifierGNU, verifierBSD},
			archMap:   bothArches,
		},
	}

	// The SECOND form: the script this repository SERVES at /install.sh. It is the
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
// releasedPlatformCase reads the platform list off the installer's
// require_released_platform: the case pattern right after `case "$1-$2" in`.
var releasedPlatformCase = regexp.MustCompile(`case "\$1-\$2" in\s*\n\s*([a-z0-9| -]+)\)`)

// The hero links four binaries and a checksums manifest by NAME. The manifest
// says those names come from the release workflow, and the build deliberately
// does not read that workflow — a page that parses CI configuration to decide
// what to link is a worse dependency than a constant. So the claim is held here
// instead: the linked names must be exactly the `abcd-<os>-<arch>` matrix the
// installer itself admits (require_released_platform in site-src/install.sh.tmpl).
//
// A platform added to the release matrix with no link on the page, or a link to
// a platform nothing publishes, fails this test — which is the whole content of
// the claim.
func TestLinkedAssetsMatchTheReleaseMatrix(t *testing.T) {
	// The platforms the installer admits are the release matrix as the one
	// script every documented install runs states it.
	template := readFile(t, filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	admitted := releasedPlatformCase.FindStringSubmatch(template)
	if admitted == nil {
		t.Fatal("site-src/install.sh.tmpl has no require_released_platform case naming the released platforms")
	}
	want := map[string]bool{}
	for _, p := range strings.Split(admitted[1], "|") {
		want["abcd-"+strings.TrimSpace(p)] = true
	}
	if !want["abcd-darwin-arm64"] || !want["abcd-linux-amd64"] {
		t.Fatalf("the installer admits %v; expected macOS and Linux on amd64 and arm64", keysOf(want))
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
			// A one-liner redirects into the record; the installer writes a
			// private temporary file beside it and moves it into place.
			if !strings.Contains(s.script, "> "+pathEntryRecord) &&
				!strings.Contains(s.script, `mv -f "$entry_tmp" `+pathEntryRecord) {
				t.Errorf("%s does not write its record to %s", s.source, pathEntryRecord)
			}
			if strings.Contains(s.script, oldHome) || strings.Contains(s.script, `"$HOME/.abcd/`) {
				t.Errorf("%s names the old folder %s; abcd stops while it stands", s.source, oldHome)
			}
		})
	}
	if seen == 0 {
		t.Fatal("no install form records the owned PATH copy; the installer does")
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

// The documented install forms are rendered from site-src/install.sh.tmpl,
// the one source (iss-2609261423214723): the release-download prefix and the
// curl hardening flags are read off the template's own checksums.txt fetch,
// so a template that moves either moves every documented form with it. The
// person ruled on 2026-10-28 that README and the install guide show curl's
// plain form, and that the guide offers the hardened form as an option.
//
// installerFetch matches that fetch: its flags before -fsSLO, and the
// release-download prefix before checksums.txt.
var installerFetch = regexp.MustCompile(`curl ((?:-[^\s]+ (?:'[^']*' )?)+)-fsSLO "(https://[^"\s]+/releases/latest/download/)` +
	regexp.QuoteMeta(checksumsAsset) + `"`)

// installForms returns the documented command and its hardened form, rendered
// from the template's text: the plain form fetches the release's installer
// with curl's -fsSL and pipes it into sh; the hardened form adds the
// template's own fetch flags and --noproxy '*', which stands for the proxy
// variables the template unsets before it fetches anything.
func installForms(t *testing.T, tmpl string) (plain, hardened string) {
	t.Helper()
	m := installerFetch.FindStringSubmatch(tmpl)
	if m == nil {
		t.Fatalf("site-src/install.sh.tmpl has no curl fetch of %s to render the documented install command from", checksumsAsset)
	}
	url := m[2] + installScriptName
	return "curl -fsSL " + url + " | sh",
		"curl " + m[1] + "--noproxy '*' -fsSL " + url + " | sh"
}

// templateInstallForms renders the documented forms from this repository's
// template.
func templateInstallForms(t *testing.T) (plain, hardened string) {
	t.Helper()
	return installForms(t, readFile(t, filepath.Join(repoRoot(), "site-src", "install.sh.tmpl")))
}

// The README's install block sits between these markers and is the rendering
// of renderReadmeInstallBlock, byte for byte, so a hand edit fails
// TestReadmeInstallBlockIsRendered.
const (
	installBlockBegin = "<!-- BEGIN GENERATED: install-command -->"
	installBlockEnd   = "<!-- END GENERATED: install-command -->"
)

// renderReadmeInstallBlock is what the README carries between the markers.
func renderReadmeInstallBlock(plain string) string {
	return "\n<!-- Written by `go test ./internal/core/site -run TestReadmeInstallBlockIsRendered -update` " +
		"from site-src/install.sh.tmpl; edit the template, never this block. -->\n" +
		"```sh\n" + plain + "\n```\n"
}

// spliceInstallBlock returns readme with its one install block rendered from
// plain, every byte outside the block kept. A README with no block, an
// unclosed one, or more than one is an error: the gate never guesses where
// the block belongs.
func spliceInstallBlock(readme, plain string) (string, error) {
	switch n := strings.Count(readme, installBlockBegin); n {
	case 0:
		return "", fmt.Errorf("README.md carries no %s marker", installBlockBegin)
	case 1:
	default:
		return "", fmt.Errorf("README.md carries %d %s markers; it must carry one", n, installBlockBegin)
	}
	start := strings.Index(readme, installBlockBegin) + len(installBlockBegin)
	stop := strings.Index(readme[start:], installBlockEnd)
	if stop < 0 {
		return "", fmt.Errorf("README.md opens %s and never closes it with %s", installBlockBegin, installBlockEnd)
	}
	return readme[:start] + renderReadmeInstallBlock(plain) + readme[start+stop:], nil
}

// checkInstallBlock is the gate: nil when readme carries exactly the block the
// template renders, and an error naming the drift otherwise.
func checkInstallBlock(readme, plain string) error {
	want, err := spliceInstallBlock(readme, plain)
	if err != nil {
		return err
	}
	if want != readme {
		return fmt.Errorf("README.md's install block is not the rendering of site-src/install.sh.tmpl; " +
			"run `go test ./internal/core/site -run TestReadmeInstallBlockIsRendered -update`")
	}
	return nil
}

// TestReadmeInstallBlockIsRendered holds the README's install block to the
// rendering of the template (iss-2609261423214723). With -update it rewrites
// the block instead of failing.
func TestReadmeInstallBlockIsRendered(t *testing.T) {
	path := filepath.Join(repoRoot(), "README.md")
	readme := readFile(t, path)
	plain, _ := templateInstallForms(t)
	err := checkInstallBlock(readme, plain)
	if err == nil {
		return
	}
	if *update {
		synced, serr := spliceInstallBlock(readme, plain)
		if serr != nil {
			t.Fatal(serr)
		}
		if werr := os.WriteFile(path, []byte(synced), 0o644); werr != nil {
			t.Fatal(werr)
		}
		t.Log("rewrote README.md's install block from site-src/install.sh.tmpl")
		return
	}
	t.Error(err)
}

// TestHandEditedInstallBlockIsRefused proves the gate refuses what it exists
// to refuse: a block edited by hand, a block whose markers are gone, and a
// second block.
func TestHandEditedInstallBlockIsRefused(t *testing.T) {
	plain, _ := templateInstallForms(t)
	rendered := "intro\n" + installBlockBegin + renderReadmeInstallBlock(plain) + installBlockEnd + "\noutro\n"
	if err := checkInstallBlock(rendered, plain); err != nil {
		t.Fatalf("the rendered block must pass the gate: %v", err)
	}
	for name, readme := range map[string]string{
		"hand-edited command": strings.Replace(rendered, " | sh", " | bash", 1),
		"hand-edited note":    strings.Replace(rendered, "never this block", "this block", 1),
		"markers removed":     strings.NewReplacer(installBlockBegin, "", installBlockEnd, "").Replace(rendered),
		"end marker removed":  strings.Replace(rendered, installBlockEnd, "", 1),
		"block twice":         rendered + rendered,
	} {
		t.Run(name, func(t *testing.T) {
			if err := checkInstallBlock(readme, plain); err == nil {
				t.Errorf("the gate passed a README whose install block was %s", name)
			}
		})
	}
}

// installBlocks returns every block that downloads from the release, in page
// order.
func installBlocks(blocks []fencedBlock) []string {
	var found []string
	for _, b := range blocks {
		if downloadURL.MatchString(b.body) {
			found = append(found, strings.TrimSpace(b.body))
		}
	}
	return found
}

// TestDocumentedInstallCommandFetchesTheReleasedInstaller holds the README and
// the install guide to exactly the forms they document, in page order (the
// README shows the command alone, the guide shows it and then its hardened
// form), each byte for byte the template's rendering, and each form to the
// release-download prefix go.mod names.
func TestDocumentedInstallCommandFetchesTheReleasedInstaller(t *testing.T) {
	plain, hardened := templateInstallForms(t)
	for _, c := range []string{plain, hardened} {
		if !strings.Contains(c, " "+releaseDownloadPrefix(t)+installScriptName+" ") {
			t.Fatalf("the documented command %q does not fetch from the installer's release-download prefix %q",
				c, releaseDownloadPrefix(t))
		}
	}
	if !strings.HasPrefix(hardened, "curl -q --proto '=https' --proto-redir '=https' --tlsv1.2 --noproxy '*' ") {
		t.Errorf("the hardened form %q does not carry the template's fetch flags and --noproxy '*'", hardened)
	}
	documentedInstallForms := map[string][]string{
		"README.md": {plain},
		filepath.Join("docs", "how-to", "install.md"): {plain, hardened},
	}
	for doc, want := range documentedInstallForms {
		t.Run(doc, func(t *testing.T) {
			got := installBlocks(fencedBlocks(readFile(t, filepath.Join(repoRoot(), doc))))
			if !equalStrings(got, want) {
				t.Errorf("%s documents\n\t%s\nwant\n\t%s", doc, strings.Join(got, "\n\t"), strings.Join(want, "\n\t"))
			}
		})
	}
}

// TestTheHomePageShowsTheDocumentedInstallCommand renders this repository's
// own site and holds its install chapter to the documented command: the page
// builds its install tabs from the guide's headings under `## CLI`, and only a
// tab whose section holds the command shows it, with the read-the-script link
// beside it. A heading change in the guide that left no such tab would ship a
// home page with no install command on it.
func TestTheHomePageShowsTheDocumentedInstallCommand(t *testing.T) {
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: repoRoot(), OutDir: out, Stamp: fixtureStamp, LeaveUI: true}); err != nil {
		t.Fatalf("build this repository's site: %v", err)
	}
	html := readFile(t, filepath.Join(out, "index.html"))
	start := strings.Index(html, `<div role="tabpanel" id="panel-macos-and-linux"`)
	if start < 0 {
		t.Fatal("the home page has no macOS and Linux install panel")
	}
	panel := html[start:]
	if end := strings.Index(panel[1:], `<div role="tabpanel"`); end >= 0 {
		panel = panel[:end+1]
	}
	if !strings.Contains(panel, "releases/latest/download/"+installScriptName+" | sh") {
		t.Error("the macOS and Linux panel does not show the documented install command")
	}
	if !strings.Contains(panel, `<a href="/`+installScriptName+`">`) {
		t.Error("the macOS and Linux panel has no read-the-script link beside the command")
	}
}
