package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// bootstrapTag / bootstrapCommit / bootstrapRelease are the fixture release the
// test server serves. The commits are made-up 40-hex strings (the script only
// ever tests their shape) and deliberately DIFFER from each other, so a script
// that confused the plugin root's stamp with the release's commit — the two
// values the skew notice compares — could not pass by coincidence.
const (
	bootstrapTag     = "v9.9.9"
	bootstrapCommit  = "0123456789abcdef0123456789abcdef01234567"
	bootstrapRelease = "fedcba9876543210fedcba9876543210fedcba98"
)

// bootstrapReleaseOrigin / bootstrapAPIOrigin are the two fetch origins the
// shipped script hardcodes. They are the ONLY seam these tests have, and they
// are a textual one: the script itself reads no environment variable to decide
// where to fetch from, because two successive attempts to keep such a variable
// behind a loopback allowlist were both defeated in adversarial review. A test
// therefore rewrites these literals in a throwaway COPY of the script (see
// bootstrapFixtureScript) rather than driving a redirection seam that would
// then also exist in production.
const (
	bootstrapReleaseOrigin = "https://github.com/intentdriven/abcd"
	bootstrapAPIOrigin     = "https://api.github.com/repos/intentdriven/abcd"
)

// bootstrapTrustScrub is the shipped script's line that removes curl's CA-override
// variables from the environment before any fetch, and bootstrapProxyScrub is its
// companion for the proxy variables and CURL_HOME.
//
// The trust line is the ONE line bootstrapFixtureScript neutralises in its
// throwaway copy, because the fixture is a self-signed local TLS server and
// CURL_CA_BUNDLE is the only way to make a real curl trust it. That is a widening
// of what curl will ACCEPT, never a naming of a different server, so the copy is
// still exercising the same fetch path; the proxy/CURL_HOME line stays intact in
// the copy, which is what makes the poisoned-config case below meaningful.
// TestBootstrapFetchOriginsAreConstants holds both lines in the shipped bytes.
const (
	bootstrapTrustScrub = "unset CURL_CA_BUNDLE SSL_CERT_FILE SSL_CERT_DIR"
	bootstrapProxyScrub = "unset HTTPS_PROXY https_proxy HTTP_PROXY http_proxy ALL_PROXY all_proxy CURL_HOME"
)

// bootstrapScript locates the committed hooks/bootstrap.sh from this test file's
// own on-disk position (internal/surface/cli/ -> three levels up), so the tests
// derive from the script that actually ships rather than from a hand-written
// stand-in. Callers that need to RUN it against a fixture go through
// bootstrapFixtureScript.
func bootstrapScript(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to locate the test source file")
	}
	script := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "hooks", "bootstrap.sh"))
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("the committed bootstrap script must exist: %v", err)
	}
	return script
}

// bootstrapRequires skips a case whose host cannot run the script at all.
func bootstrapRequires(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl unavailable")
	}
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64":
	default:
		t.Skipf("no released binary for %s/%s: the supported-matrix case covers this", runtime.GOOS, runtime.GOARCH)
	}
}

// bootstrapFixtureScript writes a per-test COPY of the committed script with its
// two hardcoded fetch origins rewritten to the fixture's base URL, and returns
// the copy's path.
//
// This is the whole test seam, and it exists on THIS side of the boundary on
// purpose: the shipped script must contain no branch, variable, or flag that can
// point a fetch somewhere else, because the binary and the checksums.txt that
// "verifies" it come from the same origin and the result is run unattended as
// the Bash shell guard on every tool call. Rewriting a copy keeps the production
// bytes identical to what runs on a user's machine while still exercising the
// full download/verify/install path.
//
// Every substitution is checked, both before and after. A silent no-op — the
// literal drifting in the script, a typo in the constant — would leave every
// case below pointed at the REAL GitHub while reporting a pass, which is exactly
// the fixture-hides-the-bug failure an earlier round already shipped once.
func bootstrapFixtureScript(t *testing.T, base string) string {
	t.Helper()
	src, err := os.ReadFile(bootstrapScript(t))
	if err != nil {
		t.Fatalf("reading the committed bootstrap script: %v", err)
	}
	body := string(src)
	// The QUOTED, complete assignment value — never a bare hostname and never a
	// bare prefix. A prefix match would happily rewrite a longer origin
	// (…/abcd-cli-renamed) into a mangled URL and carry on; requiring the closing
	// quote means the literal is matched whole or not at all. Each must appear
	// exactly once, or the script has changed shape in a way this harness no
	// longer models and would otherwise silently test against the real GitHub.
	for _, origin := range []string{bootstrapAPIOrigin, bootstrapReleaseOrigin} {
		quoted := `"` + origin + `"`
		if n := strings.Count(body, quoted); n != 1 {
			t.Fatalf("the fixture substitution expects exactly one %s in hooks/bootstrap.sh, found %d — the script's fetch origins have changed and this harness would otherwise silently test against the real GitHub", quoted, n)
		}
		body = strings.Replace(body, quoted, `"`+base+`"`, 1)
	}
	for _, origin := range []string{bootstrapAPIOrigin, bootstrapReleaseOrigin} {
		if strings.Contains(body, origin) {
			t.Fatalf("the rewritten script still names %q, so the fixture substitution did not take effect", origin)
		}
	}
	if n := strings.Count(body, base); n != 2 {
		t.Fatalf("the rewritten script must name the fixture base exactly twice (release origin + API origin), found %d", n)
	}
	// The fixture's self-signed certificate reaches curl through CURL_CA_BUNDLE,
	// which the shipped script scrubs. Neutralise that ONE line (`:` is the POSIX
	// no-op) so the copy can talk to the fixture at all; everything else about the
	// lockdown — -q on every call, and the proxy/CURL_HOME scrub — stays exactly as
	// shipped, so the cases below still run against it.
	if n := strings.Count(body, bootstrapTrustScrub); n != 1 {
		t.Fatalf("the fixture substitution expects exactly one %q in hooks/bootstrap.sh, found %d — the CA-scrub line has changed shape and the fixture would otherwise be unreachable", bootstrapTrustScrub, n)
	}
	body = strings.Replace(body, bootstrapTrustScrub, ":", 1)
	if !strings.Contains(body, bootstrapProxyScrub) {
		t.Fatalf("the rewritten script must keep the proxy/CURL_HOME scrub intact, otherwise the poisoned-config case proves nothing")
	}
	path := filepath.Join(t.TempDir(), "bootstrap.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing the rewritten bootstrap script: %v", err)
	}
	return path
}

// bootstrapAsset is the release asset name the script derives from uname.
func bootstrapAsset() string { return "abcd-" + runtime.GOOS + "-" + runtime.GOARCH }

// bootstrapFixture is the stand-in release host plus the trust material a real
// curl needs to talk to it.
type bootstrapFixture struct {
	base string // https://127.0.0.1:<port>
	ca   string // PEM file holding the fixture's self-signed certificate
	hits *int32 // every request the script makes, counted
	// artefactHits counts only the requests that download the release BINARY
	// (the platform asset and its CDN redirect target) — the spc-35 cache cases
	// assert these stay at zero while still allowing the one best-effort tag
	// resolve the refresh detector makes and the checksums.txt fetch the online
	// cache authentication needs.
	artefactHits *int32
	// manifestHits counts checksums.txt fetches, kept SEPARATE from the binary
	// download: the online cache-authentication path (spc-35, adr-46 decision 3)
	// fetches the published manifest to check the cached hash before trusting
	// the cache, WITHOUT downloading the artefact, so a cache hit must show a
	// manifest fetch and zero asset fetches.
	manifestHits *int32
	// failLatest, when set non-zero, makes /releases/latest answer 500 — the
	// offline-resolve case: the refresh detector's best-effort resolve fails
	// and a cache hit must still provision the root.
	failLatest *int32
}

