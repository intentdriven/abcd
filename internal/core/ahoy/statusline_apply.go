package ahoy

// The status-line wiring `ahoy install` writes and `ahoy uninstall` restores
// (spc-70 ac-8; itd-200 "the install step asks"). Two files, one consent:
//
//   - ~/.abcd.noindex/statusline.json, the user-level setting — the bundled defaults
//     with the element switches the prompts took and `previous_command` set to
//     whatever the harness ran before, written 0600 because its reader refuses
//     a file others can write;
//   - <harness-home>/settings.json, where `statusLine` is pointed at
//     `'<entry>' statusline`, every other key preserved verbatim.
//
// The writes are one transaction: the harness file is parsed and both payloads
// rendered before either lands, so an unparseable harness file refuses before
// the setting is written, and a harness write that fails takes a setting this
// run created back out. Declining writes nothing, and nothing records the
// decline — the offer stands again next time.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// entryTrust is the trust check the status-line entry must pass before the
// harness is pointed at it (statusLineEntryTrust). A variable so a test can
// drive a refusal without arranging a file another account owns.
var entryTrust = statusLineEntryTrust

// writeHarnessSettings replaces the harness settings file atomically, keeping
// its mode. A variable so a test can make the write land something other than
// what was asked, which is what the read-back after it exists to catch.
var writeHarnessSettings = fsutil.WriteFileAtomicPreserveMode

// elementPromptPrefix keys the per-element on/off prompts, so a scripted answer
// stream and the transcript name the element they answer.
const elementPromptPrefix = "statusline."

// stepStatusLine wires the status line on consent, or repairs abcd's own line
// when it dangles or runs an abcd that fails the trust checks.
//
// The offer runs only against an answered prompt: --yes approves the category
// but never reaches here (autoYes), because the wiring rewrites a harness-wide
// user setting and takes element choices only a prompt can carry. The repair
// is ordinary ConfigChange work — required, and honoured under --yes — because
// a status command naming an abcd that is gone blanks the user's line in every
// repository, and one naming an abcd that fails the trust checks runs a binary
// nobody vouched for on every refresh of every session.
//
// It runs AFTER stepPathEntry, so the entry it points the harness at is the one
// this run actually left on PATH.
func (a *applyCtx) stepStatusLine() {
	if a.approved[ConfigChange] && (a.has(statusLineDanglingGapID) || a.has(StatusLineUntrustedGapID)) {
		a.repairStatusLine()
		return
	}
	if a.autoYes || !a.approved[StatusLine] || !a.has(StatusLineOfferGapID) {
		return
	}
	a.offerStatusLine()
}

// offerStatusLine is ac-8: the reason and the question, the element switches,
// then the two writes — and nothing at all on a decline. The harness file is
// re-read here rather than trusted from detection, and every refusal comes
// BEFORE the questions: there is nothing to consent to when the wiring cannot
// be written.
func (a *applyCtx) offerStatusLine() {
	hs := readHarnessSettings()
	switch hs.state {
	case statusLineNoHarness:
		return // the harness settings vanished since detection; nothing to offer
	case statusLineUnreadable:
		a.refuse("refused to wire the status line: " + displayPath(hs.path) + " " + hs.reason +
			"; nothing was written. Repair the file and re-run `abcd ahoy install`.")
		return
	case statusLineInstalled, statusLineDangling:
		return // already abcd's; the dangling repair has its own gap
	}
	if hs.line != nil && (hs.lineType != "command" || !hs.hasCommand) {
		a.refuse("refused to wire the status line: the statusLine in " + displayPath(hs.path) +
			" has type " + termsafe.Sanitize(strconv.Quote(hs.lineType)) + ", which abcd does not understand, so nothing was written. " +
			"Change or remove it by hand and re-run `abcd ahoy install`.")
		return
	}
	entry, why := a.statusLineEntry()
	if entry == "" {
		a.refuse("refused to wire the status line: " + why + "; nothing was written. " +
			"Put abcd on your PATH with `abcd ahoy install` first, then run it again to wire the status line.")
		return
	}
	if !a.prompter.Confirm(statusLineOfferQuestion(hs, entry)) {
		return // declined: nothing written, nothing recorded
	}
	switches := make(map[statusline.ElementKey]bool)
	for _, k := range statusline.Order() {
		if k == statusline.KeyPresence {
			continue // the badge is element one and is not switchable
		}
		ans := a.prompter.Prompt(elementPromptPrefix+string(k), []string{"on", "off"}, "on")
		// Default on: EOF, a bare Enter and anything that is not the word "off"
		// keep the element, so a run that says nothing loses nothing.
		switches[k] = strings.ToLower(strings.TrimSpace(ans)) != "off"
	}
	a.wireStatusLine(hs, entry, switches, hs.command)
}

