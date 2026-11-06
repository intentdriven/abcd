package site

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The served installer, run. The surface tests above hold its text to the
// other install forms; these run the committed template in a temp HOME with
// curl and gh stubbed on PATH, so its two security behaviours are proven by
// what it does rather than by what it says:
//
//   - iss-2610120617521209 (the product thinker's ruling of 2026-10-12): the
//     download is verified with `gh attestation verify` when gh is present,
//     and a failed verification installs nothing; without gh, or with a gh
//     that is not signed in, the checksum-verified binary installs with a
//     warning naming the hand check.
//   - iss-2610120601505029: a verified install records the newest-release mark
//     the bootstrap and abcd update keep, with the same atomic owner-only
//     write, and only ever raises it.
//   - a fetched binary whose --version names another release installs nothing.

const (
	installedTag = "v0.14.2"
	// releaseCommit is the commit the stub API resolves installedTag to; gh
	// must be held to it (--source-digest), because the release workflow's
	// signature alone passes any release's genuine build under any tag.
	releaseCommit = "0123456789abcdef0123456789abcdef01234567"
	ghPolicyFlags = "--hostname github.com --repo intentdriven/abcd --cert-identity https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main --source-ref refs/heads/main --deny-self-hosted-runners"
)

type ghBehaviour int

const (
	ghVerifies ghBehaviour = iota
	ghRefuses
	ghSignedOut
	ghAbsent
	// ghVerifiesCommitUnresolved: gh would verify, but the API does not
	// answer the release tag's commit, so the check cannot be bound to it.
	ghVerifiesCommitUnresolved
	// ghNoSourceDigest: a gh whose attestation verify predates
	// --source-digest, so it cannot bind the check to the release commit.
	ghNoSourceDigest
	// ghNoSourceRef: a gh whose attestation verify has --source-digest but
	// no --source-ref, so it cannot hold the check to the ref the release
	// was built from.
	ghNoSourceRef
	// ghNoCertIdentity and ghNoDenySelfHosted: a gh whose attestation verify
	// lacks one of the other two policy flags.
	ghNoCertIdentity
	ghNoDenySelfHosted
	// ghAbsentTagUnresolved: no gh, and GitHub does not answer which tag the
	// latest release names.
	ghAbsentTagUnresolved
)

// installerRun is one run of the template under a sandbox.
type installerRun struct {
	home, binDir string
	out          string
	err          error
	ghLog        string
	release      []byte // the binary the stub release serves
}