// bootstrapServer stands in for the release host, shaped like the real one
// rather than like the script's convenience: /releases/latest answers a 302 to
// /releases/tag/<tag> (the only place the tag is legible), and the tag-pinned
// asset URL redirects to an OPAQUE CDN path carrying no tag segment — which is
// what GitHub actually does (release-assets.githubusercontent.com/<numbers>?…),
// and the reason reading the tag off curl's effective URL resolved to "unknown"
// in production while passing against a friendlier fixture. checksums.txt is
// served only from the tag-pinned path, so a script that fetched it from
// /releases/latest/download instead gets a 404 rather than a silent pass.
// hits counts every request, which is how the no-network cases are proved.
//
// It speaks TLS, not plain http, because the script pins `--proto '=https'
// --proto-redir '=https'` on every call unconditionally and nothing can clear
// that. Trust is supplied through CURL_CA_BUNDLE (see runBootstrap): a CA bundle
// only ever ADDS an issuer curl will accept — it can never name a different
// server — so it is not the class of seam that was removed from the script.
func bootstrapServer(t *testing.T, body []byte, manifest string) *bootstrapFixture {
	t.Helper()
	var count, artefacts, manifests, failLatest int32
	asset := bootstrapAsset()
	pinned := "/releases/download/" + bootstrapTag + "/"
	mux := http.NewServeMux()
	// Deliberately outside the counter: the harness's own reachability probe
	// must not read as a fetch the script made.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		switch r.URL.Path {
		case "/releases/latest":
			if atomic.LoadInt32(&failLatest) != 0 {
				http.Error(w, "unavailable", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/releases/tag/"+bootstrapTag, http.StatusFound)
		case pinned + asset:
			atomic.AddInt32(&artefacts, 1)
			http.Redirect(w, r, "/cdn-blob/1292161760/10d722b0?sig=x", http.StatusFound)
		case "/cdn-blob/1292161760/10d722b0":
			atomic.AddInt32(&artefacts, 1)
			_, _ = w.Write(body)
		case pinned + "checksums.txt":
			atomic.AddInt32(&manifests, 1)
			_, _ = w.Write([]byte(manifest))
		case "/commits/" + bootstrapTag:
			_, _ = w.Write([]byte(`{"sha":"` + bootstrapRelease + `","node_id":"x"}`))
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)

	ca := filepath.Join(t.TempDir(), "fixture-ca.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(ca, pemBytes, 0o600); err != nil {
		t.Fatalf("writing the fixture CA: %v", err)
	}
	fx := &bootstrapFixture{base: srv.URL, ca: ca, hits: &count, artefactHits: &artefacts, manifestHits: &manifests, failLatest: &failLatest}

	// Prove the harness itself works before any case blames the script. Without
	// this, a curl build that ignored CURL_CA_BUNDLE would surface as "the latest
	// release tag could not be resolved" and read like a script defect.
	probe := exec.Command("curl", "-fsS", "--proto", "=https", fx.base+"/healthz")
	probe.Env = append(fx.env(), "PATH="+os.Getenv("PATH"))
	if out, err := probe.CombinedOutput(); err != nil {
		t.Fatalf("the fixture TLS server is unreachable from curl with CURL_CA_BUNDLE set, so the harness cannot exercise the pinned-HTTPS script: %v (output %q)", err, out)
	}
	return fx
}

// env is the trust material the fixture needs, and nothing else. CURL_CA_BUNDLE
// and SSL_CERT_FILE are curl/OpenSSL's own variables, honoured by any curl
// invocation in the process environment; the script neither names nor branches
// on them. Both are set because the two curl TLS backends in play read different
// ones.
func (f *bootstrapFixture) env() []string {
	return []string{"CURL_CA_BUNDLE=" + f.ca, "SSL_CERT_FILE=" + f.ca}
}

// bootstrapManifest renders a sha256sum-style manifest over body, listing the
// platform asset plus one unrelated entry so a match is never accidental.
func bootstrapManifest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) + "  " + bootstrapAsset() + "\n" +
		strings.Repeat("0", 64) + "  abcd-other-platform\n"
}

// bootstrapRoot makes a plugin root whose basename is a 40-hex commit stamp —
// the shape the harness's plugin cache uses, and the input for plugin_sha.
func bootstrapRoot(t *testing.T) string {
	t.Helper()
	return bootstrapRootNamed(t, bootstrapCommit)
}

// bootstrapRootNamed makes a plugin root with an arbitrary basename, which is
// how the "the commit-stamped-cache warrant did not hold" case is reached.
func bootstrapRootNamed(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// runBootstrap executes a fixture-pointed copy of the script with an explicitly
// constructed environment — no inherited proxy or plugin variables — and returns
// its merged output and exit code. extraPath is prepended to PATH so a case can
// shim a command.
func runBootstrap(t *testing.T, root string, fx *bootstrapFixture, extraPath string) (string, int) {
	t.Helper()
	bootstrapRequires(t)
	return runScript(t, bootstrapFixtureScript(t, fx.base), root, fx.env(), extraPath)
}

// runScript executes a bootstrap script directly (not via `sh <path>`, so the
// executable bit and the shebang are exercised too) under a constructed
// environment.
func runScript(t *testing.T, script, root string, extraEnv []string, extraPath string) (string, int) {
	t.Helper()
	return runScriptIn(t, "", script, root, extraEnv, extraPath)
}

// runScriptIn is runScript with the child's WORKING DIRECTORY pinned (empty
// inherits this process's, which is the package directory inside the checkout).
// A case that hands the script a relative path in its environment — the
// relative HOME the attestation's trust floor has to refuse — needs that
// directory to be a temp dir it controls, because the whole point of the shape
// is that the child resolves the value against wherever it happens to run.
func runScriptIn(t *testing.T, dir, script, root string, extraEnv []string, extraPath string) (string, int) {
	t.Helper()
	pathValue := os.Getenv("PATH")
	if extraPath != "" {
		pathValue = extraPath + string(os.PathListSeparator) + pathValue
	}
	cmd := exec.Command(script)
	cmd.Dir = dir
	cmd.Env = dedupEnvKeepLast(append([]string{
		"PATH=" + pathValue,
		"HOME=" + t.TempDir(),
		"CLAUDE_PLUGIN_ROOT=" + root,
	}, extraEnv...))
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatalf("running the bootstrap: %v (output %s)", err, out)
		}
	}
	return string(out), code
}

// dedupEnvKeepLast collapses duplicate KEY= entries keeping the LAST value, so a
// caller that appends HOME= (or CURL_HOME=) after the constructed defaults
// really does override them regardless of how the child's libc resolves a
// duplicated key.
func dedupEnvKeepLast(env []string) []string {
	idx := map[string]int{}
	var out []string
	for _, kv := range env {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if at, ok := idx[key]; ok {
			out[at] = kv
			continue
		}
		idx[key] = len(out)
		out = append(out, kv)
	}
	return out
}

// metaValues parses the key=value .binary-meta the bootstrap writes.
func metaValues(t *testing.T, root string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".binary-meta"))
	if err != nil {
		t.Fatalf("the bootstrap must write .binary-meta on a successful install: %v", err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			out[k] = v
		}
	}
	return out
}

