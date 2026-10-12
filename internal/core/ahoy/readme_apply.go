package ahoy

// Setup writes the stored README parts, the flag answers the pair, and
// uninstall removes them (spc-2610031704338417 step 3, for
// itd-2610031348087517).
//
// The owner's answer is `readme.parts` in .abcd/config.json:
//
//   - absent: not yet answered. Detection raises the optional readme.offered
//     gap. A person at a terminal, in a run that is not --yes, is asked the
//     question pair (readme_ask.go); every other run reports the gap under
//     optional_skipped with the flag that answers it;
//   - `[]`: answered no. Never asked again, and abcd's parts already in
//     README.md are removed as a narrowed set removes them;
//   - a list: the chosen parts, which every setup run derives from the project
//     and rewrites (decision 1).
//
// `--readme-parts` answers the pair in any run, --yes included: it is the
// owner's consent itself, so the readme category has no approval question
// (open question 5), and a value that differs from the stored one is echoed
// as a change and makes an otherwise current repository do the work.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Readme covers the README parts abcd keeps current. It is never asked as a
// category: the README question is the consent (open question 5).
const Readme GapCategory = "readme"

// ReadmeOfferGapID is the README question, raised while it is unanswered.
const ReadmeOfferGapID = "readme.offered"

// readmeFileName is the one README abcd keeps parts in (scope condition 1).
const readmeFileName = "README.md"

// ReadmePartsChoices is what --readme-parts takes, as a summary names it.
const ReadmePartsChoices = "licence|links|contents|release-badge|build-badge|abcd-badge|none"

// readmeNone is --readme-parts' answer of no to every part.
const readmeNone = "none"

// ParseReadmeParts reads a --readme-parts value: a comma-separated list of
// part names, returned once each in the fixed order, or `none`, returned as an
// empty, non-nil list. An unknown name, an empty one, or none beside a name is
// refused naming the known names.
func ParseReadmeParts(v string) ([]string, error) {
	known := strings.Join(readmePartOrder, ", ")
	if strings.TrimSpace(v) == readmeNone {
		return []string{}, nil
	}
	chosen := map[string]bool{}
	for _, name := range strings.Split(v, ",") {
		name = strings.TrimSpace(name)
		switch {
		case name == readmeNone:
			return nil, fmt.Errorf("--readme-parts: none stands alone, answering no to every part; it cannot sit beside a part name (known: %s)", known)
		case name == "":
			return nil, fmt.Errorf("--readme-parts names an empty part (known: %s, or none)", known)
		case readmePartIndex(name) < 0:
			return nil, fmt.Errorf("--readme-parts names %q, which is not a part abcd writes (known: %s, or none)", termsafe.Sanitize(name), known)
		}
		chosen[name] = true
	}
	out := []string{}
	for _, name := range readmePartOrder {
		if chosen[name] {
			out = append(out, name)
		}
	}
	return out, nil
}

// readmeAnswer is the README pair's answer for one run: what is stored, what
// the flag says, and the declared visibility the badges go by.
type readmeAnswer struct {
	stored    []string
	storedSet bool
	storedBad string // why a stored readme.parts cannot be read; "" when it can
	flag      []string
	flagSet   bool
	vis       string
}

// loadReadmeAnswer reads the stored answer and parses the flag (flag "" is
// the flag not given). An error is a config that cannot be read, or a flag
// value ParseReadmeParts refuses.
func loadReadmeAnswer(cwd, flag string) (readmeAnswer, error) {
	var a readmeAnswer
	if flag != "" {
		parts, err := ParseReadmeParts(flag)
		if err != nil {
			return a, err
		}
		a.flag, a.flagSet = parts, true
	}
	cfg, err := readConfig(cwd)
	if err != nil {
		return a, err
	}
	sub := subMap(cfg, "readme")
	a.vis, _ = stringVal(subMap(cfg, "repo"), "visibility")
	v, ok := sub["parts"]
	if !ok {
		return a, nil
	}
	a.storedSet = true
	list, isList := v.([]any)
	if !isList {
		a.storedBad = "readme.parts in .abcd/config.json is not a list of part names, so the README parts were left as they are; set it with abcd ahoy install --readme-parts " + ReadmePartsChoices
		return a, nil
	}
	a.stored = []string{}
	for _, e := range list {
		name, isName := e.(string)
		if !isName {
			a.storedBad = "readme.parts in .abcd/config.json holds something that is not a part name, so the README parts were left as they are; set it with abcd ahoy install --readme-parts " + ReadmePartsChoices
			return a, nil
		}
		a.stored = append(a.stored, name)
	}
	return a, nil
}

