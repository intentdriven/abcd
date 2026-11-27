// Package refusal is the registry of abcd's refusal codes
// (itd-2610301428378830, spc-2610301431108175 step 1).
//
// When abcd refuses a call (exit 2, or exit 3 when someone else holds what
// the call needs), the refusal carries a short code, a one-line hint and the
// next command to try, and `abcd explain <code>` prints the code's
// explanation. Every code is one entry of the table below, and the front
// door reads the hint, the next command and the explanation from it, so a
// refusal and its explanation cannot disagree.
//
// A code is a short name, never a number (adr-2610301638558260): the verb, a
// slash and the reason, in lower-case kebab-case, such as
// capture/missing-remedy, or usage/... for the command framework's own
// refusals. Once a release has carried a code it is never renamed and never
// given to a different refusal: a refusal that changes meaning gets a new
// code, and one that goes away keeps its entry, marked retired, so explain
// still answers and the name is never handed out again. The words of an
// entry can change in any release; only the code is a promise.
//
// The registry is held by tests: no two entries share a code, every code
// follows the grammar, every Code constant is registered, and the committed
// ledger (LedgerPath), written from the table by cmd/refusal-codes-sync, is
// the one the release guard reads at the last tag.
//
// The package is transport-free: it holds data and two small helpers, Wrap
// and CodeOf, which let a core package name the code where the refusal is
// decided and the front door find it however the error was wrapped.
package refusal

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Code is a refusal code: `<verb>/<reason>`.
type Code string

// None is the absence of a code. A refusal built with None at exit 2 or 3 is
// carried out as Uncoded.
const None Code = ""

// Uncoded is the code a refusal carries when it was built without one at
// exit 2 or 3. It is never registered, so `abcd explain` refuses it and the
// tests that hold the registry and the front door fail on it.
const Uncoded Code = "internal/uncoded"

// NoNext is the next command of an entry where no command can help, such as
// a refusal that says to wait for someone else.
const NoNext = "none"

// CommandSlot is the placeholder in a next-command template that FillNext
// replaces with the refused call's own command path, below the root. It is
// the only part of a next command filled at run time, and it is never filled
// from the caller's arguments.
const CommandSlot = "{command}"

// The registered codes of the table below. Each constant is an entry of the
// table, or of a family file's register call; a test fails on a constant the
// registry does not hold.
const (
	// The command framework's usage refusals, raised by every command.
	UsageUnknownCommand      Code = "usage/unknown-command"
	UsageUnknownFlag         Code = "usage/unknown-flag"
	UsageMissingArgument     Code = "usage/missing-argument"
	UsageExtraArgument       Code = "usage/extra-argument"
	UsageFlagConflict        Code = "usage/flag-conflict"
	UsageMissingRequiredFlag Code = "usage/missing-required-flag"
	UsageInvalidFlagValue    Code = "usage/invalid-flag-value"

	// capture's own refusals.
	CaptureMissingRemedy         Code = "capture/missing-remedy"
	CaptureRemedyMachineSpecific Code = "capture/remedy-machine-specific"
	CaptureUnknownIssue          Code = "capture/unknown-issue"
	// CaptureUnknownReadingRecord is an id the reading ledger does not hold:
	// a reading item, an admission, a disposition or a reframe.
	CaptureUnknownReadingRecord Code = "capture/unknown-reading-record"
	CaptureNotOpen              Code = "capture/not-open"
	CaptureInsufficientGrounds  Code = "capture/insufficient-grounds"
	CaptureInvalidInput         Code = "capture/invalid-input"
	CaptureNotCharacterized     Code = "capture/not-characterized"

	// explain's own refusals.
	ExplainUnknownPath       Code = "explain/unknown-path"
	ExplainUnrecognizedInput Code = "explain/unrecognized-input"
	ExplainUnknownCode       Code = "explain/unknown-code"

	// start's own refusals.
	StartOutsideATerminal  Code = "start/outside-a-terminal"
	StartProfileUnusable   Code = "start/profile-unusable"
	StartRepositoryUnready Code = "start/repository-unready"
	StartHostNotRunnable   Code = "start/host-not-runnable"
	StartNotLoggedIn       Code = "start/not-logged-in"
	StartNoVerdict         Code = "start/no-verdict"
	StartWallNotHeld       Code = "start/wall-not-held"
)