// TestBootstrapFastPathTouchesNoNetwork is the steady-state AC: a plugin root
// that already holds an executable binary costs one file test and nothing else.
func TestBootstrapFastPathTouchesNoNetwork(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
	binary := filepath.Join(root, "abcd")
	if err := os.WriteFile(binary, []byte("already here"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Errorf("a present binary must exit 0, got %d (output %q)", code, out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("a present binary must cost no network, got %d request(s)", n)
	}
	got, err := os.ReadFile(binary)
	if err != nil || string(got) != "already here" {
		t.Errorf("the fast path must leave the existing binary untouched; got %q (%v)", got, err)
	}
}

// TestBootstrapInstallsVerifiedBinary is the fresh-install AC: download, verify
// against the release manifest, install executable, and record the provenance
// the skew notice reads. The exit code is 2, not 0: a SessionStart hook's stdout
// is model context and only a non-zero exit shows its stderr to the human, so
// the one-time `abcd ahoy install` suggestion would otherwise never be read by
// anyone who could act on it. SessionStart does not block on a non-zero exit.
func TestBootstrapInstallsVerifiedBinary(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("#!/bin/sh\nexit 0\n")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("a successful install is a notice, not a fault: it must exit 0, got %d (output %q)", code, out)
	}
	// The substitution is proved BEHAVIOURALLY as well as textually: a copy that
	// still named the real origin could not have reached this fixture at all.
	if n := atomic.LoadInt32(fx.hits); n == 0 {
		t.Fatal("the fixture served no request, so the script was not pointed at it — the substitution silently missed")
	}
	binary := filepath.Join(root, "abcd")
	fi, err := os.Stat(binary)
	if err != nil {
		t.Fatalf("the bootstrap must install the binary into the plugin root: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("the installed binary must be mode 0755, got %v", fi.Mode().Perm())
	}
	got, err := os.ReadFile(binary)
	if err != nil || string(got) != string(body) {
		t.Errorf("the installed binary must be the verified download; got %q (%v)", got, err)
	}
	if !strings.Contains(out, "ahoy install") {
		t.Errorf("the success message must suggest `abcd ahoy install` for terminal PATH setup; output %q", out)
	}

	meta := metaValues(t, root)
	// release_tag has to be resolved from the release endpoint, not scraped off
	// the asset download's effective URL: the real one redirects to an opaque CDN
	// path, so a scraped tag is always "unknown" and the whole skew feature — and
	// the tag-pinned checksums URL that keeps the binary and its manifest from
	// coming from two different releases — dies with it.
	if meta["release_tag"] != bootstrapTag {
		t.Errorf("release_tag = %q, want %q resolved from the release endpoint", meta["release_tag"], bootstrapTag)
	}
	if meta["release_sha"] != bootstrapRelease {
		t.Errorf("release_sha = %q, want %q (the commit the release API reports)", meta["release_sha"], bootstrapRelease)
	}
	// spc-35 dropped plugin_sha (and its raw-basename companion) from the
	// record: the version-skew notice compares the LIVE plugin root's basename
	// at render time, so a provisioning-time snapshot of the root is recorded
	// nowhere any more.
	for _, gone := range []string{"plugin_sha", "plugin_root_basename"} {
		if v, has := meta[gone]; has {
			t.Errorf("%s = %q must no longer be recorded — the skew notice reads the live plugin root", gone, v)
		}
	}
	// The hash the verification just accepted is recorded, so a human (or a later
	// verb) can tell the binary that was verified from one that replaced it.
	// Nothing re-checks it today; the fast path deliberately stays a file test.
	sum := sha256.Sum256(body)
	if want := hex.EncodeToString(sum[:]); meta["binary_sha256"] != want {
		t.Errorf("binary_sha256 = %q, want the verified hash %q", meta["binary_sha256"], want)
	}
	if _, err := time.Parse(time.RFC3339, meta["fetched_at"]); err != nil {
		t.Errorf("fetched_at = %q, want an RFC3339 UTC timestamp (%v)", meta["fetched_at"], err)
	}
	if _, err := os.Stat(filepath.Join(root, ".bootstrap.lock")); !os.IsNotExist(err) {
		t.Error("the lock must be removed on the success path")
	}
}

// bootstrapRepoFile locates a committed file at the repository root from this
// test file's own on-disk position, the same way bootstrapScript does, so the
// assertions below read the bytes that actually ship.
func bootstrapRepoFile(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed to locate the test source file")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", rel))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the committed %s must exist: %v", rel, err)
	}
	return path
}

// bootstrapReadmeInstruction is the shape the install guide's one-time PATH
// setup instruction has to take. The page cannot know the absolute plugin root at
// authoring time, so it carries the same INVOCATION with the one part it cannot
// know left as a placeholder the notice fills in — not a bare `abcd`, which is
// the name that does not resolve, and not `${CLAUDE_PLUGIN_ROOT}` either, which
// is set for a hook and unset in the terminal the sentence is addressed to.
// Single quotes, matching the bytes the notice prints, so the two surfaces do
// not show the reader two different shapes of the same command.
const bootstrapReadmeInstruction = `'<plugin-root>/abcd' ahoy install`

// bootstrapShSingleQuote mirrors internal/core/ahoy's shSingleQuote: the POSIX
// single-quote wrapping that survives every character a path can hold. It is
// restated here rather than imported (it is unexported, in another package) so
// this test states the bytes it EXPECTS independently of the code that produces
// them — a shared helper would agree with the script by construction and prove
// nothing about it.
func bootstrapShSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// bootstrapEveryInvocationIsPathQualified reports each occurrence of `ahoy
// install` in body that is NOT reached through prefix. Splitting on the verb
// rather than searching for the good form is deliberate: a text may hold the
// runnable instruction AND still leave the unrunnable one standing beside it,
// and only an every-occurrence check sees the second one.
func bootstrapEveryInvocationIsPathQualified(body, prefix string) []string {
	parts := strings.Split(body, "ahoy install")
	var bad []string
	for _, before := range parts[:len(parts)-1] {
		if strings.HasSuffix(before, prefix) {
			continue
		}
		if len(before) > 80 {
			before = "…" + before[len(before)-80:]
		}
		bad = append(bad, before+"ahoy install")
	}
	return bad
}

// TestBootstrapPrintsARunnableInstruction is iss-207: the ONE instruction that
// resolves the no-binary-on-PATH state could not be run in that state. The
// notice said "run `abcd ahoy install` once" while `abcd` is, by the notice's
// own premise, not a name the shell can resolve — so the sentence fails with
// "command not found" for exactly the reader it is written for. On the first
// manual install this was not a cosmetic failure: the agent reading it could not
// run the printed command, invented a `go run` incantation reaching into the
// harness's plugin cache, and told the user to run that instead — a
// source-build path needing a Go toolchain, which is not the documented install
// at all.
//
// The script already holds the absolute path as $binary, so the fix is to print
// it. Both surfaces are held here: the notice, whose path is concrete, and the
// README, whose corresponding sentence carries the same invocation with a
// placeholder for the one part a committed file cannot know.
func TestBootstrapPrintsARunnableInstruction(t *testing.T) {
	t.Run("the notice names the absolute binary", func(t *testing.T) {
		root := bootstrapRoot(t)
		body := []byte("payload")
		fx := bootstrapServer(t, body, bootstrapManifest(body))

		out, code := runBootstrap(t, root, fx, "")
		if code != 0 {
			t.Fatalf("a successful install is a notice, not a fault: it must exit 0, got %d (output %q)", code, out)
		}
		binary := filepath.Join(root, "abcd")
		prefix := bootstrapShSingleQuote(binary) + " "
		if !strings.Contains(out, prefix+"ahoy install") {
			t.Errorf("the notice must print the absolute plugin-root binary the script already holds, so the instruction runs in the state it describes; want %q in output %q", prefix+"ahoy install", out)
		}
		for _, bad := range bootstrapEveryInvocationIsPathQualified(out, prefix) {
			t.Errorf("the notice still prints an `ahoy install` a reader cannot run — `abcd` is not on PATH, which is the very state this sentence addresses: %q", bad)
		}
		// A2's contract, held here because this test rewrites the same sentence:
		// the transcript renders only the first line of a hook's stderr, so the
		// success has to lead it. itd-154 deliberately does NOT put a
		// provisioning announcement ahead of it for that reason — nothing may
		// take this line, from the success or from a refusal's cause.
		if got := firstLine(out); !strings.HasPrefix(got, "abcd bootstrap: installed") {
			t.Errorf("the success must still lead the first visible line; first line = %q", got)
		}
	})

	// The prose surface is the install guide, which is where the plugin route is
	// documented. README is swept too: it is the other committed page a reader
	// meets first, and the unrunnable form must not reappear on either.
	t.Run("the install guide carries the same concrete form", func(t *testing.T) {
		const guide = "docs/how-to/install.md"
		body := mustReadFile(t, bootstrapRepoFile(t, guide))
		if !strings.Contains(body, bootstrapReadmeInstruction) {
			t.Errorf("%s must give the one-time PATH setup as %q; a bare `abcd ahoy install` cannot be run by the reader it is written for", guide, bootstrapReadmeInstruction)
		}
		prefix := `'<plugin-root>/abcd' `
		for _, page := range []string{guide, "README.md"} {
			text := mustReadFile(t, bootstrapRepoFile(t, page))
			for _, bad := range bootstrapEveryInvocationIsPathQualified(text, prefix) {
				t.Errorf("%s still instructs an `ahoy install` that cannot be run before `abcd` is on PATH: %q", page, bad)
			}
		}
		// The placeholder has to be instantiable from THIS file. The notice that
		// prints the path in full prints once per plugin root (every later session
		// takes the bootstrap's fast path), so a reader who meets this sentence on
		// day five has no notice on screen — and the plugin root's location cannot
		// be named here, because abcd's published prose stays host-agnostic and
		// docs-lint blocks naming a specific harness. So the page owes two
		// things: what the placeholder IS in terms the reader can act on, and a
		// route that needs no plugin root at all.
		if !strings.Contains(body, "directory the agent harness unpacked") {
			t.Errorf("%s must say what <plugin-root> is in host-agnostic terms; a placeholder defined only as \"what the notice printed\" cannot be instantiated once the notice has scrolled away", guide)
		}
		if !strings.Contains(body, "needs no plugin root") {
			t.Errorf("%s must point the reader who cannot instantiate <plugin-root> at the install route that does not need one", guide)
		}
	})

	// The printed command is not merely path-qualified, it is QUOTED for a shell,
	// and the double-quoted form this started as survives a space or an apostrophe
	// while `$`, a backtick or a `"` inside the path still expand or terminate the
	// string. The repo already owns the robust form (internal/core/ahoy's
	// shSingleQuote), so the notice uses it. The proof is not a string comparison:
	// the printed command is handed to a real `sh`, which either runs the binary
	// at that path or does not.
	t.Run("the printed command runs when pasted, whatever the path holds", func(t *testing.T) {
		// Every metacharacter that defeats double quoting, plus the apostrophe
		// that defeats naive single quoting, in one directory name.
		root := bootstrapRootNamed(t, `it's $HOME "and" `+"`backticks`")
		body := []byte("#!/bin/sh\nexit 0\n")
		fx := bootstrapServer(t, body, bootstrapManifest(body))

		out, code := runBootstrap(t, root, fx, "")
		if code != 0 {
			t.Fatalf("a successful install is a notice, not a fault: it must exit 0, got %d (output %q)", code, out)
		}
		command := bootstrapShSingleQuote(filepath.Join(root, "abcd")) + " ahoy install"
		if !strings.Contains(out, " "+command) {
			t.Fatalf("the notice must print the shell-quoted command as its own word; want %q in output %q", command, out)
		}
		// The installed fixture binary is `exit 0`, so a shell that resolves the
		// path runs it and exits 0. A path that broke out of its quoting instead
		// yields "command not found", a syntax error, or an unrelated exit.
		if err := exec.Command("sh", "-c", command).Run(); err != nil {
			t.Errorf("the command the notice printed does not run when pasted into a shell: %v (command %q)", err, command)
		}
	})
}

