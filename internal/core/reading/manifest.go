package reading

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SchemaVersion is the shape version of both artefacts an assembly writes.
//
// It is ONE constant for two shapes, so a change to either restamps both. At
// version 2 the manifest item gained a kind and the bundle's shape did not
// move; the bundle was restamped anyway. At version 3 BOTH shapes moved, the
// bundle gaining the scope a reading was given and the manifest gaining the
// effective scope, its hash and the override stamp — so at that version the
// shared constant cost nothing. At version 4 the manifest item gained its byte
// length, so the shared constant restamps the bundle again. That is a known consequence of the
// shared constant, accepted rather than fixed inside a change that needed only
// one half of it — splitting the two is a larger change, and making the split
// silently is how a shape version stops meaning anything (spc-68). At version 5
// BOTH shapes move again: the scope operand and its override stamp are
// withdrawn, so the manifest's `scope`, `scope_hash` and `scope_overridden`
// become `preset` and `preset_hash`, and the bundle's `scope` becomes `preset`
// (adr-2609021016286571). At version 6 the manifest item gains its `scan` mark,
// the fact of whether the exclusion floor examined that item, so the shared
// constant restamps the bundle once more (itd-194, adr-56 as refined
// 2026-09-02). At version 7 the RUN RECORD gains its `bounds` list, the
// departures a run made from the reading the design documents state; the bundle
// and the manifest are untouched and are restamped by the shared constant once
// more, which is why `AssemblerVersionCore` does not move with it
// (spc-2609020626048722). At version 8 BOTH shapes move for the comparative
// channel: the closed `Kind` vocabulary gains `candidate`, which `DecodeManifest`
// refuses when it does not know it, the bundle item gains `candidate` and
// `field`, and the manifest gains `candidate_run`, `candidates`, `exercised`,
// `candidate_fields` and `criteria` with its item gaining `candidate`
// (adr-2609021016272867, spc-2609020626039834). At version 9 the MANIFEST gains
// `candidate_run_target`, the commit the derived widening run itself read: the
// derivation now admits a run whose own records were committed since it read, so
// the run's target and the assembly's can differ and a reader needs both to
// check the selection (iss-2609021833302981). The bundle is untouched and is
// restamped by the shared constant once more.
const SchemaVersion = 9

// The two artefact type tags. They are carried in the documents themselves so a
// reader of a loose file can tell the two apart without its filename.
const (
	BundleType   = "abcd.reading.bundle"
	ManifestType = "abcd.reading.manifest"
)

// BundleItem is one passed item as the reading receives it: a key, a material
// class, and the text. It carries NO repository path, by construction — the key
// is an ordinal and the kind names a class, never a location (invariant 15).
type BundleItem struct {
	ItemKey string `json:"item_key"`
	Kind    Kind   `json:"kind"`
	// Candidate and Field are set on a CANDIDATE item and on nothing else: which
	// rdi-N of the derived widening run this text belongs to, and whether it is
	// the configuration or what admits it.
	//
	// A comparative reading returns one item per candidate-criterion pair and its
	// body cites a `candidate_id`, so it has to be able to name the candidate it
	// is characterising in the terms it was handed. Neither value is a repository
	// path — an item id is an ordinal minted by the ingest verb, and a field name
	// is a record's own key — so brief invariant 15 holds, and
	// TestNoBundleFieldIsAScopeSelector pins both against becoming selectors
	// (spc-2609020626039834).
	Candidate string `json:"candidate,omitempty"`
	Field     string `json:"field,omitempty"`
	Text      string `json:"text"`
}