// Entry is one registered code.
type Entry struct {
	// Code is the code itself, the one part a release promises.
	Code Code `json:"code"`
	// Summary is the one-line hint a refusal carrying the code shows.
	Summary string `json:"summary"`
	// Explanation is a few lines `abcd explain <code>` prints: what the
	// refusal means and how to get past it.
	Explanation string `json:"explanation"`
	// Next is the command to try next, or NoNext where no command can help.
	// It may hold CommandSlot, which the front door fills. A next holding a
	// placeholder (`<fix>`, a word in angle brackets) is a template whose
	// placeholders the caller fills in before running it (IsTemplate); any
	// other next runs as it stands.
	Next string `json:"next"`
	// RaisedBy names the command paths that raise the code. It is empty for
	// a usage code, which every command raises.
	RaisedBy []string `json:"raised_by,omitempty"`
	// Retired is the release tag the code was retired in, empty while the
	// code is live. A retired entry stays in the table for good.
	Retired string `json:"retired,omitempty"`
	// ReplacedBy is, for a retired code, the code that took its place, if
	// one did.
	ReplacedBy Code `json:"replaced_by,omitempty"`
}

// explainCommand is the next command of every usage code: the refused
// command's own facts.
const explainCommand = "abcd explain " + CommandSlot

// captureNew is the next command of a refusal of a new issue.
const captureNew = `abcd capture "<text>" --remedy "<fix>"`

