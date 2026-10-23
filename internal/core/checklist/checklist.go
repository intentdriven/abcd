// Package checklist is a working copy's checklist on disk: the list of what a
// session is working on, what it waits on and what it has promised, kept in the
// local tier so it survives compaction and crashes (itd-2610170638184486,
// spc-2610230805429429 step 1). This file holds the v1 schema, its strict
// decode and its validation; store.go holds the checked write and the
// falling-back read.
//
// The file is JSON with typed fields rather than prose, so it can be validated
// and scanned for secrets field by field (the spec's "Settled here"). A
// credential is recorded only by its kind, scope and expiry, each from a closed
// vocabulary or a timestamp, never as free text (adr-2610222026265719, brief
// invariant 24).
//
// No refusal in this package repeats the value it refused. A value that fails
// the schema is untrusted, and a secret pasted into the wrong field is exactly
// what the schema would see first; naming the field and the rule is enough to
// find it.
package checklist

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// Schema is the v1 schema's name, carried in every file's `schema` field. A
// reader meeting any other value refuses the file rather than guessing at it.
const Schema = "abcd/checklist/v1"

// MaxTitleRunes caps an item's title, in characters (the spec's "The file").
const MaxTitleRunes = 200

// maxIDLen, maxUpdatedByRunes, maxHandleLen and maxBranchLen bound the short
// fields. They are shapes, not a size refusal on the file: the spec refuses no
// write for the file's total size, and these only keep an identifier an
// identifier.
const (
	maxIDLen          = 64
	maxUpdatedByRunes = 128
	maxHandleLen      = 64
	maxBranchLen      = 255
)

// State is an item's state: one of the four the spec names.
type State string

const (
	Working State = "working"
	Waiting State = "waiting"
	Done    State = "done"
	Dropped State = "dropped"
)

// States returns the four states in the order they are documented: the one
// enumeration a refusal or a surface's completion reads.
func States() []State { return []State{Working, Waiting, Done, Dropped} }

// CredentialKind is what sort of credential an item needs. The list is the
// three adr-2610222026265719 names as examples; the ADR fixes that the list is
// closed, not its members, so a new kind is an edit here.
type CredentialKind string

const (
	AppInstallationToken CredentialKind = "app-installation-token"
	ModelServiceKey      CredentialKind = "model-service-key"
	SigningKey           CredentialKind = "signing-key"
)

// CredentialKinds returns the closed list of credential kinds.
func CredentialKinds() []CredentialKind {
	return []CredentialKind{AppInstallationToken, ModelServiceKey, SigningKey}
}

// CredentialScope is one thing a credential reaches. adr-2610222026265719
// describes scope as "a repository and the acts allowed, a service" and
// requires a fixed list; it gives none, so this one is defined here: the acts
// an App installation token is granted on a repository, a model service, and
// signing. The repository itself is not recorded, because a fixed list cannot
// hold its name and the ADR allows no free text in a credential.
type CredentialScope string

const (
	ScopeContentsRead  CredentialScope = "contents-read"
	ScopeContentsWrite CredentialScope = "contents-write"
	ScopePullRequests  CredentialScope = "pull-requests"
	ScopeIssues        CredentialScope = "issues"
	ScopeReleases      CredentialScope = "releases"
	ScopeWorkflows     CredentialScope = "workflows"
	ScopeModelService  CredentialScope = "model-service"
	ScopeSigning       CredentialScope = "signing"
)

// CredentialScopes returns the closed list of credential scopes.
func CredentialScopes() []CredentialScope {
	return []CredentialScope{
		ScopeContentsRead, ScopeContentsWrite, ScopePullRequests, ScopeIssues,
		ScopeReleases, ScopeWorkflows, ScopeModelService, ScopeSigning,
	}
}

// NoExpiry is the expiry of a credential that does not expire: the ADR's "or
// that it has none", spelt as a word rather than an absent field so a forgotten
// expiry is a refusal, not a silent "never".
const NoExpiry = "none"

// Checklist is the v1 file. Timestamps are kept as RFC 3339 strings and parsed
// by Validate, not by the decoder: encoding/json's time error quotes the value
// it could not parse, and a refusal here never quotes a value.
type Checklist struct {
	Schema    string `json:"schema"`
	UpdatedAt string `json:"updated_at"`
	// UpdatedBy is the host's session id for the session that last wrote.
	UpdatedBy string `json:"updated_by"`
	Items     []Item `json:"items"`
}

// Item is one entry. WaitsOn, NextLook, Promise, OwnerRefs and Credential are
// optional.
type Item struct {
	// ID is a short stable slug the verbs address the item by.
	ID    string `json:"id"`
	Title string `json:"title"`
	State State  `json:"state"`
	// WaitsOn is a record id, `pr:<n>`, `person:<role>` or `slot:<kind>`.
	WaitsOn  string `json:"waits_on,omitempty"`
	NextLook string `json:"next_look,omitempty"`
	// Promise is a trigger and the act it promises; the remainder of the spec
	// types the trigger, and step 1 keeps both as one-line text.
	Promise *Promise `json:"promise,omitempty"`
	// OwnerRefs names the work the item is about: a record id (a reservation),
	// `pr:<n>` or `branch:<name>`.
	OwnerRefs  []string    `json:"owner_refs,omitempty"`
	Credential *Credential `json:"credential,omitempty"`
}