// runInstaller runs the installer as abcd site build renders it from the
// committed template (the script abcdev.app serves at /install.sh), with HOME,
// curl and gh sandboxed.
// The stub curl answers the two release downloads from a fixture directory by
// the URL's last segment; the release binary is a script that reports
// installedTag; prepare runs against the sandbox home before the install.
func runInstaller(t *testing.T, gh ghBehaviour, prepare func(home string)) installerRun {
	t.Helper()
	script := filepath.Join(t.TempDir(), "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, renderInstallScript(tmpl, fixtureStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	return runInstallForm(t, []string{"sh", script}, gh, prepare)
}

// runInstallForm runs argv, one form of the install command, in the sandbox
// runInstaller describes. The stub release serves the binary under every
// released OS name, so each form runs on either OS.
func runInstallForm(t *testing.T, argv []string, gh ghBehaviour, prepare func(home string)) installerRun {
	t.Helper()
	return runInstallOf(t, argv, gh, releaseBinary("abcd "+installedTag), prepare)
}

// releaseBinary is a stub of the released binary whose --version reports the
// given first line. It knows only --version, as abcd does: anything else is an
// unknown command, exit 2, so a form that ran `abcd version` would fail here as
// it fails for a reader (iss-2610120704269048).
func releaseBinary(firstLine string) []byte {
	return []byte("#!/bin/sh\nif [ \"$1\" = --version ]; then echo '" + firstLine + "'; echo '  install:   pinned'; exit 0; fi\n" +
		"echo \"Error: unknown command \\\"$1\\\" for \\\"abcd\\\"\" >&2\nexit 2\n")
}

// runInstallerOf runs the rendered installer with the stub release serving binary.
func runInstallerOf(t *testing.T, gh ghBehaviour, binary []byte) installerRun {
	t.Helper()
	return runInstallOf(t, []string{"sh", renderedInstaller(t)}, gh, binary, nil)
}

// runInstallOf is runInstallForm with the stub release serving binary.
func runInstallOf(t *testing.T, argv []string, gh ghBehaviour, binary []byte, prepare func(home string)) installerRun {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the installer runs on macOS and Linux only")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	r := installerRun{home: t.TempDir()}
	r.binDir = filepath.Join(r.home, ".local", "bin")
	if prepare != nil {
		prepare(r.home)
	}

	r.release = binary
	release := t.TempDir()
	sum := sha256.Sum256(r.release)
	manifest := ""
	for _, goos := range []string{"darwin", "linux"} {
		asset := "abcd-" + goos + "-" + runtime.GOARCH
		if err := os.WriteFile(filepath.Join(release, asset), r.release, 0o644); err != nil {
			t.Fatal(err)
		}
		manifest += hex.EncodeToString(sum[:]) + "  " + asset + "\n"
	}
	manifest += strings.Repeat("0", 64) + "  abcd-other-platform\n"
	if err := os.WriteFile(filepath.Join(release, "checksums.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	// The release also publishes the installer, which the documented command
	// fetches and pipes into sh (iss-2610240827243654).
	// The release publishes the template byte for byte (release.yml copies it),
	// so the sandbox serves exactly that, not the site's stamped copy.
	installer := []byte(readFile(t, filepath.Join(repoRoot(), "site-src", "install.sh.tmpl")))
	if err := os.WriteFile(filepath.Join(release, installScriptName), installer, 0o644); err != nil {
		t.Fatal(err)
	}

	stubs := t.TempDir()
	// The stub curl answers the latest-release redirect with installedTag's
	// page, the API's commit lookup for installedTag with releaseCommit (or a
	// failure, when the case says the API does not answer), and any other URL
	// from the release fixture by its last segment, except the installer, which
	// it writes to stdout as `curl -fsSL <url> | sh` reads it.
	commitAnswer := "printf '%s' '" + releaseCommit + "'"
	if gh == ghVerifiesCommitUnresolved {
		commitAnswer = "exit 22"
	}
	tagAnswer := "printf '%s' 'https://github.com/intentdriven/abcd/releases/tag/" + installedTag + "'"
	if gh == ghAbsentTagUnresolved {
		tagAnswer = "exit 22"
	}
	curl := "#!/bin/sh\nfor a in \"$@\"; do url=\"$a\"; done\n" +
		"case \"$url\" in\n" +
		"https://github.com/intentdriven/abcd/releases/latest) " + tagAnswer + "; exit 0 ;;\n" +
		"https://api.github.com/repos/intentdriven/abcd/commits/" + installedTag + ") " + commitAnswer + "; exit 0 ;;\n" +
		"https://api.github.com/*) exit 22 ;;\n" +
		"*/" + installScriptName + ") cat '" + release + "'/" + installScriptName + "; exit 0 ;;\n" +
		"esac\n" +
		"cp '" + release + "'/\"${url##*/}\" \"${url##*/}\"\n"
	if err := os.WriteFile(filepath.Join(stubs, "curl"), []byte(curl), 0o755); err != nil {
		t.Fatal(err)
	}
	r.ghLog = filepath.Join(t.TempDir(), "gh.log")
	if gh != ghAbsent && gh != ghAbsentTagUnresolved {
		verdict := "exit 0"
		switch gh {
		case ghRefuses:
			verdict = "echo 'Error: verifying with issuer \"sigstore.dev\": no matching attestation' >&2; exit 1"
		case ghSignedOut:
			verdict = "echo 'To get started with GitHub CLI, please run:  gh auth login' >&2; exit 4"
		}
		help := "printf '%s\\n' '      --cert-identity string' '      --deny-self-hosted-runners' '      --signer-workflow string' '      --source-digest string' '      --source-ref string'"
		switch gh {
		case ghNoSourceDigest:
			help = "printf '%s\\n' '      --signer-workflow string'"
		case ghNoSourceRef:
			help = "printf '%s\\n' '      --signer-workflow string' '      --source-digest string'"
		case ghNoCertIdentity:
			help = "printf '%s\\n' '      --deny-self-hosted-runners' '      --signer-workflow string' '      --source-digest string' '      --source-ref string'"
		case ghNoDenySelfHosted:
			help = "printf '%s\\n' '      --cert-identity string' '      --signer-workflow string' '      --source-digest string' '      --source-ref string'"
		}
		script := "#!/bin/sh\n" +
			"[ \"$1 $2 $3\" = 'attestation verify --help' ] && { " + help + "; exit 0; }\n" +
			"if [ \"$1 $2\" = 'attestation verify' ]; then\n" +
			"\tsum=$( (shasum -a 256 \"$3\" 2>/dev/null || sha256sum \"$3\") | cut -d' ' -f1)\n" +
			"\tprintf '%s\\t%s\\n' \"$sum\" \"$*\" >> '" + r.ghLog + "'\n" +
			"\t" + verdict + "\nfi\nexit 1\n"
		if err := os.WriteFile(filepath.Join(stubs, "gh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{
		"HOME=" + r.home,
		"PATH=" + stubs + string(os.PathListSeparator) + installerPathWithout(t, os.Getenv("PATH"), "gh", "curl"),
	}
	out, err := cmd.CombinedOutput()
	r.out, r.err = string(out), err
	return r
}

// installerPathWithout is PATH as one directory of symlinks to every command
// on base except the named ones, so gh and curl are only ever the stubs,
// wherever the machine keeps its own (/usr/bin on a Linux runner). A machine has
// thousands of commands, so the directory is made once for each base and list:
// the installers only run what it holds, and it is read-only so one cannot do
// otherwise.
func installerPathWithout(t *testing.T, base string, names ...string) string {
	t.Helper()
	key := base + "\x00" + strings.Join(names, "\x00")
	installerPaths.mu.Lock()
	defer installerPaths.mu.Unlock()
	if dir, ok := installerPaths.dirs[key]; ok {
		return dir
	}
	dir, err := os.MkdirTemp("", "installer-path-")
	if err != nil {
		t.Fatal(err)
	}
	drop := map[string]bool{}
	for _, n := range names {
		drop[n] = true
	}
	seen := map[string]bool{}
	for _, d := range filepath.SplitList(base) {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if drop[name] || seen[name] {
				continue
			}
			full := filepath.Join(d, name)
			fi, err := os.Stat(full)
			if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
				continue
			}
			seen[name] = true
			if err := os.Symlink(full, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	if installerPaths.dirs == nil {
		installerPaths.dirs = map[string]string{}
	}
	installerPaths.dirs[key] = dir
	return dir
}

// installerPaths holds the directories installerPathWithout made.
var installerPaths struct {
	mu   sync.Mutex
	dirs map[string]string
}

// removeInstallerPaths deletes them; TestMain calls it once the tests have run.
func removeInstallerPaths() {
	installerPaths.mu.Lock()
	defer installerPaths.mu.Unlock()
	for _, dir := range installerPaths.dirs {
		_ = os.Chmod(dir, 0o755)
		_ = os.RemoveAll(dir)
	}
	installerPaths.dirs = nil
}

func (r installerRun) installed(t *testing.T) bool {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(r.binDir, "abcd"))
	return err == nil && string(got) == string(r.release)
}

func (r installerRun) ghCalls(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(r.ghLog)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func (r installerRun) mark(t *testing.T) (string, bool) {
	t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(r.home, "release-mark"))
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func TestInstallerInstallsADownloadGhVerifies(t *testing.T) {
	r := runInstaller(t, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("a verified download installs: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the verified binary must be installed:\n%s", r.out)
	}
	calls := r.ghCalls(t)
	if len(calls) != 1 {
		t.Fatalf("gh attestation verify must run once, got %q", calls)
	}
	sum := sha256.Sum256(r.release)
	if !strings.HasPrefix(calls[0], hex.EncodeToString(sum[:])+"\t") {
		t.Errorf("gh must verify the downloaded binary: %q", calls[0])
	}
	if !strings.Contains(calls[0], ghPolicyFlags) {
		t.Errorf("gh must be held to the release workflow's policy %q: %q", ghPolicyFlags, calls[0])
	}
	if !strings.Contains(calls[0], "--source-digest "+releaseCommit) {
		t.Errorf("gh must be bound to the latest release's commit %s: %q", releaseCommit, calls[0])
	}
	if strings.Contains(r.out, "WARNING") {
		t.Errorf("a verified install carries no warning:\n%s", r.out)
	}
}

// TestInstallerRefusesWhenTheReleaseCommitCannotBeResolved: with a gh that
// can verify, a latest release whose commit the API does not answer cannot
// have its provenance bound to it, so nothing is installed.
func TestInstallerRefusesWhenTheReleaseCommitCannotBeResolved(t *testing.T) {
	r := runInstaller(t, ghVerifiesCommitUnresolved, nil)
	if r.err == nil {
		t.Fatalf("an unresolvable release commit must refuse when gh can verify:\n%s", r.out)
	}
	if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
		t.Errorf("an unbound check must install nothing: %v", err)
	}
	if calls := r.ghCalls(t); len(calls) != 0 {
		t.Errorf("gh must not verify without the release's commit: %q", calls)
	}
	for _, want := range []string{"commit", "Nothing is installed"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the refusal must say %q:\n%s", want, r.out)
		}
	}
}

// TestInstallerWarnsWhenGhCannotBindTheCommit: a gh without --source-digest
// counts as one that cannot verify.
func TestInstallerWarnsWhenGhCannotBindTheCommit(t *testing.T) {
	r := runInstaller(t, ghNoSourceDigest, nil)
	if r.err != nil {
		t.Fatalf("a gh too old to bind the commit counts as no gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	if calls := r.ghCalls(t); len(calls) != 0 {
		t.Errorf("a gh that cannot bind the commit must not be asked to verify: %q", calls)
	}
	assertInstallerWarning(t, r.out, "--source-digest")
}

// TestInstallerWarnsWhenGhCannotPinTheSourceRef: a gh without --source-ref
// counts as one that cannot verify.
func TestInstallerWarnsWhenGhCannotPinTheSourceRef(t *testing.T) {
	r := runInstaller(t, ghNoSourceRef, nil)
	if r.err != nil {
		t.Fatalf("a gh that cannot pin the source ref counts as no gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	if calls := r.ghCalls(t); len(calls) != 0 {
		t.Errorf("a gh that cannot pin the source ref must not be asked to verify: %q", calls)
	}
	assertInstallerWarning(t, r.out, "--source-ref")
}

// TestInstallerWarnsWhenGhLacksAPolicyFlag: a gh without --cert-identity or
// --deny-self-hosted-runners counts as one that cannot verify, rather than
// failing part-way through the check.
func TestInstallerWarnsWhenGhLacksAPolicyFlag(t *testing.T) {
	for gh, flag := range map[ghBehaviour]string{ghNoCertIdentity: "--cert-identity", ghNoDenySelfHosted: "--deny-self-hosted-runners"} {
		t.Run(flag, func(t *testing.T) {
			r := runInstaller(t, gh, nil)
			if r.err != nil {
				t.Fatalf("a gh without %s counts as no gh: %v\n%s", flag, r.err, r.out)
			}
			if !r.installed(t) {
				t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
			}
			if calls := r.ghCalls(t); len(calls) != 0 {
				t.Errorf("a gh without %s must not be asked to verify: %q", flag, calls)
			}
			assertInstallerWarning(t, r.out, flag)
		})
	}
}

func TestInstallerRefusesADownloadGhCannotVerify(t *testing.T) {
	r := runInstaller(t, ghRefuses, nil)
	if r.err == nil {
		t.Fatalf("a failed verification must refuse:\n%s", r.out)
	}
	if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
		t.Errorf("a failed verification must install nothing: %v", err)
	}
	if _, ok := r.mark(t); ok {
		t.Error("a refused install records no release mark")
	}
	for _, want := range []string{"build provenance", "no matching attestation", "Nothing is installed"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the refusal must say %q:\n%s", want, r.out)
		}
	}
}

func TestInstallerWarnsWithoutGh(t *testing.T) {
	r := runInstaller(t, ghAbsent, nil)
	if r.err != nil {
		t.Fatalf("abcd does not require gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	assertInstallerWarning(t, r.out, "gh is not installed")
}

func TestInstallerWarnsWhenGhIsNotSignedIn(t *testing.T) {
	r := runInstaller(t, ghSignedOut, nil)
	if r.err != nil {
		t.Fatalf("a gh that is not signed in counts as no gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	assertInstallerWarning(t, r.out, "not signed in")
}

func assertInstallerWarning(t *testing.T, out, why string) {
	t.Helper()
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"WARNING", "not checked against the signed build record", why, "gh attestation verify", "--repo intentdriven/abcd --cert-identity https://github.com/intentdriven/abcd/.github/workflows/release.yml@refs/heads/main --source-ref refs/heads/main --deny-self-hosted-runners"} {
		if !strings.Contains(flat, want) {
			t.Errorf("the warning must say %q:\n%s", want, out)
		}
	}
}

// TestInstallerEndsWithTheInstalledVersion: the install's last line is the
// version the installed binary reports through --version, the one way abcd
// prints it, so an install that succeeded ends in success rather than in an
// unknown-command error (iss-2610120704269048).
func TestInstallerEndsWithTheInstalledVersion(t *testing.T) {
	r := runInstaller(t, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("a verified install must exit 0: %v\n%s", r.err, r.out)
	}
	lines := strings.Split(strings.TrimSpace(r.out), "\n")
	if last := lines[len(lines)-1]; strings.TrimSpace(last) != "install:   pinned" || !strings.Contains(r.out, "abcd "+installedTag+"\n") {
		t.Errorf("the install must end with the installed binary's --version report:\n%s", r.out)
	}
	if strings.Contains(r.out, "unknown command") {
		t.Errorf("the install ran a command abcd does not have:\n%s", r.out)
	}
}

// --- the binary held to the release it came from ---

// assertInstalledNothing holds a refused install to leaving no trace: no binary,
// no record of it and no release mark.
func assertInstalledNothing(t *testing.T, r installerRun) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
		t.Errorf("a refused install must install nothing: %v", err)
	}
	if _, err := os.Lstat(abcdhome.Path(r.home, "path-entry")); !os.IsNotExist(err) {
		t.Errorf("a refused install records no path-entry: %v", err)
	}
	if _, ok := r.mark(t); ok {
		t.Error("a refused install records no release mark")
	}
}

// TestInstallerRefusesABinaryReportingAnotherRelease: the fetched binary's own
// --version must name the tag the latest release names, whether or not gh could
// check its provenance.
func TestInstallerRefusesABinaryReportingAnotherRelease(t *testing.T) {
	const older = "v0.13.4"
	for name, gh := range map[string]ghBehaviour{"gh verifies": ghVerifies, "gh signed out": ghSignedOut, "no gh": ghAbsent} {
		t.Run(name, func(t *testing.T) {
			r := runInstallerOf(t, gh, releaseBinary("abcd "+older))
			if r.err == nil {
				t.Fatalf("a binary reporting %s under %s must refuse:\n%s", older, installedTag, r.out)
			}
			assertInstalledNothing(t, r)
			for _, want := range []string{older, installedTag, "Nothing is installed"} {
				if !strings.Contains(r.out, want) {
					t.Errorf("the refusal must say %q:\n%s", want, r.out)
				}
			}
		})
	}
}

// TestInstallerRefusesABinaryReportingNoRelease: a build that names no release
// (an unstamped one, or one whose first line is not abcd's) is not the latest
// release's, and whatever else it printed is not repeated to the reader.
func TestInstallerRefusesABinaryReportingNoRelease(t *testing.T) {
	for _, line := range []string{"abcd (dev build)", "abcd v0.14.2-with \\033 escape", "something else"} {
		t.Run(line, func(t *testing.T) {
			r := runInstallerOf(t, ghVerifies, releaseBinary(line))
			if r.err == nil {
				t.Fatalf("a binary reporting %q must refuse:\n%s", line, r.out)
			}
			assertInstalledNothing(t, r)
			if !strings.Contains(r.out, "Nothing is installed") {
				t.Errorf("the refusal must say so:\n%s", r.out)
			}
			if strings.Contains(r.out, line) {
				t.Errorf("the refusal repeats what the binary printed:\n%s", r.out)
			}
		})
	}
}

// TestInstallerRefusesABinaryThatFailsToReport: a binary that exits with an
// error at --version reports no release.
func TestInstallerRefusesABinaryThatFailsToReport(t *testing.T) {
	r := runInstallerOf(t, ghAbsent, []byte("#!/bin/sh\nexit 3\n"))
	if r.err == nil {
		t.Fatalf("a binary that cannot report its version must refuse:\n%s", r.out)
	}
	assertInstalledNothing(t, r)
}

// homeStoppedBinary is a stub of the released binary as abcd behaves while its old
// home stands: --version says the stop on stderr, prints nothing on stdout and
// exits 1, for any command, before the command runs.
func homeStoppedBinary(tag string) []byte {
	return []byte("#!/bin/sh\nif [ -e \"$HOME/.abcd\" ]; then echo 'abcd: the old folder still stands' >&2; exit 1; fi\n" +
		"echo 'abcd " + tag + "'\n")
}

// oldHomeShapes are the two homes abcd stops in: the old folder alone, and the
// old folder beside the current one.
var oldHomeShapes = map[string]func(t *testing.T, home string){
	"the old folder": func(t *testing.T, home string) {
		t.Helper()
		if err := os.Mkdir(filepath.Join(home, ".abcd"), 0o700); err != nil {
			t.Fatal(err)
		}
	},
	"both folders": func(t *testing.T, home string) {
		t.Helper()
		for _, d := range []string{filepath.Join(home, ".abcd"), abcdhome.Path(home)} {
			if err := os.Mkdir(d, 0o700); err != nil {
				t.Fatal(err)
			}
		}
	},
}

// TestInstallerComparesTheReleaseWhateverTheHomeHolds: abcd's stop for an old home
// comes before --version, and says nothing about which release the binary is. The
// comparison reads the binary under a home holding neither folder, so the latest
// release still installs, and the install's closing report shows the stop.
func TestInstallerComparesTheReleaseWhateverTheHomeHolds(t *testing.T) {
	for name, shape := range oldHomeShapes {
		t.Run(name, func(t *testing.T) {
			r := runInstallOf(t, []string{"sh", renderedInstaller(t)}, ghVerifies, homeStoppedBinary(installedTag), func(home string) { shape(t, home) })
			if !r.installed(t) {
				t.Fatalf("the latest release must install whatever the home holds:\n%s", r.out)
			}
			if strings.Contains(r.out, "Nothing is installed") {
				t.Errorf("the comparison read the stop as a verdict on the release:\n%s", r.out)
			}
			if !strings.Contains(r.out, "the old folder still stands") {
				t.Errorf("the install's closing report must show the stop:\n%s", r.out)
			}
		})
	}
}

// TestInstallerRefusesAnotherReleaseWhateverTheHomeHolds: reading the binary under
// another home keeps the comparison in force where abcd's stop stands.
func TestInstallerRefusesAnotherReleaseWhateverTheHomeHolds(t *testing.T) {
	for name, shape := range oldHomeShapes {
		t.Run(name, func(t *testing.T) {
			r := runInstallOf(t, []string{"sh", renderedInstaller(t)}, ghVerifies, homeStoppedBinary("v0.13.4"), func(home string) { shape(t, home) })
			if r.err == nil {
				t.Fatalf("a binary reporting another release must refuse:\n%s", r.out)
			}
			if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
				t.Errorf("a refused install must install nothing: %v", err)
			}
			if !strings.Contains(r.out, "v0.13.4") {
				t.Errorf("the refusal must name what the binary reported:\n%s", r.out)
			}
		})
	}
}

// TestInstallerLeavesTheHomeUntouchedByTheComparison: the binary being compared
// is not installed yet, so its run sees the temporary directory as its home.
func TestInstallerLeavesTheHomeUntouchedByTheComparison(t *testing.T) {
	seen := filepath.Join(t.TempDir(), "homes")
	binary := []byte("#!/bin/sh\ncase \"$0\" in ./*) printf '%s\\n' \"$HOME\" > '" + seen + "' ;; esac\necho 'abcd " + installedTag + "'\n")
	r := runInstallerOf(t, ghVerifies, binary)
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.out)
	}
	raw, err := os.ReadFile(seen)
	if err != nil {
		t.Fatalf("the comparison did not run the binary: %v\n%s", err, r.out)
	}
	if got := strings.TrimSpace(string(raw)); got == r.home || !filepath.IsAbs(got) {
		t.Errorf("the comparison ran under home %q, want an absolute path other than the person's %q", got, r.home)
	}
}

// TestInstallerRunsNoBinaryItsProvenanceRefused: the version comes from running
// the fetched binary, so that waits until its provenance has passed.
func TestInstallerRunsNoBinaryItsProvenanceRefused(t *testing.T) {
	ran := filepath.Join(t.TempDir(), "ran")
	r := runInstallerOf(t, ghRefuses, []byte("#!/bin/sh\n: > '"+ran+"'\necho 'abcd "+installedTag+"'\n"))
	if r.err == nil {
		t.Fatalf("a failed verification must refuse:\n%s", r.out)
	}
	if _, err := os.Lstat(ran); !os.IsNotExist(err) {
		t.Errorf("the binary ran before its provenance passed: %v\n%s", err, r.out)
	}
}

// TestInstallerSaysWhenItHasNoTagToCompare: with no gh and no answer on the tag,
// the checksum stands alone as it did, and a warning says the release was not
// compared. With a gh that can verify, the same silence already refuses
// (TestInstallerRefusesWhenTheReleaseCommitCannotBeResolved).
func TestInstallerSaysWhenItHasNoTagToCompare(t *testing.T) {
	r := runInstallerOf(t, ghAbsentTagUnresolved, releaseBinary("abcd "+installedTag))
	if r.err != nil {
		t.Fatalf("no answer on the tag must not stop a checksum-only install: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	for _, want := range []string{"WARNING", "not compared with the latest release"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the warning must say %q:\n%s", want, r.out)
		}
	}
}

// TestInstallerSaysWhenTheBinaryCannotRunBeforeInstall: a temporary directory
// mounted noexec gives exit 126, which is about the mount and says nothing of the
// binary, so the install goes on and says what it did not compare. The stub gives
// that exit only when run from the temporary directory, as such a mount would.
func TestInstallerSaysWhenTheBinaryCannotRunBeforeInstall(t *testing.T) {
	binary := []byte("#!/bin/sh\ncase \"$0\" in ./*) exit 126 ;; esac\necho 'abcd " + installedTag + "'\n")
	r := runInstallerOf(t, ghVerifies, binary)
	if r.err != nil {
		t.Fatalf("a temporary directory that cannot run binaries must not stop the install: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the binary must be installed:\n%s", r.out)
	}
	for _, want := range []string{"WARNING", "not compared with the latest release"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the warning must say %q:\n%s", want, r.out)
		}
	}
}

