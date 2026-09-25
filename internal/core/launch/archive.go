package launch

// archive.go — the pinned plugin archive (adr-2609231048308186's decision; the
// 2026-09-23 ruling E1).
//
// A release publishes its plugin as ONE zip, `<plugin>-plugin-v<version>.zip`,
// and the committed marketplace catalog names that zip by address and by
// SHA-256. The harness downloads the zip and refuses it when the digest differs,
// so what an install or update at vX receives is exactly vX's payload, stamped
// with vX's version — the thing itd-67 promised and a relative-path source into
// the unversioned working tree could never deliver.
//
// The digest is only checkable because the archive is REPRODUCIBLE: the ship
// renders it from its tree to learn the digest it commits, and the release
// workflow renders it again from the tagged commit and refuses to publish unless
// the two agree. Everything that could make two honest renders differ is fixed
// here: entries are written in sorted order, uncompressed (a compressor's output
// may change between toolchain releases; stored bytes cannot), with one fixed
// timestamp and a mode normalised to 0644 or 0755.
//
// The catalog itself is LEFT OUT of the zip. It is the file that names the
// archive's digest, so an archive that carried it would have to contain its own
// hash.

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// archiveEpoch is the one timestamp every archive entry carries. It is the
// earliest instant a zip's DOS date field can represent, so no reader
// normalises it into something else.
var archiveEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// archiveSourceKind is the marketplace source type that names a zip by URL and
// digest.
const archiveSourceKind = "archive"