// parts is the set this run keeps: the flag's, else the stored one; false
// when neither answers (or the stored one cannot be read).
func (a readmeAnswer) parts() ([]string, bool) {
	switch {
	case a.flagSet:
		return a.flag, true
	case a.storedSet && a.storedBad == "":
		return a.stored, true
	}
	return nil, false
}

// flagChanges reports whether the flag differs from what is stored.
func (a readmeAnswer) flagChanges() bool {
	return a.flagSet && (!a.storedSet || a.storedBad != "" || !slices.Equal(a.flag, a.stored))
}

// readmePartsText renders a set for the echoed change.
func readmePartsText(parts []string, set bool) string {
	switch {
	case !set:
		return "not set"
	case len(parts) == 0:
		return readmeNone
	}
	return termsafe.Sanitize(strings.Join(parts, ","))
}

// detectReadme raises the README question while it is unanswered. A config
// abcd cannot read raises its own gap; this one waits for it.
func detectReadme(cwd string) []Gap {
	cfg, err := readConfig(cwd)
	if err != nil {
		return nil
	}
	if _, ok := subMap(cfg, "readme")["parts"]; ok {
		return nil
	}
	return []Gap{{
		ID: ReadmeOfferGapID, Category: Readme, Scope: "repo",
		Title: "the README question is not answered",
		Detail: "abcd can keep parts of README.md current that it reads from the project itself (a licence line, " +
			"links to the changelog and the other guides, a table of contents, and, for a project declared public, " +
			"release and build badges), each inside its own marked place, and only where the owner chooses them.",
		FixHint:  "answer it with abcd ahoy install --readme-parts " + ReadmePartsChoices + "; none answers no and is not asked again.",
		Required: false, Resolvable: true,
	}}
}

// readmeDerive derives the chosen parts, returning what the write wants: the
// bodies every check passed.
func readmeDerive(cwd string, parts []string, vis string) ([]readmeDerived, []readmePartWant) {
	derived := deriveReadmeParts(cwd, readmeSettings{Parts: parts, Visibility: vis})
	var want []readmePartWant
	for _, d := range derived {
		if d.Reason == "" {
			want = append(want, readmePartWant{Name: d.Name, Body: d.Body})
		}
	}
	return derived, want
}

// readmeWouldChange reports whether setup has README work to do in an
// otherwise current repository: a --readme-parts value that differs from the
// stored answer, a readme.created whose file is gone, or a chosen part that
// would be written, rewritten or removed.
// It plans without writing.
func readmeWouldChange(cwd string, opts InstallOptions) bool {
	ans, err := loadReadmeAnswer(cwd, opts.ReadmeParts)
	if err != nil {
		return false
	}
	if ans.flagChanges() || readmeRecordOutlivesFile(cwd) {
		return true
	}
	parts, ok := ans.parts()
	if !ok {
		return false
	}
	_, want := readmeDerive(cwd, parts, ans.vis)
	_, res, err := planReadmeFile(filepath.Join(cwd, readmeFileName), want)
	return err == nil && res.Wrote
}

// readmeStandingNotes is what a run with no README work still says: each
// chosen part omitted and why, and each part left as the file holds it. The
// early return carries them, since no apply step runs to say them.
func readmeStandingNotes(cwd string, opts InstallOptions) []string {
	ans, err := loadReadmeAnswer(cwd, opts.ReadmeParts)
	if err != nil {
		return nil
	}
	parts, ok := ans.parts()
	if !ok {
		if ans.storedBad != "" {
			return []string{ans.storedBad}
		}
		return nil
	}
	derived, want := readmeDerive(cwd, parts, ans.vis)
	_, res, perr := planReadmeFile(filepath.Join(cwd, readmeFileName), want)
	var notes []string
	notes = append(notes, readmeDerivedNotes(derived)...)
	if perr != nil {
		return append(notes, readmeLeftNote(cwd, perr))
	}
	for _, o := range res.Parts {
		if o.Action == readmeActionLeft || o.Action == readmeActionRefused {
			notes = append(notes, o.Note)
		}
	}
	return notes
}

// readmeDerivedNotes is each omitted part's reason and each note a derivation
// made. Both are terminal-safe as derived; a part's body is never in them.
func readmeDerivedNotes(derived []readmeDerived) []string {
	var notes []string
	for _, d := range derived {
		if d.Reason != "" {
			notes = append(notes, d.Reason)
		}
		notes = append(notes, d.Notes...)
	}
	return notes
}

func readmeLeftNote(cwd string, err error) string {
	return "README.md was left as it is: " + errText(withoutFolder(filepath.Join(cwd, readmeFileName), err))
}

