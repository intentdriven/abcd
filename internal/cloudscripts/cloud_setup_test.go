package cloudscripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The setup step (scripts/cloud/connectors/claude/setup.sh, C1 and C11),
// spc-2610120941441282 step 4. The host runs it as root from a SessionStart
// hook; here it runs as the test's own user in newCloudCase's clone, with a
// fake for every command that would need root or the network: id, useradd,
// chown, runuser, uname, curl, sha256sum, tar and install. The checksum and the
// download are recorded, not performed. abcd's token-identity verb
// (`go run ./cmd/abcd reserve --holder --json`) is a fake answering what a
// test sets, so no test needs the forge.

const (
	setupScript = "scripts/cloud/connectors/claude/setup.sh"

	// The identity a hosted container comes with: the vendor's default. This
	// file is the one place in this branch that spells it, so a test can prove
	// the setup never repeats it.
	vendorName = "Claude"
	vendorMail = "noreply@anthropic.com"

	tokenLogin   = "octo-person"
	tokenNoReply = "4242+octo-person@users.noreply.github.com"
)

func holderJSON(login, noreply, source string) string {
	return `{"login":"` + login + `","id":4242,"noreply":"` + noreply + `","source":"` + source + `"}`
}

// newSetupCase is newCloudCase's clone with the files the setup reads, the
// vendor identity in the checkout's own configuration, and the fakes.
//
//	FAKE_UID          what `id -u` prints (default 0, root)
//	FAKE_USERADD      useradd's exit code when set (it fails without making the user)
//	FAKE_SHA          the checksum tool's exit code (default 0)
//	FAKE_HOLDER       what the token-identity verb prints on standard output
//	FAKE_HOLDER_EXIT  its exit code (default 0)
//	FAKE_BUILD        the build of abcd's exit code when set (it fails)
//	FAKE_ROOT_ONLY    a directory standing in for root's home: the user abcd
//	                  cannot read what it holds, so a command run through
//	                  runuser that names a file there is refused, and the
//	                  token-identity verb, which reaches the forge over HTTPS,
//	                  fails as TLS does when a CA variable names a file there
//
// git's system configuration is a file of the test's own ($STATE/gitconfig,
// through GIT_CONFIG_SYSTEM), so the setup's root-wide trust of the checkout
// is read back from there and never reaches the machine's.
//
// Made users are files under $STATE; `install` puts the gitleaks the fake tar
// unpacked first on PATH, so a second run finds it in place.
func newSetupCase(t *testing.T) *cloudCase {
	t.Helper()
	c := newCloudCase(t)
	top := exec.Command("git", "rev-parse", "--show-toplevel")
	top.Env = gittest.Env(t)
	out, err := top.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	root := strings.TrimSpace(string(out))
	for _, rel := range []string{".github/workflows/ci.yml", "scripts/pinned-toolchain.sh"} {
		c.copyFrom(root, rel)
	}
	c.git("config", "user.name", vendorName)
	c.git("config", "user.email", vendorMail)

	state := filepath.Join(filepath.Dir(c.dir), "state")
	goroot := filepath.Join(filepath.Dir(c.dir), "goroot")
	for _, d := range []string{state, filepath.Join(goroot, "bin")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	declared := c.declaredGo()
	for _, tool := range []string{"go", "gofmt"} {
		body := "#!/bin/sh\necho \"go version " + declared + " linux/amd64\"\n"
		if err := os.WriteFile(filepath.Join(goroot, "bin", tool), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	c.env = append(c.env, "STATE="+state, "BIN_DIR="+c.bin, "FAKE_GOROOT="+goroot,
		"GIT_CONFIG_SYSTEM="+filepath.Join(state, "gitconfig"))

	c.fake("id", `echo "id $*" >>"$CALLS"
case "$*" in
"-u") echo "${FAKE_UID:-0}" ;;
"-u abcd") [ -e "$STATE/user-abcd" ] || exit 1; echo 1001 ;;
*) echo "fake id: unexpected $*" >&2; exit 3 ;;
esac`)
	c.fake("useradd", `echo "useradd $*" >>"$CALLS"
if [ -n "${FAKE_USERADD:-}" ]; then echo "useradd: cannot lock /etc/passwd" >&2; exit "$FAKE_USERADD"; fi
: >"$STATE/user-abcd"`)
	c.fake("chown", `echo "chown $*" >>"$CALLS"`)
	c.fake("runuser", `[ "${1:-}" = -u ] && [ "${2:-}" = abcd ] && [ "${3:-}" = -- ] || { echo "fake runuser: unexpected $*" >&2; exit 3; }
shift 3
echo "as-abcd $*" >>"$CALLS"
if [ -n "${FAKE_ROOT_ONLY:-}" ]; then
	for a in "$@"; do
		case "$a" in "$FAKE_ROOT_ONLY"/*) echo "fake runuser: $a: Permission denied" >&2; exit 1 ;; esac
	done
fi
exec "$@"`)
	c.fake("uname", `case "$*" in -m) echo x86_64 ;; -s) echo Linux ;; *) exec /usr/bin/uname "$@" ;; esac`)
	c.fake("curl", `out="" url="" prev=""
for a in "$@"; do
	[ "$prev" = -o ] && out="$a"
	prev="$a"
	url="$a"
done
echo "curl $url" >>"$CALLS"
b="${url##*/}"
case "$b" in
*_checksums.txt)
	v="${b#gitleaks_}"; v="${v%_checksums.txt}"
	printf 'aaaa  gitleaks_%s_linux_arm64.tar.gz\nbbbb  gitleaks_%s_linux_x64.tar.gz\n' "$v" "$v" >"$out" ;;
*) echo tarball >"$out" ;;
esac`)
	c.fake("sha256sum", `echo "sha256sum $*" >>"$CALLS"
cat >>"$CALLS.sha"
exit "${FAKE_SHA:-0}"`)
	c.fake("tar", `echo "tar $*" >>"$CALLS"
for a in "$@"; do case "$a" in gitleaks_*.tar.gz) v="${a#gitleaks_}"; v="${v%%_*}" ;; esac; done
printf '#!/bin/sh\necho "%s"\n' "$v" >gitleaks
chmod +x gitleaks`)
	c.fake("install", `echo "install $*" >>"$CALLS"
cat "$1" >"$BIN_DIR/gitleaks"
chmod +x "$BIN_DIR/gitleaks"`)
	c.fake("go", `echo "go $*" >>"$CALLS"
case "$*" in
"env GOVERSION") echo go0.0.0 ;;
"env GOROOT") echo "$FAKE_GOROOT" ;;
"build -o "*" ./cmd/abcd")
	if [ -n "${FAKE_BUILD:-}" ]; then echo "fake go: the build failed" >&2; exit "$FAKE_BUILD"; fi
	printf '#!/bin/sh\necho built abcd\n' >"$3" && chmod +x "$3" ;;
"run ./cmd/abcd reserve --holder --json")
	for v in SSL_CERT_FILE GIT_SSL_CAINFO CURL_CA_BUNDLE REQUESTS_CA_BUNDLE NODE_EXTRA_CA_CERTS; do
		eval "f=\${$v:-}"
		[ -z "$f" ] || echo "$v=$f" >>"$STATE/holder-ca"
		case "$f" in "${FAKE_ROOT_ONLY:-/nonexistent}"/*)
			echo "abcd reserve: x509: certificate signed by unknown authority ($v=$f: permission denied)" >&2; exit 1 ;;
		esac
	done
	if [ -n "${FAKE_HOLDER:-}" ]; then printf '%s\n' "$FAKE_HOLDER"; fi
	echo "abcd reserve: git's identity is $(git config user.name) <$(git config user.email)>" >&2
	exit "${FAKE_HOLDER_EXIT:-0}" ;;
*) echo "fake go: unexpected $*" >&2; exit 3 ;;
esac`)
	c.resetCalls()
	return c
}

var setupRemote = []string{remoteMarker}

// localConfig is the checkout's own configuration, as a listing.
func (c *cloudCase) localConfig() string {
	c.t.Helper()
	return c.git("config", "--local", "--list")
}

// localIdentity is the identity half of the checkout's own configuration.
func (c *cloudCase) localIdentity() string {
	c.t.Helper()
	var keep []string
	for _, l := range strings.Split(c.localConfig(), "\n") {
		if strings.HasPrefix(l, "user.") || strings.HasPrefix(l, "author.") || strings.HasPrefix(l, "committer.") {
			keep = append(keep, l)
		}
	}
	return strings.Join(keep, "\n")
}

// refsAndHead is every ref and HEAD, so a test can prove nothing was committed.
func (c *cloudCase) refsAndHead() string {
	c.t.Helper()
	return c.git("for-each-ref", "--format=%(refname) %(objectname)") + "\n" + c.git("rev-parse", "HEAD")
}

func TestSetupDoesNothingOffCloud(t *testing.T) {
	for _, env := range [][]string{nil, {"CLAUDE_CODE_REMOTE="}, {"CLAUDE_CODE_REMOTE=false"}, {"CLAUDE_CODE_REMOTE=1"}} {
		c := newSetupCase(t)
		before, refs := c.localConfig(), c.refsAndHead()
		stamps := treeStamps(t, c.dir)
		out, code := c.script(env, setupScript)
		if code != 0 || out != "" {
			t.Errorf("setup off the cloud (%v) exited %d and printed %q; it exits 0 silently", env, code, out)
		}
		if log := c.callLog(); log != "" {
			t.Errorf("setup off the cloud (%v) ran commands:\n%s", env, log)
		}
		if c.localConfig() != before || c.refsAndHead() != refs {
			t.Errorf("setup off the cloud (%v) changed the checkout's configuration or refs", env)
		}
		after := treeStamps(t, c.dir)
		for p, s := range stamps {
			if after[p] != s {
				t.Errorf("setup off the cloud (%v) touched %s", env, p)
			}
		}
		if len(after) != len(stamps) {
			t.Errorf("setup off the cloud (%v) made files", env)
		}
	}
}

func TestSetupSetsIdentityFromToken(t *testing.T) {
	c := newSetupCase(t)
	out, code := c.script(append(setupRemote, "FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token")), setupScript)
	if code != 0 {
		t.Fatalf("setup exited %d\n%s", code, out)
	}
	if got := c.git("config", "--local", "--get", "user.name"); got != tokenLogin {
		t.Errorf("user.name = %q, want the token's login %q", got, tokenLogin)
	}
	if got := c.git("config", "--local", "--get", "user.email"); got != tokenNoReply {
		t.Errorf("user.email = %q, want the token's no-reply address %q", got, tokenNoReply)
	}
	log := c.callLog()
	if !strings.Contains(log, "as-abcd go run ./cmd/abcd reserve --holder --json") {
		t.Errorf("the identity was not looked up through the one primitive, as the user abcd:\n%s", log)
	}
	for _, key := range []string{"user.name", "user.email"} {
		if !regexp.MustCompile(`(?m)^as-abcd git -C \S+ config --local ` + regexp.QuoteMeta(key) + ` `).MatchString(log) {
			t.Errorf("%s was not written with git config --local, as abcd:\n%s", key, log)
		}
	}
	if strings.Contains(log, "--global") {
		t.Errorf("setup wrote configuration outside the checkout:\n%s", log)
	}
	for _, l := range regexp.MustCompile(`(?m)^.*--system.*$`).FindAllString(log, -1) {
		if !strings.Contains(l, " safe.directory") {
			t.Errorf("setup wrote system configuration other than the checkout's trust: %s", l)
		}
	}
	if !strings.Contains(out, tokenLogin) {
		t.Errorf("setup does not say whose identity it set\n%s", out)
	}
	if strings.Contains(out, vendorName+" ") || strings.Contains(out, vendorMail) {
		t.Errorf("setup repeated the container's identity\n%s", out)
	}
}

func TestSetupLookupFailureSetsNothingAndSaysSoOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"the lookup fails", []string{"FAKE_HOLDER_EXIT=1"}},
		{"the lookup answers from git, not the token", []string{"FAKE_HOLDER=" + holderJSON(vendorName, vendorMail, "git")}},
		{"the lookup answers from the machine key", []string{"FAKE_HOLDER=" + holderJSON(tokenLogin, "", "config")}},
		{"the answer is not an account", []string{"FAKE_HOLDER=" + holderJSON("x y", "nobody@example.com", "token")}},
		{"the no-reply is another account's", []string{"FAKE_HOLDER=" + holderJSON(tokenLogin, "1+someone-else@users.noreply.github.com", "token")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newSetupCase(t)
			before, refs := c.localIdentity(), c.refsAndHead()
			if !strings.Contains(before, vendorMail) {
				t.Fatalf("the fixture's checkout does not carry the container's identity:\n%s", before)
			}
			out, code := c.script(append(setupRemote, tc.env...), setupScript)
			if code == 0 {
				t.Errorf("setup exited 0 with no identity in place\n%s", out)
			}
			if got := c.localIdentity(); got != before {
				t.Errorf("setup changed the checkout's identity:\nbefore:\n%s\nafter:\n%s", before, got)
			}
			if got := c.refsAndHead(); got != refs {
				t.Errorf("setup committed or moved a ref")
			}
			if strings.Contains(out, vendorMail) || regexp.MustCompile(`\b`+vendorName+`\b`).MatchString(out) {
				t.Errorf("setup's output repeats the container's identity\n%s", out)
			}
			lines := 0
			for _, l := range strings.Split(out, "\n") {
				if strings.Contains(l, "could not be taken from the forge token") {
					lines++
				}
			}
			if lines != 1 {
				t.Errorf("setup said %d times that the identity could not be taken from the forge token, want once\n%s", lines, out)
			}
			// Every other prerequisite is still put in place.
			if !strings.Contains(c.callLog(), "core.hooksPath .githooks") {
				t.Errorf("a failed identity lookup stopped the rest of the setup:\n%s", c.callLog())
			}
		})
	}
}

