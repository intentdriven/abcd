package ahoy

// The guards around the one write abcd makes into the harness's user settings
// (iss-2610050556383525, the product thinker's ruling of 2026-10-05): abcd
// touches that file only for the status line, only with consent, and guarded
// against what can go wrong there.
//
//   - The consent is given over the change itself: the confirm shows the one
//     entry's value now and after (statusLineOfferQuestion).
//   - The file's exact bytes are copied into ~/.abcd.noindex/backups before it
//     is replaced, and a copy that cannot be kept refuses the write.
//   - The replaced file is read back, and one that does not say what was
//     written is put back from the copy.
//   - Only the PATH entry ~/.abcd.noindex/path-entry records is wired, and only
//     when it passes the trust check — never the plugin's own binary, whose
//     versioned cache directory a plugin update deletes (statusLineEntry).

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// statusLineOfferLead opens the offer and statusLineOfferAsk closes it; the
// change between them is the person's own, so the offer is recognised by the
// two ends (isStatusLineOfferQuestion).
var (
	statusLineOfferLead = "In an abcd-managed repository your assistant's status line becomes abcd's own row, " +
		"led by a badge saying whether abcd is here and whose answer the work is waiting on; " +
		"in every other repository the line you have now runs untouched. " +
		"Switch parts of it off any time in " + statusline.SettingsDisplay + "."
	statusLineOfferAsk = "Install abcd's status line?"
)

// statusLineOfferQuestion is the reason, the one change and the question, as
// the confirm's text: core never prints, so what ac-8 and risk 4 require the
// person to see travels as the question the prompter renders. The change is a
// list of two lines — the entry now and after — so it reads at a glance.
// Paths are shown in the tilde form; the command now is quoted as it stands.
func statusLineOfferQuestion(hs harnessSettings, entry string) string {
	now := "none"
	if hs.line != nil {
		now = termsafe.Sanitize(strconv.Quote(hs.command))
	}
	return statusLineOfferLead + "\n" +
		"It changes one entry, " + harnessStatusKey + ", in " + displayPath(hs.path) + ":\n" +
		"  now: " + now + "\n" +
		"  after: " + displayPath(entry) + " " + statusVerb + "\n" +
		"abcd records the command you have now and `abcd ahoy uninstall` puts it back; " +
		"a copy of the file is kept in " + abcdhome.Display("backups") + "/ before it changes, and declining writes nothing.\n" +
		statusLineOfferAsk
}

// isStatusLineOfferQuestion reports whether text is the status-line offer.
func isStatusLineOfferQuestion(text string) bool {
	return strings.HasPrefix(text, statusLineOfferLead) && strings.HasSuffix(text, "\n"+statusLineOfferAsk)
}

// statusLineEntry is the absolute path the harness command will run, or why
// there is none abcd will wire. It is the entry ~/.abcd.noindex/path-entry
// records, present, abcd's own by the classification every install step uses,
// and passing the trust check. Never the plugin-root binary: it lives in a
// versioned plugin cache directory a plugin update deletes, which would leave
// the line dangling in every session. Never a bare `abcd` either: the
// harness's shell PATH is not the user's.
func (a *applyCtx) statusLineEntry() (entry, why string) {
	rec, ok := readPathEntry()
	if !ok {
		return "", "abcd has no PATH entry on record (" + abcdhome.Display("path-entry") + " names none) for it to run, " +
			"and abcd never points it at the plugin's own copy, which a plugin update deletes"
	}
	if present, err := fsutil.Exists(rec.path); err != nil || !present {
		return "", "the PATH entry on record, " + displayPath(rec.path) + ", is not there"
	}
	switch classifyBinTarget(rec.path, a.det.pluginRoot) {
	case binTargetOwnedSymlink, binTargetOwnedCopy, binTargetDevShim:
	default:
		return "", "the PATH entry on record, " + displayPath(rec.path) + ", is not abcd's own"
	}
	if ok, reason := entryTrust(rec.path); !ok {
		return "", displayPath(rec.path) + " is not safe for the harness to run on every refresh: " + reason
	}
	return rec.path, ""
}

// freshForWrite is freshDocForWrite with the bytes the write will replace:
// the document to render from, and the file's current bytes for the backup,
// read last of all, so the copy kept is the file the write replaces. A file
// that moved again between the two reads is, like any other move, a state
// nobody consented to, and ok is false.
func (hs harnessSettings) freshForWrite() (doc map[string]any, raw []byte, ok bool) {
	doc, ok = hs.freshDocForWrite()
	if !ok {
		return nil, nil, false
	}
	cur := readHarnessSettings()
	if cur.doc == nil || cur.path != hs.path || !reflect.DeepEqual(cur.doc, doc) {
		return nil, nil, false
	}
	return doc, cur.raw, true
}

// backupsRel is the folder in abcd's home the copies are kept in.
var backupsRel = abcdhome.Rel("backups")