// wireStatusLine performs the two writes as one transaction. Both payloads are
// rendered first and a copy of the harness file is kept in abcd's backups
// folder (no copy, no write); the setting lands, then the harness file, which
// is read back; a harness write that fails or does not read back as written
// puts the file back from the copy and removes a setting this run created, so
// no half-state survives.
//
// Two refusals come first. A previous command that reaches abcd's own status
// verb is never recorded: the status verb runs the previous command outside
// a managed checkout, so recording itself as previous makes it run itself
// without end (statusVerbRe). And the harness document is re-read immediately
// before the writes (freshForWrite): the snapshot hs predates the prompts,
// and a write the live harness made meanwhile is merged under, not reverted.
func (a *applyCtx) wireStatusLine(hs harnessSettings, entry string, switches map[statusline.ElementKey]bool, previous string) {
	if reachesStatusVerb(previous) {
		a.refuse("refused to wire the status line: the current status command in " + displayPath(hs.path) + ", " +
			termsafe.Sanitize(strconv.Quote(previous)) + ", reaches abcd's own status verb, and recording it as the previous command " +
			"would make that verb run itself without end; nothing was written. Edit the file to remove it first, then re-run `abcd ahoy install`.")
		return
	}
	settingPath := userStatusLineSettingPath()
	if settingPath == "" {
		a.refuse("the status line was not wired: the home directory could not be resolved, so " + statusline.SettingsDisplay + " has nowhere to go.")
		return
	}
	// The setting's reader refuses a file behind a symlinked ~/.abcd.noindex, so a write
	// through the link would land wherever the link points and never be read.
	if err := fsutil.HomeScopeLink(userHome(), statusline.SettingsRelPath); err != nil {
		a.refuse("refused to wire the status line: " + err.Error() + "; nothing was written.")
		return
	}
	settingBytes, created, err := statusLineSettingBytes(userHome(), switches, previous)
	if err != nil {
		a.refuse("refused to wire the status line: " + errText(err) + "; nothing was written.")
		return
	}
	doc, current, ok := hs.freshForWrite()
	if !ok {
		a.refuse("refused to wire the status line: " + displayPath(hs.path) + " changed while the prompts were open and its status line " +
			"is no longer what was read, so nothing was written; re-run `abcd ahoy install`.")
		return
	}
	harnessBytes, err := harnessSettingsBytes(doc, harnessLineWith(hs.line, statusCommandFor(entry)))
	if err != nil {
		a.refuse("refused to wire the status line: " + displayPath(hs.path) + " could not be re-encoded (" + errText(err) + "); nothing was written.")
		return
	}
	// The copy is kept before either file is written: a copy that cannot be
	// kept refuses the whole wiring, the user-level setting included.
	backup, pruneNote, err := keepHarnessCopy(hs.path, current)
	if err != nil {
		a.refuse("refused to wire the status line: " + err.Error() + ".")
		return
	}
	if pruneNote != "" {
		a.inform(pruneNote + ".")
	}
	// ~/.abcd.noindex is created, judged and opened relative to home's descriptor and
	// the setting is written (and, on a failed harness write, removed) through
	// it, so a link swapped in after the check above is refused rather than
	// written through (iss-2609281310017733).
	settingLeaf := path.Base(statusline.SettingsRelPath)
	var settingDir *os.Root
	if settingBytes != nil {
		var werr error
		settingDir, werr = fsutil.EnsureHomeScope(userHome(), path.Dir(statusline.SettingsRelPath), abcdhome.DirMode)
		if errors.Is(werr, fsutil.ErrHomeScopeSymlinked) {
			a.refuse("refused to wire the status line: " + werr.Error() + "; nothing was written.")
			return
		}
		if werr == nil {
			defer settingDir.Close()
			if created {
				// 0600, and never wider: the setting's reader refuses a file others
				// can write, so a default 0644 would be a setting that is never read.
				werr = fsutil.WriteFileAtomicInRoot(settingDir, settingLeaf, settingBytes, 0o600)
			} else {
				werr = fsutil.WriteFileAtomicPreserveModeInRoot(settingDir, settingLeaf, settingBytes)
			}
		}
		if werr != nil {
			a.refuse("could not write " + statusline.SettingsDisplay + " (" + errText(werr) + "); the status line was not wired.")
			return
		}
	}
	if err := replaceKeptHarnessSettings(hs.path, backup, current, doc, harnessBytes); err != nil {
		tail := "."
		if created && settingDir != nil {
			_ = settingDir.Remove(settingLeaf)
			tail = "; " + statusline.SettingsDisplay + " was not left behind."
		}
		a.refuse("refused to wire the status line: " + err.Error() + tail)
		return
	}
	a.note(writeStatusLineBackup, backup)
	if settingBytes != nil {
		a.note(writeStatusLine, settingPath)
	}
	a.note(writeStatusLine, hs.path)
}

