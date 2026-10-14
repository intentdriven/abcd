package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shell installers verify a download's build provenance with `gh
// attestation verify` when gh is on PATH (iss-2610120617521209). A test that
// runs one must decide what gh is, or it runs whatever gh the machine has: on
// a developer's machine a signed-in gh that would ask GitHub about fixture
// bytes and refuse them, on a CI runner an unauthenticated one. So every
// bootstrap run in this package puts a stub first on PATH, the verifying one
// unless the case asks for another, and the gh-absent case builds a PATH with
// no gh on it at all.

// ghMode is how the stub answers `gh attestation verify <file> …`.
type ghMode int

const (
	ghVerified        ghMode = iota // exit 0, as a verified attestation does
	ghFails                         // exit 1 with gh's own failure line
	ghUnauthenticated               // exit 4, gh's "authentication required"
	ghNoSourceDigest                // a gh whose attestation verify predates --source-digest
)

// ghStub is a stub gh in its own directory, recording every verify call.
type ghStub struct {
	dir string
	log string
}

// ghCall is one recorded `gh attestation verify`: its arguments and the
// SHA-256 of the file it was asked about, taken when it was asked.
type ghCall struct {
	args string
	sha  string
}

// newGhStub writes a gh that answers `attestation verify --help` (the probe
// that the attestation command exists, and that it can bind a check to a
// commit) with its flag list, and a verify call as mode says.
func newGhStub(t *testing.T, mode ghMode) *ghStub {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(t.TempDir(), "gh-calls.log")
	verdict := "exit 0"
	switch mode {
	case ghFails:
		verdict = "echo 'Loaded digest sha256:x for file' >&2; echo 'Error: failed to fetch attestations from intentdriven/abcd: no attestations found for subject' >&2; exit 1"
	case ghUnauthenticated:
		verdict = "echo 'To get started with GitHub CLI, please run:  gh auth login' >&2; exit 4"
	}
	help := "printf '%s\\n' '      --signer-workflow string' '      --source-digest string' '      --source-ref string'"
	if mode == ghNoSourceDigest {
		help = "printf '%s\\n' '      --signer-workflow string'"
	}
	script := "#!/bin/sh\n" +
		"[ \"$1 $2 $3\" = 'attestation verify --help' ] && { " + help + "; exit 0; }\n" +
		"if [ \"$1 $2\" = 'attestation verify' ]; then\n" +
		"\tsum=$( (shasum -a 256 \"$3\" 2>/dev/null || sha256sum \"$3\") | cut -d' ' -f1)\n" +
		"\tprintf '%s\\t%s\\n' \"$sum\" \"$*\" >> '" + log + "'\n" +
		"\t" + verdict + "\n" +
		"fi\n" +
		"echo \"gh stub: unexpected call: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return &ghStub{dir: dir, log: log}
}

// verifyCalls reads the recorded verify calls.
func (g *ghStub) verifyCalls(t *testing.T) []ghCall {
	t.Helper()
	raw, err := os.ReadFile(g.log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var calls []ghCall
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if sha, args, ok := strings.Cut(line, "\t"); ok {
			calls = append(calls, ghCall{args: args, sha: sha})
		}
	}
	return calls
}

// pathWithout returns a PATH holding every command on base except the named
// ones, as symlinks in one directory, so a case can run on this machine's
// tools with gh absent, wherever gh is installed (/usr/bin on a Linux runner).
func pathWithout(t *testing.T, base string, names ...string) string {
	t.Helper()
	drop := map[string]bool{}
	for _, n := range names {
		drop[n] = true
	}
	dir := t.TempDir()
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
	return dir
}