// TestBootstrapRefusesChecksumMismatch is the trust bar: a download whose SHA-256
// disagrees with the release manifest never reaches the binary path, and the
// message says so.
func TestBootstrapRefusesChecksumMismatch(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("tampered"), bootstrapManifest([]byte("the published bytes")))

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Errorf("a checksum mismatch must fail loudly, got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "SHA-256") {
		t.Errorf("the refusal must name the checksum mismatch; output %q", out)
	}
	assertNothingInstalled(t, root, out)
}

// TestBootstrapRefusesAbsentManifestEntry is the other half of the same bar: a
// manifest that does not list this platform's asset is unverifiable, so nothing
// is installed.
func TestBootstrapRefusesAbsentManifestEntry(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), strings.Repeat("0", 64)+"  abcd-other-platform\n")

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Errorf("an unlisted asset must fail loudly, got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "lists no entry") {
		t.Errorf("the refusal must say the manifest lists no entry for this platform, not a download or mismatch failure; output %q", out)
	}
	if !strings.Contains(out, bootstrapAsset()) {
		t.Errorf("the refusal must name the asset the manifest lacks; output %q", out)
	}
	assertNothingInstalled(t, root, out)
}

// assertNothingInstalled is the shared refusal contract: no binary, no meta file,
// no leftover lock or temp dir, and an actionable message rather than a raw
// shell error.
func assertNothingInstalled(t *testing.T, root, out string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "abcd")); !os.IsNotExist(err) {
		t.Error("a refused download must never be installed")
	}
	if _, err := os.Stat(filepath.Join(root, ".binary-meta")); !os.IsNotExist(err) {
		t.Error("a refused download must write no .binary-meta")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".bootstrap") {
			t.Errorf("a refused run must leave no %s behind", e.Name())
		}
	}
	if strings.Contains(out, "No such file or directory") {
		t.Errorf("a raw shell error must never be the message a user gets; output %q", out)
	}
	if !strings.Contains(out, "go build ./cmd/abcd") {
		t.Errorf("the refusal must name the build-from-source recovery; output %q", out)
	}
}