// --- the newest-release mark (iss-2610120601505029) ---

func TestInstallerRecordsTheReleaseMark(t *testing.T) {
	for _, gh := range []ghBehaviour{ghVerifies, ghAbsent} {
		r := runInstaller(t, gh, nil)
		if r.err != nil {
			t.Fatalf("install failed: %v\n%s", r.err, r.out)
		}
		got, ok := r.mark(t)
		if !ok || got != "release_tag="+installedTag+"\n" {
			t.Fatalf("a fresh install must record the installed release as the mark; got %q (%v)\n%s", got, ok, r.out)
		}
		fi, err := os.Stat(abcdhome.Path(r.home, "release-mark"))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("the mark must be mode 0600: %v %v", fi.Mode(), err)
		}
		if di, err := os.Stat(abcdhome.Path(r.home)); err != nil || di.Mode().Perm() != 0o700 {
			t.Errorf("the home it lives in must be mode 0700: %v %v", di.Mode(), err)
		}
		left, _ := filepath.Glob(abcdhome.Path(r.home, ".release-mark.*"))
		if len(left) != 0 {
			t.Errorf("the atomic write must leave no temp file: %v", left)
		}
	}
}

func TestInstallerOnlyRaisesTheMark(t *testing.T) {
	for _, tc := range []struct{ had, want string }{
		{"v0.13.4", installedTag}, // older: raised
		{"v0.15.0", "v0.15.0"},    // newer: kept
	} {
		r := runInstaller(t, ghVerifies, func(home string) {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abcdhome.Path(home, "release-mark"), []byte("release_tag="+tc.had+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		})
		if r.err != nil {
			t.Fatalf("install failed: %v\n%s", r.err, r.out)
		}
		if got, _ := r.mark(t); got != "release_tag="+tc.want+"\n" {
			t.Errorf("with a mark of %s the install must leave %s; got %q", tc.had, tc.want, got)
		}
	}
}

// TestInstallerRefusesASymlinkedHome: abcd refuses a ~/.abcd.noindex that is
// a symlink rather than following it, and the installer, which records the
// owned PATH copy in ~/.abcd.noindex/path-entry, refuses one before it fetches
// anything, as the one-liners do: nothing is installed, nothing is written
// through the link, and the refusal says why. Every reader ignores a record
// behind the link (iss-2609281017573862), so an install that went ahead would
// report a finished install no hook recognises.
func TestInstallerRefusesASymlinkedHome(t *testing.T) {
	elsewhere := t.TempDir()
	r := runInstaller(t, ghVerifies, func(home string) {
		if err := os.Symlink(elsewhere, abcdhome.Path(home)); err != nil {
			t.Fatal(err)
		}
	})
	if r.err == nil {
		t.Fatalf("a symlinked ~/.abcd.noindex must refuse the install:\n%s", r.out)
	}
	if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
		t.Errorf("a refused install must install nothing: %v", err)
	}
	if calls := r.ghCalls(t); len(calls) != 0 {
		t.Errorf("the refusal must come before anything is fetched or verified: %q", calls)
	}
	for _, f := range []string{"release-mark", "path-entry"} {
		if _, err := os.Lstat(filepath.Join(elsewhere, f)); !os.IsNotExist(err) {
			t.Errorf("%s was written through the symlink: %v", f, err)
		}
	}
	for _, want := range []string{"symlink", "Nothing is installed"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the refusal must say %q:\n%s", want, r.out)
		}
	}
}

