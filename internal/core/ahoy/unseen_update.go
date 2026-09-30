package ahoy

import (
	"os"
	"path/filepath"
	"regexp"
)

// ReleaseTagShape is the accepted release-tag alphabet, one definition for the
// updater (a tag travels into a URL path there) and for the unseen-update
// notice below (a tag travels onto a terminal there).
var ReleaseTagShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)

// updateShownPrefix names the claim the session check takes for one release's
// unseen update: update-shown-<tag> beside the cache's binary-meta, or
// .update-shown-<tag> beside a plugin root's own .binary-meta in the degraded
// per-root mode. The tag has ReleaseTagShape before it is joined, so it can
// carry no separator and cannot begin with a dot.
const updateShownPrefix = "update-shown-"

// TakeUnseenUpdate reports the one release swap nobody was told about, and
// marks it told. It is the ruling CJ1b's single exception to the session
// check's read-only rule, and the claim it creates is the only write.
//
// An update is announced by whatever performed the swap, when the swap
// completes: `abcd update` prints update.UpdatedFormat, and hooks/bootstrap.sh
// leads its success notice with it. The one swap whose output no one reads is
// the bootstrap salvage the per-prompt, per-command and pre-compaction hooks
// run with their output discarded (hooks/hooks.json); that run records
// transition_unseen=yes beside previous_tag in the binary-meta it writes (the
// cache's, or the plugin root's .binary-meta when there is no data dir), and
// this is what shows it, once, at the next session start. A swap whose own
// line was relayed carries no flag, so this shows it never and writes nothing.
//
// The directory the record comes from is an environment value, so it passes
// dataDirHazard before anything is read from it or written into it, and both
// tags must have ReleaseTagShape: a record an HTTP redirect filled shows
// nothing it cannot vouch for. Once is one exclusive create per release
// (O_CREATE|O_EXCL), so of any number of sessions starting together exactly
// one takes the claim; a claim path that already exists, or a create that
// fails for any other reason, reports nothing, because a notice that cannot
// remember it was shown would repeat every session. Claims for earlier
// releases are left in place: removing one would be a second write, and each
// is an empty file.
func TakeUnseenUpdate(pluginRoot, cwd string) (from, to string, ok bool) {
	dir, meta, claim := "", "", ""
	if data := pluginDataDir(pluginRoot).dir; data != "" {
		dir = data
		meta = filepath.Join(data, "cache", "binary-meta")
		claim = filepath.Join(data, "cache", updateShownPrefix)
	} else if pluginRoot != "" {
		dir = pluginRoot
		meta = filepath.Join(pluginRoot, ".binary-meta")
		claim = filepath.Join(pluginRoot, "."+updateShownPrefix)
	}
	if dir == "" || dataDirHazard(dir, cwd) != "" {
		return "", "", false
	}
	if metaField(meta, "transition_unseen") != "yes" {
		return "", "", false
	}
	from, to = metaField(meta, "previous_tag"), metaField(meta, "release_tag")
	if !ReleaseTagShape.MatchString(from) || !ReleaseTagShape.MatchString(to) || from == to {
		return "", "", false
	}
	f, err := os.OpenFile(claim+to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", false
	}
	if err := f.Close(); err != nil {
		return "", "", false
	}
	return from, to, true
}