// TestBootstrapUnsupportedPlatformChangesNothing shims uname so the script sees a
// platform no release covers. It must say so plainly, name the matrix, change
// nothing, and exit 2 — an unsupported platform is a reported condition, not a
// hook fault, but a SessionStart hook that exits 0 has only put its message into
// model context, where the human who would build from source never sees it.
// A non-zero SessionStart exit is non-blocking, so nothing is disrupted; the
// accepted cost is that this notice then repeats every session for as long as
// the platform is unsupported (itd-105's "every hook reports, for now" posture,
// with the one-notice-per-session aspiration tracked in iss-168).
func TestBootstrapUnsupportedPlatformChangesNothing(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
	shim := t.TempDir()
	fake := "#!/bin/sh\ncase \"$1\" in\n-s) echo Haiku ;;\n-m) echo sparc64 ;;\n*) echo Haiku ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(shim, "uname"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, shim)
	if code != 0 {
		t.Errorf("an unsupported platform must exit 0 (a notice is not a fault), got %d (output %q)", code, out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("an unsupported platform must download nothing, got %d request(s)", n)
	}
	for _, want := range []string{"darwin", "linux", "amd64", "arm64", "go build ./cmd/abcd"} {
		if !strings.Contains(out, want) {
			t.Errorf("the message must name %q (the supported matrix and the way out); output %q", want, out)
		}
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Errorf("an unsupported platform must change nothing in the plugin root; entries %v (%v)", entries, err)
	}
}

// TestBootstrapLockContentionExitsQuietly: a second concurrent session that loses
// the mkdir race does nothing at all — no download, no output, no exit code that
// would render as a hook failure.
func TestBootstrapLockContentionExitsQuietly(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
	lock := filepath.Join(root, ".bootstrap.lock")
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Errorf("losing the lock race must exit 0, got %d (output %q)", code, out)
	}
	if out != "" {
		t.Errorf("losing the lock race must be silent; output %q", out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("losing the lock race must cost no network, got %d request(s)", n)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Error("the loser must not remove the winner's live lock")
	}
}

// TestBootstrapUnusableLockPathRefusesLoudly is the other side of that silence.
// A lock that cannot be taken and is NOT there afterwards is not a race: the
// plugin root is unwritable, or something that is not a directory occupies the
// lock path. That condition is permanent and repeats every session, so sharing
// the loser's silent exit 0 with it would leave a plugin root that can never be
// provisioned and never says why.
func TestBootstrapUnusableLockPathRefusesLoudly(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
	// A regular FILE at the lock path: mkdir fails exactly as it does on a
	// read-only plugin root, and no lock directory exists afterwards. Unlike a
	// permission bit this reproduces for root as well as an ordinary user.
	if err := os.WriteFile(filepath.Join(root, ".bootstrap.lock"), []byte("not a lock"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Fatalf("an unusable lock path must not be mistaken for a lost race; got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "not writable") {
		t.Errorf("the refusal must say the plugin root is not writable; output %q", out)
	}
	if !strings.Contains(out, root) {
		t.Errorf("the refusal must name the plugin root %q; output %q", root, out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("a run that never took the lock must fetch nothing, got %d request(s)", n)
	}
}

// TestBootstrapSweepsOrphanedTempDirs: SIGKILL runs no trap, so a killed run
// leaves its PID-stamped temp directory behind holding a partially downloaded,
// UNVERIFIED binary. Nothing else ever removes it, so it accumulates in the
// plugin root forever. A run that holds the lock sweeps them.
func TestBootstrapSweepsOrphanedTempDirs(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("payload")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	orphan := filepath.Join(root, ".bootstrap.tmp.99999")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "abcd-unverified"), []byte("not verified"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("the install must still proceed, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("a temp directory orphaned by a killed run must be swept, %s survives (%v)", orphan, err)
	}
}

// TestBootstrapFastPathRejectsADirectory: `[ -x ]` is true of a DIRECTORY too, so
// a plugin root holding a directory named abcd would take the fast path forever
// and never provision anything. The check has to be a regular file, matching
// internal/core/ahoy's isExecutableFile — the two must agree about what
// "reachable binary" means or ahoy reports a gap the bootstrap refuses to fill.
// It must also REFUSE outright rather than proceed: `mv -f` onto an existing
// directory moves the downloaded file INTO it instead of replacing it, which
// would otherwise report success while the hooks stay broken.
func TestBootstrapFastPathRejectsADirectory(t *testing.T) {
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
	if err := os.MkdirAll(filepath.Join(root, "abcd"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Fatalf("a directory at the binary path must not read as a provisioned binary; output %q", out)
	}
	if !strings.Contains(out, "not a regular file") {
		t.Errorf("the refusal must name the obstruction; output %q", out)
	}
	if n := atomic.LoadInt32(fx.hits); n != 0 {
		t.Errorf("a pre-existing obstruction must be caught before any download, got %d request(s)", n)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd", bootstrapAsset())); !os.IsNotExist(err) {
		t.Error("the downloaded asset must never be moved INTO the obstructing directory")
	}
}

// TestBootstrapStaleLockIsBroken: a lock left by a crashed run would otherwise
// make the plugin root unprovisionable forever, so one older than ten minutes is
// taken over.
func TestBootstrapStaleLockIsBroken(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("payload")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	lock := filepath.Join(root, ".bootstrap.lock")
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-30 * time.Minute)
	if err := os.Chtimes(lock, stale, stale); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("a stale lock must be broken and the install proceed, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); err != nil {
		t.Errorf("a stale lock must not block provisioning: %v", err)
	}
}

// TestBootstrapStripsControlCharactersFromMessages: both message sinks
// interpolate values the script does not author — the plugin-root path, uname's
// output, the release tag read off a redirect. A raw ESC in one of them can
// recolour, reposition, or visually rewrite what the reader is shown, on a
// message whose entire job is to be believed. A bare CR is the same class of
// attack by a different mechanism (it returns the cursor to column zero and
// overprints what the reader has already been shown — internal/termsafe's
// SanitizeBlock names this specifically), and DEL is stripped alongside it.
func TestBootstrapStripsControlCharactersFromMessages(t *testing.T) {
	t.Run("refuse", func(t *testing.T) {
		root := bootstrapRootNamed(t, "plugin\x1b[31mroot\rEVIL\x7f")
		fx := bootstrapServer(t, []byte("tampered"), bootstrapManifest([]byte("published")))

		out, code := runBootstrap(t, root, fx, "")
		if code == 0 {
			t.Fatalf("expected a refusal; output %q", out)
		}
		for _, r := range []rune{0x1b, '\r', 0x7f} {
			if strings.ContainsRune(out, r) {
				t.Errorf("the refusal must strip control character %U from the values it echoes; output %q", r, out)
			}
		}
	})
	t.Run("notice", func(t *testing.T) {
		root := bootstrapRoot(t)
		fx := bootstrapServer(t, []byte("payload"), bootstrapManifest([]byte("payload")))
		shim := t.TempDir()
		fake := "#!/bin/sh\ncase \"$1\" in\n-s) printf 'Hai\\033[31mku\\n' ;;\n-m) echo sparc64 ;;\n*) printf 'Hai\\033[31mku\\n' ;;\nesac\n"
		if err := os.WriteFile(filepath.Join(shim, "uname"), []byte(fake), 0o755); err != nil {
			t.Fatal(err)
		}

		out, code := runBootstrap(t, root, fx, shim)
		if code != 0 {
			t.Fatalf("expected the unsupported-platform notice; got %d (output %q)", code, out)
		}
		if strings.ContainsRune(out, 0x1b) {
			t.Errorf("the notice must strip control characters from the values it echoes; output %q", out)
		}
	})
}

// TestBootstrapMetaHoldsExactlyItsDeclaredFields: .binary-meta is a key=value
// text file another process parses with last-line-wins semantics, so its shape
// is a contract, not a formatting nicety. spc-35 reduced the record to four
// fields — release_tag, release_sha, binary_sha256, fetched_at — by dropping
// the provisioning-time plugin_sha (and its raw-basename companion, which
// existed only to diagnose plugin_sha's 40-hex gate): the skew notice now
// compares the LIVE plugin root at render time, and dropping the basename also
// removes the one value a hostile directory name could once have used to forge
// an extra key into the record.
func TestBootstrapMetaHoldsExactlyItsDeclaredFields(t *testing.T) {
	root := bootstrapRootNamed(t, "evil\nrelease_sha="+strings.Repeat("f", 40))
	body := []byte("payload")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("the install must still succeed despite the hostile basename, got %d (output %q)", code, out)
	}
	raw := strings.TrimSpace(mustReadFile(t, filepath.Join(root, ".binary-meta")))
	lines := strings.Split(raw, "\n")
	if len(lines) != 4 {
		t.Fatalf(".binary-meta must hold exactly its four declared fields, got %d lines: %q", len(lines), raw)
	}
	for i, key := range []string{"release_tag", "release_sha", "binary_sha256", "fetched_at"} {
		if !strings.HasPrefix(lines[i], key+"=") {
			t.Errorf("line %d must be %s=…, got %q", i+1, key, lines[i])
		}
	}
}

// mustReadFile is a small helper so the tests above can assert on
// .binary-meta's raw shape, not just its parsed values.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestBootstrapIgnoresCurlsOwnConfigSurface is the regression for the bypass an
// independent security review reproduced: curl reads a config surface of its own
// that has nothing to do with this script's argv. Unless `-q` is passed as the
// FIRST argument, every invocation loads `$CURL_HOME/.curlrc` (curl >= 7.73,
// falling back to `$HOME/.curlrc`), and a `connect-to` or `resolve` line there
// re-points the connection while the URL on the command line still literally
// reads https://github.com/… — so `--proto '=https' --proto-redir '=https'`
// still holds, and the checksum verification is VACUOUS, because the same
// poisoned config supplies both the binary and the checksums.txt that "verifies"
// it. The forged binary is then run unattended as the Bash shell guard on every
// tool call.
//
// The fixture is the published release; a second, byte-different server stands in
// for whoever wrote the config. The poisoned .curlrc carries its own `cacert`, as
// the reproduction did, so the redirection is not merely attempted but would
// actually complete on TLS. The bar is that it changes nothing: the published
// bytes are installed, and the forger is never contacted at all.
func TestBootstrapIgnoresCurlsOwnConfigSurface(t *testing.T) {
	bootstrapRequires(t)
	published := []byte("#!/bin/sh\nexit 0\n")
	forged := []byte("#!/bin/sh\nid > /tmp/pwned\n")

	// Both names, because curl consults CURL_HOME first and falls back to HOME,
	// and `-q` is the one answer that covers both without enumerating them.
	for _, home := range []string{"CURL_HOME", "HOME"} {
		name := home
		t.Run(name, func(t *testing.T) {
			root := bootstrapRoot(t)
			fx := bootstrapServer(t, published, bootstrapManifest(published))
			forger := bootstrapServer(t, forged, bootstrapManifest(forged))

			dir := t.TempDir()
			rc := "connect-to = \"" + bootstrapHostPort(fx.base) + ":" + bootstrapHostPort(forger.base) + "\"\n" +
				"cacert = \"" + forger.ca + "\"\n"
			if err := os.WriteFile(filepath.Join(dir, ".curlrc"), []byte(rc), 0o600); err != nil {
				t.Fatal(err)
			}

			// runScript de-duplicates its environment keeping the LAST value, so the
			// HOME case really does replace the constructed one.
			out, code := runScript(t, bootstrapFixtureScript(t, fx.base), root,
				append(fx.env(), name+"="+dir), "")
			if code != 0 {
				t.Fatalf("the published release must still install, got %d (output %q)", code, out)
			}
			if n := atomic.LoadInt32(forger.hits); n != 0 {
				t.Errorf("a %s/.curlrc redirected %d fetch(es) to a server this script never names: curl's own config surface is a fetch-origin seam", name, n)
			}
			if n := atomic.LoadInt32(fx.hits); n == 0 {
				t.Error("the published release server was never contacted, so this case proves nothing")
			}
			got, err := os.ReadFile(filepath.Join(root, "abcd"))
			if err != nil {
				t.Fatalf("the bootstrap must install the published binary: %v", err)
			}
			if string(got) == string(forged) {
				t.Fatalf("the binary installed at the guard path is the FORGED one: a %s/.curlrc supplied both the payload and the manifest that verified it", name)
			}
			if string(got) != string(published) {
				t.Errorf("the installed binary is neither the published nor the forged payload; got %q", got)
			}
		})
	}
}

// bootstrapHostPort reduces a fixture base URL to the host:port pair a curl
// `connect-to` line is written in terms of.
func bootstrapHostPort(base string) string {
	return strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
}

// TestBootstrapReportsAnObstructedBinaryMeta: `mv -f` onto an existing DIRECTORY
// moves the file INTO it rather than replacing it — the same hazard the binary
// path already refuses. At the .binary-meta path the consequence is quieter and
// therefore worse: the provenance record lands at .binary-meta/binary-meta where
// nothing will ever read it, and the success notice claims provenance was
// recorded when it was not. The install itself is genuine and must not be undone,
// so this is reported in the notice rather than refused.
func TestBootstrapReportsAnObstructedBinaryMeta(t *testing.T) {
	root := bootstrapRoot(t)
	body := []byte("payload")
	fx := bootstrapServer(t, body, bootstrapManifest(body))
	meta := filepath.Join(root, ".binary-meta")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		t.Fatal(err)
	}

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("an obstructed .binary-meta must not fail the install itself, got %d (output %q)", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "abcd")); err != nil {
		t.Errorf("the verified binary must still be installed: %v", err)
	}
	entries, err := os.ReadDir(meta)
	if err != nil {
		t.Fatalf("the obstruction must be left as it was found: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the provenance record was moved INTO the obstructing directory (%v), where nothing will ever read it", entries)
	}
	if !strings.Contains(out, "provenance") {
		t.Errorf("the notice must say the provenance record was not written; output %q", out)
	}
	if !strings.Contains(out, "not a regular file") {
		t.Errorf("the notice must name the obstruction, not report a generic write failure; output %q", out)
	}
}

// TestBootstrapScriptIsExecutable: the harness invokes the committed script by
// path, not as `sh <path>`, so a lost executable bit is a fresh-install failure
// no other test would catch — every case above runs a rewritten COPY this test
// file chmods itself.
func TestBootstrapScriptIsExecutable(t *testing.T) {
	fi, err := os.Stat(bootstrapScript(t))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("hooks/bootstrap.sh must be committed executable, got mode %v", fi.Mode().Perm())
	}
}