func TestSetupReadsGitleaksPinFromCI(t *testing.T) {
	pinRe := regexp.MustCompile(`(?m)^\s*GITLEAKS_VERSION:\s*([0-9][0-9.]*)\s*$`)
	t.Run("the committed pin", func(t *testing.T) {
		c := newSetupCase(t)
		ci, err := os.ReadFile(filepath.Join(c.dir, ".github", "workflows", "ci.yml"))
		if err != nil {
			t.Fatal(err)
		}
		m := pinRe.FindSubmatch(ci)
		if m == nil {
			t.Fatal("ci.yml pins no GITLEAKS_VERSION")
		}
		want := string(m[1])
		out, code := c.script(append(setupRemote, "FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token")), setupScript)
		if code != 0 {
			t.Fatalf("setup exited %d\n%s", code, out)
		}
		log := c.callLog()
		base := "gitleaks_" + want + "_linux_x64.tar.gz"
		for _, url := range []string{
			"https://github.com/gitleaks/gitleaks/releases/download/v" + want + "/" + base,
			"https://github.com/gitleaks/gitleaks/releases/download/v" + want + "/gitleaks_" + want + "_checksums.txt",
		} {
			if !strings.Contains(log, "curl "+url+"\n") {
				t.Errorf("setup did not fetch %s:\n%s", url, log)
			}
		}
		sha, err := os.ReadFile(c.calls + ".sha")
		if err != nil || string(sha) != "bbbb  "+base+"\n" {
			t.Errorf("the checksum tool was given %q (%v), want the one line for %s, as the CI job does", sha, err, base)
		}
		if !strings.Contains(log, "sha256sum -c -\n") || !strings.Contains(log, "install gitleaks /usr/local/bin/gitleaks\n") {
			t.Errorf("gitleaks was not checked and installed as the CI job does:\n%s", log)
		}
		if !strings.Contains(out, "gitleaks "+want) {
			t.Errorf("setup does not say which gitleaks it put in place\n%s", out)
		}
	})

	t.Run("a changed pin is followed", func(t *testing.T) {
		c := newSetupCase(t)
		ci, _ := os.ReadFile(filepath.Join(c.dir, ".github", "workflows", "ci.yml"))
		c.write(".github/workflows/ci.yml", pinRe.ReplaceAllString(string(ci), "      GITLEAKS_VERSION: 9.8.7"), 0o644)
		c.script(setupRemote, setupScript)
		if !strings.Contains(c.callLog(), "/v9.8.7/gitleaks_9.8.7_linux_x64.tar.gz\n") {
			t.Errorf("setup did not follow the pin ci.yml carries:\n%s", c.callLog())
		}
	})

	for _, tc := range []struct{ name, ci string }{
		{"no pin", "jobs: {}\n"},
		{"two pins", "a:\n  GITLEAKS_VERSION: 8.24.3\nb:\n  GITLEAKS_VERSION: 8.25.0\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newSetupCase(t)
			c.write(".github/workflows/ci.yml", tc.ci, 0o644)
			out, code := c.script(setupRemote, setupScript)
			if code == 0 {
				t.Errorf("setup exited 0 without one gitleaks pin\n%s", out)
			}
			if strings.Contains(c.callLog(), "curl ") {
				t.Errorf("setup downloaded with no single pin to follow:\n%s", c.callLog())
			}
			if !strings.Contains(out, "GITLEAKS_VERSION") {
				t.Errorf("setup does not name the pin it could not read\n%s", out)
			}
		})
	}

	t.Run("a checksum mismatch installs nothing", func(t *testing.T) {
		c := newSetupCase(t)
		out, code := c.script(append(setupRemote, "FAKE_SHA=1"), setupScript)
		if code == 0 {
			t.Errorf("setup exited 0 with gitleaks unverified\n%s", out)
		}
		if strings.Contains(c.callLog(), "install ") || strings.Contains(c.callLog(), "tar ") {
			t.Errorf("setup unpacked or installed a download that failed its checksum:\n%s", c.callLog())
		}
	})
}