// repairStatusLine closes the dangling and untrusted gaps: abcd's own status
// command names an abcd that is gone, or one that fails the trust checks
// (untrustedStatusLine), so it is repointed at the recorded, trusted PATH
// entry (statusLineEntry; never the plugin's own binary), or — when abcd has
// none to offer — handed back to the previous command the setting recorded,
// or removed outright when nothing was recorded (a blank line is what the
// dangling command already produces, and a missing key at least lets the
// harness render its own default rather than run a binary nobody vouched
// for). The command it replaces is never recorded as the previous one: it is
// abcd's own line, and a stale build recorded there would run again on every
// refresh outside a managed checkout. Like the wiring, it keeps a copy of the
// file first and reads the file back after (replaceHarnessSettings).
func (a *applyCtx) repairStatusLine() {
	hs := readHarnessSettings()
	if hs.state != statusLineDangling && !untrustedStatusLine(hs) {
		return // the state moved since detection; there is nothing to repair
	}
	var line map[string]any
	if entry, _ := a.statusLineEntry(); entry != "" {
		line = harnessLineWith(hs.line, statusCommandFor(entry))
	} else {
		previous, err := recordedPreviousCommand()
		if err != nil {
			a.refuse("refused to repair the status line: " + errText(err) + ", so " + displayPath(hs.path) + " was left as it is.")
			return
		}
		if previous != "" {
			line = harnessLineWith(hs.line, previous)
		}
	}
	doc, current, ok := hs.freshForWrite()
	if !ok {
		a.refuse("refused to repair the status line: " + displayPath(hs.path) + " changed under the repair and its status line " +
			"is no longer what was read, so it was left as it is; re-run `abcd ahoy install`.")
		return
	}
	data, err := harnessSettingsBytes(doc, line)
	if err != nil {
		a.refuse("refused to repair the status line: " + displayPath(hs.path) + " could not be re-encoded (" + errText(err) + ").")
		return
	}
	backup, pruneNote, err := replaceHarnessSettings(hs.path, current, doc, data)
	if pruneNote != "" {
		a.inform(pruneNote + ".")
	}
	if err != nil {
		a.refuse("refused to repair the status line: " + err.Error() + ".")
		return
	}
	a.note(writeStatusLineBackup, backup)
	a.note(writeStatusLine, hs.path)
}

