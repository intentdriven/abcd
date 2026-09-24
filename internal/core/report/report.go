// Package report is the written account a repository abcd manages files back to
// abcd itself, and the inbox it waits in (itd-2609221656361680,
// spc-2609221657168936).
//
// A report is a markdown file: a machine-readable block between `---` lines
// beside prose the reporter writes. abcd issues the template (Template), checks
// a filled one against it (Parse), and files it into the user account's machine
// store:
//
//	~/.abcd/inbox/<received-stamp>-<sender-key>.md   one waiting report
//	~/.abcd/inbox/promoted/<same name>               a report filed as a capture
//	~/.abcd/inbox/promoted.jsonl                     which capture each became
//
// The inbox lives beside the history, transcript, worktree, sources, labs and
// run stores, never in either repository's tree. The sender key is the sending
// repository's full root-commit SHA, the key the other machine-scoped stores
// use. Nothing in a report names where it is written or read: the file name is
// derived by abcd from the clock and the key, and every field is data that is
// rendered, never opened.
//
// A report arrives from another repository, so everything in it is untrusted
// input. Its size is bounded (MaxBytes), a control byte anywhere refuses it, a
// field that names a filesystem location refuses it, and every front door
// sanitises what it prints. Nothing is filed as a record until a person or a
// session runs Promote, and what Promote files carries the sender's root-commit
// key and a generic description, never the sender's name.
//
// Core never writes to stdout; the CLI front door formats what these functions
// return.
package report

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the template version this abcd writes and reads. A report
// carrying any other version is listed as unreadable, naming the version, and is
// never dropped or half-read.
const SchemaVersion = 1

// MaxBytes bounds a whole report. A report is written at the moment of a
// finding (scope condition 2); a run's whole account is a document the report
// points at, not the report's body.
const MaxBytes = 32 << 10

// maxFiledBytes bounds a report read back from the inbox. The stored file is
// the accepted report plus the envelope abcd stamps and the escaping its quoted
// scalars take, so it may exceed MaxBytes; twice the bound holds every report
// MaxBytes admits, and still bounds a file planted in the inbox by other means.
const maxFiledBytes = 2 * MaxBytes

// Per-field bounds, in bytes.
const (
	maxShortField = 200
	maxRemedy     = 2000
	maxEvidence   = 20
	maxPointer    = 300
	maxVersion    = 64
	// maxKey clips a key name a refusal echoes: an unknown key is the
	// reporter's text, and the inbox lists the refusal as a reason.
	maxKey = 64
)

// Kinds is the closed vocabulary of what a report asks for.
var Kinds = []string{"defect", "enhancement"}

// ErrRefused is the class of every refusal a report can earn: a malformed
// block, a field outside its bounds, an unknown template version at filing, an
// id that names no report. Nothing is written for a refused act. The CLI maps
// it to exit 2.
var ErrRefused = errors.New("refused")

// FieldError is a refusal that names the field it is about.
type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Reason)
}

// Unwrap classes every FieldError as a refusal.
func (e *FieldError) Unwrap() error { return ErrRefused }

// VersionError is a report whose template version this abcd does not know. It is
// a refusal at filing and an "unreadable" listing in the inbox.
type VersionError struct {
	Version string
}

func (e *VersionError) Error() string {
	return fmt.Sprintf("template version %s is not one this abcd knows (it reads version %d)", e.Version, SchemaVersion)
}

// Unwrap classes every VersionError as a refusal.
func (e *VersionError) Unwrap() error { return ErrRefused }

func fieldErr(field, format string, a ...any) error {
	return &FieldError{Field: field, Reason: fmt.Sprintf(format, a...)}
}