// stepReadme asks the question pair while it is unanswered, of a person at a
// terminal (askReadme), stores that answer or a --readme-parts one, then
// writes the chosen parts into README.md. It runs after stepMarker and before
// stepSymlink, the place the README pair takes in setup's fixed order
// (decision 7).
func (a *applyCtx) stepReadme() {
	ans, err := loadReadmeAnswer(a.cwd, a.readmeFlag)
	if err != nil {
		if a.readmeFlag != "" {
			if _, ferr := ParseReadmeParts(a.readmeFlag); ferr != nil {
				a.refuse(ferr.Error())
				return
			}
		}
		a.refuseMalformedConfig(err)
		return
	}
	a.forgetGoneReadme()
	if a.asksReadme(ans) {
		parts, answered := a.askReadme(ans.vis)
		if !answered {
			return // decided later: nothing stored, so the next run asks again
		}
		// The answer is stored as the flag's would be, and echoed the same way.
		ans.flag, ans.flagSet = parts, true
	}
	if ans.flagChanges() && !a.storeReadmeParts(ans) {
		return
	}
	parts, ok := ans.parts()
	if !ok {
		if ans.storedBad != "" {
			a.refuse(ans.storedBad)
		}
		return
	}
	a.syncReadme(parts, ans.vis)
}

// storeReadmeParts saves the answer, the flag's or the one the pair was
// given, under the config's lock, echoing the change. Nothing is written into
// README.md for an answer not saved.
func (a *applyCtx) storeReadmeParts(ans readmeAnswer) bool {
	err := withConfigLock(a.cwd, func() error {
		cfg, err := readConfig(a.cwd)
		if err != nil {
			return err
		}
		if cfg == nil {
			return errors.New(".abcd/config.json does not exist yet; answer y to creating abcd's starter files and run abcd ahoy install again")
		}
		setSub(cfg, "readme", "parts", ans.flag)
		return writeConfig(a.cwd, cfg)
	})
	if err != nil {
		a.refuse("the README answer was not saved, so README.md was left as it is: " + errText(err))
		return false
	}
	a.echoChange("readme.parts", readmePartsText(ans.stored, ans.storedSet && ans.storedBad == ""), readmePartsText(ans.flag, true))
	a.note(writeSettings, configPath(a.cwd))
	return true
}

// syncReadme derives the chosen parts and makes README.md hold exactly the
// ones derived: written, rewritten, kept, or removed when no longer chosen or
// derivable, each part edited by hand left and named. A README created here is
// recorded as abcd's (open question 4).
func (a *applyCtx) syncReadme(parts []string, vis string) {
	derived, want := readmeDerive(a.cwd, parts, vis)
	for _, n := range readmeDerivedNotes(derived) {
		a.inform(n)
	}
	path := filepath.Join(a.cwd, readmeFileName)
	res, err := syncReadmeParts(path, want)
	if err != nil {
		a.refuse(readmeLeftNote(a.cwd, err))
		return
	}
	wrote := false
	for _, o := range res.Parts {
		switch o.Action {
		case readmeActionWritten, readmeActionRewritten:
			wrote = true
			a.inform(o.Note)
		case readmeActionRemoved:
			a.inform(o.Note)
		case readmeActionLeft, readmeActionRefused:
			a.refuse(o.Note)
		}
	}
	if !res.Wrote {
		return
	}
	switch {
	case res.Created:
		a.note(writeReadmeCreated, path)
		a.recordReadmeCreated(true)
	case wrote:
		a.note(writeReadmeParts, path)
	default:
		a.note(writeReadmePartsRemoved, path)
	}
}

// recordReadmeCreated sets or clears readme.created, the record that README.md
// is a file abcd made, which uninstall may delete once it holds nothing.
func (a *applyCtx) recordReadmeCreated(created bool) {
	if err := setReadmeCreated(a.cwd, created); err != nil {
		a.refuse("abcd created README.md but could not record that it did, so uninstall will leave the file: " + errText(err))
	}
}

// forgetGoneReadme clears readme.created when README.md is gone. The record
// vouches for the one file abcd made; a README.md written after it is the
// owner's, which uninstall must never delete on its word.
func (a *applyCtx) forgetGoneReadme() {
	if !readmeRecordOutlivesFile(a.cwd) {
		return
	}
	if err := setReadmeCreated(a.cwd, false); err != nil {
		a.refuse("README.md, which abcd created, is gone, but " + readmeRecordStands(err) + ", so uninstall would delete a blank README.md written in its place")
		return
	}
	a.note(writeSettings, configPath(a.cwd))
}

// readmeRecordOutlivesFile reports whether readme.created is recorded while
// README.md is gone.
func readmeRecordOutlivesFile(cwd string) bool {
	if !readmeCreatedByAbcd(cwd) {
		return false
	}
	_, err := os.Lstat(filepath.Join(cwd, readmeFileName))
	return errors.Is(err, fs.ErrNotExist)
}

// readmeRecordStands says that readme.created could not be cleared, and why.
func readmeRecordStands(err error) string {
	return "the record that abcd created it was not cleared: " + errText(err)
}