// Promise is an item's "when this happens, do that".
type Promise struct {
	Trigger string `json:"trigger"`
	Act     string `json:"act"`
}

// Credential names a credential an item needs by its kind, scope and expiry
// only (adr-2610222026265719). There is deliberately no field for anything
// else, and the strict decode refuses one in the file.
type Credential struct {
	Kind   CredentialKind    `json:"kind"`
	Scope  []CredentialScope `json:"scope"`
	Expiry string            `json:"expiry"`
}

// FieldError is a schema refusal. Field is the path to the offending field
// (`items[2].state`), Problem the rule it breaks; neither carries the value.
type FieldError struct {
	Field   string
	Problem string
}

func (e *FieldError) Error() string { return "checklist: " + e.Field + " " + e.Problem }

// Decode is the strict decode: exactly one JSON document, no repeated key, no
// field the schema does not name (jsonstrict.Decode, as every trust-boundary
// reader decodes). It checks the shape only; Validate checks the values. The
// error may name an unknown key by the file's own spelling, so a caller never
// shows it: a redaction keeps a fingerprint, which is still a part of a secret.
func Decode(data []byte) (Checklist, error) {
	var c Checklist
	if err := jsonstrict.Decode(data, &c); err != nil {
		return Checklist{}, err
	}
	return c, nil
}

