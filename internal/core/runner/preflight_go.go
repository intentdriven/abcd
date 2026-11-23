package runner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// preflightGoPrefix begins the name of the folder, beside a stage's home, that
// holds one preflight's Go build cache and GOPATH (MakePreflightGo).
const preflightGoPrefix = "preflight-go-"

// MakePreflightGo makes the folder one preflight's Go build cache and GOPATH
// live in, and returns its path: beside the stage's home, under a name drawn
// at random, made by a mkdir that fails on any entry already at that name, a
// link included, and read back as a real directory. A drive makes it after
// the stage's home was emptied (ResetStageHome), and removes it once the
// preflight ends (RemovePreflightGo). A folder of this kind an earlier
// preflight left, as a drive stopped part-way leaves it, is removed first.
func MakePreflightGo(p Profile) (string, error) {
	root, err := stageScope(p)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if f, err := root.Open("."); err == nil {
		entries, _ := f.ReadDir(-1)
		_ = f.Close()
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), preflightGoPrefix) {
				restoreOwnerAccess(root, e.Name())
				_ = root.RemoveAll(e.Name())
			}
		}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", &WallError{WallSettingsFile, "the preflight's Go folder could not be named"}
	}
	name := preflightGoPrefix + hex.EncodeToString(b[:])
	if err := root.Mkdir(name, 0o700); err != nil {
		return "", &WallError{WallSettingsFile, "the preflight's Go folder could not be made afresh: " + fsutil.RedactHome(err.Error())}
	}
	if fi, err := root.Lstat(name); err != nil || !fi.IsDir() || fi.Mode()&fs.ModeSymlink != 0 {
		return "", &WallError{WallSettingsFile, "the preflight's Go folder is no real directory"}
	}
	for _, sub := range []string{"build", "path"} {
		if err := root.Mkdir(name+"/"+sub, 0o700); err != nil {
			return "", &WallError{WallSettingsFile, "the preflight's Go folder could not be laid out: " + fsutil.RedactHome(err.Error())}
		}
	}
	return filepath.Join(p.StageRoot, name), nil
}

// RemovePreflightGo removes the folder MakePreflightGo made at dir, through the
// stage's scope, so a link inside it is removed and never followed.
func RemovePreflightGo(p Profile, dir string) error {
	name := filepath.Base(dir)
	if filepath.Dir(dir) != filepath.Clean(p.StageRoot) || !strings.HasPrefix(name, preflightGoPrefix) {
		return &WallError{WallSettingsFile, "the folder named is not a preflight's Go folder"}
	}
	root, err := stageScope(p)
	if err != nil {
		return err
	}
	defer root.Close()
	restoreOwnerAccess(root, name)
	if err := root.RemoveAll(name); err != nil {
		return &WallError{WallSettingsFile, "the preflight's Go folder could not be removed: " + fsutil.RedactHome(err.Error())}
	}
	return nil
}

// verifyTimeout bounds each `go mod verify` VerifyModules runs.
const verifyTimeout = 5 * time.Minute

// verifyOutputCap bounds what VerifyModules keeps of each run's output.
const verifyOutputCap = 64 << 10

// verifyShownLines bounds how many of go's lines a ModuleCacheError names.
const verifyShownLines = 10

// verifiedFiles are the files of a module or workspace root that VerifyModules
// holds unchanged.
var verifiedFiles = []string{"go.mod", "go.sum", "go.work", "go.work.sum"}

// ModuleCacheError is a module cache `go mod verify` did not pass, or a
// verification that changed what it must only read: the preflight is not run.
type ModuleCacheError struct {
	// Modified names each module whose directory or zip go found changed, as
	// go names it (path and version).
	Modified []string
	// Lines are go's own words, sanitised and capped.
	Lines []string
	// Changed names a file of the checkout the verification wrote, put back as
	// it was.
	Changed string
}

func (e *ModuleCacheError) Error() string {
	switch {
	case e.Changed != "":
		return "`go mod verify` changed " + e.Changed + ", which it must only read, so the preflight was not run; it was put back as it was"
	case len(e.Modified) > 0:
		return "the module cache holds modules changed since Go unpacked them (" + strings.Join(e.Modified, ", ") + "), so the preflight was not run: " + strings.Join(e.Lines, "; ")
	}
	return "the module cache could not be verified with `go mod verify`, so the preflight was not run: " + strings.Join(e.Lines, "; ")
}

