package ahoy

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
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
// It wraps the existing unexported classify/install machinery: dryRun first
// asks whether the file's folder can take the files a write creates beside it
// (markerFolderRefusal, the write's own first check, iss-2610032202263648),
// then maps classifyMarker(path) → current→(false,nil), missing/outdated→
// (true,nil), symlink, unplaceable or unreadable→(false, err); a real run calls
// installMarkerFile(path) → its error wrapped as (false, err), else (wrote,
// nil). An unreadable file's refusal is the write's own: the dry run reads it
// once more for the reason and words it as the write does, with the folder
// taken out (iss-2610032048280232, iss-2610032202259011).
func EnsureMarker(path string, dryRun bool) (changed bool, err error) {
	if dryRun {
		if ferr := markerFolderRefusal(path); ferr != nil {
			return false, fmt.Errorf("cannot write marker to %s: %w", filepath.Base(path), ferr)
		}
		switch classifyMarker(path) {
		case markerCurrent:
			return false, nil
		case markerMissing, markerOutdated:
			return true, nil
		case markerUnreadable:
			_, rerr := fsutil.ReadGuarded(path, maxAhoyFileBytes)
			if rerr == nil {
				rerr = errors.New("it changed while it was read")
			}
			return false, fmt.Errorf("cannot write marker to %s: %w", filepath.Base(path),
				withoutFolder(path, fmt.Errorf("it could not be read: %w", rerr)))
		case markerSymlink:
			return false, fmt.Errorf("cannot write marker to %s: it is a symlink", filepath.Base(path))
		case markerUnplaceable:
			return false, fmt.Errorf("cannot write marker to %s: a fenced block or HTML comment is never closed, "+
				"so the block would land inside it", filepath.Base(path))
		default:
			return false, fmt.Errorf("cannot classify marker at %s", filepath.Base(path))
		}
	}
	wrote, err := installMarkerFile(path)
	if err != nil {
		return false, fmt.Errorf("cannot write marker to %s: %w", filepath.Base(path), err)
	}
	return wrote, nil
}