// --- the owned PATH copy's record (iss-2610240827243654) ---

// TestInstallerRecordsThePathEntry: a verified install records the binary it
// installed in ~/.abcd.noindex/path-entry, as the one-liners do — the path it
// copied the binary to and the SHA-256 the release manifest vouched for — so
// the hooks and ahoy recognise the copy as abcd's own, whichever form
// installed it. A --bin-dir install records that directory's copy.
func TestInstallerRecordsThePathEntry(t *testing.T) {
	for _, gh := range []ghBehaviour{ghVerifies, ghAbsent} {
		r := runInstaller(t, gh, nil)
		if r.err != nil {
			t.Fatalf("install failed: %v\n%s", r.err, r.out)
		}
		sum := sha256.Sum256(r.release)
		raw, err := os.ReadFile(abcdhome.Path(r.home, "path-entry"))
		if err != nil {
			t.Fatalf("a verified install must record ~/.abcd.noindex/path-entry: %v\n%s", err, r.out)
		}
		want := "path=" + filepath.Join(physical(t, r.binDir), "abcd") + "\nbinary_sha256=" + hex.EncodeToString(sum[:]) + "\n"
		if string(raw) != want {
			t.Errorf("path-entry must hold exactly the installed path and its digest:\ngot  %q\nwant %q", raw, want)
		}
		fi, err := os.Stat(abcdhome.Path(r.home, "path-entry"))
		if err != nil || fi.Mode().Perm()&0o022 != 0 {
			t.Errorf("path-entry must not be group- or world-writable, or every reader ignores it: %v %v", fi.Mode(), err)
		}
	}
}