var (
	slugRe   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	prRe     = regexp.MustCompile(`^pr:[1-9][0-9]{0,9}$`)
	handleRe = regexp.MustCompile(`^(?:person|slot):[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	// branchRe is the characters a git branch name commonly carries; the
	// structural rules (no "..", no leading "-") are checked beside it.
	branchRe = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
)

// Validate checks every field against the v1 schema and answers the first
// offending one as a *FieldError, in file order.
func (c Checklist) Validate() error {
	if c.Schema != Schema {
		return refuse("schema", "must be "+Schema)
	}
	if err := checkStamp("updated_at", c.UpdatedAt); err != nil {
		return err
	}
	if err := checkText("updated_by", c.UpdatedBy, maxUpdatedByRunes); err != nil {
		return err
	}
	seen := map[string]int{}
	for i, it := range c.Items {
		at := fmt.Sprintf("items[%d]", i)
		if err := it.validate(at); err != nil {
			return err
		}
		if j, dup := seen[it.ID]; dup {
			return refuse(at+".id", fmt.Sprintf("repeats the id of items[%d]; ids must be unique", j))
		}
		seen[it.ID] = i
	}
	return nil
}

func (it Item) validate(at string) error {
	if len(it.ID) > maxIDLen || !slugRe.MatchString(it.ID) {
		return refuse(at+".id", fmt.Sprintf("must be a slug of lower-case letters and digits joined by single hyphens, at most %d characters", maxIDLen))
	}
	if err := checkText(at+".title", it.Title, MaxTitleRunes); err != nil {
		return err
	}
	if !validState(it.State) {
		return refuse(at+".state", "must be one of "+join(States()))
	}
	if it.WaitsOn != "" && !validWaitsOn(it.WaitsOn) {
		return refuse(at+".waits_on", "must be a record id ("+recordid.CitedIDForms()+"), pr:<number>, person:<role> or slot:<kind>, the role or kind a lower-case slug")
	}
	if it.NextLook != "" {
		if err := checkStamp(at+".next_look", it.NextLook); err != nil {
			return err
		}
	}
	if it.Promise != nil {
		if err := checkText(at+".promise.trigger", it.Promise.Trigger, 0); err != nil {
			return err
		}
		if err := checkText(at+".promise.act", it.Promise.Act, 0); err != nil {
			return err
		}
	}
	for i, ref := range it.OwnerRefs {
		if !validOwnerRef(ref) {
			return refuse(fmt.Sprintf("%s.owner_refs[%d]", at, i), "must be a record id ("+recordid.CitedIDForms()+"), pr:<number> or branch:<name>")
		}
	}
	if it.Credential != nil {
		return it.Credential.validate(at + ".credential")
	}
	return nil
}

func (cr Credential) validate(at string) error {
	if !validKind(cr.Kind) {
		return refuse(at+".kind", "must be one of "+join(CredentialKinds()))
	}
	if len(cr.Scope) == 0 {
		return refuse(at+".scope", "must name at least one of "+join(CredentialScopes()))
	}
	seen := map[CredentialScope]bool{}
	for i, s := range cr.Scope {
		field := fmt.Sprintf("%s.scope[%d]", at, i)
		if !validScope(s) {
			return refuse(field, "must be one of "+join(CredentialScopes()))
		}
		if seen[s] {
			return refuse(field, "repeats a scope already listed")
		}
		seen[s] = true
	}
	if cr.Expiry != NoExpiry {
		if !isStamp(cr.Expiry) {
			return refuse(at+".expiry", "must be an RFC 3339 timestamp or "+NoExpiry)
		}
	}
	return nil
}

func refuse(field, problem string) error { return &FieldError{Field: field, Problem: problem} }

func checkStamp(field, v string) error {
	if !isStamp(v) {
		return refuse(field, "must be an RFC 3339 timestamp")
	}
	return nil
}

// maxStampLen is the longest RFC 3339 stamp with nanoseconds and an offset.
const maxStampLen = len("2006-01-02T15:04:05.999999999-07:00")

// isStamp reports whether v is an RFC 3339 timestamp of at most nanosecond
// precision. WHY the length cap: time.Parse takes any number of fractional
// digits, and a stamp is never a place for free data (invariant 24).
func isStamp(v string) bool {
	if len(v) > maxStampLen {
		return false
	}
	_, err := time.Parse(time.RFC3339, v)
	return err == nil
}

// checkText is the rule for a free-text field: present, valid UTF-8, one line
// with no control character (the render puts each field on a line of its own,
// and the scan maps a finding back to its field by line), and within maxRunes
// characters when maxRunes is above zero.
func checkText(field, v string, maxRunes int) error {
	switch {
	case strings.TrimSpace(v) == "":
		return refuse(field, "must not be empty")
	case !utf8.ValidString(v):
		return refuse(field, "must be valid UTF-8")
	case strings.IndexFunc(v, unicode.IsControl) >= 0:
		return refuse(field, "must be one line with no control characters")
	case maxRunes > 0 && utf8.RuneCountInString(v) > maxRunes:
		return refuse(field, fmt.Sprintf("must be at most %d characters", maxRunes))
	}
	return nil
}

func validState(s State) bool {
	for _, v := range States() {
		if s == v {
			return true
		}
	}
	return false
}

func validKind(k CredentialKind) bool {
	for _, v := range CredentialKinds() {
		if k == v {
			return true
		}
	}
	return false
}

func validScope(s CredentialScope) bool {
	for _, v := range CredentialScopes() {
		if s == v {
			return true
		}
	}
	return false
}

func validWaitsOn(v string) bool {
	return recordid.CitedIDRe.MatchString(v) || prRe.MatchString(v) ||
		(len(v) <= maxHandleLen && handleRe.MatchString(v))
}

func validOwnerRef(v string) bool {
	if recordid.CitedIDRe.MatchString(v) || prRe.MatchString(v) {
		return true
	}
	name, ok := strings.CutPrefix(v, "branch:")
	return ok && validBranch(name)
}

// validBranch is a conservative reading of git's ref-name rules: the common
// characters only, and none of the structures git refuses.
func validBranch(name string) bool {
	return len(name) <= maxBranchLen && branchRe.MatchString(name) &&
		!strings.Contains(name, "..") && !strings.Contains(name, "//") &&
		!strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "/") &&
		!strings.HasSuffix(name, "/") && !strings.HasSuffix(name, ".") &&
		!strings.HasSuffix(name, ".lock")
}

func join[T ~string](vs []T) string {
	words := make([]string, len(vs))
	for i, v := range vs {
		words[i] = string(v)
	}
	return strings.Join(words, ", ")
}

// field is one string the scan reads, with the path a refusal names it by.
type field struct {
	name, value string
}

// stringFields lists every string the checklist holds, closed vocabularies and
// timestamps included, in file order. The scan reads all of them: a field the
// schema has already constrained cannot carry a token shape today, but the
// rule (adr-2610222026265719) is that every field is scanned, not every field
// this release thinks is risky.
func (c Checklist) stringFields() []field {
	fs := []field{{"schema", c.Schema}, {"updated_at", c.UpdatedAt}, {"updated_by", c.UpdatedBy}}
	for i, it := range c.Items {
		at := fmt.Sprintf("items[%d]", i)
		fs = append(fs,
			field{at + ".id", it.ID},
			field{at + ".title", it.Title},
			field{at + ".state", string(it.State)},
			field{at + ".waits_on", it.WaitsOn},
			field{at + ".next_look", it.NextLook},
		)
		if it.Promise != nil {
			fs = append(fs, field{at + ".promise.trigger", it.Promise.Trigger}, field{at + ".promise.act", it.Promise.Act})
		}
		for j, ref := range it.OwnerRefs {
			fs = append(fs, field{fmt.Sprintf("%s.owner_refs[%d]", at, j), ref})
		}
		if cr := it.Credential; cr != nil {
			fs = append(fs, field{at + ".credential.kind", string(cr.Kind)})
			for j, s := range cr.Scope {
				fs = append(fs, field{fmt.Sprintf("%s.credential.scope[%d]", at, j), string(s)})
			}
			fs = append(fs, field{at + ".credential.expiry", cr.Expiry})
		}
	}
	return fs
}