// entries is the registry, in the order explain reads it: this table, then
// each family file's entries (register). The ledger sorts it by code.
var entries = []Entry{
	{
		Code:    UsageUnknownCommand,
		Summary: "the first argument names no command or sub-command here",
		Explanation: "The command takes no positional argument, or the word given names none of its\n" +
			"sub-commands. The refusal names the command and lists the sub-commands it has.\n" +
			"Run `abcd explain` on the command to see what it takes.",
		Next: explainCommand,
	},
	{
		Code:    UsageUnknownFlag,
		Summary: "a flag given is not one this command takes",
		Explanation: "The command does not declare the flag, or an argument begins with `---` or `--=`\n" +
			"and so names no flag. A flag may have been renamed or retired since the page you\n" +
			"read was written; the running binary's `abcd explain` lists the flags it takes.",
		Next: explainCommand,
	},
	{
		Code:    UsageMissingArgument,
		Summary: "the command needs more positional arguments than were given",
		Explanation: "The command's usage line names positional arguments it needs, and fewer were\n" +
			"given, or one given was empty. Run `abcd explain` on the command to see its usage\n" +
			"line and a worked example.",
		Next: explainCommand,
	},
	{
		Code:    UsageExtraArgument,
		Summary: "the command takes fewer positional arguments than were given",
		Explanation: "More positional arguments were given than the command's usage line allows. A\n" +
			"value meant for a flag may have lost its flag, or text with spaces may need\n" +
			"quoting. Run `abcd explain` on the command to see its usage line.",
		Next: explainCommand,
	},
	{
		Code:    UsageFlagConflict,
		Summary: "two flags were given that cannot be used together",
		Explanation: "The command declares the flags named in the refusal as alternatives: set one of\n" +
			"them, not both. Run `abcd explain` on the command to see its flags.",
		Next: explainCommand,
	},
	{
		Code:    UsageMissingRequiredFlag,
		Summary: "a flag the command needs was not given",
		Explanation: "The command needs the flag the refusal names, or one flag of the group it names,\n" +
			"and none was given. Run `abcd explain` on the command to see which flags are\n" +
			"required and the values each takes.",
		Next: explainCommand,
	},
	{
		Code:    UsageInvalidFlagValue,
		Summary: "a flag's value is missing, or is not one the flag takes",
		Explanation: "The flag needs a value and none followed it, or the value does not parse as the\n" +
			"flag's type or is not one of the values it accepts. The refusal names the flag\n" +
			"and, for a closed set, the values it accepts; `abcd explain` lists them too.",
		Next: explainCommand,
	},
	{
		Code:    CaptureMissingRemedy,
		Summary: "a new issue needs --remedy, the fix it proposes, in one line",
		Explanation: "Every new issue carries the fix it proposes. Give it with --remedy \"<fix>\" in\n" +
			"one line; nothing was captured. `abcd capture remedy <iss-N> <text>` rewrites it\n" +
			"later.",
		Next:     captureNew,
		RaisedBy: []string{"abcd capture"},
	},
	{
		Code:    CaptureRemedyMachineSpecific,
		Summary: "the --remedy given is the value only abcd's automatic filers write",
		Explanation: "That value marks an issue an automatic filer wrote with no fix, and a drain skips\n" +
			"such an issue. Name the fix the issue proposes instead; nothing was captured.",
		Next:     captureNew,
		RaisedBy: []string{"abcd capture"},
	},
	{
		Code:    CaptureUnknownIssue,
		Summary: "no issue in this checkout's ledger has that id",
		Explanation: "The issue id (iss-N) is in none of the ledger's status folders here. Check the\n" +
			"id, or list the ledger; when the record is on another branch, `abcd peers` shows\n" +
			"where. An id of the reading ledger (rdi-N, adm-N, dsp-N, rfm-N) carries\n" +
			"capture/unknown-reading-record instead.",
		Next: "abcd capture list --all",
		RaisedBy: []string{"abcd capture defer", "abcd capture link", "abcd capture owed", "abcd capture promote",
			"abcd capture remedy", "abcd capture resolve", "abcd capture wontfix"},
	},
	{
		Code:    CaptureUnknownReadingRecord,
		Summary: "no reading item, admission, disposition or reframe here has that id",
		Explanation: "The id (rdi-N, adm-N, dsp-N or rfm-N) names no record the reading ledger holds\n" +
			"here. Check the id and its kind: `abcd explain` on the command shows which kind\n" +
			"each argument takes. When the record is on another branch, `abcd peers` shows\n" +
			"where.",
		Next: explainCommand,
		RaisedBy: []string{"abcd capture admit", "abcd capture disposition", "abcd capture promote",
			"abcd capture reframe", "abcd capture surprise"},
	},
	{
		Code:    CaptureNotOpen,
		Summary: "the issue is no longer open, so it cannot move again",
		Explanation: "The issue was found but not in open/: it was already resolved or wontfixed, or\n" +
			"another run moved it while this one worked. Nothing was written. List the ledger\n" +
			"to see where it stands.",
		Next:     "abcd capture list --all",
		RaisedBy: []string{"abcd capture resolve", "abcd capture wontfix", "abcd capture defer", "abcd capture remedy"},
	},
	{
		Code:    CaptureInsufficientGrounds,
		Summary: "the grounds given are missing, outside their vocabulary or too thin",
		Explanation: "The verb records the grounds for what it does, and those given were absent, not one\n" +
			"of the accepted kinds, malformed, or below the substance floor. Nothing was\n" +
			"written. The refusal names the accepted kinds; `abcd explain` shows the flag.",
		Next: explainCommand,
		RaisedBy: []string{"abcd capture admit", "abcd capture promote", "abcd capture reframe",
			"abcd capture resolve", "abcd capture surprise", "abcd capture wontfix"},
	},
	{
		Code:    CaptureInvalidInput,
		Summary: "a value the request carries is outside its shape or names nothing",
		Explanation: "An impact outside its set, a provenance reference that is malformed or names no\n" +
			"record, a release tag or a production mode that is not one: the refusal names\n" +
			"which. Nothing was written.",
		Next: explainCommand,
		RaisedBy: []string{"abcd capture", "abcd capture admit", "abcd capture defer",
			"abcd capture disposition", "abcd capture link", "abcd capture mentions",
			"abcd capture owed", "abcd capture reframe", "abcd capture remedy", "abcd capture resolve",
			"abcd capture surprise"},
	},
	{
		Code:    CaptureNotCharacterized,
		Summary: "the item has no committed comparative run yet, so it cannot be admitted",
		Explanation: "A disposition or an admission at the widening position waits for a committed\n" +
			"comparative run that names the item. The refusal names the run it waits on;\n" +
			"commit that run first. Nothing was written.",
		Next:     NoNext,
		RaisedBy: []string{"abcd capture disposition", "abcd capture admit"},
	},
	{
		Code:    ExplainUnknownPath,
		Summary: "the words name no command in this binary's tree",
		Explanation: "A word of the path names no sub-command of the command before it, or follows a\n" +
			"command that takes none. The refusal names the nearest real path, or the\n" +
			"sub-commands where the path went wrong.",
		Next:     "abcd explain abcd",
		RaisedBy: []string{"abcd explain"},
	},
	{
		Code:    ExplainUnrecognizedInput,
		Summary: "the command reads no filled-in input through that flag",
		Explanation: "--input names a flag through which the command reads a filled-in JSON input. The\n" +
			"refusal names the inputs the command does read, or says it reads none.",
		Next:     explainCommand,
		RaisedBy: []string{"abcd explain"},
	},
	{
		Code:    ExplainUnknownCode,
		Summary: "no refusal code by that name is registered in this binary",
		Explanation: "A word holding a slash is read as a refusal code, and this binary registers none\n" +
			"by that name. The refusal names the nearest registered code. A code from a newer\n" +
			"release is unknown to an older binary.",
		Next:     NoNext,
		RaisedBy: []string{"abcd explain"},
	},
	{
		Code:    StartOutsideATerminal,
		Summary: "a host session's marker is set, or a stream is not a terminal",
		Explanation: "A host session's marker is set in the environment, or standard input, output or\n" +
			"error is not a terminal. Those two are all that is read: they catch a start made by\n" +
			"mistake and prove nothing about who typed it. A person types `abcd start` in a plain\n" +
			"terminal. Nothing was started.",
		Next:     NoNext,
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartProfileUnusable,
		Summary: "the profile is not one a session can start on",
		Explanation: "The account holds several profiles and none was named, holds none by the name\n" +
			"given, or the directory is one abcd does not trust or that holds something a launch\n" +
			"would load. The refusal says which. Nothing was started.",
		Next:     NoNext,
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartRepositoryUnready,
		Summary: "this is not a managed checkout with a conventions file to hand over",
		Explanation: "A session starts in a git checkout abcd manages, with AGENTS.md at its root: a\n" +
			"regular file, not a link, with no NUL byte, at most 100 KiB. The refusal says which\n" +
			"does not hold. Nothing was started.",
		Next:     "abcd ahoy",
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartHostNotRunnable,
		Summary: "the host, or the copy of abcd its hooks run, cannot be run",
		Explanation: "The host is not on PATH or lies where abcd never runs one, abcd could not name or\n" +
			"keep a copy of its own binary, or the machine's runner configuration could not be\n" +
			"read. The refusal says which. Nothing was started.",
		Next:     NoNext,
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartNotLoggedIn,
		Summary: "the profile is not logged in, and the login is the person's",
		Explanation: "The refusal prints the host's own login command for the profile, which the person\n" +
			"runs once in their own shell and which opens no session. abcd never performs the\n" +
			"login and never reads, stores or forwards its token. Nothing was started.",
		Next:     NoNext,
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartNoVerdict,
		Summary: "the host gave no answer a start can stand on",
		Explanation: "The host did not answer about the login, or the short launch met a limit, did not\n" +
			"finish, was interrupted, or could not be written to the launch ledger. It says\n" +
			"nothing about the profile. Nothing waits and nothing was tried again.",
		Next:     "abcd start",
		RaisedBy: []string{"abcd start"},
	},
	{
		Code:    StartWallNotHeld,
		Summary: "the short launch, or this checkout's guard, did not show that the wall holds",
		Explanation: "The launch held a tool, a server, a plugin or a key the list does not name, loaded\n" +
			"something from its working tree, reported no list to read, or did not show the guard\n" +
			"refusing a command it is meant to refuse; or the guard in this checkout is off or\n" +
			"does not refuse that command. The refusal names which. No session was opened.",
		Next:     NoNext,
		RaisedBy: []string{"abcd start"},
	},
}