// VerifyModules checks, with `go mod verify` in dir under env, the preflight's
// own, the repository's own dependencies the person's module cache holds: each
// module of the main module's build list (every workspace module's, with a
// go.work in force) whose directory or zip the cache holds is held to the hash
// Go recorded when it unpacked it, and Go's build then holds that hash to
// go.sum. A module the cache does not hold is passed over. Nothing else in the
// cache is checked: not the Go toolchain a go.mod or GOTOOLCHAIN selects, not a
// tool `go install` or `go run` takes by version, and no program on PATH.
//
// It runs first with GOPROXY=off, so a cache that holds the module graph is read
// with no network, and with GOFLAGS=-mod=mod in module mode, so go.sum is read
// and not written; in workspace mode, which refuses -mod=mod, with the GOFLAGS
// env carries less any -mod. Where that run stops only because the cache lacks
// a module's metadata (`module lookup disabled by GOPROXY=off`), it runs again
// with the GOPROXY env carries, which fetches that metadata, and can download a
// toolchain the checkout selects. Each run is bounded by verifyTimeout.
//
// A module go reports changed refuses the preflight and is named; so does any
// other failure, in go's words, and a run that leaves go.mod, go.sum, go.work
// or go.work.sum in dir other than it found them, each put back first. Nothing
// is verified, and note says why, where the build reads no dependency from the
// module cache (vendorMode), where dir holds no go.mod, and where no go is on
// PATH. go is the one PATH finds, held to the in-checkout rule (tools.LookPath).
func VerifyModules(ctx context.Context, dir string, env []string) (note string, err error) {
	if fi, err := os.Lstat(filepath.Join(dir, "go.mod")); err != nil || !fi.Mode().IsRegular() {
		return "the checkout has no go.mod at its root, so no module in the module cache was verified", nil
	}
	goBin, err := tools.LookPath("go", dir)
	if errors.Is(err, tools.ErrInsideWorkingTree) {
		return "", &ModuleCacheError{Lines: []string{"the go PATH finds lies inside a working tree, so it is never run"}}
	}
	if err != nil {
		return "no go is on PATH, so no module in the module cache was verified", nil
	}
	before := map[string][]byte{}
	for _, name := range verifiedFiles {
		if raw, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			before[name] = raw
		}
	}
	work, err := goWork(ctx, goBin, dir, env)
	if changed := putBack(dir, before); changed != "" {
		return "", &ModuleCacheError{Changed: changed}
	}
	if err != nil {
		return "", &ModuleCacheError{Lines: []string{"go could not say whether a go.work is in force: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}}
	}
	flags := envFlags(env)
	if vendorMode(dir, work, flags) {
		return "the build reads its dependencies from vendor/, not the module cache, so no module in the module cache was verified", nil
	}
	mode := "GOFLAGS=-mod=mod"
	if work != "" {
		mode = "GOFLAGS=" + strings.Join(slices.DeleteFunc(slices.Clone(flags), isModFlag), " ")
	}
	out, err := verifyOnce(ctx, goBin, dir, withEnv(env, mode, "GOPROXY=off"))
	if err != nil && strings.Contains(out, "module lookup disabled by GOPROXY=off") && !modifiedLine(out) {
		out, err = verifyOnce(ctx, goBin, dir, withEnv(env, mode))
	}
	if changed := putBack(dir, before); changed != "" {
		return "", &ModuleCacheError{Changed: changed}
	}
	if err == nil {
		return "", nil
	}
	e := &ModuleCacheError{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name, ok := modifiedModule(line); ok {
			e.Modified = append(e.Modified, termsafe.Sanitize(name))
		}
		if len(e.Lines) < verifyShownLines {
			e.Lines = append(e.Lines, termsafe.Sanitize(gitutil.ScrubURLCredentials(fsutil.RedactHome(line))))
		}
	}
	if len(e.Lines) == 0 {
		e.Lines = []string{termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return "", e
}

// goWork is the go.work go finds in force for dir under env (`go env GOWORK`),
// "" in module mode.
func goWork(parent context.Context, goBin, dir string, env []string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, verifyTimeout)
	defer cancel()
	// #nosec G204 -- goBin is the go PATH finds, held to the in-checkout rule;
	// the arguments are fixed.
	cmd := exec.CommandContext(ctx, goBin, "env", "GOWORK")
	cmd.Dir, cmd.Env, cmd.Stdin = dir, env, nil
	var buf bytes.Buffer
	w := &bounded{w: &buf, remaining: verifyOutputCap}
	cmd.Stdout, cmd.Stderr = w, w
	procgroup.Lead(cmd)
	cmd.Cancel = func() error { return procgroup.Kill(cmd) }
	cmd.WaitDelay = pipeGrace
	err := cmd.Run()
	procgroup.KillLeft(cmd)
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(buf.String()))
	}
	switch v := strings.TrimSpace(buf.String()); v {
	case "", "off":
		return "", nil
	default:
		return v, nil
	}
}