// Report is one parsed report: the reporter's block and prose, and, for a
// report read back from the inbox, the envelope abcd stamped on it at filing.
type Report struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	Severity      string   `json:"severity"`
	Category      string   `json:"category"`
	Title         string   `json:"title"`
	AbcdVersion   string   `json:"abcd_version"`
	Surface       string   `json:"surface"`
	Remedy        string   `json:"remedy,omitempty"`
	Evidence      []string `json:"evidence"`
	Prose         string   `json:"prose"`

	// The envelope. Empty on a submission; required on a filed report.
	ReceivedAt string `json:"received_at,omitempty"`
	SenderKey  string `json:"sender_key,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
}

// The block's keys, in the order the template and the stored file write them.
const (
	keyVersion    = "schema_version"
	keyKind       = "kind"
	keySeverity   = "severity"
	keyCategory   = "category"
	keyTitle      = "title"
	keyAbcd       = "abcd_version"
	keySurface    = "surface"
	keyRemedy     = "remedy"
	keyEvidence   = "evidence"
	keyReceivedAt = "received_at"
	keySenderKey  = "sender_key"
	keySenderName = "sender_name"
)

var reporterKeys = []string{keyVersion, keyKind, keySeverity, keyCategory, keyTitle, keyAbcd, keySurface, keyRemedy, keyEvidence}

var envelopeKeys = []string{keyReceivedAt, keySenderKey, keySenderName}

// Template returns the skeleton a reporter fills: the block with every key the
// validator reads, its closed vocabularies named in comments, and a prose
// placeholder. abcdVersion is the running binary's version, filled in so the
// reporter need not look it up. The skeleton as issued is refused by Parse
// (its title and prose are empty), so an unfilled template can never be filed.
func Template(abcdVersion string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "# abcd report template, version %d. Fill every field; keep each on one line.\n", SchemaVersion)
	fmt.Fprintf(&b, "%s: %d\n", keyVersion, SchemaVersion)
	fmt.Fprintf(&b, "# %s: %s\n", keyKind, strings.Join(Kinds, " | "))
	fmt.Fprintf(&b, "%s: defect\n", keyKind)
	fmt.Fprintf(&b, "# %s: %s\n", keySeverity, strings.Join(issueschema.Severities, " | "))
	fmt.Fprintf(&b, "%s: minor\n", keySeverity)
	fmt.Fprintf(&b, "# %s: %s\n", keyCategory, strings.Join(issueschema.Categories, " | "))
	fmt.Fprintf(&b, "%s: bug\n", keyCategory)
	b.WriteString("# title: one line saying what you found.\n")
	fmt.Fprintf(&b, "%s: \"\"\n", keyTitle)
	b.WriteString("# abcd_version: the abcd in play when you found it.\n")
	fmt.Fprintf(&b, "%s: %s\n", keyAbcd, frontmatter.QuoteScalar(oneLine(abcdVersion, maxVersion)))
	b.WriteString("# surface: the verb, hook or page in play, e.g. abcd capture.\n")
	fmt.Fprintf(&b, "%s: \"\"\n", keySurface)
	b.WriteString("# remedy: optional; what you would change.\n")
	fmt.Fprintf(&b, "%s: \"\"\n", keyRemedy)
	b.WriteString("# evidence: optional pointers (record ids, commit SHAs, URLs), one per line as `  - pointer`.\n")
	b.WriteString("#   Never a path on this machine: a pointer is quoted in the report, never opened.\n")
	fmt.Fprintf(&b, "%s: []\n", keyEvidence)
	b.WriteString("---\n\n")
	b.WriteString(prosePlaceholder + "\n")
	return []byte(b.String())
}

// prosePlaceholder is the template's prose. Comments are removed from the
// prose at parse, so a report whose prose is only this is refused as empty, and
// the placeholder never reaches the inbox or a capture.
const prosePlaceholder = "<!-- What happened, what you expected, and how to see it again. Write it here, below the block. -->"

// oneLine clips s to one line of at most max bytes, for a value abcd itself
// writes into the template.
func oneLine(s string, max int) string {
	s = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s))
	for len(s) > max {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return s
}

// Parse reads a report as a reporter submits it: the envelope keys are abcd's
// and a submission carrying one is refused. An unknown template version is a
// *VersionError; any other defect is a *FieldError naming the field.
func Parse(data []byte) (Report, error) {
	return parse(data, false)
}

// parseFiled reads a report back from the inbox, where the envelope is
// required.
func parseFiled(data []byte) (Report, error) {
	return parse(data, true)
}

// blockLine is one key of the block as read.
type blockLine struct {
	value string   // the raw same-line value
	items []string // the block list under the key, when it has one
	list  bool     // a block list followed the key
}

func parse(data []byte, filed bool) (Report, error) {
	limit := MaxBytes
	if filed {
		limit = maxFiledBytes
	}
	if len(data) > limit {
		return Report{}, fieldErr("report", "is %d bytes; a report is at most %d (point at a longer account instead of pasting it)", len(data), limit)
	}
	if !utf8.Valid(data) {
		return Report{}, fieldErr("report", "is not valid UTF-8")
	}
	text := frontmatter.TrimBOM(string(data))
	for i, r := range text {
		if (r < 0x20 && r != '\n' && r != '\t' && r != '\r') || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return Report{}, fieldErr("report", "carries control byte %#02x at offset %d; a report is plain text", r, i)
		}
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || !frontmatter.IsDelimiter(lines[0]) {
		return Report{}, fieldErr("block", "the report must open with the template's `---` block")
	}
	block := map[string]*blockLine{}
	var order []string
	var last string
	closeAt := -1
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if frontmatter.IsDelimiter(line) {
			closeAt = i
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			item, ok := strings.CutPrefix(trimmed, "- ")
			if !ok && trimmed != "-" {
				return Report{}, fieldErr(orBlock(last), "line %d is indented but is not a `- item` of a list", i+1)
			}
			if last == "" || strings.TrimSpace(frontmatter.StripComment(block[last].value)) != "" {
				return Report{}, fieldErr(orBlock(last), "line %d is a list item under a key that holds a value", i+1)
			}
			block[last].list = true
			block[last].items = append(block[last].items, strings.TrimSpace(frontmatter.StripComment(item)))
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return Report{}, fieldErr("block", "line %d is not a `key: value` line", i+1)
		}
		if _, dup := block[key]; dup {
			return Report{}, fieldErr(oneLine(key, maxKey), "appears twice")
		}
		block[key] = &blockLine{value: value}
		order = append(order, key)
		last = key
	}
	if closeAt < 0 {
		return Report{}, fieldErr("block", "the `---` block is never closed")
	}

	// The version first, before any other key is judged: a report written to a
	// later template may carry keys this abcd has never heard of, and it must
	// read as "a version I do not know", not as "an unknown key".
	vl, ok := block[keyVersion]
	if !ok {
		return Report{}, fieldErr(keyVersion, "is missing")
	}
	vraw, _ := scalar(vl)
	if vl.list || vraw == "" {
		return Report{}, fieldErr(keyVersion, "must be a whole number")
	}
	version, err := strconv.Atoi(vraw)
	if err != nil {
		return Report{}, &VersionError{Version: oneLine(vraw, maxVersion)}
	}
	if version != SchemaVersion {
		return Report{}, &VersionError{Version: strconv.Itoa(version)}
	}

	for _, k := range order {
		switch {
		case slices.Contains(reporterKeys, k):
		case slices.Contains(envelopeKeys, k):
			if !filed {
				return Report{}, fieldErr(k, "is abcd's to write when the report is filed; remove it")
			}
		default:
			return Report{}, fieldErr(oneLine(k, maxKey), "is not a field of template version %d", SchemaVersion)
		}
	}

	r := Report{SchemaVersion: version}
	get := func(key string, required bool, max int) (string, error) {
		bl, present := block[key]
		if !present {
			if required {
				return "", fieldErr(key, "is missing")
			}
			return "", nil
		}
		if bl.list {
			return "", fieldErr(key, "must be a single value, not a list")
		}
		v, ok := scalar(bl)
		if !ok {
			return "", fieldErr(key, "is not a readable single-line value")
		}
		if required && v == "" {
			return "", fieldErr(key, "is empty")
		}
		if len(v) > max {
			return "", fieldErr(key, "is %d bytes; at most %d", len(v), max)
		}
		if err := refuseHidden(key, v); err != nil {
			return "", err
		}
		if err := refusePath(key, v); err != nil {
			return "", err
		}
		return v, nil
	}
	enum := func(key string, set []string) (string, error) {
		v, err := get(key, true, maxShortField)
		if err != nil {
			return "", err
		}
		if !slices.Contains(set, v) {
			return "", fieldErr(key, "%q is not one of %s", v, strings.Join(set, ", "))
		}
		return v, nil
	}
	if r.Kind, err = enum(keyKind, Kinds); err != nil {
		return Report{}, err
	}
	if r.Severity, err = enum(keySeverity, issueschema.Severities); err != nil {
		return Report{}, err
	}
	if r.Category, err = enum(keyCategory, issueschema.Categories); err != nil {
		return Report{}, err
	}
	if r.Title, err = get(keyTitle, true, maxShortField); err != nil {
		return Report{}, err
	}
	if r.AbcdVersion, err = get(keyAbcd, true, maxVersion); err != nil {
		return Report{}, err
	}
	if r.Surface, err = get(keySurface, true, maxShortField); err != nil {
		return Report{}, err
	}
	if r.Remedy, err = get(keyRemedy, false, maxRemedy); err != nil {
		return Report{}, err
	}
	if r.Evidence, err = evidence(block[keyEvidence]); err != nil {
		return Report{}, err
	}
	if filed {
		if r.ReceivedAt, err = get(keyReceivedAt, true, maxShortField); err != nil {
			return Report{}, err
		}
		if r.SenderKey, err = get(keySenderKey, true, maxShortField); err != nil {
			return Report{}, err
		}
		if !senderKeyRe.MatchString(r.SenderKey) {
			return Report{}, fieldErr(keySenderKey, "is not a root-commit key")
		}
		if r.SenderName, err = get(keySenderName, true, maxShortField); err != nil {
			return Report{}, err
		}
		if !senderNameRe.MatchString(r.SenderName) {
			return Report{}, fieldErr(keySenderName, "is not a repository name")
		}
	}

	prose := strings.Join(lines[closeAt+1:], "\n")
	if err := refuseHidden("prose", prose); err != nil {
		return Report{}, err
	}
	// Comments are removed, not kept: the template's placeholder is one, and a
	// reporter who writes below it leaves it in place. A comment renders as
	// nothing, so one that reached the capture a promotion commits would be text
	// in the record that no reader of it sees.
	prose = strings.TrimSpace(htmlCommentRe.ReplaceAllString(prose, ""))
	if prose == "" {
		return Report{}, fieldErr("prose", "is empty; write the account below the block")
	}
	if strings.Contains(prose, "<!--") {
		return Report{}, fieldErr("prose", "opens an HTML comment it never closes, which would hide the rest of the account")
	}
	r.Prose = prose
	return r, nil
}

func orBlock(key string) string {
	if key == "" {
		return "block"
	}
	return key
}

// htmlCommentRe matches an HTML comment, which the template's placeholder is.
var htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)

// scalar reads a key's same-line value: a trailing comment removed, quotes
// decoded, "" for an empty or explicitly empty value. ok is false for a shape
// that is not a single-line string (a flow collection other than `[]`, a
// block-scalar header, an unclosed quote).
func scalar(bl *blockLine) (string, bool) {
	v := strings.TrimSpace(frontmatter.StripComment(bl.value))
	switch v {
	case "", `""`, "''", "~", "null":
		return "", true
	}
	s, ok := frontmatter.ScalarString(v)
	return strings.TrimSpace(s), ok
}

// evidence reads the evidence list: absent, `[]`, or a block list of pointers.
func evidence(bl *blockLine) ([]string, error) {
	if bl == nil {
		return []string{}, nil
	}
	if !bl.list {
		v := strings.TrimSpace(frontmatter.StripComment(bl.value))
		if v == "" || v == "[]" {
			return []string{}, nil
		}
		return nil, fieldErr(keyEvidence, "must be `[]` or a list of `  - pointer` lines")
	}
	out := []string{}
	for _, raw := range bl.items {
		v, ok := scalar(&blockLine{value: raw})
		if !ok {
			return nil, fieldErr(keyEvidence, "item %q is not a readable single-line value", raw)
		}
		if v == "" {
			continue
		}
		if len(v) > maxPointer {
			return nil, fieldErr(keyEvidence, "an item is %d bytes; at most %d", len(v), maxPointer)
		}
		if err := refuseHidden(keyEvidence, v); err != nil {
			return nil, err
		}
		if err := refusePath(keyEvidence, v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if len(out) > maxEvidence {
		return nil, fieldErr(keyEvidence, "has %d pointers; at most %d", len(out), maxEvidence)
	}
	return out, nil
}

// pathRe matches a value that names a location on a filesystem: an absolute
// path, a UNC path in either slash, a home-relative one, the environment's home
// ($HOME, %USERPROFILE%), a Windows drive, a path after a colon, a file URL, or
// a traversal segment. A slash between words, a tilde before a number, a clock
// time and an http(s) URL are words, not locations.
//
// It is a hygiene check over the block's fields, best effort, and not a
// refusal boundary: the prose is never matched against it, and a determined
// reporter can spell a location it does not recognise. Nothing depends on it
// being complete, because no consumer opens any field or any prose: every value
// is rendered or quoted, never read as a path, fetched or executed. What it
// buys is a report that stays about records, commits and URLs, since a report
// is read on a machine that is not the one it describes.
var pathRe = regexp.MustCompile(`(?i)` +
	`(^|[\s(\[<"'=,])(//?[^\s/]|~[a-z0-9._-]*/|\\\\|[a-z]:[\\/])` + // absolute, UNC, home-relative, drive
	`|:/[^\s/]` + // a path after a colon; a URL's "://" is not one
	`|\$\{?home\b|%(userprofile|homepath|homedrive|appdata|localappdata)%` + // the environment's home
	`|file:` + // a file URL
	`|(^|[\\/\s])\.\.([\\/]|$)`) // a traversal segment

// refuseHidden refuses a value carrying a bidirectional control or a
// zero-width rune, judged by termsafe's own predicate: such a value displays
// differently from its bytes, so a reader of the report, or of the capture it
// becomes, would see something the report does not say. The byte-order mark
// that opens a file is trimmed before any field is read, so only one inside
// the text reaches this check.
func refuseHidden(key, v string) error {
	for i, r := range v {
		if termsafe.IsHidden(r) {
			return fieldErr(key, "carries the invisible or direction-changing character U+%04X at byte %d; a report is plain text", r, i)
		}
	}
	return nil
}

func refusePath(key, v string) error {
	if pathRe.MatchString(v) {
		return fieldErr(key, "names a filesystem location; a report points at records, commits and URLs, never at a path on a machine")
	}
	return nil
}

// senderKeyRe is a full root-commit SHA, SHA-1 or SHA-256.
var senderKeyRe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// senderNameRe is the shape a sender's name is held to: a directory name with
// nothing a terminal or a path could make anything of.
var senderNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// serialize renders a filed report canonically: every value abcd parsed,
// written back by abcd, so the stored file never carries a byte the parser did
// not accept.
func serialize(r Report) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "%s: %d\n", keyVersion, r.SchemaVersion)
	for _, kv := range [][2]string{
		{keyKind, r.Kind}, {keySeverity, r.Severity}, {keyCategory, r.Category},
		{keyTitle, r.Title}, {keyAbcd, r.AbcdVersion}, {keySurface, r.Surface},
		{keyRemedy, r.Remedy},
	} {
		fmt.Fprintf(&b, "%s: %s\n", kv[0], frontmatter.QuoteScalar(kv[1]))
	}
	if len(r.Evidence) == 0 {
		fmt.Fprintf(&b, "%s: []\n", keyEvidence)
	} else {
		fmt.Fprintf(&b, "%s:\n", keyEvidence)
		for _, e := range r.Evidence {
			fmt.Fprintf(&b, "  - %s\n", frontmatter.QuoteScalar(e))
		}
	}
	for _, kv := range [][2]string{
		{keyReceivedAt, r.ReceivedAt}, {keySenderKey, r.SenderKey}, {keySenderName, r.SenderName},
	} {
		fmt.Fprintf(&b, "%s: %s\n", kv[0], frontmatter.QuoteScalar(kv[1]))
	}
	b.WriteString("---\n\n")
	b.WriteString(r.Prose)
	b.WriteString("\n")
	return []byte(b.String())
}