// uninstallStatusLine is the uninstall half: a status line that is abcd's is
// handed back to the command recorded before abcd took the row, or removed when
// none was recorded. A foreign line is left alone, an absent harness has
// nothing to restore, and a setting that cannot be read leaves the harness
// untouched — the previous command is the one thing this restore must not
// guess. The user-level setting itself stays: it is the user's configuration.
// The restore keeps a copy of the file first, reads it back after, and names
// the copy on the receipt (replaceHarnessSettings).
func uninstallStatusLine() StatusLineReceipt {
	hs := readHarnessSettings()
	switch hs.state {
	case statusLineNoHarness:
		return StatusLineReceipt{Note: "no harness settings file; nothing to restore"}
	case statusLineUnreadable:
		return StatusLineReceipt{Note: displayPath(hs.path) + " " + hs.reason + "; left untouched"}
	case statusLineAbsent:
		return StatusLineReceipt{Note: "no status line configured; nothing to restore"}
	case statusLineForeign:
		return StatusLineReceipt{Note: "status line is not abcd's; left untouched"}
	}
	previous, err := recordedPreviousCommand()
	if err != nil {
		return StatusLineReceipt{Note: "the previous command was not restored: " + errText(err) + "; " + displayPath(hs.path) + " left untouched"}
	}
	var line map[string]any
	if previous != "" {
		line = harnessLineWith(hs.line, previous)
	}
	doc, current, ok := hs.freshForWrite()
	if !ok {
		return StatusLineReceipt{Note: displayPath(hs.path) + " changed under the restore and its status line is no longer what was read; " +
			"left untouched — re-run `abcd ahoy uninstall`"}
	}
	data, err := harnessSettingsBytes(doc, line)
	if err != nil {
		return StatusLineReceipt{Note: displayPath(hs.path) + " could not be re-encoded (" + errText(err) + "); left untouched"}
	}
	backup, pruneNote, err := replaceHarnessSettings(hs.path, current, doc, data)
	if err != nil {
		return StatusLineReceipt{Note: "restore failed: " + err.Error()}
	}
	kept := "; the file as it was is kept at " + displayPath(backup)
	if pruneNote != "" {
		kept += "; " + pruneNote
	}
	if previous == "" {
		return StatusLineReceipt{Restored: true, Note: "removed abcd's status line; none was configured before it" + kept}
	}
	return StatusLineReceipt{Restored: true, Note: "restored the previous status command" + kept}
}

// ---------------------------------------------------------------------------
// the two files
// ---------------------------------------------------------------------------

// statusLineSettingBytes renders the user-level setting to write, or nil when
// the existing file needs no change. A file that already exists is the user's
// configuration and is left as it is — save for an empty `previous_command`,
// which is filled from the harness so uninstall has something to restore. A
// new file is the bundled defaults with the switches the prompts took. The
// created flag says which, so a failed transaction knows what to take back.
//
// An existing file is read through the setting's own guarded reader, and one
// the guard refuses — not a regular file, writable by others, another uid's —
// is an error here rather than a file to fill: writing into a file that is
// not the caller's word would be taking somebody else's configuration as
// theirs.
func statusLineSettingBytes(home string, switches map[statusline.ElementKey]bool, previous string) (data []byte, created bool, err error) {
	raw, err := readUserStatusLineSetting(home)
	switch {
	case err != nil:
		return nil, false, err
	case raw == nil:
		set := statusline.Defaults()
		for k, on := range switches {
			set.Elements[k] = on
		}
		set.PreviousCommand = previous
		data, merr := json.MarshalIndent(set, "", "  ")
		if merr != nil {
			return nil, false, merr
		}
		return append(data, '\n'), true, nil
	}
	doc, err := decodeJSONObject(raw)
	if err != nil {
		return nil, false, &ahoyError{statusline.SettingsDisplay + " could not be read — " + err.Error()}
	}
	if pc, _ := doc["previous_command"].(string); pc != "" || previous == "" {
		return nil, false, nil // nothing to fill; the file stands
	}
	doc["previous_command"] = previous
	data, err = encodeJSONObject(doc)
	return data, false, err
}

// readUserStatusLineSetting reads the user-level setting through
// statusline.ReadSettingsFile — the ONE reader of that file, so the guard the
// row's render applies (regular file, not writable by others, this uid's) is
// the guard the install and uninstall steps apply. nil with no error is an
// absent file; a file the guard refuses is an error naming the reason, never
// a silent fallback, because what the callers take from the file is a shell
// command the harness will run.
func readUserStatusLineSetting(home string) ([]byte, error) {
	raw, why, err := statusline.ReadSettingsFile(home)
	if err != nil {
		return nil, err
	}
	if why != "" {
		return nil, &ahoyError{statusline.SettingsDisplay + " was not read — " + why}
	}
	return raw, nil
}

