// Package scribe builds the ledger scribe's context and ingests what the scribe
// returns (itd-2609020625402599, spc-2609020626045177).
//
// The scribe (agents/scribe.md) transcribes a returned reading's records and the
// researcher's dispositions into the ledger's declared shapes, and its access
// rule is the reading assembler's exact inverse: ledger content only, never the
// shipped repository. Until this package that rule was held by the definition
// alone. Here it is held by construction, on the reading assembler's idiom:
//
//   - Assemble builds the context by POSITIVE inclusion at directory grain, from
//     an allow list derived from the ledger's own directory constants
//     (issueschema.LedgerDirs), plus the researcher's supplied text. Nothing
//     outside the list is walked, and assertAllowList refuses any item whose path
//     is outside it whatever route it arrived by. A manifest names every path
//     passed, by hash, and is parked in the local tier.
//   - Ingest validates the scribe's four outputs and writes dispositions,
//     admissions and surprises through the capture verbs' own functions, adding
//     no validation path of its own beyond the one thing only it can check: that
//     the scribe authored nothing. Every ground, exit condition and surprise the
//     payload carries must already stand in the supplied text. The manifest is
//     promoted beside the run last, inside the read block.
//
// Both artefacts carry the per-run context stamp (core/sessionkind), so a
// retained transcript that held this context says so, and the transcript
// store's separation check can see it (adr-2609021016275803).
//
// `scribe` is its own top-level verb rather than a sub-verb of `reading`,
// because the two contexts must never share a front door.
package scribe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The three artefact type tags. They are carried in the documents so a loose
// file can be told apart without its name.
const (
	ContextType  = "abcd.scribe.context/1"
	ManifestType = "abcd.scribe.manifest/1"
	OutputType   = "abcd.scribe.output/1"
)

// SchemaVersion is the shape version of the context and the manifest.
const SchemaVersion = 1

// DefaultRunDir is the local-tier directory a session's context and manifest
// are parked in, one subdirectory per reading run. The local tier is on the
// reading assembler's exclusion floor, so nothing parked here reaches a reading.
const DefaultRunDir = ".abcd/.work.local/scratch/scribe-runs"

// The parked filenames. ManifestFileName is also the name the manifest is
// promoted under beside the run.
const (
	ContextFileName  = "context.json"
	ManifestFileName = "scribe-manifest.json"
)

// DefinitionPath is the scribe definition whose hash the manifest records.
const DefinitionPath = "agents/scribe.md"

// ErrSymlink refuses a symlinked directory or leaf inside the allow list: a link
// is a route out of the ledger that a prefix check on its name cannot see.
var ErrSymlink = errors.New("scribe: a symlink inside the ledger")

// AllowList is every directory the scribe's context may draw from, repository
// relative: the issue ledger's own directory list under its root. It is DERIVED
// from issueschema.LedgerDirs, so a family the ledger declares later is on this
// list the day its constant is, and the reading assembler's comparative
// exclusion rows describe the same set from the same function.
func AllowList() []string {
	dirs := issueschema.LedgerDirs()
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, capture.LedgerRelPath+"/"+d)
	}
	return out
}

// LedgerEntry is one ledger record as the scribe receives it. The scribe files
// by ledger path, so its material may name one; it is the reading bundle that
// may not.
type LedgerEntry struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// Supplied is the researcher's own material, verbatim.
type Supplied struct {
	Dispositions string `json:"dispositions"`
}

// Context is `abcd.scribe.context/1`: the scribe session's whole working set.
type Context struct {
	Type          string `json:"_type"`
	SchemaVersion int    `json:"schema_version"`
	// ContextStamp is the scribe kind's per-run stamp over the ledger array, in
	// the position the reading bundle carries its own.
	ContextStamp string        `json:"context_stamp"`
	Run          string        `json:"run"`
	Ledger       []LedgerEntry `json:"ledger"`
	Supplied     Supplied      `json:"supplied"`
}

// ManifestItem names one ledger record passed, by path, length and hash.
type ManifestItem struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// SuppliedHashes is the manifest's account of the supplied text.
type SuppliedHashes struct {
	DispositionsSHA256 string `json:"dispositions_sha256"`
}

// Exclusion is one class of material the context refuses, and the signal that
// holds the refusal.
type Exclusion struct {
	Source string `json:"source"`
	Signal string `json:"signal"`
}

