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

// VerifyModules runs `go mod verify` in dir under env, the preflight's own, with
// GOFLAGS=-mod=mod in place of any it carries: Go checks each module of the
// build list its cache holds, directory and zip, against the hash it recorded
// on download, and passes over one it does not hold. It runs first with
// GOPROXY=off, so a cache that holds the module graph is read with no network.
// Where that run stops only because the cache lacks a module's metadata
// (`module lookup disabled by GOPROXY=off`), it runs again with the GOPROXY
// env carries, which fetches that metadata and never a module's source, so
// the modules the cache holds are verified and the rest are left to the
// preflight's own download, which checks them against go.sum. A module go
// reports changed refuses the preflight and is named; so does any other
// failure, in go's words, and a run that leaves go.mod, go.sum, go.work or
// go.work.sum in dir other than it found them, each put back first. A dir with
// no go.mod, or no go on PATH, has nothing to verify. go is the one PATH finds,
// held to the in-checkout rule (tools.LookPath).
func VerifyModules(ctx context.Context, dir string, env []string) error {
	if fi, err := os.Lstat(filepath.Join(dir, "go.mod")); err != nil || !fi.Mode().IsRegular() {
		return nil
	}
	goBin, err := tools.LookPath("go", dir)
	if errors.Is(err, tools.ErrInsideWorkingTree) {
		return &ModuleCacheError{Lines: []string{"the go PATH finds lies inside a working tree, so it is never run"}}
	}
	if err != nil {
		return nil
	}
	before := map[string][]byte{}
	for _, name := range verifiedFiles {
		if raw, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			before[name] = raw
		}
	}
	out, err := verifyOnce(ctx, goBin, dir, withEnv(env, "GOFLAGS=-mod=mod", "GOPROXY=off"))
	if err != nil && strings.Contains(out, "module lookup disabled by GOPROXY=off") && !modifiedLine(out) {
		out, err = verifyOnce(ctx, goBin, dir, withEnv(env, "GOFLAGS=-mod=mod"))
	}
	if changed := putBack(dir, before); changed != "" {
		return &ModuleCacheError{Changed: changed}
	}
	if err == nil {
		return nil
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
	return e
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
