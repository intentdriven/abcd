package ahoy

import (
	"fmt"
	"path/filepath"
)

// EnsureMarker installs, refreshes, or (dryRun) predicts the CURRENT abcd marker
// block in the file at path. It NEVER copies foreign prose — only the canonical
// block travels, so an embark can re-inject the block into a target CLAUDE.md
// without carrying a lifeboat's authored text. With dryRun it writes nothing and
// reports whether a real run WOULD change the file (the embark probe path);
// without it, it performs the write (the embark write path) — one code path, so
// probe cannot mispredict. changed reports whether the file was/would be written;
// a symlinked or unwritable target returns a non-nil error and changed=false.
//
// It wraps the existing unexported classify/install machinery: dryRun maps
// classifyMarker(path) → current→(false,nil), missing/outdated→(true,nil),
// symlink or unplaceable→(false, err); a real run calls installMarkerFile(path) → ok==false→
// (false, err), else (wrote, nil).
func EnsureMarker(path string, dryRun bool) (changed bool, err error) {
	if dryRun {
		switch classifyMarker(path) {
		case markerCurrent:
			return false, nil
		case markerMissing, markerOutdated:
			return true, nil
		case markerSymlink:
			return false, fmt.Errorf("cannot write marker to %s: it is a symlink", filepath.Base(path))
		case markerUnplaceable:
			return false, fmt.Errorf("cannot write marker to %s: a fenced block or HTML comment is never closed, "+
				"so the block would land inside it", filepath.Base(path))
		default:
			return false, fmt.Errorf("cannot classify marker at %s", filepath.Base(path))
		}
	}
	wrote, ok := installMarkerFile(path)
	if !ok {
		return false, fmt.Errorf("cannot write marker to %s", filepath.Base(path))
	}
	return wrote, nil
}