// Manifest is `abcd.scribe.manifest/1`: what the assembly passed, by path and
// hash, and what it refused. It carries no record text.
type Manifest struct {
	Type             string         `json:"_type"`
	SchemaVersion    int            `json:"schema_version"`
	ContextStamp     string         `json:"context_stamp"`
	Run              string         `json:"run"`
	DefinitionSHA256 string         `json:"definition_sha256"`
	ContextSHA256    string         `json:"context_sha256"`
	Supplied         SuppliedHashes `json:"supplied"`
	Items            []ManifestItem `json:"items"`
	AllowList        []string       `json:"allow_list"`
	Exclusions       []Exclusion    `json:"exclusions"`
}

// allowListSignal is the signal every exclusion row rests on: the collector
// walks nothing but the allow list, and assertAllowList refuses any item outside
// it by prefix.
const allowListSignal = "not walked: no allow-list directory lies in it, and assertAllowList refuses " +
	"any item outside the allow list by path prefix"

// Exclusions is the manifest's exclusion assertion. Each row names a class of
// material by its location, never a home path.
func Exclusions() []Exclusion {
	return []Exclusion{
		{Source: "the shipped tree (every path outside .abcd/)", Signal: allowListSignal},
		{Source: ".abcd/development (the durable record: brief, intents, specs, decisions, readings)",
			Signal: allowListSignal},
		{Source: ".abcd/work outside " + capture.LedgerRelPath + " (the shared working tier)",
			Signal: allowListSignal},
		{Source: ".abcd/.work.local (the local tier, the per-repo transcript store included)",
			Signal: allowListSignal},
		{Source: "the session-transcript store under the user's home",
			Signal: "unreachable: outside the repository tree, and no walk starts outside it"},
	}
}

// encode is the one definition of canonical bytes for the scribe's artefacts,
// the reading assembler's: struct field order, two-space indent, no HTML
// escaping, one trailing newline.
func encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("scribe: encoding an artefact: %w", err)
	}
	return buf.Bytes(), nil
}

// decodeStrict decodes one document, refusing a repeated key at any depth,
// unknown fields and trailing content. The scribe output, the manifest and the
// context all decode through it.
func decodeStrict(data []byte, into any, what string) error {
	// A repeated key at any depth is refused, not read last-wins: encoding/json
	// would take {"state":"declined","state":"accepted"} as accepted. jsonstrict
	// is the one check every trust-boundary reader shares (iss-2609261036363114).
	if err := jsonstrict.NoDuplicateKeys(data); err != nil {
		return fmt.Errorf("scribe: decoding %s: %w; nothing is written", what, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("scribe: decoding %s: %w", what, err)
	}
	if dec.More() {
		return fmt.Errorf("scribe: decoding %s: trailing content after the document", what)
	}
	return nil
}

// DecodeManifest reads a scribe manifest strictly.
func DecodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := decodeStrict(data, &m, "the scribe manifest"); err != nil {
		return Manifest{}, err
	}
	if m.Type != ManifestType || m.SchemaVersion != SchemaVersion {
		return Manifest{}, fmt.Errorf("scribe: the manifest is %q version %d, want %q version %d",
			echo(m.Type), m.SchemaVersion, ManifestType, SchemaVersion)
	}
	return m, nil
}

// decodeContext reads a scribe context strictly.
func decodeContext(data []byte) (Context, error) {
	var c Context
	if err := decodeStrict(data, &c, "the scribe context"); err != nil {
		return Context{}, err
	}
	if c.Type != ContextType || c.SchemaVersion != SchemaVersion {
		return Context{}, fmt.Errorf("scribe: the context is %q version %d, want %q version %d",
			echo(c.Type), c.SchemaVersion, ContextType, SchemaVersion)
	}
	return c, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// maxEchoedBytes caps a payload-derived string quoted into a message, so a
// refusal cannot be made the size of the payload.
const maxEchoedBytes = 120

// echo neutralises and caps one payload-derived string for a message.
func echo(s string) string {
	s = termsafe.Sanitize(s)
	if len(s) > maxEchoedBytes {
		cut := maxEchoedBytes
		for cut > 0 && !utf8Start(s[cut]) {
			cut--
		}
		s = s[:cut] + "…"
	}
	return s
}

// utf8Start reports whether b begins a UTF-8 sequence, so a cap never splits
// a rune.
func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

// joinEchoed renders a list of payload-derived names for a message.
func joinEchoed(in []string) string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, echo(s))
	}
	return strings.Join(out, ", ")
}
