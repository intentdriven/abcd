package machineload

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Free disk is the load gate's other resource (iss-2610281502581386): on 28
// Oct 2026 the Go build cache had grown to 177 GiB, the data volume fell to
// 658 MiB free, and every preflight on the machine then failed or wrote
// truncated files with no gate naming the cause. This file only measures;
// the floor and the refusal live with the gate (internal/core/implement,
// load_disk.go).

// Volume is one reading of the volume holding a path.
type Volume struct {
	// Path is the path read: the one asked for, or, when it does not exist,
	// its nearest existing ancestor (NearestExisting).
	Path string
	// Free is the bytes an unprivileged writer can still use on the volume
	// (the blocks available to a non-root user times the block size), the
	// figure `df` reports as available.
	Free uint64
	// Total is the volume's size in bytes.
	Total uint64
	// ID tells two volumes apart: two paths on one volume read the same ID.
	ID string
	// Mount is where the volume is mounted, when the platform says so cheaply
	// (macOS); "" otherwise.
	Mount string
}

// ReadVolume reads the volume holding path, or, when path does not exist,
// the volume holding its nearest existing ancestor. On a platform other than
// macOS and Linux it reports ErrUnsupported.
func ReadVolume(path string) (Volume, error) {
	p := NearestExisting(path)
	v, err := statVolume(p)
	v.Path = p
	return v, err
}

// NearestExisting is path itself when it exists, else its nearest ancestor
// that does: the volume a directory not yet created will be written to. A
// path whose every ancestor is missing (only a relative path can be) is
// returned cleaned, for the read to fail on.
func NearestExisting(path string) string {
	p := filepath.Clean(path)
	for {
		if _, err := os.Stat(p); err == nil || !errors.Is(err, fs.ErrNotExist) {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return p
		}
		p = parent
	}
}

// goEnvTimeout bounds the one `go env GOCACHE` run; it takes tens of
// milliseconds.
const goEnvTimeout = 10 * time.Second

// GoCacheDir is the Go build cache's directory as `go env GOCACHE` reports
// it, which honours both the GOCACHE variable and a value saved with `go env
// -w`; "" when caching is off or the directory does not exist yet. The go
// command runs in the temporary directory with GOTOOLCHAIN=local, so no
// repository's go.mod is read and no toolchain is fetched to answer. Where
// no go command can be run, the cache is where go would put it: GOCACHE, or
// go-build in the user cache directory.
func GoCacheDir() string {
	return goCacheDir(goEnvGOCACHE, os.Getenv, os.UserCacheDir)
}

func goCacheDir(goEnv func() (string, error), getenv func(string) string, userCache func() (string, error)) string {
	dir, err := goEnv()
	if err != nil || dir == "" {
		dir = getenv("GOCACHE")
		if dir == "" {
			if c, err := userCache(); err == nil && c != "" {
				dir = filepath.Join(c, "go-build")
			}
		}
	}
	if dir == "" || dir == "off" || !filepath.IsAbs(dir) {
		return ""
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return ""
	}
	return filepath.Clean(dir)
}

// goEnvGOCACHE runs `go env GOCACHE`.
func goEnvGOCACHE() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), goEnvTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "env", "GOCACHE")
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CacheSize is a measure of the Go build cache's size.
type CacheSize struct {
	// Bytes is the size: exact when Sampled equals Subdirs, else estimated
	// from the subdirectories walked.
	Bytes uint64
	// Sampled is how many of the cache's subdirectories were walked, and
	// Subdirs how many it has.
	Sampled int
	Subdirs int
}

// Estimated reports whether Bytes is extrapolated from a sample.
func (c CacheSize) Estimated() bool { return c.Sampled < c.Subdirs }

// GoCacheBudget bounds the walk GoCacheSize makes. A full walk is too slow to
// make before a refusal: on the machine that filed iss-2610281502581386 a
// 291 GiB cache held about 927,000 files and `du -sk` over it took 66
// seconds. The cache spreads its entries over 256 subdirectories by the first
// byte of each entry's hash, so the entries are spread evenly, and a few
// subdirectories walked in full give the size to within a few percent.
const GoCacheBudget = 1500 * time.Millisecond

// GoCacheSize measures the Go build cache at dir. The cache keeps its entries
// in 256 subdirectories named by the first byte of each entry's hash (00 to
// ff), so the size of the subdirectories walked within budget, scaled by how
// many there are, estimates the whole; a cache small enough to walk in
// full is measured exactly. At least one subdirectory is always walked, so
// there is always a figure. Files at the top of the cache (its README and
// trim stamp) are not counted.
func GoCacheSize(dir string, budget time.Duration) (CacheSize, error) {
	return goCacheSize(dir, budget, time.Now)
}

func goCacheSize(dir string, budget time.Duration, now func() time.Time) (CacheSize, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return CacheSize{}, err
	}
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() && isHexByte(e.Name()) {
			subdirs = append(subdirs, e.Name())
		}
	}
	sort.Strings(subdirs)
	out := CacheSize{Subdirs: len(subdirs)}
	if len(subdirs) == 0 {
		return out, nil
	}
	deadline := now().Add(budget)
	var sum uint64
	for _, name := range subdirs {
		if out.Sampled > 0 && !now().Before(deadline) {
			break
		}
		files, err := os.ReadDir(filepath.Join(dir, name))
		if err != nil {
			// A subdirectory removed while it is read (a trim, or a
			// `go clean -cache` running beside this) counts as empty.
			out.Sampled++
			continue
		}
		for _, f := range files {
			if !f.Type().IsRegular() {
				continue
			}
			if info, err := f.Info(); err == nil {
				sum += uint64(info.Size())
			}
		}
		out.Sampled++
	}
	out.Bytes = sum
	if out.Estimated() {
		out.Bytes = uint64(float64(sum) * float64(out.Subdirs) / float64(out.Sampled))
	}
	return out, nil
}

// isHexByte reports whether name is two lowercase hex digits, the name of one
// of the Go build cache's subdirectories.
func isHexByte(name string) bool {
	if len(name) != 2 {
		return false
	}
	for _, c := range name {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}