// bootstrapEnvRefPattern matches every $NAME / ${NAME} reference a shell line
// makes, and bootstrapAssignPattern matches a `name=` assignment at the head of
// a (trimmed) line. They are the two primitives the allowlist below is built on.
var (
	bootstrapEnvRefPattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)`)
	bootstrapAssignPattern = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=`)
)

// bootstrapEnvRefNames returns, in source order, every name the fragment
// references as $NAME or ${NAME}.
func bootstrapEnvRefNames(s string) []string {
	var names []string
	for _, m := range bootstrapEnvRefPattern.FindAllStringSubmatch(s, -1) {
		names = append(names, m[1])
	}
	return names
}

// bootstrapSafeNames returns the names <body> may be said to define ITSELF —
// the set a $NAME reference is allowed to resolve to without being an
// environment read.
//
// A name is safe only if every assignment of it takes its value from constants,
// from CLAUDE_PLUGIN_ROOT / CLAUDE_PLUGIN_DATA / HOME (FILESYSTEM locations —
// destinations, never fetch origins; HOME locates the home-scoped owned-copy
// provenance record spc-35 keeps outside the data dir), or from names that are
// themselves already safe AT THAT POINT in the file. Anything else is an
// environment read wearing a local name, so the name is tainted and stays
// tainted.
//
// The propagation is run to a FIXPOINT rather than in one pass, because taint
// travels through chains. A single-pass, single-line self-reference check
// catches `mirror="${mirror:-}"` and nothing longer: two names that reference
// each other
//
//	repo_url="${GH_MIRROR:-$gh_default}"
//	GH_MIRROR="$repo_url"
//
// leave neither line self-referential, yet GH_MIRROR is read straight out of the
// environment and lands in the fetch origin. Iterating until no name is newly
// tainted closes the two-hop case and every longer chain with it: the first pass
// taints repo_url (its value names GH_MIRROR, which is not defined before use
// and so is an external read), and taint then propagates to GH_MIRROR because
// its value names a tainted name.
func bootstrapSafeNames(body string) map[string]bool {
	type assignment struct {
		name string
		rhs  string
	}
	var assignments []assignment
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		m := bootstrapAssignPattern.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		assignments = append(assignments, assignment{name: m[1], rhs: trimmed[len(m[0]):]})
	}
	tainted := map[string]bool{}
	safe := map[string]bool{}
	for {
		changed := false
		// defined is the "safe so far" set for THIS pass: a name only counts as
		// script-local for references that come after its own assignment, so a
		// value naming a name defined later is read from the environment.
		defined := map[string]bool{}
		for _, a := range assignments {
			if tainted[a.name] {
				delete(defined, a.name)
				continue
			}
			external := false
			for _, ref := range bootstrapEnvRefNames(a.rhs) {
				switch {
				case ref == a.name:
					// Reads the environment under its own name before
					// falling back to a default.
					external = true
				case ref == "CLAUDE_PLUGIN_ROOT", ref == "CLAUDE_PLUGIN_DATA", ref == "HOME":
				case tainted[ref], !defined[ref]:
					external = true
				}
				if external {
					break
				}
			}
			if external {
				tainted[a.name] = true
				delete(defined, a.name)
				changed = true
				continue
			}
			defined[a.name] = true
		}
		safe = defined
		if !changed {
			break
		}
	}
	return safe
}

// bootstrapEnvRead is one unrecognised environment read: the name, and the first
// non-comment line that reads it.
type bootstrapEnvRead struct {
	name string
	line string
}

// bootstrapUnrecognisedEnvReads applies the allowlist: every $NAME reference in
// a non-comment line must be CLAUDE_PLUGIN_ROOT, CLAUDE_PLUGIN_DATA, or a name
// bootstrapSafeNames vouches for. Anything else is reported, whatever it is
// called.
func bootstrapUnrecognisedEnvReads(body string) []bootstrapEnvRead {
	safe := bootstrapSafeNames(body)
	seen := map[string]bool{}
	var reads []bootstrapEnvRead
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, name := range bootstrapEnvRefNames(line) {
			if name == "CLAUDE_PLUGIN_ROOT" || name == "CLAUDE_PLUGIN_DATA" || name == "HOME" || safe[name] || seen[name] {
				continue
			}
			seen[name] = true
			reads = append(reads, bootstrapEnvRead{name: name, line: line})
		}
	}
	return reads
}

// bootstrapCurlCallPattern recognises a line that INVOKES curl, as opposed to
// one that merely mentions it (`command -v curl`, a refusal message).
//
// Matching the substring "curl -" instead would silently skip any call whose
// first argument is not a flag — `curl "$url" -o /dev/null` — so a call missing
// -q or the transport pin could sit in the script and never be scanned, while
// the call count still looked healthy because the scan simply undercounted.
// curl is therefore matched as a whole word in COMMAND position: at the head of
// the line, after a shell operator or command substitution, after a path
// prefix, or after one of the keywords a command can follow.
var bootstrapCurlCallPattern = regexp.MustCompile(
	"(?:^|[|&;(){}/`]|\\b(?:if|then|else|elif|do|while|until|exec|env|time|command|nohup)[ \t]+)[ \t]*curl(?:[ \t]|$)")

// bootstrapCurlCalls returns every non-comment line of body that invokes curl.
func bootstrapCurlCalls(body string) []string {
	var calls []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || !bootstrapCurlCallPattern.MatchString(trimmed) {
			continue
		}
		calls = append(calls, trimmed)
	}
	return calls
}

// bootstrapCurlQFirst matches -q in the ONE position curl honours it in: the
// first argument.
var bootstrapCurlQFirst = regexp.MustCompile(`curl[ \t]+-q(?:[ \t]|$)`)

// bootstrapCurlCallProblems reports what a single curl invocation is missing.
func bootstrapCurlCallProblems(call string) []string {
	var problems []string
	if !strings.Contains(call, "--proto '=https' --proto-redir '=https'") {
		problems = append(problems, "does not pin HTTPS literally")
	}
	if !bootstrapCurlQFirst.MatchString(call) {
		problems = append(problems, "does not pass -q as its FIRST argument, so it loads $CURL_HOME/.curlrc and the fetch origin is no longer this script's to choose")
	}
	return problems
}

