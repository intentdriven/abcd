package ahoy

import (
	"path/filepath"
	"regexp"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ReleaseTagShape is the accepted release-tag alphabet, one definition for the
// updater (a tag travels into a URL path there) and for the unseen-update
// notice below (a tag travels onto a terminal there).
var ReleaseTagShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)

// updateShownFile is the marker the session check writes beside the cache's
// binary-meta: the release whose unseen update it has shown, as one
// release_tag= line.
const updateShownFile = "update-shown"

// TakeUnseenUpdate reports the one release swap nobody was told about, and
// marks it told. It is the ruling CJ1b's single exception to the session
// check's read-only rule, and the marker it writes is the only write.
//
// An update is announced by whatever performed the swap, when the swap
// completes: `abcd update` prints update.UpdatedFormat, and hooks/bootstrap.sh
// leads its success notice with it. The one swap whose output no one reads is
// the bootstrap salvage the per-prompt, per-command and pre-compaction hooks
// run with their output discarded (hooks/hooks.json); that run records
// transition_unseen=yes beside previous_tag in the cache's binary-meta, and
// this is what shows it, once, at the next session start. A swap whose own
// line was relayed carries no flag, so this shows it never and writes nothing.
//
// The data dir is an environment value, so it passes dataDirHazard before
// anything is read from it or written into it, and both tags must have
// ReleaseTagShape: a record an HTTP redirect filled shows nothing it cannot
// vouch for. The marker is written before the update is reported, and a marker
// that cannot be written reports nothing: once is the ruling, and a notice
// that cannot remember it was shown would repeat every session.
func TakeUnseenUpdate(pluginRoot, cwd string) (from, to string, ok bool) {
	data := pluginDataDir(pluginRoot).dir
	if data == "" || dataDirHazard(data, cwd) != "" {
		return "", "", false
	}
	cache := filepath.Join(data, "cache")
	meta := filepath.Join(cache, "binary-meta")
	if metaField(meta, "transition_unseen") != "yes" {
		return "", "", false
	}
	from, to = metaField(meta, "previous_tag"), metaField(meta, "release_tag")
	if !ReleaseTagShape.MatchString(from) || !ReleaseTagShape.MatchString(to) || from == to {
		return "", "", false
	}
	marker := filepath.Join(cache, updateShownFile)
	if metaField(marker, "release_tag") == to {
		return "", "", false
	}
	if err := fsutil.WriteFileAtomic(marker, []byte("release_tag="+to+"\n"), 0o600); err != nil {
		return "", "", false
	}
	return from, to, true
}