// envFlags are the flags of the last GOFLAGS env carries, split on blanks.
func envFlags(env []string) []string {
	var flags []string
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "GOFLAGS="); ok {
			flags = strings.Fields(v)
		}
	}
	return flags
}

// isModFlag says whether a GOFLAGS flag sets -mod.
func isModFlag(f string) bool {
	return strings.HasPrefix(f, "-mod=") || strings.HasPrefix(f, "--mod=")
}

// vendorMode says whether Go builds from vendor/ rather than the module cache,
// as Go decides it: a -mod=vendor in flags, or no -mod at all with
// vendor/modules.txt beside the go.mod (dir's) or the go.work in force (work)
// and that file's go directive at 1.14 or later (1.22 for a go.work).
func vendorMode(dir, work string, flags []string) bool {
	mod := ""
	for _, f := range flags {
		if isModFlag(f) {
			_, mod, _ = strings.Cut(f, "=")
		}
	}
	switch mod {
	case "vendor":
		return true
	case "":
	default:
		return false
	}
	root, file, least := dir, filepath.Join(dir, "go.mod"), 14
	if work != "" {
		root, file, least = filepath.Dir(work), work, 22
	}
	if fi, err := os.Lstat(filepath.Join(root, "vendor", "modules.txt")); err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return goDirectiveMinor(file) >= least
}

// goDirectiveMinor is the minor version of the go directive in the go.mod or
// go.work at path (21 for `go 1.21.3`), 16 when it states none, as Go then
// assumes, and 0 when the file cannot be read.
func goDirectiveMinor(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "go" {
			continue
		}
		rest, ok := strings.CutPrefix(f[1], "1.")
		if !ok {
			return 0
		}
		minor, _, _ := strings.Cut(rest, ".")
		n := 0
		for _, r := range minor {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		return n
	}
	return 16
}

// verifyOnce runs `go mod verify` once, as a process group of its own killed
// whole at ctx's end or verifyTimeout, and returns its combined output, capped.
func verifyOnce(parent context.Context, goBin, dir string, env []string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, verifyTimeout)
	defer cancel()
	// #nosec G204 -- goBin is the go PATH finds, held to the in-checkout rule;
	// the arguments are fixed.
	cmd := exec.CommandContext(ctx, goBin, "mod", "verify")
	cmd.Dir, cmd.Env, cmd.Stdin = dir, env, nil
	var buf bytes.Buffer
	w := &bounded{w: &buf, remaining: verifyOutputCap}
	cmd.Stdout, cmd.Stderr = w, w
	procgroup.Lead(cmd)
	cmd.Cancel = func() error { return procgroup.Kill(cmd) }
	cmd.WaitDelay = pipeGrace
	err := cmd.Run()
	procgroup.KillLeft(cmd)
	if err == nil && w.overflowed {
		err = fmt.Errorf("go mod verify said more than %d bytes", verifyOutputCap)
	}
	return buf.String(), err
}

// withEnv is env with each KEY=value of set in place of any the same key
// already has.
func withEnv(env []string, set ...string) []string {
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		k, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(set, func(s string) bool { return strings.HasPrefix(s, k+"=") })
	})
	return append(out, set...)
}

// putBack writes each file of before back into dir where its content changed,
// and removes one of verifiedFiles that was absent and now stands, naming the
// first it found changed, "" when none did.
func putBack(dir string, before map[string][]byte) string {
	changed := ""
	for _, name := range verifiedFiles {
		path := filepath.Join(dir, name)
		now, err := os.ReadFile(path)
		was, had := before[name]
		switch {
		case had && err == nil && bytes.Equal(now, was):
			continue
		case had:
			_ = os.WriteFile(path, was, 0o644)
		case err == nil:
			_ = os.Remove(path)
		default:
			continue
		}
		if changed == "" {
			changed = name
		}
	}
	return changed
}

// modifiedModule reads a line of `go mod verify` naming a module whose
// directory or zip changed (`<path> <version>: dir has been modified (...)`),
// and returns the module as path@version.
func modifiedModule(line string) (string, bool) {
	head, rest, ok := strings.Cut(line, ": ")
	if !ok || !(strings.HasPrefix(rest, "dir has been modified") || strings.HasPrefix(rest, "zip has been modified")) {
		return "", false
	}
	path, version, ok := strings.Cut(head, " ")
	if !ok || path == "" || version == "" || strings.Contains(version, " ") {
		return "", false
	}
	return path + "@" + version, true
}

// modifiedLine says whether out names any module changed.
func modifiedLine(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if _, ok := modifiedModule(strings.TrimSpace(line)); ok {
			return true
		}
	}
	return false
}
