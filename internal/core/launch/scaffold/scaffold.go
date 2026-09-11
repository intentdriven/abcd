package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxWorkflowBytes caps a guarded read of an existing scaffolded file. A
// workflow or runbook is a short text file; a larger one is not one we wrote, and
// the drift check refuses it rather than streaming it unbounded.
const maxWorkflowBytes = 1 << 20

// ErrScaffoldBlocked is returned when a scaffold run refuses because an existing
// file differs from the machinery and --confirm was not given.
var ErrScaffoldBlocked = errors.New("scaffold refused: an existing file was hand-edited (pass --confirm to overwrite)")

// FileStatus is one scaffolded file's disposition in the report. Every value
// means what it says on disk: "written" is written, never merely planned.
type FileStatus string

const (
	// StatusWritten — the file was actually written (created, or overwritten under
	// confirm). Only set AFTER the write succeeds.
	StatusWritten FileStatus = "written"
	// StatusCurrent — the file already matches the machinery; nothing was written.
	StatusCurrent FileStatus = "current"
	// StatusRefused — the file exists and differs, and --confirm was not given, so
	// it was left untouched.
	StatusRefused FileStatus = "refused"
	// StatusSkipped — the file WOULD have been written, but the run refused (a
	// sibling was hand-edited) or a write faulted first, so it was NOT written. The
	// scaffold is all-or-nothing, so this reports honestly that nothing landed.
	StatusSkipped FileStatus = "skipped"
)

// disposition is the pre-write classification of a target: what it is on disk,
// independent of --confirm. It never claims a write happened.
type disposition int

const (
	dispAbsent  disposition = iota // no file → will be created
	dispCurrent                    // byte-identical to the machinery → no-op
	dispDiffers                    // present and different → refuse or (with --confirm) overwrite
)

// FileOutcome is the per-file result of a scaffold run.
type FileOutcome struct {
	Path   string     `json:"path"`
	Status FileStatus `json:"status"`
	// Detail explains a refusal (why the file was left alone).
	Detail string `json:"detail,omitempty"`
}

// Report is the outcome of a scaffold run.
type Report struct {
	Substitutions Substitutions `json:"-"`
	DefaultBranch string        `json:"default_branch"`
	// GoVersion is what the scaffolded workflows will RESOLVE, not a value written
	// into them: they point setup-go at go.mod, so this reports the go directive
	// the run read. It is reported because an adopter should see which toolchain
	// their release lane is about to use, and see it before the first tag.
	GoVersion string        `json:"go_version"`
	Files     []FileOutcome `json:"files"`
	Wrote     int           `json:"wrote"`
	Refused   int           `json:"refused"`
	// NoOp is true when every file was already current (the idempotent re-run).
	NoOp bool `json:"no_op"`
}

// Request is the input to a scaffold run.
type Request struct {
	RepoRoot string
	// Confirm overwrites a file that exists and differs from the machinery (the
	// transparent-confirm on a hand-edited workflow). Absent, such a file is
	// refused and left untouched, and the run reports ErrScaffoldBlocked.
	Confirm bool
}