func TestSetupRefusesWhenUserCannotBeCreated(t *testing.T) {
	c := newSetupCase(t)
	out, code := c.script(append(setupRemote, "FAKE_USERADD=1"), setupScript)
	if code != 1 {
		t.Fatalf("setup exited %d when the user could not be created, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "abcd") || !strings.Contains(out, "could not be created") {
		t.Errorf("setup does not say the user could not be created\n%s", out)
	}
	log := c.callLog()
	for _, never := range []string{"chown ", "as-abcd ", "curl "} {
		if strings.Contains(log, never) {
			t.Errorf("setup went on after the user could not be created (%s):\n%s", strings.TrimSpace(never), log)
		}
	}
}

func TestSetupRefusesWhenNotRoot(t *testing.T) {
	c := newSetupCase(t)
	out, code := c.script(append(setupRemote, "FAKE_UID=1000"), setupScript)
	if code != 1 || !strings.Contains(out, "root") {
		t.Errorf("setup as a normal user exited %d\n%s", code, out)
	}
	if strings.Contains(c.callLog(), "useradd") {
		t.Errorf("setup as a normal user tried to make a user:\n%s", c.callLog())
	}
}

func TestSetupIsIdempotent(t *testing.T) {
	c := newSetupCase(t)
	env := append(setupRemote, "FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token"))
	first, code := c.script(env, setupScript)
	if code != 0 {
		t.Fatalf("the first setup exited %d\n%s", code, first)
	}
	log := c.callLog()
	if !regexp.MustCompile(`(?m)^useradd .*\babcd$`).MatchString(log) {
		t.Errorf("the first setup made no user abcd:\n%s", log)
	}
	if m := regexp.MustCompile(`(?m)^useradd .*$`).FindString(log); strings.Contains(m, "sudo") || strings.Contains(m, "wheel") || strings.Contains(m, " -G") {
		t.Errorf("the user is given more than a normal user's groups: %s", m)
	}
	if !regexp.MustCompile(`(?m)^chown -R abcd:abcd \S+$`).MatchString(log) {
		t.Errorf("the checkout was not given to abcd:\n%s", log)
	}
	if !strings.Contains(log, "as-abcd bash ") || !strings.Contains(log, "pinned-toolchain.sh "+strings.TrimPrefix(c.declaredGo(), "go")) {
		t.Errorf("the toolchain was not resolved through scripts/pinned-toolchain.sh as abcd:\n%s", log)
	}
	if got := c.git("config", "--local", "--get", "core.hooksPath"); got != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", got)
	}
	for _, want := range []string{"user abcd", "gitleaks", "toolchain", "core.hooksPath", tokenLogin} {
		if !strings.Contains(first, want) {
			t.Errorf("the first setup prints no line for %q\n%s", want, first)
		}
	}
	config := c.localConfig()

	c.resetCalls()
	second, code := c.script(env, setupScript)
	if code != 0 {
		t.Fatalf("the second setup exited %d\n%s", code, second)
	}
	log = c.callLog()
	for _, never := range []string{"useradd ", "curl ", "install "} {
		if strings.Contains(log, never) {
			t.Errorf("the second setup did again what was already in place (%s):\n%s", strings.TrimSpace(never), log)
		}
	}
	if got := c.localConfig(); got != config {
		t.Errorf("the second setup changed the configuration:\nfirst:\n%s\nsecond:\n%s", config, got)
	}
	if n, m := strings.Count(first, "\n"), strings.Count(second, "\n"); n != m {
		t.Errorf("the second setup printed %d lines, the first %d; one line per prerequisite each time\n%s\n--\n%s", m, n, first, second)
	}
}