// setReadmeCreated sets readme.created, or removes it, under the config's
// lock.
func setReadmeCreated(cwd string, created bool) error {
	return withConfigLock(cwd, func() error {
		cfg, err := readConfig(cwd)
		if err != nil {
			return err
		}
		if cfg == nil {
			return errors.New(".abcd/config.json does not exist")
		}
		sub := subMap(cfg, "readme")
		if created {
			sub["created"] = true
		} else {
			if _, ok := sub["created"]; !ok {
				return nil
			}
			delete(sub, "created")
		}
		cfg["readme"] = sub
		return writeConfig(cwd, cfg)
	})
}

// ReadmeReceipt records what uninstall did with README.md's parts.
type ReadmeReceipt struct {
	Removed []string         `json:"removed"` // parts taken out, their text matching its hash
	Left    []ReadmePartLeft `json:"left"`    // parts left, each with its reason and line
	Deleted bool             `json:"deleted"` // a README.md abcd created, emptied and deleted
	Note    string           `json:"note"`    // the receipt in words, terminal-safe
}

// ReadmePartLeft is one part uninstall left in README.md.
type ReadmePartLeft struct {
	Part   string `json:"part"`
	Reason string `json:"reason"`
	Line   int    `json:"line"`
}

// uninstallReadme removes every part whose text still matches its hash,
// whatever the stored answer says, leaves and names the others, and deletes a
// README.md abcd created once nothing but blank lines remains. A duplicated
// or unclosed mark leaves the file untouched and names the line; an owner's
// README.md holding no part is never written. Once the file abcd created is
// gone, deleted here or found gone, readme.created is cleared from
// .abcd/config.json, and a record that cannot be cleared is said in the note.
func uninstallReadme(cwd string) ReadmeReceipt {
	r := ReadmeReceipt{Removed: []string{}, Left: []ReadmePartLeft{}}
	path := filepath.Join(cwd, readmeFileName)
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		r.Note = "no README.md"
		if readmeCreatedByAbcd(cwd) {
			if err := setReadmeCreated(cwd, false); err != nil {
				r.Note += "; " + readmeRecordStands(err)
			}
		}
		return r
	}
	res, err := removeReadmeParts(path)
	if err != nil {
		r.Note = readmeLeftNote(cwd, err)
		return r
	}
	var words []string
	var faults []string
	for _, o := range res.Parts {
		line := 0
		if len(o.Lines) > 0 {
			line = o.Lines[0]
		}
		switch o.Action {
		case readmeActionRemoved:
			r.Removed = append(r.Removed, o.Name)
		case readmeActionLeft:
			r.Left = append(r.Left, ReadmePartLeft{Part: o.Name, Reason: string(o.State), Line: line})
			words = append(words, o.Name+" "+string(o.State)+", left")
		case readmeActionRefused:
			r.Left = append(r.Left, ReadmePartLeft{Part: o.Name, Reason: string(o.State), Line: line})
			faults = append(faults, o.Note)
		}
	}
	if len(r.Removed) > 0 {
		words = append([]string{strings.Join(r.Removed, ", ") + " removed"}, words...)
	}
	words = append(words, faults...)
	if !res.Refused && readmeCreatedByAbcd(cwd) {
		switch deleted, derr := deleteEmptyReadme(path); {
		case derr != nil:
			words = append(words, "README.md, which abcd created, was kept: "+errText(withoutFolder(path, derr)))
		case deleted:
			r.Deleted = true
			words = append(words, "README.md, which abcd created, deleted")
			// The record vouched for a file that is gone; left, it would vouch
			// for the next README.md, which is the owner's.
			if err := setReadmeCreated(cwd, false); err != nil {
				words = append(words, readmeRecordStands(err))
			}
		}
	}
	if len(words) == 0 {
		words = []string{"no abcd part in README.md"}
	}
	r.Note = strings.Join(words, "; ")
	return r
}

// readmeCreatedByAbcd reports whether the config records that abcd made
// README.md. A config abcd cannot read proves nothing.
func readmeCreatedByAbcd(cwd string) bool {
	cfg, err := readConfig(cwd)
	if err != nil {
		return false
	}
	created, _ := boolVal(subMap(cfg, "readme"), "created")
	return created
}

// deleteEmptyReadme deletes the README at path when it is a regular file
// holding nothing but blank lines, under the file's lock.
func deleteEmptyReadme(path string) (bool, error) {
	deleted := false
	err := withRewriteLock(path, func() error {
		fi, err := os.Lstat(path)
		if err != nil || !fi.Mode().IsRegular() {
			return nil
		}
		data, err := fsutil.ReadGuarded(path, maxAhoyFileBytes)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(data)) != 0 {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted, err
}