// TestBootstrapFetchOriginsAreConstants is the structural half of the security
// bar the two rounds before this one failed. The shipped script must hold its
// fetch origins as literals with NO environment-driven way to redirect them, and
// must pin HTTPS (redirects included) on every call with no toggle: the binary
// and the checksums.txt that "verifies" it come from one origin, so anything
// that can name that origin supplies both the payload and its own manifest, and
// the result is executed unattended as the Bash shell guard on every tool call.
func TestBootstrapFetchOriginsAreConstants(t *testing.T) {
	src, err := os.ReadFile(bootstrapScript(t))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)

	if strings.Contains(body, "ABCD_BOOTSTRAP") {
		t.Error("the shipped script must name no ABCD_BOOTSTRAP override: the seam was removed, not narrowed")
	}
	// A DENYLIST of the names a redirection seam has used so far (ABCD_BOOTSTRAP_*)
	// only ever catches a seam that reuses one of those names — a bypass that
	// renamed the variable (e.g. $GH_MIRROR) would sail through unnoticed. The
	// real invariant this test exists to hold is narrower and checkable directly:
	// every $NAME / ${NAME} reference the script contains, after subtracting the
	// names it genuinely defines itself, must be exactly {CLAUDE_PLUGIN_ROOT,
	// CLAUDE_PLUGIN_DATA, HOME} — filesystem destinations, never a fetch origin.
	// This is an ALLOWLIST: an unrecognised environment reference under any name
	// fails it.
	//
	// "Genuinely defines itself" is the whole difficulty, and it is
	// bootstrapSafeNames' job: a name assigned from the environment is not
	// script-local however local its assignment looks, and taint travels along
	// chains of such assignments. TestBootstrapEnvAllowlistTaintsReferenceChains
	// is that helper's own detector.
	for _, read := range bootstrapUnrecognisedEnvReads(body) {
		t.Errorf("unrecognised environment reference $%s in a non-comment line — every environment read must be CLAUDE_PLUGIN_ROOT, CLAUDE_PLUGIN_DATA, HOME, or a name this script assigns itself: %q", read.name, read.line)
	}
	for _, origin := range []string{bootstrapReleaseOrigin, bootstrapAPIOrigin} {
		if n := strings.Count(body, origin); n != 1 {
			t.Errorf("the script must hold %q as a single literal, found %d", origin, n)
		}
	}
	// An environment reference is not the only origin seam: curl reads names the
	// script never mentions. $CURL_HOME/.curlrc (or $HOME/.curlrc) can re-point a
	// connection with `connect-to`/`resolve` while the URL on the command line
	// still reads github.com, and the proxy variables can route the same fetch
	// through a server of the setter's choosing — with the CA overrides supplying
	// the trust that makes it succeed. Because the binary and the checksums.txt
	// that verifies it travel the same route, any of those makes verification
	// vacuous. The allowlist above cannot see them, so they are asserted directly:
	// every call disables the config file, and the names are gone from the
	// environment before the first fetch.
	for _, scrub := range []string{bootstrapProxyScrub, bootstrapTrustScrub} {
		if n := strings.Count(body, scrub); n != 1 {
			t.Errorf("the script must scrub curl's environment surface exactly once with %q, found %d", scrub, n)
		}
	}
	for _, name := range []string{
		"HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy",
		"CURL_HOME", "CURL_CA_BUNDLE", "SSL_CERT_FILE",
	} {
		if !strings.Contains(bootstrapProxyScrub+" "+bootstrapTrustScrub, name) {
			t.Errorf("%s is a name curl reads on its own and must be scrubbed before any fetch", name)
		}
	}
	// Every curl invocation pins the transport and disables curl's own config
	// file, unconditionally. A variable holding the flags is what the previous
	// round used, and clearing it on one path is what reopened redirect downgrade.
	// -q must be the FIRST argument: curl honours it in no other position.
	// TestBootstrapCurlScanSeesEveryInvocation is the scan's own detector.
	calls := bootstrapCurlCalls(body)
	for _, call := range calls {
		for _, problem := range bootstrapCurlCallProblems(call) {
			t.Errorf("every curl call must pin HTTPS and pass -q first; %q %s", call, problem)
		}
	}
	if len(calls) < 4 {
		t.Errorf("expected the script's four curl calls to be found, saw %d — the scan above is no longer matching", len(calls))
	}
}

// TestBootstrapEnvAllowlistTaintsReferenceChains is the detector for the
// allowlist ABOVE, and exists because that allowlist is the only thing standing
// between a renamed fetch-origin variable and a green test run. A guard nothing
// tests is a guard nobody knows is broken, and this one already shipped broken
// twice: first subtracting any assigned name (so `mirror="${mirror:-}"` read as
// script-local), then subtracting any name whose OWN line did not name itself —
// which a merge-gate reviewer walked through in two hops, with a curl shim
// recording the redirected fetch.
//
// The fragments below are not the shipped script and never run; they are the
// shapes the allowlist must and must not report, fed to it directly.
func TestBootstrapEnvAllowlistTaintsReferenceChains(t *testing.T) {
	const origin = `"https://github.com/intentdriven/abcd"`
	cases := []struct {
		name string
		body string
		// want is the names the allowlist must report; empty means the
		// fragment is legitimate and must be reported clean.
		want []string
	}{
		{
			name: "constants and their derivations are script-local",
			body: "plugin_root=\"${CLAUDE_PLUGIN_ROOT:-}\"\nrepo_url=" + origin + "\nreleases_url=\"$repo_url/releases\"\ncurl -q \"$releases_url/latest\"\n",
		},
		{
			name: "a bare environment read is reported",
			body: "curl -q \"$GH_MIRROR/releases/latest\"\n",
			want: []string{"GH_MIRROR"},
		},
		{
			name: "one hop: a self-referential assignment is reported",
			body: "repo_url=" + origin + "\nmirror=\"${mirror:-}\"\n[ -n \"$mirror\" ] && repo_url=\"$mirror\"\n",
			want: []string{"mirror"},
		},
		{
			// The merge-gate reviewer's reproduction, verbatim. Neither line
			// names itself, so a per-line self-reference check passes it —
			// while GH_MIRROR=https://attacker.invalid/x redirects the fetch.
			name: "two hops: mutually referring names are reported",
			body: "gh_default=" + origin + "\nrepo_url=\"${GH_MIRROR:-$gh_default}\"\nGH_MIRROR=\"$repo_url\"\ncurl -q \"$repo_url/releases/latest\"\n",
			want: []string{"GH_MIRROR", "repo_url"},
		},
		{
			// The same trick with a third name inserted, because closing a
			// fixed hop count is not closing the class.
			name: "three hops through a laundering name",
			body: "gh_default=" + origin + "\nbase=\"${GH_MIRROR:-$gh_default}\"\nrepo_url=\"$base\"\nGH_MIRROR=\"$repo_url\"\ncurl -q \"$repo_url/releases/latest\"\n",
			want: []string{"GH_MIRROR", "base", "repo_url"},
		},
		{
			// Assignment order is load-bearing on the ASSIGNMENT side: a value
			// naming a name defined LATER reads the environment, whatever that
			// name is set to afterwards, so repo_url is tainted and every use
			// of it is reported. $mirror itself is not reported by name,
			// because the reference scan stays order-insensitive on purpose —
			// a shell function body runs long after the line that defines it,
			// so "referenced before assigned" is not a defect there.
			name: "a forward reference is an environment read",
			body: "repo_url=\"$mirror\"\nmirror=" + origin + "\ncurl -q \"$repo_url/releases/latest\"\n",
			want: []string{"repo_url"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := map[string]bool{}
			for _, read := range bootstrapUnrecognisedEnvReads(tc.body) {
				got[read.name] = true
			}
			for _, name := range tc.want {
				if !got[name] {
					t.Errorf("the allowlist must report $%s as an unrecognised environment read; it reported %v", name, keysOf(got))
				}
				delete(got, name)
			}
			for name := range got {
				t.Errorf("the allowlist reported $%s, which this fragment defines itself — a false positive here fails the shipped script for no reason", name)
			}
		})
	}

	// The same two-hop reproduction planted into the REAL script's bytes, which
	// is how the merge-gate reviewer ran it: with GH_MIRROR set, the planted
	// script fetches from wherever GH_MIRROR says, and the allowlist as it stood
	// reported nothing at all.
	t.Run("planted into the committed script", func(t *testing.T) {
		src, err := os.ReadFile(bootstrapScript(t))
		if err != nil {
			t.Fatal(err)
		}
		const constant = "repo_url=" + `"https://github.com/intentdriven/abcd"`
		planted := strings.Replace(string(src), constant,
			"gh_default="+`"https://github.com/intentdriven/abcd"`+"\nrepo_url=\"${GH_MIRROR:-$gh_default}\"\nGH_MIRROR=\"$repo_url\"", 1)
		if planted == string(src) {
			t.Fatalf("the planted bypass could not be spliced in: %q no longer appears in the script, so this detector is testing nothing", constant)
		}
		found := false
		for _, read := range bootstrapUnrecognisedEnvReads(planted) {
			if read.name == "GH_MIRROR" {
				found = true
			}
		}
		if !found {
			t.Error("the allowlist must report $GH_MIRROR in the planted script; it reported nothing, so a renamed fetch origin would pass this suite green")
		}
	})
}