// register adds one family's entries to the registry. The table above holds
// the usage, capture and explain codes; every other verb family keeps its
// constants and its entries in a file of its own, codes_<family>.go, whose
// init calls register, so two changes that each add a family's codes touch
// no common line. Go sets every package-level variable before it runs any
// init, and runs a package's init functions in the order of their files'
// names, so the registry order is the table, then each family file in name
// order, each file's entries in the order it lists them (held by
// TestEntriesLoadInFileOrder).
func register(es ...Entry) { entries = append(entries, es...) }

// Entries returns every registered entry, in registry order. The slice and
// its entries are copies.
func Entries() []Entry {
	out := make([]Entry, len(entries))
	for i, e := range entries {
		e.RaisedBy = slices.Clone(e.RaisedBy)
		out[i] = e
	}
	return out
}

// Lookup returns the entry registered for c.
func Lookup(c Code) (Entry, bool) {
	for _, e := range entries {
		if e.Code == c {
			e.RaisedBy = slices.Clone(e.RaisedBy)
			return e, true
		}
	}
	return Entry{}, false
}

// nameRe is a code's grammar: `<verb>/<reason>`, each part lower-case
// kebab-case.
var nameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*/[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// validName reports whether s follows the code grammar.
func validName(s string) bool { return nameRe.MatchString(s) }

// placeholderRe is a placeholder of a next-command template: a word in angle
// brackets that the caller replaces with a value of their own.
var placeholderRe = regexp.MustCompile(`<[a-z][a-z0-9-]*>`)

// IsTemplate reports whether next is a template, a command holding
// placeholders the caller fills in before running it, rather than a command
// to run as it stands.
func IsTemplate(next string) bool { return placeholderRe.MatchString(next) }

// FillNext fills a next-command template from the refused call's command
// path (`abcd capture remedy`): CommandSlot becomes the words below the root,
// or the root's own name for the root, as `abcd explain` takes them.
func FillNext(next, commandPath string) string {
	if !strings.Contains(next, CommandSlot) {
		return next
	}
	words := commandPath
	if _, rest, ok := strings.Cut(commandPath, " "); ok {
		words = rest
	}
	return strings.ReplaceAll(next, CommandSlot, words)
}

// codedError carries a code on an error a core package returns.
type codedError struct {
	code Code
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

// Wrap returns err carrying the code c, keeping its message and its chain,
// so the front door finds the code with CodeOf however the error is wrapped
// on the way out. A nil err stays nil.
func Wrap(c Code, err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: c, err: err}
}

// CodeOf returns the code Wrap put on err, or None.
func CodeOf(err error) Code {
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return None
}

// LedgerPath is the committed ledger of codes, from the repository root. The
// release guard reads it at the last tag and refuses a cut in which a code it
// lists is missing.
const LedgerPath = ".abcd/development/release/refusal-codes.json"

// LedgerFormatVersion is the ledger's format version.
const LedgerFormatVersion = 1

// Ledger is the committed list of codes: only the codes and their retirement,
// since an entry's words may change in any release and the code may not.
type Ledger struct {
	FormatVersion int           `json:"format_version"`
	Codes         []LedgerEntry `json:"codes"`
}

// LedgerEntry is one code of the ledger.
type LedgerEntry struct {
	Code       Code   `json:"code"`
	Retired    string `json:"retired,omitempty"`
	ReplacedBy Code   `json:"replaced_by,omitempty"`
}

// BuildLedger is the ledger of the registry, sorted by code.
func BuildLedger() Ledger { return ledgerOf(entries) }

// ledgerOf is the ledger of entries, sorted by code.
func ledgerOf(entries []Entry) Ledger {
	l := Ledger{FormatVersion: LedgerFormatVersion, Codes: make([]LedgerEntry, 0, len(entries))}
	for _, e := range entries {
		l.Codes = append(l.Codes, LedgerEntry{Code: e.Code, Retired: e.Retired, ReplacedBy: e.ReplacedBy})
	}
	slices.SortFunc(l.Codes, func(a, b LedgerEntry) int { return strings.Compare(string(a.Code), string(b.Code)) })
	return l
}

// EncodeLedger renders a ledger as the committed file holds it.
func EncodeLedger(l Ledger) ([]byte, error) {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// DecodeLedger reads a committed ledger, refusing a format version it does
// not know, a code that does not follow the grammar, and a replaced_by that
// is set on a live code, names itself, or names a code the ledger does not
// list.
func DecodeLedger(data []byte) (Ledger, error) {
	var l Ledger
	if err := json.Unmarshal(data, &l); err != nil {
		return Ledger{}, fmt.Errorf("the refusal-code ledger is not JSON: %w", err)
	}
	if l.FormatVersion != LedgerFormatVersion {
		return Ledger{}, fmt.Errorf("the refusal-code ledger is format version %d; this binary reads %d", l.FormatVersion, LedgerFormatVersion)
	}
	listed := make(map[Code]bool, len(l.Codes))
	for _, c := range l.Codes {
		if !validName(string(c.Code)) {
			return Ledger{}, fmt.Errorf("the refusal-code ledger holds %q, which is not a code", c.Code)
		}
		listed[c.Code] = true
	}
	for _, c := range l.Codes {
		switch {
		case c.ReplacedBy == None:
		case c.Retired == "":
			return Ledger{}, fmt.Errorf("the refusal-code ledger names a replacement for %s, which is not retired", c.Code)
		case c.ReplacedBy == c.Code || !listed[c.ReplacedBy]:
			return Ledger{}, fmt.Errorf("the refusal-code ledger replaces %s with a code it does not otherwise list", c.Code)
		}
	}
	return l, nil
}