// systemTrust is every safe.directory value git's system configuration holds.
func (c *cloudCase) systemTrust() []string {
	c.t.Helper()
	out, err := c.run(c.dir, "git", "config", "--system", "--get-all", "safe.directory")
	if err != nil {
		return nil
	}
	return strings.Fields(out)
}

// TestSetupTrustsTheCheckoutForRoot: once the checkout belongs to abcd, git
// run as root in it (the host's own commands, or a hand-back run as root)
// refuses it as of dubious ownership. So the setup trusts exactly that
// checkout in git's system configuration, once however often it runs.
func TestSetupTrustsTheCheckoutForRoot(t *testing.T) {
	c := newSetupCase(t)
	env := append(setupRemote, "FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token"))
	first, code := c.script(env, setupScript)
	if code != 0 {
		t.Fatalf("setup exited %d\n%s", code, first)
	}
	if got := c.systemTrust(); len(got) != 1 || got[0] != c.dir {
		t.Fatalf("git's system configuration trusts %q, want the checkout alone, %q", got, c.dir)
	}
	if !regexp.MustCompile(`(?m)^git config --system --add safe.directory \S+$`).MatchString(c.callLog()) {
		t.Errorf("the trust was not added with git config --system --add, as root:\n%s", c.callLog())
	}
	if !strings.Contains(first, "safe.directory") {
		t.Errorf("setup prints no line for the checkout's trust\n%s", first)
	}
	c.resetCalls()
	second, code := c.script(env, setupScript)
	if code != 0 {
		t.Fatalf("the second setup exited %d\n%s", code, second)
	}
	if got := c.systemTrust(); len(got) != 1 || got[0] != c.dir {
		t.Errorf("after a second setup git's system configuration trusts %q, want the checkout once", got)
	}
	if strings.Contains(c.callLog(), "--add safe.directory") {
		t.Errorf("the second setup added the trust again:\n%s", c.callLog())
	}
}