// physical resolves dir's symlinks, as the installer's `cd -P && pwd -P` does,
// so an expectation names the path the hooks will compare (t.TempDir sits
// under a symlinked /var on macOS).
func physical(t *testing.T, dir string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// renderedInstaller writes the rendered install script to a temporary file.
func renderedInstaller(t *testing.T) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, renderInstallScript(tmpl, fixtureStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// TestInstallerRecordsAPrivatePathEntryUnderALooseUmask: the hooks refuse a
// group-writable record, so the installer writes path-entry 0600 whatever the
// umask, rather than leaving a record its own readers ignore.
func TestInstallerRecordsAPrivatePathEntryUnderALooseUmask(t *testing.T) {
	script := renderedInstaller(t)
	r := runInstallForm(t, []string{"sh", "-c", `umask 002 && exec sh "$0"`, script}, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.out)
	}
	fi, err := os.Stat(abcdhome.Path(r.home, "path-entry"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("path-entry must be 0600 under umask 002, or the hooks refuse it: %v %v", fi.Mode(), err)
	}
}

// TestInstallerRecordsThePhysicalBinDir: the hooks match the recorded path
// against `command -v abcd`, so a --bin-dir given with a trailing slash
// through a symlinked directory is recorded as the directory's physical path.
func TestInstallerRecordsThePhysicalBinDir(t *testing.T) {
	script := renderedInstaller(t)
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	r := runInstallForm(t, []string{"sh", script, "--bin-dir", link + "/"}, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.out)
	}
	raw, err := os.ReadFile(abcdhome.Path(r.home, "path-entry"))
	want := "path=" + filepath.Join(physical(t, real), "abcd") + "\n"
	if err != nil || !strings.HasPrefix(string(raw), want) {
		t.Errorf("path-entry must record the physical directory; got %q, want prefix %q (%v)", raw, want, err)
	}
}

// TestInstallerRecordsTheBinDirInThePathEntry: --bin-dir moves the record
// with the binary.
func TestInstallerRecordsTheBinDirInThePathEntry(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the installer runs on macOS and Linux only")
	}
	script := filepath.Join(t.TempDir(), "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, renderInstallScript(tmpl, fixtureStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "elsewhere")
	r := runInstallForm(t, []string{"sh", script, "--bin-dir", dir}, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.out)
	}
	raw, err := os.ReadFile(abcdhome.Path(r.home, "path-entry"))
	if err != nil || !strings.HasPrefix(string(raw), "path="+filepath.Join(physical(t, dir), "abcd")+"\n") {
		t.Errorf("a --bin-dir install must record that directory's copy; got %q (%v)\n%s", raw, err, r.out)
	}
}