// Scaffold writes the changelog-driven release machinery into RepoRoot: a
// generic (bare-repo) release.yml, auto-release.yml, and the adr-37 runbook, each
// wired to the repo's own default branch. The Go toolchain is not wired in: the
// workflows point setup-go at the repo's go.mod, so they follow its go directive
// with no re-scaffold. It is idempotent and fail-safe:
//
//   - a file absent on disk is written;
//   - a file byte-identical to the machinery is a no-op (StatusCurrent);
//   - a file that exists and DIFFERS (hand-edited, or a stale scaffold) is REFUSED
//     and left untouched unless Confirm is set, in which case it is overwritten.
//
// A run that refuses any file returns ErrScaffoldBlocked with the report, so the
// caller can render exactly what was and was not touched — no partial half-write.
func Scaffold(req Request) (Report, error) {
	branch, goVersion := DeriveRepoFacts(req.RepoRoot)
	subs := BareSubstitutions(branch)
	rendered, err := Render(subs)
	if err != nil {
		return Report{}, err
	}

	report := Report{Substitutions: subs, DefaultBranch: branch, GoVersion: goVersion}
	planned := []struct {
		rel  string
		data []byte
	}{
		{ReleaseYMLPath, rendered.ReleaseYML},
		{AutoReleaseYMLPath, rendered.AutoReleaseYML},
		{RunbookPath, rendered.Runbook},
	}

	// First pass: classify every file WITHOUT writing. A refusal on any file with
	// Confirm unset aborts the whole run before a single write, so the scaffold is
	// all-or-nothing rather than half-applied. Nothing is marked "written" here —
	// a planned write is only tentative until the second pass commits it, so the
	// report never claims a file landed that did not (the StatusWritten contract).
	outcomes := make([]FileOutcome, len(planned))
	writeNeeded := make([]bool, len(planned))
	overwrite := make([]bool, len(planned))
	refused := 0
	for i, p := range planned {
		abs := filepath.Join(req.RepoRoot, filepath.FromSlash(p.rel))
		disp, detail := classify(abs, p.data)
		switch disp {
		case dispCurrent:
			outcomes[i] = FileOutcome{Path: p.rel, Status: StatusCurrent}
		case dispAbsent:
			writeNeeded[i] = true
			outcomes[i] = FileOutcome{Path: p.rel, Status: StatusSkipped} // provisional until written
		case dispDiffers:
			if req.Confirm {
				writeNeeded[i] = true
				overwrite[i] = true
				outcomes[i] = FileOutcome{Path: p.rel, Status: StatusSkipped} // provisional until written
			} else {
				refused++
				outcomes[i] = FileOutcome{Path: p.rel, Status: StatusRefused, Detail: detail}
			}
		}
	}

	if refused > 0 {
		// Abort all-or-nothing: nothing is written. Every file that WOULD have been
		// written stays StatusSkipped with a reason, so the per-file report and the
		// wrote/refused counters match what is actually on disk.
		for i := range outcomes {
			if writeNeeded[i] {
				outcomes[i].Detail = "not written: the run refused because another file was hand-edited (all-or-nothing)"
			}
		}
		report.Files = outcomes
		report.Refused = refused
		return report, ErrScaffoldBlocked
	}

	// Second pass: commit the writes. Every file that reaches here is either a
	// create or a confirmed overwrite; a file becomes StatusWritten only once its
	// write actually succeeds.
	wrote := 0
	for i, p := range planned {
		if !writeNeeded[i] {
			continue
		}
		abs := filepath.Join(req.RepoRoot, filepath.FromSlash(p.rel))
		if err := fsutil.WriteFileAtomicPreserveMode(abs, p.data); err != nil {
			// The atomic writer never leaves a half-written file; the files written
			// before this fault are marked written, this one and any later planned
			// file stay StatusSkipped (not written), so the report matches disk.
			outcomes[i].Detail = "not written: a write faulted on this file"
			report.Files = outcomes
			report.Wrote = wrote
			return report, fmt.Errorf("scaffold: write %s: %w", p.rel, err)
		}
		outcomes[i].Status = StatusWritten
		if overwrite[i] {
			outcomes[i].Detail = "overwritten under --confirm"
		}
		wrote++
	}

	report.Files = outcomes
	report.Wrote = wrote
	report.NoOp = wrote == 0
	return report, nil
}

// classify reports whether abs is absent (→ dispAbsent, write it), byte-equal to
// want (→ dispCurrent, no-op), or present-and-different (→ dispDiffers, refuse or
// overwrite under --confirm). A non-regular leaf (symlink/FIFO/device) or an
// unreadable existing file is treated as dispDiffers — the scaffold never writes
// through a symlink silently and never assumes an unreadable file is safe to
// clobber. The string is a refusal reason, set only for dispDiffers.
func classify(abs string, want []byte) (disposition, string) {
	got, err := fsutil.ReadGuarded(abs, maxWorkflowBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return dispAbsent, ""
		}
		// A symlink leaf fails the O_NOFOLLOW open with ELOOP before ReadGuarded
		// can classify it as non-regular, so fold it in here — otherwise the
		// symlink case (named first in this function's contract) falls through to
		// the generic branch below and leaks a developer-identity path.
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
			return dispDiffers, "existing path is not a regular file (a symlink or non-regular leaf is never written through)"
		}
		if errors.Is(err, fsutil.ErrTooBig) {
			return dispDiffers, "existing file exceeds the size cap; this is not machinery abcd wrote"
		}
		// Path-free: this reason reaches the --dry-run report and its --json, a
		// success envelope the CLI's error-path scrubber never sees (iss-81).
		return dispDiffers, "existing file is unreadable: " + pathFreeReason(err)
	}
	if string(got) == string(want) {
		return dispCurrent, ""
	}
	return dispDiffers, "existing file differs from the current machinery (hand-edited or stale)"
}

// pathFreeReason renders a filesystem error without the absolute path it was
// raised against — the scaffold report and its --json must never carry a
// developer-identity path (iss-81).
func pathFreeReason(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Op + ": " + pe.Err.Error()
	}
	return err.Error()
}

// goVersionRe matches a major.minor[.patch] Go version — the only shape allowed
// into the rendered `go-version:` value, so a crafted go.mod cannot inject YAML.
// The patch is admitted so a repo pinning `go 1.25.6` scaffolds that exact
// toolchain rather than a floating minor that setup-go resolves on release day
// (iss-386); the allowlist stays injection-safe.
var goVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)

