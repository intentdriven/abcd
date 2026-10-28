package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

// maxProgramBytes bounds the program extracted from an archive. gh is about
// 55 MiB unpacked.
const maxProgramBytes = 512 << 20

// extractMember returns the bytes of the one regular file named member inside
// an archive whose sha256 has already been checked. Only that exact name is
// read: no other entry is written anywhere, a link or a directory by that name
// is refused, and the read is bounded, so the archive chooses nothing but the
// program's bytes.
func extractMember(archive []byte, kind Archive, member string) ([]byte, error) {
	switch kind {
	case TarGz:
		return extractTarGz(archive, member)
	case Zip:
		return extractZip(archive, member)
	}
	return nil, fmt.Errorf("abcd does not read %q archives", kind)
}

func extractTarGz(archive []byte, member string) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("the download is not a gzip archive: %w", err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("the archive holds no %s", member)
		}
		if err != nil {
			return nil, fmt.Errorf("the archive does not read: %w", err)
		}
		if h.Name != member && h.Name != "./"+member {
			continue
		}
		if h.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("%s in the archive is not a regular file", member)
		}
		return readBounded(tr, member)
	}
}

func extractZip(archive []byte, member string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("the download is not a zip archive: %w", err)
	}
	for _, f := range zr.File {
		if f.Name != member {
			continue
		}
		if !f.Mode().IsRegular() {
			return nil, fmt.Errorf("%s in the archive is not a regular file", member)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s in the archive does not open: %w", member, err)
		}
		defer rc.Close()
		return readBounded(rc, member)
	}
	return nil, fmt.Errorf("the archive holds no %s", member)
}

func readBounded(r io.Reader, member string) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxProgramBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s in the archive does not read: %w", member, err)
	}
	if len(b) > maxProgramBytes {
		return nil, fmt.Errorf("%s in the archive is larger than the %d-byte bound", member, maxProgramBytes)
	}
	return b, nil
}