// Bundle is the assembled input: the reading's entire working set.
//
// It carries no run identifier and no timestamp, so two assemblies of one
// repository state at one commit are byte-identical — the property itd-187's
// eval falsifies independently, and the reason the run identifier lives on the
// manifest alone.
type Bundle struct {
	Type          string   `json:"_type"`
	SchemaVersion int      `json:"schema_version"`
	Position      Position `json:"position"`
	// Preset is what THIS run was given, and it is the reading's own fact
	// rather than the auditor's. A reader told its object is the shipped tree
	// and handed a tenth of it will report the missing nine tenths as a
	// tension against the claim record, with every gate green.
	//
	// It carries NO repository path under any entry, and no provenance. See
	// BundlePreset: it is a projection rather than the applied entry precisely
	// because the obvious implementation carried a path, and which entry was
	// applied and what it hashes to is the auditor's business and lives on the
	// manifest.
	Preset BundlePreset `json:"preset"`
	Items  []BundleItem `json:"items"`
}

// BundlePreset is the applied entry as a READING sees it, and it is
// deliberately NOT the AppliedPreset the manifest carries.
//
// The manifest may name repository paths; the bundle may not, by brief
// invariant 15 — the assembled input is the reading's entire working set and
// no repository path enters its context. An entry's Path selectors ARE
// repository paths, so writing one type into both artefacts put a path
// into the reading's own working set. That is what this split exists to
// prevent, and it was a live breach before it was caught
// (iss-2608312058244357).
//
// A reading still has to know it was handed a subset: told its object is the
// shipped tree and given a tenth of it, it reports the missing nine tenths as
// a finding. So it is told the kinds and the records it was handed, and
// that a narrowing by LOCATION applied — never where. That is enough to know
// the bundle is not the whole object, and it carries no location.
type BundlePreset struct {
	Kinds   []Kind   `json:"kinds,omitempty"`
	Records []string `json:"records,omitempty"`
	// LocationNarrowings counts the location-based narrowings applied. It is a
	// count and never a list, because the list would be the paths.
	LocationNarrowings int `json:"location_narrowings,omitempty"`
}

// bundlePreset projects the applied entry down to what a reading may see.
func bundlePreset(s AppliedPreset) BundlePreset {
	var out BundlePreset
	for _, sel := range s.Selectors {
		switch {
		case sel.Kind != "":
			out.Kinds = append(out.Kinds, sel.Kind)
		case sel.Record != "":
			out.Records = append(out.Records, sel.Record)
		case sel.Path != "":
			out.LocationNarrowings++
		}
	}
	return out
}

