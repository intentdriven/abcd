package site

// abcd's own record, built on the project route, fits GitHub Pages' limits
// (itd-2610142145236613's scope condition cond-2610142146250921). The limit
// is the one adr-2610142147242544's consequences name: a site of at most
// 1 GB. The deploy hands Pages the rendered tree as one tar archive, which
// may hold no symbolic or hard links (itd-2610091010348892's grounds, from
// the upload-pages-artifact README), so the test writes that archive for
// real, uncompressed, the larger of its two sizes, and refuses a link.

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// pagesSiteLimit is GitHub Pages' published site limit, 1 GB, read as
// decimal: the smaller of the two readings, so the guard fires first.
const pagesSiteLimit int64 = 1_000_000_000

// byteCounter counts what is written to it.
type byteCounter int64

func (c *byteCounter) Write(p []byte) (int, error) { *c += byteCounter(len(p)); return len(p), nil }

func TestAbcdsPagesSiteFitsGitHubPagesLimits(t *testing.T) {
	out := t.TempDir()
	res, err := Build(Request{RepoRoot: repoRoot(), OutDir: out, Stamp: fixtureStamp, Project: true, BasePath: "/abcd/", LeaveUI: true})
	if err != nil {
		t.Fatalf("abcd's project route does not build: %v", err)
	}
	var archive byteCounter
	tw := tar.NewWriter(&archive)
	var files int
	var site, largest int64
	var largestName string
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(out, p)
		if !info.IsDir() && !info.Mode().IsRegular() {
			t.Errorf("the Pages site holds %s (%s), which is neither a file nor a directory; the Pages artefact holds no link", rel, info.Mode().Type())
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files++
		site += info.Size()
		if info.Size() > largest {
			largest, largestName = info.Size(), hdr.Name
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err == nil {
		err = tw.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	if site != res.Bytes || files != len(res.Files) {
		t.Errorf("the build reports %d files of %d bytes, and its output holds %d files of %d bytes", len(res.Files), res.Bytes, files, site)
	}
	t.Logf("abcd's Pages site: %d files, %d bytes; %d bytes as an uncompressed tar; largest file %s, %d bytes; limit %d bytes",
		files, site, int64(archive), largestName, largest, pagesSiteLimit)
	if int64(archive) > pagesSiteLimit {
		t.Errorf("abcd's Pages site is %d bytes as the archive the deploy uploads (%d bytes in %d files), over GitHub Pages' %d-byte site limit (adr-2610142147242544); the largest file is %s, %d bytes",
			int64(archive), site, files, pagesSiteLimit, largestName, largest)
	}
}