// goModVersionRe extracts the `go X.Y[.Z]` directive from go.mod, keeping the
// patch when the module declares one.
var goModVersionRe = regexp.MustCompile(`(?m)^go[ \t]+([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)

// branchNameRe is the injection-safe allowlist for a default-branch name written
// into the workflow YAML: git ref characters only, no whitespace or YAML
// metacharacters. A name outside it falls back to the safe default.
var branchNameRe = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// defaultGoVersion is the fallback when go.mod cannot be read or its version is
// not a clean major.minor.
const defaultGoVersion = "1.25"

// defaultBranch is the fallback default branch when the repo does not name one.
const defaultBranch = "main"

// DeriveRepoFacts reads the target repo's own default branch and Go version, both
// validated against an injection-safe allowlist. Anything malformed or absent
// falls back to a safe default, so a hostile ref name can never inject workflow
// content.
//
// Only the branch reaches the rendered YAML. The Go version is reported to the
// adopter — it is the toolchain their scaffolded workflows will resolve, since
// those point setup-go at go.mod — and is validated on the same terms anyway,
// because a report line is a place a hostile go.mod could otherwise write.
func DeriveRepoFacts(repoRoot string) (branch, goVersion string) {
	return deriveBranch(repoRoot), deriveGoVersion(repoRoot)
}

func deriveGoVersion(repoRoot string) string {
	data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, "go.mod"), maxWorkflowBytes)
	if err != nil {
		return defaultGoVersion
	}
	m := goModVersionRe.FindSubmatch(data)
	if m == nil || !goVersionRe.Match(m[1]) {
		return defaultGoVersion
	}
	return string(m[1])
}

// deriveBranch resolves the repo's default branch from its committed git config
// (the symref origin/HEAD points at, or the checked-out HEAD), validated to the
// injection-safe allowlist. It reads git's plaintext refs directly rather than
// shelling out, so it stays dependency-free and works on a bare checkout.
func deriveBranch(repoRoot string) string {
	gitDir, commonDir := resolveGitDir(repoRoot)
	// origin/HEAD, when set, names the remote default branch: a line
	// "ref: refs/remotes/origin/<branch>". It lives in the SHARED (common) git
	// dir, which for a linked worktree is not the worktree's own gitdir.
	if data, err := os.ReadFile(filepath.Join(commonDir, "refs", "remotes", "origin", "HEAD")); err == nil {
		if b := branchFromSymref(string(data), "refs/remotes/origin/"); b != "" {
			return b
		}
	}
	// Fall back to the checked-out branch (HEAD → "ref: refs/heads/<branch>"). HEAD
	// is per-worktree, so it is read from the worktree's own gitdir.
	if data, err := os.ReadFile(filepath.Join(gitDir, "HEAD")); err == nil {
		if b := branchFromSymref(string(data), "refs/heads/"); b != "" {
			return b
		}
	}
	return defaultBranch
}

// resolveGitDir returns the repo's git directory and its shared (common) git
// directory. In an ordinary checkout ".git" is a directory and the two coincide.
// In a linked worktree or a submodule ".git" is instead a FILE holding
// "gitdir: <path>": HEAD lives at that path, while refs/remotes lives in the
// shared dir named by the worktree's "commondir" file. Treating ".git" as always
// a directory (the pre-fix behaviour) made both ref reads fail in a worktree, so
// deriveBranch silently fell back to "main".
func resolveGitDir(repoRoot string) (gitDir, commonDir string) {
	dotGit := filepath.Join(repoRoot, ".git")
	info, err := os.Lstat(dotGit)
	if err != nil || info.IsDir() {
		return dotGit, dotGit
	}
	data, err := os.ReadFile(dotGit)
	if err != nil {
		return dotGit, dotGit
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir:") {
		return dotGit, dotGit
	}
	target := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if target == "" {
		return dotGit, dotGit
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(repoRoot, target)
	}
	gitDir = filepath.Clean(target)
	commonDir = gitDir
	// A linked worktree's gitdir carries a "commondir" file pointing at the shared
	// git dir that holds refs/remotes.
	if cd, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		if c := strings.TrimSpace(string(cd)); c != "" {
			if !filepath.IsAbs(c) {
				c = filepath.Join(gitDir, c)
			}
			commonDir = filepath.Clean(c)
		}
	}
	return gitDir, commonDir
}

// branchFromSymref extracts and validates a branch name from a "ref: <prefix><branch>"
// symbolic-ref line. An unvalidated name yields "" so the caller falls back.
func branchFromSymref(content, prefix string) string {
	line := strings.TrimSpace(content)
	line = strings.TrimPrefix(line, "ref:")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	name := strings.TrimPrefix(line, prefix)
	if name == "" || !branchNameRe.MatchString(name) {
		return ""
	}
	return name
}