// sha256HexRe is a pin's digest: 64 hex characters, either case (the harness
// accepts both; this package writes lower case).
var sha256HexRe = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// githubRepoRe is the only repository address the download URL derives from:
// https://github.com/<owner>/<repo>, nothing after it.
var githubRepoRe = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9][A-Za-z0-9-]*)/([A-Za-z0-9._-]+)$`)

// ErrArchivePinMismatch reports that the committed catalog does not name the
// archive a render just produced — no pin at all, a pin for another release, or
// a digest the render does not reproduce. It is the release gate's refusal.
var ErrArchivePinMismatch = errors.New("the committed marketplace pin does not match the rendered plugin archive")

// ErrArchiveRepositoryMismatch reports that the archive's download address does
// not sit under the releasing repository's download path for the tag. The
// address derives from plugin.json's repository, which a rename, a transfer or
// a fork leaves naming another repository: the pin still matches the render,
// and every install 404s. It is the release gate's refusal, like
// ErrArchivePinMismatch.
var ErrArchiveRepositoryMismatch = errors.New("the plugin archive's address is not the releasing repository's release")

// publishesArchiveKey is the version-location contract's declaration that the
// repository's release workflow publishes the pinned plugin archive.
const publishesArchiveKey = "publishes_plugin_archive"

// githubRepositoryRe is a bare GitHub repository name, owner/name, as the
// forge hands it to a workflow in GITHUB_REPOSITORY.
var githubRepositoryRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)

// PluginArchive is one rendered, packed plugin release archive.
type PluginArchive struct {
	// Name is the release asset's file name, <plugin>-plugin-v<version>.zip.
	Name string `json:"name"`
	// Path is where the archive was written.
	Path string `json:"path"`
	// SHA256 is the lower-case hex digest of the archive's bytes.
	SHA256 string `json:"sha256"`
	// Version is the release version stamped into the archived plugin manifest.
	Version string `json:"version"`
	// Files counts the archive's entries.
	Files int `json:"files"`
	// Bytes is the archive's size.
	Bytes int64 `json:"bytes"`
}

// ArchivePin is the marketplace source that names a release archive.
type ArchivePin struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// PluginArchiveName is the release asset name for one plugin at one version.
func PluginArchiveName(plugin, version string) string {
	return plugin + "-plugin-v" + version + ".zip"
}

// RenderPluginArchive renders the release payload through RenderPayload into
// req.Dest (a staging directory under the same rules as any render: outside the
// repository, empty or absent) and packs it into outDir as the release archive.
// It returns the archive and the render it was packed from.
//
// outDir must already exist; the archive is written there and nowhere else, and
// an archive of the same name already in outDir is refused rather than replaced.
func RenderPluginArchive(req PayloadRenderRequest, outDir string) (PluginArchive, PayloadRenderResult, error) {
	var a PluginArchive
	info, err := os.Stat(outDir)
	if err != nil {
		return a, PayloadRenderResult{}, fmt.Errorf("the archive output directory is not usable: %w", pathFreeError(err))
	}
	if !info.IsDir() {
		return a, PayloadRenderResult{}, errors.New("the archive output directory is not a directory")
	}
	res, err := RenderPayload(req)
	if err != nil {
		return a, res, err
	}
	name, err := pluginName(res.Dest)
	if err != nil {
		return a, res, err
	}
	a.Name = PluginArchiveName(name, req.Version)
	a.Version = req.Version
	a.Path = filepath.Join(outDir, a.Name)
	if _, err := os.Lstat(a.Path); err == nil {
		return a, res, fmt.Errorf("%s already exists in the output directory — refusing to replace a release archive", a.Name)
	}
	a.SHA256, a.Files, a.Bytes, err = packPluginArchive(res.Dest, a.Path)
	if err != nil {
		_ = os.Remove(a.Path)
		return a, res, err
	}
	return a, res, nil
}

// validPluginName admits a plugin name that is safe inside a file name and a
// URL path segment.
func validPluginName(name string) bool { return pluginNameRe.MatchString(name) }

// pluginNameRe is a plugin name that is safe inside a file name and a URL path
// segment.
var pluginNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// packPluginArchive writes every regular file under payloadDir, except the
// marketplace catalog, into a reproducible zip at zipPath, and returns its
// digest, entry count and size.
func packPluginArchive(payloadDir, zipPath string) (string, int, int64, error) {
	var rels []string
	err := filepath.WalkDir(payloadDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(payloadDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !d.Type().IsRegular() {
			// The render writes regular files only; anything else in its output
			// is not something it made, and packing it would ship it unexamined.
			return fmt.Errorf("the rendered payload holds a non-regular file %s", rel)
		}
		if rel == marketplaceFile {
			return nil
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return "", 0, 0, err
	}
	sort.Strings(rels)

	f, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, 0, err
	}
	hash := sha256.New()
	counter := &countingWriter{}
	zw := zip.NewWriter(io.MultiWriter(f, hash, counter))
	for _, rel := range rels {
		if err := addArchiveEntry(zw, payloadDir, rel); err != nil {
			_ = f.Close()
			return "", 0, 0, err
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return "", 0, 0, err
	}
	if err := f.Close(); err != nil {
		return "", 0, 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), len(rels), counter.n, nil
}

// addArchiveEntry stores one payload file with the fixed timestamp and a mode
// normalised to 0755 (any execute bit) or 0644.
func addArchiveEntry(zw *zip.Writer, payloadDir, rel string) error {
	abs := filepath.Join(payloadDir, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	hdr := &zip.FileHeader{Name: rel, Method: zip.Store, Modified: archiveEpoch}
	perm := fs.FileMode(0o644)
	if info.Mode().Perm()&0o111 != 0 {
		perm = 0o755
	}
	hdr.SetMode(perm)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// countingWriter counts the bytes the archive writer emits.
type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

// PrecheckPluginArchive makes the version-free refusals of the archive half of a
// ship: the plugin manifest names a plugin and a repository the release download
// address can be derived from. It performs zero writes.
func PrecheckPluginArchive(repoRoot string) error {
	_, _, err := archiveIdentity(repoRoot)
	return err
}

// DeclaresPluginArchive reports whether the repository's release publishes the
// pinned plugin archive: its version-location contract carries
// `"publishes_plugin_archive": true`. Nothing else counts. The contract alone
// says where the version lives, not that a release uploads an archive — a
// managed repository scaffolds workflows that upload none, and a catalog pinned
// there names an asset nothing publishes. An absent contract or key is false;
// a key that is not a boolean is refused rather than read as either answer.
func DeclaresPluginArchive(repoRoot string) (bool, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(versionLocationRelPath))
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	doc, err := loadJSON(path)
	if err != nil {
		return false, fmt.Errorf("version-location.json not readable: %w", err)
	}
	obj, ok := doc.(map[string]any)
	if !ok {
		return false, errors.New("version-location.json is not a JSON object")
	}
	raw, present := obj[publishesArchiveKey]
	if !present {
		return false, nil
	}
	b, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("version-location.json: %s must be true or false", publishesArchiveKey)
	}
	return b, nil
}

// ValidateGitHubRepository refuses a repository that is not a bare GitHub
// owner/name, the shape a workflow's GITHUB_REPOSITORY carries.
func ValidateGitHubRepository(repository string) error {
	if !githubRepositoryRe.MatchString(repository) || strings.HasSuffix(repository, "/.") || strings.HasSuffix(repository, "/..") {
		return fmt.Errorf("the repository %q is not a GitHub owner/name", repository)
	}
	return nil
}

// CheckArchiveRepository refuses, with ErrArchiveRepositoryMismatch, unless url
// sits under https://github.com/<repository>/releases/download/<tag>/ — the
// path the release workflow running in repository uploads tag's assets to.
// GitHub resolves owner and repository names case-insensitively, so the
// comparison is too. A repository that is not owner/name is a structural fault,
// not a mismatch.
func CheckArchiveRepository(url, repository, tag string) error {
	if err := ValidateGitHubRepository(repository); err != nil {
		return err
	}
	prefix := "https://github.com/" + repository + "/releases/download/" + tag + "/"
	if !strings.HasPrefix(strings.ToLower(url), strings.ToLower(prefix)) {
		return fmt.Errorf("%w: %s is not under %s — plugin.json's repository does not name the repository this release runs in",
			ErrArchiveRepositoryMismatch, url, prefix)
	}
	return nil
}

// ArchiveReleaseURL is the address the release workflow publishes version's
// archive at: <repository>/releases/download/v<version>/<archive name>.
func ArchiveReleaseURL(repoRoot, version string) (string, error) {
	if !IsStrictSemver(version) {
		return "", fmt.Errorf("the release version %q is not strict SemVer", version)
	}
	name, repo, err := archiveIdentity(repoRoot)
	if err != nil {
		return "", err
	}
	return repo + "/releases/download/v" + version + "/" + PluginArchiveName(name, version), nil
}

// ReleaseRepository is the https://github.com/<owner>/<repo> address the
// working tree's plugin manifest names: where the plugin's releases, and their
// assets, are published.
func ReleaseRepository(repoRoot string) (string, error) {
	_, repo, err := archiveIdentity(repoRoot)
	return repo, err
}

// archiveIdentity reads the plugin name and the repository address from the
// working tree's plugin manifest.
func archiveIdentity(repoRoot string) (name, repo string, err error) {
	name, err = pluginName(repoRoot)
	if err != nil {
		return "", "", err
	}
	doc, err := loadJSON(filepath.Join(repoRoot, filepath.FromSlash(pluginManifestFile)))
	if err != nil {
		return "", "", fmt.Errorf("%s not readable: %w", pluginManifestFile, err)
	}
	obj, _ := doc.(map[string]any)
	repo, _ = obj["repository"].(string)
	m := githubRepoRe.FindStringSubmatch(repo)
	if m == nil || m[2] == "." || m[2] == ".." {
		return "", "", fmt.Errorf("%s: repository %q is not an https://github.com/<owner>/<repo> address, so no release download URL can be derived for the pinned archive", pluginManifestFile, repo)
	}
	return name, repo, nil
}

// ReadArchivePin returns the archive source the working tree's catalog names
// for the plugin. ok is false, with no error, when the plugin's listing carries
// any other source (the relative-path "./" included).
func ReadArchivePin(repoRoot string) (ArchivePin, bool, error) {
	_, entry, err := loadCatalogEntry(repoRoot)
	if err != nil {
		return ArchivePin{}, false, err
	}
	src, ok := entry["source"].(map[string]any)
	if !ok || src["source"] != archiveSourceKind {
		return ArchivePin{}, false, nil
	}
	pin := ArchivePin{}
	pin.URL, _ = src["url"].(string)
	pin.SHA256, _ = src["sha256"].(string)
	return pin, true, nil
}

// WriteArchivePin rewrites the plugin's catalog listing so its source is the
// pinned archive. Only the source changes; every other key of the listing and
// of the catalog is kept.
func WriteArchivePin(repoRoot string, pin ArchivePin) error {
	if err := validatePin(pin); err != nil {
		return err
	}
	doc, entry, err := loadCatalogEntry(repoRoot)
	if err != nil {
		return err
	}
	entry["source"] = map[string]any{
		"source": archiveSourceKind,
		"url":    pin.URL,
		"sha256": strings.ToLower(pin.SHA256),
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicPreserveMode(filepath.Join(repoRoot, filepath.FromSlash(marketplaceFile)), append(data, '\n'))
}

// validatePin refuses a pin the harness would refuse or a reader could misread.
func validatePin(pin ArchivePin) error {
	u, err := url.Parse(pin.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || !strings.HasSuffix(u.Path, ".zip") {
		return fmt.Errorf("archive pin URL %q is not an https URL of a .zip", pin.URL)
	}
	if !sha256HexRe.MatchString(pin.SHA256) {
		return fmt.Errorf("archive pin digest %q is not 64 hex characters", pin.SHA256)
	}
	return nil
}

// loadCatalogEntry decodes the working tree's catalog and returns it with the
// listing whose name is the plugin manifest's name, so a caller can edit that
// listing in place and marshal the whole document.
func loadCatalogEntry(repoRoot string) (any, map[string]any, error) {
	name, err := pluginName(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	doc, err := loadJSON(filepath.Join(repoRoot, filepath.FromSlash(marketplaceFile)))
	if err != nil {
		return nil, nil, fmt.Errorf("%s not readable: %w", marketplaceFile, err)
	}
	obj, _ := doc.(map[string]any)
	plugins, ok := obj["plugins"].([]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s declares no plugins array", marketplaceFile)
	}
	for _, item := range plugins {
		if entry, ok := item.(map[string]any); ok && entry["name"] == name {
			return doc, entry, nil
		}
	}
	return nil, nil, fmt.Errorf("%s lists no plugin named %q", marketplaceFile, name)
}

// pluginName reads the plugin name from the working tree's plugin manifest.
func pluginName(repoRoot string) (string, error) {
	doc, err := loadJSON(filepath.Join(repoRoot, filepath.FromSlash(pluginManifestFile)))
	if err != nil {
		return "", fmt.Errorf("%s not readable: %w", pluginManifestFile, err)
	}
	obj, _ := doc.(map[string]any)
	name, _ := obj["name"].(string)
	if !validPluginName(name) {
		return "", fmt.Errorf("%s names no usable plugin name (%q)", pluginManifestFile, name)
	}
	return name, nil
}

// VerifyArchivePin proves the committed catalog names exactly this archive: the
// address the release publishes it at, and its digest. Any other state is
// ErrArchivePinMismatch, with both sides named.
func VerifyArchivePin(repoRoot string, a PluginArchive) error {
	want, err := ArchiveReleaseURL(repoRoot, a.Version)
	if err != nil {
		return err
	}
	pin, ok, err := ReadArchivePin(repoRoot)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s does not source the plugin from a pinned archive; expected url %s sha256 %s",
			ErrArchivePinMismatch, marketplaceFile, want, a.SHA256)
	}
	if pin.URL != want {
		return fmt.Errorf("%w: the pin names %s, the release publishes %s", ErrArchivePinMismatch, pin.URL, want)
	}
	if !strings.EqualFold(pin.SHA256, a.SHA256) {
		return fmt.Errorf("%w: the pin's sha256 is %s, the rendered archive's is %s", ErrArchivePinMismatch, pin.SHA256, a.SHA256)
	}
	return nil
}

// DirtyPayloadFiles names the payload files whose working-tree bytes are not
// the committed ones: modified or staged tracked files, and untracked files the
// payload would carry.
//
// The ship renders the archive from the WORKING TREE to learn the digest it
// commits, while the release gate renders it from the tagged COMMIT. A payload
// file that differs between the two makes the pin unreproducible, and the
// release would then refuse at a point where the version is already tagged. So
// the ship refuses such a tree before it writes anything.
//
// It reads the tree through DirtyTreeFiles, the dirty-tree gate's own reader,
// and keeps the payload's share: unlike that gate, this refusal has no
// --allow-dirty, because an unreproducible pin is wrong whoever allows it.
func DirtyPayloadFiles(repoRoot string, bundle Bundle) ([]string, error) {
	list, err := DirtyTreeFiles(repoRoot)
	if err != nil {
		return nil, err
	}
	dirty := make(map[string]struct{}, len(list))
	for _, p := range list {
		dirty[p] = struct{}{}
	}
	var out []string
	for _, f := range bundle.Included {
		if _, ok := dirty[f.LogicalPath]; ok {
			out = append(out, f.LogicalPath)
		}
	}
	return out, nil
}