// backupLeafPrefix names every copy: settings.json.<UTC stamp>.
const backupLeafPrefix = harnessSettingsFile + "."

// backupHarnessSettings keeps raw, the harness file's exact current bytes, in
// ~/.abcd.noindex/backups/settings.json.<UTC stamp>, private to the account,
// and returns its path. The folder is created and opened through the
// home-scope walk the setting write uses, so a symlinked level is refused
// rather than written through, and nothing is created outside abcd's home.
// The copy is created exclusively, never over an earlier one: two copies in
// one second take a numbered suffix.
func backupHarnessSettings(raw []byte) (string, error) {
	home, err := homeScopeErr()
	if err != nil {
		return "", err
	}
	dir, err := fsutil.EnsureHomeScope(home, backupsRel, abcdhome.DirMode)
	if err != nil {
		return "", err
	}
	defer dir.Close()
	stamp := backupLeafPrefix + time.Now().UTC().Format("20060102T150405Z")
	for n := 1; n <= 100; n++ {
		leaf := stamp
		if n > 1 {
			leaf += "-" + strconv.Itoa(n)
		}
		err := fsutil.CreateExclusiveIn(dir, leaf, raw, abcdhome.FileMode)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return abcdhome.Path(home, "backups", leaf), nil
	}
	return "", errors.New("a hundred copies already carry this second's stamp")
}

// replaceHarnessSettings is the guarded replace every status-line write goes
// through: keep a copy of current (the file's exact bytes now), then replace
// the file and read it back (replaceKeptHarnessSettings). It returns the
// copy's path — "" when none could be kept, and then nothing was written —
// and an error whose text completes a sentence ("…: <text>") and names the
// copy wherever one was kept.
func replaceHarnessSettings(path string, current []byte, before map[string]any, data []byte) (backup string, err error) {
	backup, err = keepHarnessCopy(path, current)
	if err != nil {
		return "", err
	}
	return backup, replaceKeptHarnessSettings(path, backup, current, before, data)
}

// keepHarnessCopy keeps current in abcd's backups folder before path is
// replaced, and refuses in plain words when it cannot: no copy, no write.
func keepHarnessCopy(path string, current []byte) (string, error) {
	backup, err := backupHarnessSettings(current)
	if err != nil {
		return "", &ahoyError{"a copy of " + displayPath(path) + " could not be kept in " + abcdhome.Display("backups") +
			"/ first (" + errText(err) + "), so nothing was written"}
	}
	return backup, nil
}

// replaceKeptHarnessSettings replaces path with data once its copy is kept at
// backup, reads it back, and puts current back when the read does not show
// what was written. before is the document data was rendered from: the
// read-back must parse as a JSON object, carry data's statusLine (or none,
// when data has none), and hold every other top-level key exactly as before.
func replaceKeptHarnessSettings(path, backup string, current []byte, before map[string]any, data []byte) error {
	shown := displayPath(backup)
	if err := writeHarnessSettings(path, data); err != nil {
		return &ahoyError{displayPath(path) + " could not be written (" + errText(err) + "); it is as it was, and a copy is at " + shown}
	}
	why := harnessReadBackMismatch(path, before, data)
	if why == "" {
		return nil
	}
	if rerr := fsutil.WriteFileAtomicPreserveMode(path, current); rerr != nil {
		return &ahoyError{displayPath(path) + " did not read back as written (" + why + "), and putting it back failed too (" +
			errText(rerr) + "): copy " + shown + " over it by hand"}
	}
	return &ahoyError{displayPath(path) + " did not read back as written (" + why + "), so it was put back from the copy at " + shown}
}

// harnessReadBackMismatch re-reads path after a write and says, in plain
// words, how it differs from what was meant, or "" when it does not.
func harnessReadBackMismatch(path string, before map[string]any, data []byte) string {
	raw, err := fsutil.ReadGuarded(path, maxHarnessSettingsBytes)
	if err != nil {
		return "it could not be read: " + errText(err)
	}
	after, err := decodeJSONObject(raw)
	if err != nil {
		return err.Error()
	}
	want, err := decodeJSONObject(data)
	if err != nil {
		return "what abcd meant to write is " + err.Error()
	}
	wantLine, wantHas := want[harnessStatusKey]
	gotLine, gotHas := after[harnessStatusKey]
	if wantHas != gotHas || !reflect.DeepEqual(wantLine, gotLine) {
		return "its " + harnessStatusKey + " is not the one abcd wrote"
	}
	for _, doc := range []map[string]any{before, after} {
		for k := range doc {
			if k == harnessStatusKey {
				continue
			}
			bv, bHas := before[k]
			av, aHas := after[k]
			if bHas != aHas || !reflect.DeepEqual(bv, av) {
				return "its " + termsafe.Sanitize(strconv.Quote(k)) + " setting is not what it was"
			}
		}
	}
	return ""
}