// ManifestItem maps one bundle item back to the file and field it came from,
// with the hash of the passed text. This mapping is the auditor's, and only the
// auditor's: it is the reason the bundle can be pathless and still checkable.
type ManifestItem struct {
	ItemKey string `json:"item_key"`
	Path    string `json:"path"`
	Field   string `json:"field,omitempty"`
	// Candidate is the reading item this text came from, on a candidate item and
	// on nothing else. It sits beside the Field this item already carried, so an
	// auditor can resolve a comparative output's `candidate_id` back to the
	// record it names without re-deriving it from the path — which is what the
	// ingest's unknown-candidate check compares against
	// (spc-2609020626039834, "Ingest at the comparative position").
	Candidate string `json:"candidate,omitempty"`
	// Kind is the item's material class, carried so a size report is checkable
	// against the manifest rather than asserted beside it. It is deliberately
	// NOT omitempty: an item without a kind is a defect, and a shape that can
	// omit the field cannot tell that defect from a well-formed item (spc-68).
	Kind Kind `json:"kind"`
	// Scan is whether the exclusion floor EXAMINED this item: `parsed` when its
	// key and heading signals were read over the bytes that travelled,
	// `unscanned` when the row that admitted it is one the floor does not parse
	// and the item travelled whole.
	//
	// It is what makes the manifest's exclusion assertion a fact about each item
	// rather than a claim about the run. Without it a scan that ran and found
	// nothing and a scan that never ran produced byte-identical attestations,
	// which is an attestation that does not attest (adr-56; brief invariant 16).
	//
	// Deliberately NOT omitempty, on the same argument Kind carries: an item
	// without a mark is a defect, and a shape that can omit the field cannot
	// tell that defect from a well-formed item.
	Scan Scan `json:"scan"`
	// Bytes is the length of the passed text. Without it the size report was
	// only HALF checkable against the manifest: an auditor could recompute the
	// per-kind item COUNTS and not the per-kind BYTES, which is the figure
	// itd-198 exists to add. The bundle carries the text and goes to the
	// reader; the manifest stays with the auditor, so an auditor holding only
	// the manifest could not corroborate the number the intent promised was
	// corroborable.
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest enumerates what an assembly passed, by path, by field and by hash,
// and asserts what it refused. It carries no item content, so committing it
// needs no redaction.
//
// It carries no timestamp FIELD, but it is not timestamp-free and must not be
// described as such: RunID embeds a mint stamp by construction (adr-45). So two
// assemblies of one repository state at one commit produce manifests that differ
// in RunID and in nothing else — Items and Exclusions are identical, and the
// bundle beside them is byte-identical. That, not manifest byte-identity, is the
// determinism a re-run can be checked against, and it is why the manifest sits
// outside the amnesia eval's comparison rather than inside it.
type Manifest struct {
	Type             string   `json:"_type"`
	SchemaVersion    int      `json:"schema_version"`
	RunID            string   `json:"run_id"`
	Position         Position `json:"position"`
	TargetCommit     string   `json:"target_commit"`
	AssemblerVersion string   `json:"assembler_version"`
	// Preset and PresetHash are the auditor's account of what this run was
	// about: the committed entry for the invoked position, as selectors, and
	// its content hash. The hash lets a reader tell two runs apart by the entry
	// they applied rather than by re-deriving it, and it means a preset edited
	// later can never make a past run unreadable.
	//
	// There is no override stamp. It counted departures from the committed
	// presets, which was worth counting while an operand could depart from
	// them; nothing can now, so a stamp would assert a distinction that does
	// not exist (adr-2609021016286571).
	Preset     AppliedPreset `json:"preset"`
	PresetHash string        `json:"preset_hash"`
	// CandidateRun is the widening run this assembly DERIVED, at the comparative
	// position and nowhere else. It is on the manifest so a reader can check the
	// selection against the record rather than take it on trust: the run named is
	// the one committed widening run at this manifest's target whose items
	// carried no disposition and no admission when the assembly ran
	// (adr-2609021016272867).
	CandidateRun string `json:"candidate_run,omitempty"`
	// CandidateRunTarget is the commit THE DERIVED RUN ITSELF read, which is this
	// manifest's own TargetCommit or an ancestor of it. The two are recorded
	// separately, and both are recorded even when they are equal, because the
	// derivation admits a run whose records were committed since it read: a
	// reader with both commits can diff them and see for themselves that nothing
	// but the readings store and the issue ledger moved between the two
	// (adr-2609021016272867, as read in DeriveCandidateRun; divergence register
	// 27; iss-2609021857343626).
	CandidateRunTarget string `json:"candidate_run_target,omitempty"`
	// Candidates is the count of items THE DERIVED RUN HOLDS — the count of
	// candidates the bundle carries when the position is exercised, and still the
	// run's own count when it is not. It is never written as zero for a run that
	// holds an item: Exercised is the field that says the position was not
	// exercised, and conflating the two would lose which of the two facts a
	// reader is looking at.
	Candidates int `json:"candidates,omitempty"`
	// Exercised is false on the staged empty run and true when the bundle carries
	// the candidates. It is a POINTER so that it is written whenever
	// CandidateRun is and absent everywhere else: a false value is then a
	// statement rather than an absence, and no widening manifest asserts an
	// `exercised` it has no business having an opinion about.
	Exercised *bool `json:"exercised,omitempty"`
	// CandidateFields are the two widening body fields projected, and Criteria
	// the criterion names parsed out of the committed discipline. Together they
	// are what the assembler SELECTED and what it WEIGHED, so a reader of the
	// manifest alone can say what the comparative reading was handed.
	CandidateFields []string       `json:"candidate_fields,omitempty"`
	Criteria        []string       `json:"criteria,omitempty"`
	Items           []ManifestItem `json:"items"`
	Exclusions      []Exclusion    `json:"exclusions"`
}

// encode is the one definition of canonical bytes for both artefacts: fixed
// field order from the struct, two-space indent, HTML escaping off, exactly one
// trailing newline. A document that encoded differently depending on how it was
// built would turn the determinism gate into a coin flip.
func encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encoding reading artefact: %w", err)
	}
	return buf.Bytes(), nil
}