// TestSetupBuildsAbcdForLaterShells: the pre-push hook checks every commit of
// a work-in-progress push with abcd lint outbound, and through `go run` each
// check would build abcd inside the push. So the setup builds abcd once, as
// abcd, into the local tier, and hands its path to the session's later shells
// through the host's file of variables (CLAUDE_ENV_FILE): ABCD_OUTBOUND_BIN for
// the attribution gate, ABCD_BIN for the cloud scripts.
func TestSetupBuildsAbcdForLaterShells(t *testing.T) {
	bin := filepath.Join(".abcd", ".work.local", "cloud", "bin", "abcd")
	holder := "FAKE_HOLDER=" + holderJSON(tokenLogin, tokenNoReply, "token")

	t.Run("built and exported once", func(t *testing.T) {
		c := newSetupCase(t)
		envFile := filepath.Join(t.TempDir(), "session.env")
		env := append(append([]string{}, setupRemote...), holder, "CLAUDE_ENV_FILE="+envFile)
		for run := 1; run <= 2; run++ {
			out, code := c.script(env, setupScript)
			if code != 0 {
				t.Fatalf("setup run %d exited %d\n%s", run, code, out)
			}
			if !strings.Contains(out, bin) {
				t.Errorf("setup run %d does not say where it built abcd\n%s", run, out)
			}
		}
		path := filepath.Join(c.dir, bin)
		if st, err := os.Stat(path); err != nil || st.Mode()&0o111 == 0 {
			t.Fatalf("no executable abcd at %s (%v)", path, err)
		}
		if !regexp.MustCompile(`(?m)^as-abcd go build -o \S+/` + regexp.QuoteMeta(filepath.ToSlash(bin)) + ` ./cmd/abcd$`).MatchString(c.callLog()) {
			t.Errorf("abcd was not built as the user abcd into the local tier:\n%s", c.callLog())
		}
		b, err := os.ReadFile(envFile)
		if err != nil {
			t.Fatalf("the variables file was not written: %v", err)
		}
		for _, want := range []string{"export ABCD_OUTBOUND_BIN='" + path + "'", "export ABCD_BIN='" + path + "'"} {
			if n := strings.Count(string(b), want+"\n"); n != 1 {
				t.Errorf("the variables file carries %q %d times after two setups, want once:\n%s", want, n, b)
			}
		}
		if strings.Count(string(b), "\n") != 2 {
			t.Errorf("the variables file carries more than the two exports:\n%s", b)
		}
	})

	t.Run("no variables file", func(t *testing.T) {
		c := newSetupCase(t)
		out, code := c.script(append(append([]string{}, setupRemote...), holder), setupScript)
		if code != 0 {
			t.Fatalf("setup exited %d\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(c.dir, bin)); err != nil {
			t.Errorf("abcd was not built: %v", err)
		}
		if !strings.Contains(out, "ABCD_OUTBOUND_BIN") {
			t.Errorf("setup does not say the binary could not be handed to later shells\n%s", out)
		}
	})

	t.Run("a build that fails", func(t *testing.T) {
		c := newSetupCase(t)
		envFile := filepath.Join(t.TempDir(), "session.env")
		out, code := c.script(append(append([]string{}, setupRemote...), holder, "FAKE_BUILD=1", "CLAUDE_ENV_FILE="+envFile), setupScript)
		if code == 0 {
			t.Errorf("setup exited 0 with abcd not built\n%s", out)
		}
		if !strings.Contains(out, "NOT IN PLACE") || !strings.Contains(out, "abcd could not be built") {
			t.Errorf("setup does not say abcd could not be built\n%s", out)
		}
		if b, err := os.ReadFile(envFile); err == nil && strings.Contains(string(b), "ABCD_") {
			t.Errorf("a binary that was not built was exported:\n%s", b)
		}
		if !strings.Contains(c.callLog(), "reserve --holder") {
			t.Errorf("a failed build stopped the rest of the setup:\n%s", c.callLog())
		}
	})
}

// TestSetupParsesTheRealHolderAnswer: the other setup tests fake the holder
// verb's answer by hand. This one takes the answer from the real verb, built
// from this checkout and run in a clone carrying the container's vendor
// identity with a gh that answers the token's account, and feeds it to the
// setup, so the two cannot drift apart in shape.
func TestSetupParsesTheRealHolderAnswer(t *testing.T) {
	bin := builtAbcd.Path(t)
	// A gh that answers the token's account, first on PATH.
	ghDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(ghDir, "gh"), []byte("#!/bin/sh\nprintf '%s' '{\"login\":\""+tokenLogin+"\",\"id\":4242}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	clone := t.TempDir()
	env := append(gittest.Env(t), "PATH="+ghDir+string(os.PathListSeparator)+os.Getenv("PATH"), "HOME="+t.TempDir())
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", vendorName}, {"config", "user.email", vendorMail}} {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = clone, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	cmd := exec.Command(bin, "reserve", "--holder", "--json")
	cmd.Dir, cmd.Env = clone, env
	answer, err := cmd.Output()
	if err != nil {
		t.Fatalf("reserve --holder --json: %v\n%s", err, answer)
	}
	if !strings.Contains(string(answer), `"source": "token"`) {
		t.Fatalf("the real verb did not answer from the token:\n%s", answer)
	}

	c := newSetupCase(t)
	out, code := c.script(append(setupRemote, "FAKE_HOLDER="+string(answer)), setupScript)
	if code != 0 {
		t.Fatalf("setup exited %d on the real verb's answer\n%s", code, out)
	}
	if got := c.git("config", "--local", "--get", "user.name"); got != tokenLogin {
		t.Errorf("user.name = %q from the real answer, want %q", got, tokenLogin)
	}
	if got := c.git("config", "--local", "--get", "user.email"); got != tokenNoReply {
		t.Errorf("user.email = %q from the real answer, want %q", got, tokenNoReply)
	}
	if strings.Contains(out, vendorMail) {
		t.Errorf("setup repeated the container's identity\n%s", out)
	}
}

// caSeen is every CA variable the token-identity verb, the setup's networked
// command run as abcd, saw in its environment, in the order it saw them.
func caSeen(t *testing.T, c *cloudCase) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(c.dir), "state", "holder-ca"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	return strings.Fields(string(b))
}

// The proxy's CA bundle a test names, and its exact bytes.
const proxyCA = "-----BEGIN CERTIFICATE-----\nMIIBproxyCAforTheHostedSession\n-----END CERTIFICATE-----\n"

// TestSetupHandsTheUserARootOnlyCABundle: in a hosted session the proxy's CA
// bundle sits under root's home, which the user abcd cannot read, so every
// networked command run as abcd failed its certificate check. The setup made
// the user, so root copies exactly that bundle, 0444 and root's own, into a
// directory only root can write, and points the same variables at it for the
// commands it runs as abcd and for the session's later shells, root's
// included. abcd never writes the copy, and verification is never turned off.
//
// The fake's root is the test's own uid: the fake id answers 0 for it.
func TestSetupHandsTheUserARootOnlyCABundle(t *testing.T) {
	c := newSetupCase(t)
	caDir := filepath.Join(filepath.Dir(c.dir), "share", "abcd", "cloud-ca")
	rootOnly := filepath.Join(filepath.Dir(c.dir), "root-home")
	if err := os.MkdirAll(rootOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(rootOnly, "proxy-ca.pem")
	if err := os.WriteFile(bundle, []byte(proxyCA), 0o600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(t.TempDir(), "session.env")
	env := append(append([]string{}, setupRemote...),
		"FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token"), "CLAUDE_ENV_FILE="+envFile,
		"FAKE_ROOT_ONLY="+rootOnly, "ABCD_CLOUD_CA_DIR="+caDir, "SSL_CERT_FILE="+bundle, "GIT_SSL_CAINFO="+bundle)

	var outs, logs []string
	for run := 1; run <= 2; run++ {
		c.resetCalls()
		out, code := c.script(env, setupScript)
		if code != 0 {
			t.Fatalf("setup run %d exited %d with the proxy's CA bundle readable only by root\n%s", run, code, out)
		}
		outs, logs = append(outs, out), append(logs, c.callLog())
	}
	if strings.Contains(logs[1], "chown 0:0 "+caDir+"/") {
		t.Errorf("the second setup rewrote a copy whose content had not changed:\n%s", logs[1])
	}
	if n, m := strings.Count(outs[0], "\n"), strings.Count(outs[1], "\n"); n != m {
		t.Errorf("the second setup printed %d lines, the first %d\n%s\n--\n%s", m, n, outs[0], outs[1])
	}
	if !strings.Contains(outs[0], "SSL_CERT_FILE") || !strings.Contains(outs[0], "GIT_SSL_CAINFO") {
		t.Errorf("setup does not say which CA variables it pointed at the user's copy\n%s", outs[0])
	}

	seen := caSeen(t, c)
	if len(seen) != 4 {
		t.Fatalf("the user's networked command saw CA variables %q over two runs, want SSL_CERT_FILE and GIT_SSL_CAINFO each run", seen)
	}
	copies := map[string]string{}
	for _, kv := range seen {
		name, path, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(path, rootOnly+string(filepath.Separator)) {
			t.Errorf("the user's command was handed %s=%s, a bundle only root can read", name, path)
			continue
		}
		if filepath.Dir(path) != caDir {
			t.Errorf("%s=%s is not in the directory only root can write, %s", name, path, caDir)
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != proxyCA {
			t.Errorf("%s=%s does not hold exactly the proxy's bundle: %q (%v)", name, path, b, err)
		}
		if st, err := os.Lstat(path); err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0o444 {
			t.Errorf("the copy %s is not a plain file of mode 0444 (%v, %v)", path, st.Mode(), err)
		} else if uid := st.Sys().(*syscall.Stat_t).Uid; int(uid) != os.Getuid() {
			t.Errorf("the copy %s belongs to uid %d, not the fake's root (%d)", path, uid, os.Getuid())
		}
		if prev, ok := copies[name]; ok && prev != path {
			t.Errorf("%s named %s on the first run and %s on the second", name, prev, path)
		}
		copies[name] = path
	}
	if st, err := os.Lstat(caDir); err != nil || !st.IsDir() || st.Mode().Perm() != 0o755 {
		t.Errorf("the copies' directory %s is not a real directory of mode 0755 (%v, %v)", caDir, st.Mode(), err)
	} else if uid := st.Sys().(*syscall.Stat_t).Uid; int(uid) != os.Getuid() {
		t.Errorf("the copies' directory belongs to uid %d, not the fake's root (%d)", uid, os.Getuid())
	}
	log := logs[0]
	for _, want := range []string{
		`(?m)^chown 0:0 ` + regexp.QuoteMeta(caDir) + `$`,
		`(?m)^chown 0:0 ` + regexp.QuoteMeta(caDir) + `/\.ca\.\S+$`,
	} {
		if !regexp.MustCompile(want).MatchString(log) {
			t.Errorf("root did not give the directory and the copy to root (%s):\n%s", want, log)
		}
	}
	for _, l := range strings.Split(logs[0]+logs[1], "\n") {
		if strings.HasPrefix(l, "as-abcd ") && strings.Contains(l, caDir) && !strings.HasPrefix(l, "as-abcd test -r ") {
			t.Errorf("abcd touched the copies' directory beyond reading: %s", l)
		}
	}

	b, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("the variables file was not written: %v", err)
	}
	for name, path := range copies {
		if n := strings.Count(string(b), "export "+name+"='"+path+"'\n"); n != 1 {
			t.Errorf("the variables file points %s at the user's copy %d times after two setups, want once:\n%s", name, n, b)
		}
	}
	if strings.Count(string(b), "\n") != 4 {
		t.Errorf("the variables file carries more than abcd's two and the two CA exports:\n%s", b)
	}
	if orig, err := os.ReadFile(bundle); err != nil || string(orig) != proxyCA {
		t.Errorf("the proxy's own bundle was changed: %q (%v)", orig, err)
	}
	for _, never := range []string{"NO_VERIFY", "sslVerify", "sslverify", "REJECT_UNAUTHORIZED", "--insecure", "/etc/ssl"} {
		if strings.Contains(string(b), never) || strings.Contains(logs[0]+logs[1], never) || strings.Contains(c.localConfig(), never) {
			t.Errorf("setup weakened TLS verification (%s):\n%s\n%s", never, b, logs[0]+logs[1])
		}
	}
}

// TestSetupLeavesAReadableCABundleAsItIs: a CA bundle the user can already
// read is not copied, not re-pointed and not touched.
func TestSetupLeavesAReadableCABundleAsItIs(t *testing.T) {
	c := newSetupCase(t)
	rootOnly := filepath.Join(filepath.Dir(c.dir), "root-home")
	shared := filepath.Join(filepath.Dir(c.dir), "etc-ssl")
	for _, d := range []string{rootOnly, shared} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	caDir := filepath.Join(filepath.Dir(c.dir), "share", "abcd", "cloud-ca")
	bundle := filepath.Join(shared, "ca.pem")
	if err := os.WriteFile(bundle, []byte(proxyCA), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(bundle)
	if err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(t.TempDir(), "session.env")
	env := append(append([]string{}, setupRemote...),
		"FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token"), "CLAUDE_ENV_FILE="+envFile,
		"FAKE_ROOT_ONLY="+rootOnly, "ABCD_CLOUD_CA_DIR="+caDir, "SSL_CERT_FILE="+bundle, "CURL_CA_BUNDLE="+bundle)
	out, code := c.script(env, setupScript)
	if code != 0 {
		t.Fatalf("setup exited %d\n%s", code, out)
	}
	seen := caSeen(t, c)
	want := []string{"SSL_CERT_FILE=" + bundle, "CURL_CA_BUNDLE=" + bundle}
	if strings.Join(seen, " ") != strings.Join(want, " ") {
		t.Errorf("the user's networked command saw %q, want the readable bundle unchanged, %q", seen, want)
	}
	b, _ := os.ReadFile(envFile)
	for _, name := range []string{"SSL_CERT_FILE", "CURL_CA_BUNDLE"} {
		if strings.Contains(string(b), name) {
			t.Errorf("a readable bundle was re-pointed for later shells:\n%s", b)
		}
	}
	if _, err := os.Stat(caDir); !os.IsNotExist(err) {
		t.Errorf("a copy was made of a bundle the user can read (%v)", err)
	}
	after, err := os.Stat(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("the readable bundle was touched: %v %v, was %v %v", after.Mode(), after.ModTime(), before.Mode(), before.ModTime())
	}
	if got, _ := os.ReadFile(bundle); string(got) != proxyCA {
		t.Errorf("the readable bundle's content changed: %q", got)
	}
}

// TestSetupRefusesASymlinkAtTheCACopy: root writes the copy, so a symlink
// standing where the copy goes is refused, never followed: the file it names
// is left as it was, and the setup says the copy could not be made.
func TestSetupRefusesASymlinkAtTheCACopy(t *testing.T) {
	c := newSetupCase(t)
	caDir := filepath.Join(filepath.Dir(c.dir), "share", "abcd", "cloud-ca")
	rootOnly := filepath.Join(filepath.Dir(c.dir), "root-home")
	for _, d := range []string{caDir, rootOnly} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(rootOnly, "proxy-ca.pem")
	if err := os.WriteFile(bundle, []byte(proxyCA), 0o600); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(filepath.Dir(c.dir), "victim")
	if err := os.WriteFile(victim, []byte("root's own file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(caDir, "SSL_CERT_FILE.pem")
	if err := os.Symlink(victim, dst); err != nil {
		t.Fatal(err)
	}
	out, code := c.script(append(append([]string{}, setupRemote...),
		"FAKE_HOLDER="+holderJSON(tokenLogin, tokenNoReply, "token"),
		"FAKE_ROOT_ONLY="+rootOnly, "ABCD_CLOUD_CA_DIR="+caDir, "SSL_CERT_FILE="+bundle), setupScript)
	if code == 0 {
		t.Errorf("setup exited 0 with a symlink where the CA copy goes\n%s", out)
	}
	if !strings.Contains(out, "NOT IN PLACE") || !strings.Contains(out, "a copy could not be made") {
		t.Errorf("setup does not say the copy could not be made\n%s", out)
	}
	if b, _ := os.ReadFile(victim); string(b) != "root's own file\n" {
		t.Errorf("the setup wrote through the symlink: %q", b)
	}
	if st, err := os.Lstat(dst); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symlink was replaced or removed (%v, %v)", st.Mode(), err)
	}
}