// recordedPreviousCommand reads `previous_command` out of the user-level
// setting: "" when the file is absent or records none, an error when the file
// is present and is refused by the guarded read, cannot be read as a JSON
// object, or records a command that reaches abcd's own status verb — handing
// that back to the harness is the recursion the wiring refuses to record,
// from the other end.
func recordedPreviousCommand() (string, error) {
	home := userHome()
	if home == "" {
		return "", nil
	}
	raw, err := readUserStatusLineSetting(home)
	if err != nil || raw == nil {
		return "", err
	}
	doc, err := decodeJSONObject(raw)
	if err != nil {
		return "", &ahoyError{statusline.SettingsDisplay + " could not be read — " + err.Error()}
	}
	previous, _ := doc["previous_command"].(string)
	if reachesStatusVerb(previous) {
		return "", &ahoyError{"the previous command recorded in " + statusline.SettingsDisplay + ", " +
			termsafe.Sanitize(strconv.Quote(previous)) + ", reaches abcd's own status verb, and handing it back to the harness " +
			"would make that verb run itself without end; edit the file to remove it first"}
	}
	return previous, nil
}

// harnessLineWith returns the statusLine object to write: a copy of the
// existing one (padding and whatever else it carries preserved) with the type
// and command set, or a fresh {type, command} when there was none.
func harnessLineWith(existing map[string]any, command string) map[string]any {
	line := make(map[string]any, len(existing)+2)
	for k, v := range existing {
		line[k] = v
	}
	line["type"] = "command"
	line["command"] = command
	return line
}

// harnessSettingsBytes renders the harness document with its statusLine set
// to line — or removed, when line is nil — and every other key untouched. The
// encoder does not escape HTML: a hook command carrying `&&` must come back
// out as `&&`, not `&&`.
func harnessSettingsBytes(doc map[string]any, line map[string]any) ([]byte, error) {
	out := make(map[string]any, len(doc)+1)
	for k, v := range doc {
		out[k] = v
	}
	if line == nil {
		delete(out, harnessStatusKey)
	} else {
		out[harnessStatusKey] = line
	}
	return encodeJSONObject(out)
}

// encodeJSONObject renders doc two-space indented with a trailing newline,
// keys sorted, HTML left alone.
func encodeJSONObject(doc map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// the local-ephemeral tier
// ---------------------------------------------------------------------------

// stepLocalTier creates .abcd/.work.local/ as a real directory, proving every
// level real rather than following a symlink (fsutil.EnsureRealDirAll). It
// runs after stepVisibility for the reason stepBanlist does: the tier is only
// worth having once the .gitignore fence that keeps it untracked is on disk.
//
// The proof starts at the checkout root resolved through its symlinks
// (fsutil.RealExistingPath), not at a.cwd: a.cwd is the shell's logical working
// directory, and a checkout entered through a symlinked path (`cd ~/proj` where
// ~/proj -> ~/src/proj) is the user's own, so only a symlink at or below the
// checkout's .abcd is refused (iss-2609261108448674). A refusal names the
// refused level repository-relative.
func (a *applyCtx) stepLocalTier() {
	if !a.approved[SafeAutocreate] || !a.has(localTierGapID) {
		return
	}
	root := fsutil.RealExistingPath(a.cwd)
	if err := fsutil.EnsureRealDirAll(root, localTierRelPath, 0o755); err != nil {
		reason := errText(err)
		var pe *os.PathError
		if errors.Is(err, fsutil.ErrNotRealDir) && errors.As(err, &pe) {
			level := "the checkout root"
			if rel, relErr := filepath.Rel(root, pe.Path); relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				level = filepath.ToSlash(rel)
			}
			reason = level + " is not a real directory (a symlink, or a file, stands there)"
		}
		a.refuse("refused to create " + localTierRelPath + "/: " + reason +
			". abcd never reaches the local tier through a symlink; remove what is there and re-run `abcd ahoy install`.")
		return
	}
	a.note(writeLocalTier, filepath.Join(a.cwd, filepath.FromSlash(localTierRelPath)))
}