// EncodeBundle renders the assembled input as canonical JSON.
func EncodeBundle(b Bundle) ([]byte, error) { return encode(b) }

// EncodeManifest renders the manifest as canonical JSON.
func EncodeManifest(m Manifest) ([]byte, error) { return encode(m) }

// DecodeManifest reads a manifest strictly: unknown fields, trailing content
// and a schema-version mismatch are all refused. All three are fail-closed on
// purpose, because a manifest is the evidence a reader judges contamination by.
//
// It has NO front door yet, and that is deliberate rather than an oversight: the
// verb that reads a manifest back is the ingest, which spc-63 owns and which is
// not in this delivery. It is exported and tested here because the writer and
// the reader of one format belong together — a decoder written later, against
// the file rather than against the encoder, is how the two drift.
func DecodeManifest(data []byte) (Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decoding reading manifest: %w", err)
	}
	if dec.More() {
		return Manifest{}, fmt.Errorf("decoding reading manifest: trailing content after the document")
	}
	if m.Type != ManifestType {
		return Manifest{}, fmt.Errorf("decoding reading manifest: _type is %q, want %q", m.Type, ManifestType)
	}
	if m.SchemaVersion != SchemaVersion {
		return Manifest{}, fmt.Errorf("decoding reading manifest: schema_version is %d, want %d",
			m.SchemaVersion, SchemaVersion)
	}
	// An item's kind is REFUSED when absent rather than defaulted, so the
	// not-omitempty decision on the write side is a property of the format
	// rather than a habit of this writer. Without this, an item carrying no
	// kind decodes clean and the strictness the type's own comment claims is
	// true of what the binary writes and false of what it reads — an
	// attestation asserting more than its examination establishes, which brief
	// invariant 16 forbids.
	known := make(map[Kind]bool, len(Kinds()))
	for _, k := range Kinds() {
		known[k] = true
	}
	// The scan mark is refused on exactly the same ground, and it is the
	// stronger case of the two: a manifest whose item carries no mark cannot say
	// whether its key and heading exclusions were established for that item, so
	// decoding it clean would hand a reader an exclusion assertion with nothing
	// behind it (itd-194 ac-4).
	knownScans := make(map[Scan]bool, len(Scans()))
	for _, s := range Scans() {
		knownScans[s] = true
	}
	for i, it := range m.Items {
		if it.Kind == "" {
			return Manifest{}, fmt.Errorf("decoding reading manifest: item %d (%s) carries no kind",
				i, it.ItemKey)
		}
		if !known[it.Kind] {
			return Manifest{}, fmt.Errorf("decoding reading manifest: item %d (%s) carries the "+
				"unknown kind %q; the vocabulary is closed", i, it.ItemKey, it.Kind)
		}
		if it.Scan == "" {
			return Manifest{}, fmt.Errorf("decoding reading manifest: item %d (%s) carries no "+
				"\"scan\" mark, so the manifest cannot say whether the exclusion floor examined it",
				i, it.ItemKey)
		}
		if !knownScans[it.Scan] {
			return Manifest{}, fmt.Errorf("decoding reading manifest: item %d (%s) carries the "+
				"unknown \"scan\" mark %q; the vocabulary is closed", i, it.ItemKey, it.Scan)
		}
	}
	return m, nil
}

// ManifestHash is the manifest's own content hash over its canonical bytes. It
// is the reference an ingest cites back.
func ManifestHash(m Manifest) (string, error) {
	data, err := EncodeManifest(m)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

// sha256Hex is the one hash this package computes: item text and manifest bytes.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