// keysOf renders a set for a failure message, sorted so the message is stable.
func keysOf(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestBootstrapCurlScanSeesEveryInvocation is the detector for the curl scan
// above. Filtering lines on the substring "curl -" made the scan's coverage
// depend on argument ORDER: a call whose first argument is its URL was skipped
// entirely, so it could be missing -q and the transport pin without the scan
// ever looking at it — and the call count, being the scan's own count, stayed
// happily consistent with itself. The scan matches curl as a command word now,
// and this holds that: every shape that IS an invocation is seen, every shape
// that merely mentions curl is not.
func TestBootstrapCurlScanSeesEveryInvocation(t *testing.T) {
	const pin = "--proto '=https' --proto-redir '=https'"
	cases := []struct {
		name         string
		line         string
		wantCall     bool
		wantProblems bool
	}{
		{
			name:     "the shipped shape",
			line:     "curl -q -fsSL " + pin + " --max-time 120 -o \"$tmp/$asset\" \"$url\"",
			wantCall: true,
		},
		{
			name:     "inside a command substitution",
			line:     "redirect=$(curl -q -fsS " + pin + " -o /dev/null \"$url\")",
			wantCall: true,
		},
		{
			// The dodge: flags after the URL, so "curl -" never appears. The
			// scan must see it AND report it, because -q is honoured in no
			// position but the first.
			name:         "flags after the url, dodging the substring",
			line:         "curl \"$url\" -q " + pin + " -o /dev/null",
			wantCall:     true,
			wantProblems: true,
		},
		{
			name:         "no pin and no -q at all",
			line:         "curl \"$url\" -o /dev/null",
			wantCall:     true,
			wantProblems: true,
		},
		{
			name:         "an absolute path to curl",
			line:         "/usr/bin/curl \"$url\" -o /dev/null",
			wantCall:     true,
			wantProblems: true,
		},
		{
			name:         "after a shell keyword",
			line:         "if curl \"$url\" -o /dev/null; then",
			wantCall:     true,
			wantProblems: true,
		},
		{
			name:         "downstream of a pipe",
			line:         "printf '%s' \"$url\" | curl \"$url\" -o /dev/null",
			wantCall:     true,
			wantProblems: true,
		},
		{
			// Mentions, not invocations: reporting these would make the scan
			// unusable and tempt the next reader to loosen it again.
			name: "a availability probe is not an invocation",
			line: "command -v curl >/dev/null 2>&1 ||",
		},
		{
			name: "prose naming curl is not an invocation",
			line: "refuse 'curl is not available, so the release binary cannot be downloaded'",
		},
		{
			name: "curlrc is not an invocation",
			line: "printf '%s' \"$CURL_HOME/.curlrc\"",
		},
		{
			name: "a commented-out call is not an invocation",
			line: "# curl \"$url\" -o /dev/null",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := bootstrapCurlCalls(tc.line + "\n")
			if tc.wantCall && len(calls) != 1 {
				t.Fatalf("the scan must see %q as a curl invocation; it saw %d", tc.line, len(calls))
			}
			if !tc.wantCall {
				if len(calls) != 0 {
					t.Fatalf("the scan must not read %q as a curl invocation; it saw %d", tc.line, len(calls))
				}
				return
			}
			problems := bootstrapCurlCallProblems(calls[0])
			if tc.wantProblems && len(problems) == 0 {
				t.Errorf("the scan must report %q as unpinned or missing -q first, and reported nothing", tc.line)
			}
			if !tc.wantProblems && len(problems) != 0 {
				t.Errorf("the scan must accept %q; it reported %v", tc.line, problems)
			}
		})
	}
}

// TestBootstrapIgnoresEveryEnvironmentOrigin is the behavioural half, and the
// direct regression for the defect two review rounds failed to close: the
// COMMITTED script (no substitution) is run with the removed override variables
// set to every shape that previously worked — a bare loopback base, the URL
// userinfo form that walked straight through the loopback glob
// (http://127.0.0.1:1@host), and a file:// origin. curl is shimmed so nothing
// leaves the machine and every invocation is recorded; the assertions are that
// the recorded URLs name GitHub and only GitHub, that the fixture server was
// never contacted, and that the transport pin is present on each call.
func TestBootstrapIgnoresEveryEnvironmentOrigin(t *testing.T) {
	bootstrapRequires(t)
	fx := bootstrapServer(t, []byte("attacker payload"), bootstrapManifest([]byte("attacker payload")))
	hostile := []struct{ name, value string }{
		// The plain loopback base the removed allowlist accepted by design.
		{"loopback base", fx.base},
		// The userinfo form an independent reviewer walked through the loopback
		// glob twice: everything before the @ is a username, so the real host is
		// abcd-bootstrap.invalid while the pattern reads 127.0.0.1.
		{"userinfo past the loopback glob", "http://127.0.0.1:1@abcd-bootstrap.invalid"},
		{"userinfo over https", "https://127.0.0.1:1@abcd-bootstrap.invalid"},
		{"local file origin", "file:///dev/null"},
	}
	for _, tc := range hostile {
		value := tc.value
		t.Run(tc.name, func(t *testing.T) {
			root := bootstrapRoot(t)
			shim := t.TempDir()
			log := filepath.Join(shim, "curl.log")
			// The shim records the argv it was handed and fails, standing in for
			// "no network": the script then refuses at tag resolution, and the log
			// is the evidence of where it TRIED to go.
			fake := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\nexit 1\n"
			if err := os.WriteFile(filepath.Join(shim, "curl"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}

			out, code := runScript(t, bootstrapScript(t), root, []string{
				"ABCD_BOOTSTRAP_BASE_URL=" + value,
				"ABCD_BOOTSTRAP_API_URL=" + value,
				"ABCD_BOOTSTRAP_URL=" + value,
			}, shim)
			if code == 0 {
				t.Fatalf("with no reachable network the script must refuse; got exit 0 (output %q)", out)
			}
			recorded, err := os.ReadFile(log)
			if err != nil {
				t.Fatalf("the script must have attempted at least one fetch: %v", err)
			}
			calls := strings.Split(strings.TrimSpace(string(recorded)), "\n")
			for _, call := range calls {
				if !strings.Contains(call, bootstrapReleaseOrigin) && !strings.Contains(call, bootstrapAPIOrigin) {
					t.Errorf("a fetch went somewhere other than the compiled origins: %q", call)
				}
				if !strings.Contains(call, "--proto =https --proto-redir =https") {
					t.Errorf("a fetch was made without the HTTPS pin: %q", call)
				}
			}
			for _, forbidden := range []string{"127.0.0.1", "abcd-bootstrap.invalid", "file://"} {
				if strings.Contains(string(recorded), forbidden) {
					t.Errorf("the environment redirected a fetch to %q; recorded calls: %q", forbidden, recorded)
				}
			}
			if n := atomic.LoadInt32(fx.hits); n != 0 {
				t.Errorf("the fixture server was contacted %d time(s): the override still redirects", n)
			}
			assertNothingInstalled(t, root, out)
		})
	}
}
