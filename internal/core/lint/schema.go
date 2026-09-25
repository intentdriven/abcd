package lint

// The record-schema family (record_schema): the mechanical shape of the record's
// identified stores — ADRs, intents, specs, issues.
//
// Every other record rule asks a question about ONE store (is this intent's
// bucket schema right, does this spec agree with its intent). None of them asks
// the questions that only make sense ACROSS the stores, which is where the record
// actually drifts (iss-39):
//
//   - a cross-reference names a record that is not in the corpus;
//   - a supersession is declared from one side only, so the record contradicts
//     itself about which decision is in force;
//   - a filename and the id inside it disagree, so the same record answers to two
//     handles;
//   - a lifecycle directory nobody declared holds records no rule ever reads.
//
// The last one is why this rule enumerates the buckets rather than allowlisting
// them per store: an undeclared directory is not "clean", it is a lifecycle state
// that silently escapes every other rule.
//
// It is a STRUCTURAL rule, so it never consults contentExempt — the historical
// part of the record is excused from content-authoring rules, not from being
// well-formed (iss-39; a superseded intent that carries no supersession is the
// exact defect the broad exemption used to hide).

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

const ruleRecordSchema = "record_schema"

var (
	// A record handle as it is written in a cross-reference field: adr-6, ADR-6,
	// itd-47, iss-39, spc-5. The number is compared numerically, so the zero-padded
	// and bare spellings of one id are the same handle. The alternation covers
	// every store the rule INDEXES — a prefix indexed but not matched here reads as
	// "no handle at all", which turns a well-formed link into a false blocker and
	// leaves its reverse direction unchecked.
	recordHandleRe = regexp.MustCompile(`(?i)\b(adr|itd|iss|spc)-(\d+)\b`)
	// The same handle, anchored: a whole frontmatter id value and nothing else.
	recordHandleFullRe = regexp.MustCompile(`(?i)^(adr|itd|iss|spc)-(\d+)$`)
	// A frontmatter id of ANY store, parsed by shape rather than against the
	// cross-reference vocabulary above. The two are deliberately different sets:
	// the vocabulary says which prefixes may appear in a cross-reference FIELD,
	// while this says what an id looks like at all. The filename ↔ id agreement is
	// a question every store has to answer, including the ones whose records are
	// never cited — and checking it against the citation vocabulary would report a
	// perfectly good rdi-N id as disagreeing with itself.
	anyHandleFullRe = regexp.MustCompile(`(?i)^([a-z]+)-(\d+)$`)
	// recordHandleKinds renders the legal prefixes for a message, composed from the
	// stores rather than spelled as a literal so it cannot drift from the pattern.
	recordHandleKinds = "adr-N, itd-N, spc-N, or iss-N"
	// Filename → id number for each store. An ADR filename is the zero-padded
	// sequential form (0012-<slug>.md); the other three carry the prose handle and
	// borrow the recordid resolver's OWN grammar, not a looser local copy.
	//
	// The prose-handle stores match recordid.FilenameNumRe exactly, so lint reads a
	// filename as a record iff the resolver does: the old `^iss-(\d+).*\.md$` and
	// its siblings accepted an arbitrary tail (`iss-5_bad.md`), which capture's
	// scanLedger silently dropped and the resolver hard-errored on when the record
	// was cited — a record the gate passed but no consumer could read
	// (iss-2608270908346617). The ADR store keeps its own pattern (it has no
	// prose-handle prefix) but is tightened to a kebab slug tail for the same
	// reason: no arbitrary bytes after the ordinal.
	adrFileNumRe    = regexp.MustCompile(`^([0-9]+)-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	intentFileNumRe = recordid.FilenameNumRe("itd")
	specFileNumRe   = recordid.FilenameNumRe("spc")
	issueFileNumRe  = recordid.FilenameNumRe("iss")
	// The reading families' filename and bucket grammars. A run directory is
	// named for the run that minted it; a disposition directory is named for the
	// ITEM it answers, which is what makes the status signal one directory probe
	// rather than a folder-membership question.
	readingItemFileNumRe = recordid.FilenameNumRe(issueschema.ReadingItemFamily)
	readingRunFileNumRe  = recordid.FilenameNumRe(issueschema.ReadingRunFamily)
	dispositionFileNumRe = recordid.FilenameNumRe(issueschema.DispositionFamily)
	readingRunBucketRe   = regexp.MustCompile(`^` + issueschema.ReadingRunFamily + `-[0-9]+$`)
	dispositionBucketRe  = regexp.MustCompile(`^` + issueschema.ReadingItemFamily + `-[0-9]+$`)
	// The step-2 families' filename grammars. An admission is bucketed by the
	// same minted run directory a reading is (readingRunBucketRe above); a
	// surprise store is FLAT, because a surprise is keyed by the `occasioned_by`
	// it carries rather than by a directory.
	admissionFileNumRe = recordid.FilenameNumRe(issueschema.AdmissionFamily)
	surpriseFileNumRe  = recordid.FilenameNumRe(issueschema.SurpriseFamily)
	// A YAML block-scalar header and nothing else: `|`, `>`, with the chomping and
	// indentation indicators the spelling allows (`|-`, `>+`, `|2-`). A key
	// carrying one holds its value on the lines BELOW it, so the same-line scanner
	// reports the header as the value — which is why a rule asking "is this
	// property empty?" has to look at the block rather than at the byte.
	// The trailing comment YAML permits after the header is part of the header,
	// not of the value: `grounds: | # nothing` over an empty block is the empty
	// string exactly as a bare `grounds: |` is, and a pattern that read only the
	// bare spellings left the reported defect passing under four legal ones.
	// The pattern is the scanner package's own, so the record readers that ask
	// "is this a single-line string?" and this gate recognise one header.
	blockScalarIndicatorRe = frontmatter.BlockScalarHeaderRe
	// The cross-reference frontmatter fields whose targets must resolve. They are
	// the record's machine-readable claims that another record exists and is a
	// live input — as distinct from prose, where naming a released or retired id
	// ("itd-38's id was released, not reserved") is legitimate narration.
	//
	// `supersedes` is deliberately NOT here. It is the field that DECLARES a
	// pruned id, so it can never fail its own resolution check; listing it would
	// read as coverage it does not provide. Its targets are checked separately,
	// against the store's allocation high-water mark.
	recordRefFields = []string{"related_adrs", "related_intents", "builds_on", "blocked_by"}
	// Every field the rule reads handles out of, so the scan parses each once.
	recordHandleFields = append([]string{"supersedes", "superseded_by"}, recordRefFields...)
	// recordGraphFields are cross-reference fields the record carries that this
	// rule does not judge — the intent↔spec pair, whose resolution is the spec
	// store's own rule — but which the exported record graph (graph.go) must
	// carry, because they are half the record's typed links. They are parsed by
	// the same scan so the graph never needs a second parser: reading them here
	// costs one regexp pass per record and adds no finding.
	recordGraphFields = []string{"spec_id", "intent"}
	// recordParsedFields is every field the scan reads handles out of, in a fixed
	// order so the graph export is deterministic.
	recordParsedFields = append(append([]string{}, recordHandleFields...), recordGraphFields...)
)

// recordStore describes one identified record store: where the buckets are, and
// how a well-formed filename spells the id.
type recordStore struct {
	// prefix is the id prefix (adr, itd, spc, iss) and the record_stores key.
	prefix string
	// noun names the record kind in a finding message.
	noun string
	// nodeType is the store's name in the exported record graph (graph.go). It
	// is the store's identity spelled for a consumer rather than for a message,
	// and it lives here so the graph never re-declares which prefixes exist.
	nodeType string
	// buckets are the lifecycle directories the store declares. A store with
	// neither buckets nor bucketRe is FLAT (the ADR store): records sit directly
	// in it.
	buckets []string
	// bucketRe declares the store's buckets by GRAMMAR instead of by list, for a
	// store whose buckets are MINTED rather than enumerated — a reading run's
	// directory and an item-keyed disposition directory are both that shape.
	// Nobody can list them ahead of time, so a store that could not declare a
	// grammar would have to leave its whole tree undeclared, which is exactly the
	// escape the undeclared-bucket check exists to close.
	bucketRe *regexp.Regexp
	// fileNumRe extracts the id number from a filename (submatch 1).
	fileNumRe *regexp.Regexp
	// fileFamily is the family prefix a filename in this store carries, spelled
	// without its hyphen — the argument recordid.SplitRecordFilename takes. It is
	// NOT always the store's prefix: an ADR filename is the bare zero-padded
	// number (0022-<slug>.md), so its family is empty. Held as data rather than
	// derived from prefix with a special case, because a store that spells its
	// filenames differently is a property of the store.
	fileFamily string
	// filename describes the convention a finding message quotes.
	filename string
	// requiredFields are the frontmatter properties every record in this store
	// must carry. A nil requiredFields means the store declares none HERE and is
	// left alone — the four stores have different schemas, and the ones whose
	// frontmatter other rules already judge (intent_lifecycle, spec_id_unique)
	// must not be re-judged against the issue's shape.
	requiredFields []string
	// knownFields is the store's additionalProperties:false allow-list. A nil
	// knownFields means the store declares no closed schema here and every key is
	// left to the rules that know it — the intent and spec stores carry fields
	// their own rules judge, and refusing them against a list this rule invented
	// would be a blocker over a schema nobody declared.
	knownFields map[string]bool
	// joins are the frontmatter properties that key this record to another
	// record, and whose targets must therefore be in the corpus. They are
	// distinct from recordRefFields, which is the fixed cross-reference
	// vocabulary every store shares: a join is the ONE value that makes a record
	// a record about something else, and each store spells its own.
	joins []recordJoin
	// readerFailsClosed declares that the store's reader VALIDATES a record before
	// it reads it, and skips one that fails — so a malformed committed record is
	// invisible to every surface of its family while it still sits in the store.
	// Two readers do this: capture's validateStrict for the issue ledger, and the
	// record dispatcher for the ADR store, which confirms the frontmatter id before
	// it will render the record.
	//
	// It is declared rather than assumed because the findings STATE that
	// consequence, and a rule may not state a consequence it has not established.
	// ONE declaration stands behind the two legs that read the record's PROPERTIES
	// — the missing required property (checkRecordRequiredFields) and the key
	// outside a closed schema (checkRecordUnknownFields) — rather than one leg
	// consulting it while the one beside it assumes, which is how the first fix
	// closed one spelling and left two (iss-2608301519254418).
	//
	// It does NOT stand behind the duplicated top-level key. The malformations
	// differ and so do the readers' answers: capture's parse refuses a duplicate
	// outright, while the ADR dispatcher reads with frontmatter.Fields, the lenient
	// scanner, and renders the record on its first value — so one declaration across
	// all three legs made the ADR store claim a refusal nobody performs
	// (iss-2608301656200729). That leg reads readerRefusesDuplicateKey instead.
	//
	// The two stores this cycle added have no such reader: the outstanding report
	// honours an admission carrying nothing but its run and its proposal
	// (reading_outstanding_test.go), and the record dispatcher reads both families
	// leniently and skips neither — so a message telling their authors the record
	// is skipped and invisible sends them to look for a refusal nobody performs
	// (iss-2608301411010342).
	// Where it is false each leg states what the malformation IS, which is true of
	// every store, and stops there.
	readerFailsClosed bool
	// readerRefusesDuplicateKey declares that the store's reader refuses a record
	// carrying a top-level key twice, and skips it. It is separate from
	// readerFailsClosed because the two answers come apart: capture's
	// parseFrontmatterBlock refuses a duplicate key by name, and the ADR dispatcher
	// — which DOES validate the id before it renders — reads the frontmatter with
	// the lenient scanner and so never sees the second line at all. An ADR carrying
	// `status` twice renders, with the first value and a nil error
	// (iss-2608301656200729).
	//
	// Where it is false the leg still reports the duplicate, on the account that is
	// true of every store: this rule's own scanner keeps the first value, so a
	// second line can silence a blocker armed on the value the first hides. That
	// clause is scoped to THIS rule's scanner rather than to "every record surface"
	// because the surfaces do not answer alike (iss-2608301813253101).
	//
	// What each store's readers make of a duplicated key is established by
	// EXERCISING them, in TestDuplicateTopLevelKeyReaderByReader. It is a test
	// rather than a list here because the list here was written four times and
	// overtaken four times: prose about another component's behaviour cannot fail,
	// so it goes stale silently and is believed BECAUSE it is specific. The last
	// one gave each store a single answer, and rdi and dsp each had a second reader
	// that contradicted the answer it was given (iss-2608301901264848). Read the
	// table for what a reader does; a wrong row there is a red gate.
	readerRefusesDuplicateKey bool
	// bucketField is the frontmatter property that must name the bucket the
	// record sits in, for a store that states its bucket twice. An admission is
	// filed under the run whose candidate set it joins AND carries that run as a
	// field, so a disagreement is the record contradicting itself about which set
	// it joined. Empty means the store makes no such double claim.
	bucketField string
}

// recordJoin is one keying field a store declares, with what the join is FOR —
// so a dangling target tells the reader what broke, not merely which key did.
type recordJoin struct {
	field string
	why   string
	// sameBucketAs names the FAMILY this join's value is a handle of, and whose
	// targets must sit in the same bucket as the record that joins them. Empty
	// means the join carries neither obligation: its value may be prose.
	//
	// Naming the family is what makes the SPELLING judgeable. A join whose value
	// may be prose can only be read leniently, and reading this one leniently left
	// six spellings of the proposal green while the reader that matches the value
	// as a string admitted nothing: three that parsed and resolved (upper-cased,
	// mixed-cased, zero-padded), and three that parsed as no handle at all and
	// took the silence prose is owed (a space inside the quotes on either side,
	// and bare prose) (iss-2608301519255871).
	//
	// An admission is meaningful only against the run whose proposals it admits,
	// and the outstanding report keys the admitted set on the PAIR — the run the
	// record is filed under, and the proposal it names (admittedProposals) — so an
	// admission filed under one run naming an item that belongs to another is keyed
	// on a pair nothing ever queries. It admits nothing, the proposal it names goes
	// on being reported as unadmitted, and no line says an answer was written
	// (iss-2608301327013320).
	//
	// That consequence is what the walk establishes, and it is the whole of what
	// the message may say. It does NOT rest on ids colliding across buckets: this
	// rule's own duplicate-ordinal leg refuses a cross-run reading collision
	// outright, and mintUnusedItemID probes every run under the ledger lock and
	// redraws on a hit, so a rule asserting the collision would be contradicting
	// itself twice over (iss-2608301519253368).
	//
	// It names a family rather than being a bare flag because the pair is a
	// PROPERTY OF THE FAMILY, and the message says so. The outstanding report walks
	// reading items and keys them by the run they sit in; it never names an issue at
	// all, and no reader keys an issue on its status directory. So a target outside
	// the declared family is left to the silence it had: its value is wrong for some
	// other reason, and inventing a consequence the operator then goes looking for
	// is the confident false statement this rule must not make
	// (iss-2608301411017768).
	//
	// checkRecordBucketField does NOT cover this and never did. That check enforces
	// FIELD == DIRECTORY; this is DIRECTORY == THE TARGET'S OWN BUCKET, and a
	// record satisfies the first while failing the second.
	sameBucketAs string
	// targetPosition is the reading POSITION the join's target must hold, where
	// what reads the join is scoped to one position. Empty means the join carries
	// no such obligation.
	//
	// The admission's is `widening`, and it is the THIRD coordinate of the pair the
	// run axis and the spelling axis already close. ReadReadingOutstanding consults
	// the admissions tree only inside `if position == issueschema.PositionWidening`,
	// so an admission naming an item at `detection`, `entailment` or `comparative`
	// resolves, matches its bucket, spells correctly, draws nothing — and is never
	// queried (iss-2608301649339636). AdmissionRequired's own doc already says
	// `proposal` names the WIDENING item admitted; this is the leg that enforces it.
	//
	// It is asked only where the target's FILENAME is a bare handle, for the padding
	// leg's reason: what reads the family never opens a file whose name is not one,
	// so nothing about what the report does with such a target is available to
	// claim. That test needs the target's family, so this obligation is carried by a
	// join that ALSO declares sameBucketAs; declared alone it would be inert, which
	// TestEveryJoinTargetPositionIsADeclaredPosition refuses.
	targetPosition string
	// oneOf names the CLOSED set of families this join's value must be a handle
	// of, verbatim, for a join whose value may name one of several families.
	// Empty means the join declares no such set.
	//
	// The surprise's `occasioned_by` is the one such join: an rdi-N, adm-N or
	// dsp-N and nothing else (spc-2609020626040342). It used to admit a
	// consequence named in prose, so a prose value passed the gate while the
	// surprise verb refuses it — a hand-written record joined to nothing. The set
	// is issueschema's one declaration (SurpriseOccasionFamilies), so the verb and
	// this gate cannot disagree about it. A value in the set then resolves on the
	// ordinary presence leg below.
	oneOf []string
}

// bucketed reports whether the store holds its records in lifecycle
// directories, by list or by grammar, rather than directly in its root.
func (s recordStore) bucketed() bool { return s.buckets != nil || s.bucketRe != nil }

// declaresBucket reports whether name is one of the store's buckets.
func (s recordStore) declaresBucket(name string) bool {
	for _, b := range s.buckets {
		if b == name {
			return true
		}
	}
	return s.bucketRe != nil && s.bucketRe.MatchString(name)
}

// bucketDesc renders the store's declared buckets for a finding message: the
// list where there is one, the grammar where the buckets are minted.
func (s recordStore) bucketDesc() string {
	if len(s.buckets) > 0 {
		return strings.Join(s.buckets, ", ")
	}
	if s.bucketRe != nil {
		return "names matching " + s.bucketRe.String()
	}
	return ""
}

// recordStores is the closed set of identified record stores. It is code, not
// config: which lifecycle states exist is the record's schema, and a config that
// could add a bucket could also hide one.
var recordStores = []recordStore{
	// `id` is required for an ADR because the record dispatcher (record.describeADR)
	// routes by the filename ordinal but CONFIRMS the frontmatter id before it will
	// render the record — so an id-less ADR reads as "not found" though its file
	// plainly sits in the store, while the lint that never asked for the id stayed
	// green (iss-2608270908344426). This is parity with the prose-handle stores,
	// whose loaders (intent.Load) fail closed on a missing id: the id is a required
	// property, and its absence must be a finding, not silent invisibility.
	{prefix: "adr", noun: "ADR", nodeType: "adr", buckets: nil, fileNumRe: adrFileNumRe, fileFamily: "", filename: "<NNNN>-<slug>.md",
		requiredFields: []string{"id"}, readerFailsClosed: true},
	{prefix: "itd", noun: "intent", nodeType: "intent", buckets: intentBucketNames, fileNumRe: intentFileNumRe, fileFamily: "itd", filename: "itd-<N>-<slug>.md"},
	{prefix: "spc", noun: "spec", nodeType: "spec", buckets: specBucketNames, fileNumRe: specFileNumRe, fileFamily: "spc", filename: "spc-<N>-<slug>.md"},
	// The issue store's required properties come from the schema's ONE definition
	// (core/issueschema), the same list the ledger reader validates against — a
	// hand-copied list here would drift the moment the schema gains a field, and
	// the drift would show up as a silently unread record, which is the defect
	// this invariant exists to catch.
	{prefix: "iss", noun: "issue", nodeType: "issue", buckets: issueStatusDirs, fileNumRe: issueFileNumRe, fileFamily: "iss", filename: "iss-<N>-<slug>.md",
		requiredFields: issueschema.Required, knownFields: issueschema.Known, readerFailsClosed: true,
		readerRefusesDuplicateKey: true},
	// The three reading families (spc-58). Each buckets by GRAMMAR because its
	// buckets are minted: a reading item and a run record live under the run that
	// produced them, and a disposition lives under the item it answers.
	//
	// Their required-field sets are absent, and that is a STATED GAP rather than a
	// delegation. What this rule gives them is structural — the bucket grammar,
	// the filename ↔ id agreement, no undeclared lifecycle directory — and their
	// CONTENT is judged by the writer that refuses a malformed record at the
	// boundary, and by review. So the guarantee the issue store has, that a record
	// the reader would refuse is not lint-green, does not yet hold here: a record
	// hand-written into these trees can carry a body no reader reads and pass this
	// gate. Declaring their required fields is what closes it, and until that
	// lands the gap belongs in writing rather than in the difference between two
	// store entries.
	{prefix: "rdi", noun: "reading item", nodeType: "reading", bucketRe: readingRunBucketRe,
		fileNumRe: readingItemFileNumRe, fileFamily: "rdi", filename: "rdi-<N>.md"},
	{prefix: "rdg", noun: "reading run", nodeType: "reading-run", bucketRe: readingRunBucketRe,
		fileNumRe: readingRunFileNumRe, fileFamily: "rdg", filename: "rdg-<N>.md"},
	{prefix: "dsp", noun: "disposition", nodeType: "disposition", bucketRe: dispositionBucketRe,
		fileNumRe: dispositionFileNumRe, fileFamily: "dsp", filename: "dsp-<N>.md"},
	// The two step-2 families (spc-67). Unlike the three above they DO declare
	// their schemas, and that is the whole of this cycle's enforcement: no verb
	// writes an admission or a surprise yet, so wiring the shapes to the gate that
	// reads committed records is what keeps a schema no code reads from being dead
	// scaffolding. A hand-written admission with a blank `grounds` is a blocker
	// finding from the day this lands; what is hand-run is WHO writes the file.
	//
	// Both lists come from core/issueschema's ONE declaration rather than from a
	// literal here, for the reason the issue store's does: a hand-copied set
	// drifts the moment one side gains a field.
	//
	// Both declare a JOIN, and the admission declares its bucket twice. The
	// admission's `proposal` used to be the one keying field the rule left
	// unresolved while it resolved the surprise's `occasioned_by` — the same
	// defect (a join naming nothing joins nothing) asked in one store and not the
	// other, which is exactly how two answers to one question start.
	{prefix: "adm", noun: "admission", nodeType: "admission", bucketRe: readingRunBucketRe,
		fileNumRe: admissionFileNumRe, fileFamily: "adm", filename: "adm-<N>.md",
		requiredFields: issueschema.AdmissionRequired, knownFields: issueschema.AdmissionKnown,
		bucketField: "run",
		joins: []recordJoin{{
			field: "proposal",
			why: "an admission is keyed to the proposal it admits, and one naming no record admits nothing in particular — " +
				"the candidate set it claims to have joined cannot be reconstructed from it",
			sameBucketAs:   issueschema.ReadingItemFamily,
			targetPosition: issueschema.PositionWidening,
		}}},
	{prefix: "srp", noun: "surprise", nodeType: "surprise",
		fileNumRe: surpriseFileNumRe, fileFamily: "srp", filename: "srp-<N>.md",
		requiredFields: issueschema.SurpriseRequired, knownFields: issueschema.SurpriseKnown,
		joins: []recordJoin{{
			field: "occasioned_by",
			why:   "a surprise is keyed to the record that occasioned it, and a join naming nothing joins nothing",
			oneOf: issueschema.SurpriseOccasionFamilies,
		}}},
}

// storeByPrefix returns the code-side store for a prefix.
func storeByPrefix(prefix string) (recordStore, bool) {
	for _, s := range recordStores {
		if s.prefix == prefix {
			return s, true
		}
	}
	return recordStore{}, false
}

// recordStorePrefixes is the set of store prefixes the scanner knows, derived
// from recordStores so the configuration validator and the walk cannot disagree
// about what a store is.
func recordStorePrefixes() map[string]bool {
	out := make(map[string]bool, len(recordStores))
	for _, s := range recordStores {
		out[s.prefix] = true
	}
	return out
}

// schemaRecord is one record file as the schema rule sees it: which store and
// bucket hold it, the id number its FILENAME claims, and its frontmatter.
type schemaRecord struct {
	rel    string
	store  recordStore
	num    int
	bucket string
	// title is the record's H1, or — for a store whose records carry none, the
	// issue ledger — its first body line. The schema rule never reads it; it is
	// read here because the scan already holds the file's lines, and the record
	// graph (graph.go) would otherwise have to open every record a second time.
	title  string
	fields map[string]fmField
	// refs holds the handles read out of each cross-reference field, parsed once
	// at scan time so the block-sequence spelling is read in ONE place. Reading
	// only fields[...].value would miss it: the shared frontmatter scanner is a
	// same-line scanner, so `supersedes:` followed by indented `- adr-12` lines
	// reads as empty — and an empty read here would make the bidirectional check
	// assert, confidently and falsely, that another file omits a link it carries.
	refs map[string][]recordRef
	// blocks holds the block-spelled value of every key whose same-line value is
	// empty, read once at scan time for the same reason refs is. Without it a rule
	// asking "is this scalar property present?" answers no for a value the file
	// plainly carries on the following lines, and goes green on a record the
	// reader refuses and skips (iss-2608300234599781).
	blocks map[string]string
}

// handle renders the record's prose handle (adr-12, itd-47).
func (r schemaRecord) handle() string {
	return r.store.prefix + "-" + strconv.Itoa(r.num)
}

// valueEmpty reports whether the record carries no value for field — the
// question isAbsentValue answers for a same-line value, widened by the one
// spelling whose value is not on the key's own line at all.
//
// `grounds: |` puts the value on the indented lines below, so the same-line
// scanner reads the indicator itself: a non-empty byte that is not a value. A
// block holding text is a value; a block holding nothing is the empty string,
// which is the same absence every other spelling is and must be refused on the
// same terms rather than passing on the strength of a `|`.
func (r schemaRecord) valueEmpty(field string, f fmField) bool {
	if blockScalarIndicatorRe.MatchString(strings.TrimSpace(f.value)) {
		return strings.TrimSpace(r.blocks[field]) == ""
	}
	return isAbsentValue(f.value)
}

// recordRef is one handle read out of a cross-reference field.
type recordRef struct {
	prefix string
	num    int
}

func (h recordRef) String() string { return h.prefix + "-" + strconv.Itoa(h.num) }

// checkRecordSchema implements the record_schema family. It reads each configured
// store once and asserts five invariants across them: bucket coverage, filename ↔
// id agreement, filename ↔ slug agreement, cross-reference resolution, and
// bidirectional supersession. A
// store that is not configured, or whose directory is absent, contributes nothing
// and is not an error — an unpopulated repository is a state, not a fault.
func checkRecordSchema(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	records, out, err := scanRecordStores(repoRoot, cfg)
	if err != nil {
		return nil, err
	}

	// index: what the corpus HAS. highWater: the highest id each store has ever
	// issued, as far as the corpus can show.
	//
	// records is sorted by path, so on an ordinal collision the FIRST record wins
	// the index slot and every later claimant is reported: two records sharing a
	// (prefix, ordinal) handle are one handle that resolves to only the first, so
	// the second is unreachable to every cross-reference and index that keys on it —
	// silently, before this check, because the map assignment just overwrote it
	// (iss-2608270908346940). For the prose-handle stores the id-unique rules
	// (issue_id_unique, intent_lifecycle, spec_id_unique) catch the frontmatter-id
	// collision; the ADR store has no such rule, so this is its only guard.
	index := map[recordRef]schemaRecord{}
	highWater := map[string]int{}
	for _, r := range records {
		ref := recordRef{r.store.prefix, r.num}
		if first, dup := index[ref]; dup {
			out = append(out, Finding{
				File: r.rel, Line: 1, RuleID: ruleRecordSchema, Severity: cfg.Severity,
				Message: "filename ordinal '" + r.handle() + "' collides with " + first.rel +
					"; two " + r.store.noun + " records sharing an id are one handle that resolves to only the first, so this one is unreachable to every cross-reference and index that keys on it",
			})
		} else {
			index[ref] = r
		}
		if r.num > highWater[r.store.prefix] {
			highWater[r.store.prefix] = r.num
		}
	}

	// retired: ids a record declares it replaced. The ADR lifecycle prunes a
	// superseded record once its successor lands and keeps the trace in the
	// successor's `supersedes` (decisions/adrs/README.md), so a handle naming a
	// pruned id resolves to that declaration rather than to a file — the record
	// has not lost track of it.
	//
	// The declaration is BOUNDED by the store's allocation high-water mark,
	// because it is otherwise self-attesting: `supersedes: [adr-9999]` would mint
	// a phantom the rule then accepts in every other record's related_adrs,
	// silently reopening the very class of reference this rule exists to close. An
	// id above the high-water mark was never issued, so nothing can have pruned
	// it, and the declaration itself is reported below.
	retired := map[recordRef]bool{}
	for _, r := range records {
		for _, h := range r.refs["supersedes"] {
			if h.num >= 1 && h.num <= highWater[h.prefix] {
				retired[h] = true
			}
		}
	}

	add := func(rel string, line int, msg string) {
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity, Message: msg,
		})
	}

	for _, r := range records {
		// judged names the properties a leg that judges CONTENT has already reported
		// for this record. What a reader does with a BLANK required property is a
		// property of the field, not of the store — capture refuses a blank severity
		// and reads a blank found_during — so the required-fields leg, which can only
		// speak store-wide, must leave the consequence to the leg that established it
		// (iss-2608301308369559). The content legs therefore run FIRST and mark what
		// they spoke about, so nobody has to keep a second list of which fields those
		// are, and a leg added later is covered by having said something.
		judged := map[string]bool{}
		out = append(out, checkRecordFilename(r, cfg.Severity, judged)...)
		out = append(out, checkRecordFilenameSlug(r, cfg.Severity, judged)...)
		out = append(out, checkIssueRecordShape(r, cfg.Severity, judged)...)
		out = append(out, checkRecordRequiredFields(r, cfg.Severity, judged)...)
		out = append(out, checkRecordUnknownFields(r, cfg.Severity)...)
		out = append(out, checkRecordJoins(r, index, retired, cfg)...)
		out = append(out, checkRecordBucketField(r, cfg.Severity)...)

		// Cross-references: a named record must be in the corpus, or declared
		// retired by the record that replaced it.
		for _, field := range recordRefFields {
			f := r.fields[field]
			for _, h := range r.refs[field] {
				if _, ok := index[h]; ok || retired[h] {
					continue
				}
				add(r.rel, f.line, field+" names '"+h.String()+"', which is not a record in the corpus and no record declares it superseded; a cross-reference is a claim that the record exists")
			}
		}

		// A pruned id must be one the store actually issued. Without this, the
		// retirement declaration is a blank cheque: any id written here becomes
		// resolvable everywhere else in the corpus.
		sup := r.fields["supersedes"]
		for _, h := range r.refs["supersedes"] {
			if _, ok := index[h]; ok {
				continue
			}
			if h.num >= 1 && h.num <= highWater[h.prefix] {
				continue // a plausible pruned id: below the store's high-water mark
			}
			add(r.rel, sup.line, "supersedes names '"+h.String()+"', which is neither a record in the corpus nor an id the "+
				h.prefix+" store has issued (its highest is "+h.prefix+"-"+strconv.Itoa(highWater[h.prefix])+
				"); a pruned record's id must be one that was allocated")
		}

		// Supersession, direction A→B: the successor must exist (a live decision is
		// never a pruned one) and must name this record back.
		sb := r.fields["superseded_by"]
		targets := r.refs["superseded_by"]
		if len(targets) == 0 {
			if !isAbsentValue(sb.value) {
				add(r.rel, sb.line, "superseded_by '"+sb.value+"' is not a record handle (want "+recordHandleKinds+")")
			}
			continue
		}
		for _, h := range targets {
			target, ok := index[h]
			if !ok {
				add(r.rel, sb.line, "superseded_by names '"+h.String()+"', which is not a record in the corpus; a successor decision must be present")
				continue
			}
			if !refsContain(target.refs["supersedes"], recordRef{r.store.prefix, r.num}) {
				add(r.rel, sb.line, "one-way supersession: this record declares superseded_by '"+h.String()+
					"' but "+target.rel+" does not list '"+r.handle()+"' in supersedes; both directions must be present")
			}
		}
	}

	// Supersession, direction B→A: a record that claims to replace another must be
	// named by it. A target that is not in the corpus was pruned with the record's
	// blessing (see above) and has nothing left to answer with.
	for _, r := range records {
		sup := r.fields["supersedes"]
		for _, h := range r.refs["supersedes"] {
			target, ok := index[h]
			if !ok {
				continue
			}
			if !refsContain(target.refs["superseded_by"], recordRef{r.store.prefix, r.num}) {
				add(r.rel, sup.line, "one-way supersession: this record declares supersedes '"+h.String()+
					"' but "+target.rel+" does not carry 'superseded_by: "+r.handle()+"'; both directions must be present")
			}
		}
	}

	return out, nil
}

// checkRecordFilename asserts that a record's frontmatter id agrees with the id
// its filename claims. The filename is the handle every cross-reference and every
// index row resolves through, so a disagreement means one record answers to two
// ids — and the rules that key on the filename and the rules that key on the
// field then lint two different records. An absent id is a different (and larger)
// schema question than this rule's, so only a present one is compared.
func checkRecordFilename(r schemaRecord, severity string, judged map[string]bool) []Finding {
	f := r.fields["id"]
	// Absence here means NOT WRITTEN, which is why it is isNull and not
	// isAbsentValue: an EMPTY id is a value that disagrees with the filename, and
	// the delegation below only reaches a store that declares a required set. The
	// intent, spec and three reading stores declare none, so reading `id: ""` as
	// absence would send the delegation nowhere and leave it green in every rule.
	if isNull(strings.TrimSpace(f.value)) {
		return nil
	}
	want := r.handle()
	got := issueScalar(f.value)
	// Compared as a PARSED handle, not as a string: `adr-0012` and `adr-12` are one
	// id written two ways (the rest of the rule already compares numerically), and
	// a string comparison would report the record's own zero-padded spelling as a
	// disagreement with itself.
	if m := anyHandleFullRe.FindStringSubmatch(got); m != nil {
		if n, err := strconv.Atoi(m[2]); err == nil &&
			strings.EqualFold(m[1], r.store.prefix) && n == r.num {
			return nil
		}
	}
	line := f.line
	if line == 0 {
		line = 1
	}
	mark(judged, "id")
	return []Finding{{
		File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity,
		Message: "filename claims id '" + want + "' but frontmatter declares '" + got +
			"'; a " + r.noun() + " filename is " + r.store.filename,
	}}
}

// mark records that a leg of this rule has reported on field for the record
// being checked, so the required-fields leg does not add a second and weaker
// finding about the same value. It tolerates a nil set, so a leg stays callable
// without one.
func mark(judged map[string]bool, field string) {
	if judged != nil {
		judged[field] = true
	}
}

// checkRecordFilenameSlug asserts that a record's frontmatter slug agrees with
// the slug its filename carries — the other half of the agreement
// checkRecordFilename pins for the id. Both halves are one value the store wrote
// twice, and a record renamed by hand keeps a stale handle in its name while its
// field says something else: readers that locate a record by filename and readers
// that trust the field then describe two different records, and neither says so.
//
// Until this landed the question was asked only by the LEDGER READER
// (core/capture's validateInvariants), which means a drifted record passed
// record-lint and every CI gate and then dropped to a Skipped line at read time —
// invisible to `capture list`, `capture status` and every other surface. Turning
// that silent skip into a red gate is why the check belongs in the structural
// rule as well as in the reader, exactly as checkRecordRequiredFields does.
//
// The filename is split by recordid.SplitRecordFilename, the SAME call the reader
// makes, so the gate and the reader cannot reach different verdicts on one record.
// The comparison is exact, with no prefix tolerance: every store applies its slug
// length cap while DERIVING the slug, before that one value forks into the
// filename and the frontmatter, so a filename is never a truncated form of a
// longer field and tolerating a prefix would license the drift this catches.
//
// Two cases are silent here, for different reasons. A record carrying no slug is
// a different (and larger) schema question, owned by checkRecordRequiredFields
// for the stores that declare the property — absence is not disagreement.
//
// A filename this splitter cannot read is silent too, and that one is a KNOWN
// GAP, not a delegation. The store's own filename rule in scanRecordStores does
// not cover it: that rule matches store.fileNumRe, whose issue pattern
// (`^iss-(\d+).*\.md$`) accepts an ARBITRARY tail, so a non-kebab name like
// iss-2-another_finding.md clears it, produces no id finding either (the id is
// compared numerically), and is skipped here — while capture's own reader holds
// it to the strict grammar. That divergence between the gate's filename grammar
// and the reader's is recorded as iss-2608270908346617 and owns the fix; the
// reader half of it (such a file reaching neither Issues nor Skipped) is closed,
// so the ledger at least reports the name it cannot read. Tightening this rule's
// filename grammar to match belongs on that record, because it changes what the
// gate refuses across all four stores.
func checkRecordFilenameSlug(r schemaRecord, severity string, judged map[string]bool) []Finding {
	f := r.fields["slug"]
	// isNull, not isAbsentValue, for checkRecordFilename's reason: an empty slug
	// is a value that disagrees, and the stores that would otherwise catch it
	// declare no required set to delegate to.
	if isNull(strings.TrimSpace(f.value)) {
		return nil
	}
	_, fnSlug, ok := recordid.SplitRecordFilename(r.store.fileFamily, filepath.Base(r.rel))
	if !ok {
		return nil
	}
	got := issueScalar(f.value)
	if fnSlug == got {
		return nil
	}
	line := f.line
	if line == 0 {
		line = 1
	}
	mark(judged, "slug")
	return []Finding{{
		File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity,
		Message: "filename carries slug '" + fnSlug + "' but frontmatter declares '" + got +
			"'; a " + r.noun() + " filename is " + r.store.filename +
			", and the two spellings must be the same value",
	}}
}

// checkRecordRequiredFields asserts that a record carries every frontmatter
// property its store declares required (iss-2608261437041050). Only a store that
// declares a list is judged; the rest are left to the rules that know their own
// schemas.
//
// The defect it closes is a SILENT one, which is why it belongs in the structural
// rule rather than in the reader alone: the issue ledger's reader validates each
// record and skips the ones that fail, so a committed record missing a required
// property disappears from `capture list`, `capture status` and every other
// surface — while sitting in the ledger, counted by nothing, reported by nothing.
// A record nobody can read is not a lax record, it is a lost one.
//
// A property present but EMPTY is a finding too, on different grounds, and the
// two say so differently. An OMITTED property is refused by the reader's
// required-property check WHERE THERE IS ONE, so that message's account of the
// consequence — skipped, invisible to every surface — is made only by a store
// that declares readerFailsClosed. It holds for every field of such a store: the
// reader's own required-property loop asks nothing about which property it is.
//
// A BLANK one has no such store-wide account, and this leg must not invent one.
// The reader's required-property loop type-checks without judging content, but
// the checks either side of it DO judge: capture refuses `severity: ""` on its
// enum, `slug: ""` on its grammar, `id: ""` on its pattern and `schema_version:
// ""` on its version, and reads a found_during written as a bare single-quote
// pair, an empty flow mapping or an explicit null tag as a value, because its
// decoder leaves all three intact. Thirty-nine of the forty-two
// required-issue-field × blank-spelling combinations (six spellings: the three
// quoted ones, the indented block, the empty flow mapping and the null tag) are
// refusals and three are acceptances, so a single
// sentence about "what the reader does with a blank" is a confident false
// statement in whichever set it does not match — first claiming a refusal that
// never happens, then, once that was fixed, an acceptance that never happens
// (iss-2608301308369559). So this leg states only what a blank IS, which is true
// of all forty-two; the consequence is stated by the leg that judges the field,
// and where such a leg has already spoken (judged) this one stays silent rather
// than adding a second, weaker finding on the same line.
//
// "Empty" is judged on the value the YAML scalar carries rather than on its bytes
// — see isAbsentValue for the quoted spellings, and blockScalarIndicatorRe for the
// one shape whose value is not on the key's own line at all.
func checkRecordRequiredFields(r schemaRecord, severity string, judged map[string]bool) []Finding {
	var out []Finding
	for _, field := range r.store.requiredFields {
		f, present := r.fields[field]
		if present && !r.valueEmpty(field, f) {
			continue
		}
		if present && judged[field] {
			continue
		}
		line := f.line
		if line == 0 {
			line = 1
		}
		msg := "frontmatter is missing required property '" + field + "'; the schema declares it required " +
			"because the record has to state it, and a record that omits it never states it"
		if r.store.readerFailsClosed {
			msg = "frontmatter is missing required property '" + field + "'; the " + r.store.noun +
				" reader validates before it reads, so a record without it is skipped — invisible to every " +
				r.store.noun + " surface while it still sits in the store"
		}
		if present {
			msg = "required property '" + field + "' carries no value once its YAML scalar is read; " +
				"the schema declares it required because the record has to state it, and a blank states nothing"
		}
		out = append(out, Finding{
			File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity, Message: msg,
		})
	}
	return out
}

// checkRecordUnknownFields asserts that a record carries no frontmatter property
// outside its store's allow-list, for every store that declares one.
//
// It is the other half of checkRecordRequiredFields: a key outside the allow-list
// is a key the schema does not declare, so no surface of that family reads it and
// whatever it carries is carried nowhere. The check is store-declared rather than
// hard-coded to the issue ledger because the question is identical wherever a
// schema is closed, and asking it twice in two places is how the two answers start
// to differ.
//
// It fails the same way too WHERE THE STORE'S READER FAILS CLOSED: a closed schema
// is what such a reader validates against before it reads, so the record is skipped
// — invisible to every surface of its own family while it still sits in the store.
// That second account is gated on readerFailsClosed for the reason the
// missing-property account is: the admission reader COUNTS a record carrying an
// unknown key, and the one reader of surprise records — the record dispatcher —
// reads them leniently and skips none (iss-2608301519254418).
func checkRecordUnknownFields(r schemaRecord, severity string) []Finding {
	if r.store.knownFields == nil {
		return nil
	}
	var out []Finding
	for key, f := range r.fields {
		if r.store.knownFields[key] {
			continue
		}
		line := f.line
		if line == 0 {
			line = 1
		}
		msg := "unknown frontmatter property '" + key + "'; the " + r.store.noun +
			" schema is closed, so a key outside it is one the schema does not declare and no " +
			r.store.noun + " surface reads — whatever it states, the record states nowhere"
		if r.store.readerFailsClosed {
			msg = "unknown frontmatter property '" + key + "'; the " + r.store.noun +
				" schema is closed, so a key outside it makes this a record the reader refuses and skips — invisible to every " +
				r.store.noun + " surface while it still sits in the store"
		}
		// A retired key names what replaced it and the verb that rewrites it: the
		// record carries a spelling an older abcd wrote, and the remedy is known.
		if successor, retired := issueschema.Retired[key]; retired &&
			(r.store.prefix == "iss" || r.store.prefix == issueschema.ReadingItemFamily) {
			msg += "; it is retired, renamed to '" + successor + "'; " + issueschema.MigrateHint
		}
		out = append(out, Finding{
			File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity, Message: msg,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Message < out[j].Message })
	return out
}

// checkRecordJoins asserts that every KEYING field a store declares resolves,
// where it names a record at all.
//
// A join is the whole of what makes a record a record about something else: a
// surprise entry is a separate record rather than a field on the thing that
// occasioned it, and an admission is an answer to one proposal rather than a note
// in a drawer. In both cases the only thing tying it back is this value, and a
// join naming a record the corpus does not hold joins nothing.
//
// The check is STORE-DECLARED rather than written per family, because the
// question is identical wherever a record is keyed to another. Asking it in one
// store and not the next is how the admission's `proposal` came to be the one
// keying field nothing resolved while the surprise's `occasioned_by` was
// (iss-2608300935215868).
//
// Resolution asks four questions, because a join can fail in four ways.
//
// The first is SPELLING, and it is asked only of a join that declares a family
// (sameBucketAs): such a join's value is a handle of that family by declaration,
// so it is judged as a STRING, because what reads the family matches it as one —
// the outstanding report keys the admitted set on the value as written
// (admittedProposals), while the questions below parse it as a handle and resolve
// its ordinal as a number. Six spellings fell between the two readings and were
// green. `RDI-2`, `Rdi-2` and `rdi-02` parsed, resolved and matched their bucket,
// so the gate approved them; a space inside the quotes on either side, and bare
// prose, parsed as no handle at all and took the silence prose is owed. Every one
// of them admitted nothing (iss-2608301519255871).
//
// It is asked in two places, because the two halves need different evidence. What
// the VALUE alone settles — the family's own prefix, lower case, nothing around
// it — is asked here. Whether its PADDING is the padding that admits is a fact
// about the target's FILE, so it is asked once the target is resolved, below.
//
// The second is PRESENCE: a target that is not in the corpus joins nothing.
//
// The fourth is the POSITION, where the join declares one: what reads such a join
// consults it only for a target at that position, so a target at any other is
// never queried and the record counts for nothing — the third coordinate of the
// pair the run and spelling axes already close (iss-2608301649339636).
//
// The third is the BUCKET. A target that is in the corpus but in ANOTHER BUCKET
// joins something nobody will ever look for: what reads that family keys what it
// finds on the PAIR — the bucket the record is filed under, and the target it
// names — so a record reaching across buckets is keyed on a pair no reader
// queries. It is declared per join AND per target family (sameBucketAs), because
// that pair-keying is a property of the family and the message names it.
//
// Prose is legitimate and stays silent WHERE THE JOIN DECLARES NO FAMILY AND NO
// CLOSED SET: only a value that is a record handle of a store this scan reads is
// resolved. A surprise's occasion is NOT such a join — it declares its closed set
// (oneOf), so a prose occasion is a finding (spc-2609020626040342). A handle a record declares it PRUNED is resolved
// too, on the same terms the cross-reference loop resolves it, so one rule gives
// one answer about it.
func checkRecordJoins(r schemaRecord, index map[recordRef]schemaRecord, retired map[recordRef]bool, cfg RuleConfig) []Finding {
	var out []Finding
	for _, join := range r.store.joins {
		f := r.fields[join.field]
		// Absence is decided on the RAW value, because isAbsentValue strips the
		// value itself: handing it an already-stripped scalar strips twice, and a
		// value that is two apostrophes inside double quotes then reads as absent
		// here while checkRecordRequiredFields — which strips once — reads it as
		// present. Every leg stood down on it and an admission that admits nothing
		// drew no finding at all (iss-2608301656192369).
		if isAbsentValue(f.value) {
			continue
		}
		value := issueScalar(f.value)
		line := f.line
		if line == 0 {
			line = 1
		}
		// The spelling, where the join declares the family its value belongs to.
		// Asked before anything is resolved, because every later question reads the
		// value as a parsed handle while the reader of the family reads it as a
		// string: a value the two read differently is one the gate approves and no
		// reader can act on.
		if join.sameBucketAs != "" && !spellsHandleOf(join.sameBucketAs, value) {
			noun := joinFamilyNoun(join.sameBucketAs)
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity,
				Message: join.field + " declares '" + value + "', which is not a " + noun + " handle (want " +
					join.sameBucketAs + "-<N>, lower case with nothing around it); what reads this " +
					r.noun() + " matches the value as written against the name the " + noun +
					"'s own file carries, so a value that is not one of those names admits nothing",
			})
			continue
		}
		// The closed set, where the join declares one: the value must be verbatim a
		// handle of one of its families, so prose and a fourth family are findings
		// rather than the silence an undeclared join gives them.
		if len(join.oneOf) > 0 && !spellsHandleOfAny(join.oneOf, value) {
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity,
				Message: join.field + " declares '" + value + "', which is not a handle of " + handleList(join.oneOf) +
					" (lower case with nothing around it); " + join.why,
			})
			continue
		}
		m := anyHandleFullRe.FindStringSubmatch(value)
		if m == nil {
			continue
		}
		prefix := strings.ToLower(m[1])
		// A family this scan does not read supports no verdict either way: the
		// record might be perfectly present in a store nobody configured, and
		// reporting it missing would be a confident false statement.
		if _, known := storeByPrefix(prefix); !known || cfg.RecordStores[prefix] == "" {
			continue
		}
		num, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		ref := recordRef{prefix, num}
		target, ok := index[ref]
		if !ok {
			// A handle a record declares it PRUNED resolves to that declaration rather
			// than to a file, exactly as the cross-reference loop in checkRecordSchema
			// reads it. Without this the one rule gives two answers about one pruned
			// handle — accepting it in related_adrs and blocking on it in a join
			// (iss-2608301327012166).
			if retired[ref] {
				continue
			}
			out = append(out, Finding{
				File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity,
				Message: join.field + " names '" + ref.String() +
					"', which is not a record in the corpus; " + join.why,
			})
			continue
		}
		// The PADDING half of the spelling question, which only the resolved target
		// can answer. What reads the family keys the record on its FILENAME — the
		// outstanding report captures the item name out of `rdi-<N>.md` and never
		// opens the item's own id property — while this rule's filename ↔ id leg
		// compares those two numerically, so a zero-padded FILE is legitimate. Which
		// spelling admits is therefore decided by the file, not by the join, and it
		// is read off the file.
		//
		// A target whose filename is not itself a bare handle is left in the silence
		// it had: the reader of the family does not read such a file at all, so no
		// spelling of this join admits it and none is more right than another. That
		// divergence between this rule's filename grammar and the report's is
		// iss-2608300929274006's to close.
		stemIsHandle := false
		if join.sameBucketAs != "" {
			stem := strings.TrimSuffix(filepath.Base(target.rel), ".md")
			stemIsHandle = spellsHandleOf(join.sameBucketAs, stem)
			if stemIsHandle && value != stem {
				out = append(out, Finding{
					File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity,
					Message: join.field + " declares '" + value + "' while the " + target.noun() +
						" it names is filed as '" + filepath.Base(target.rel) + "'; what reads this " + r.noun() +
						" matches the value as written against the name that file carries, so this spelling admits " +
						"nothing and the " + target.noun() + " it names goes on being reported as unanswered",
				})
				continue
			}
		}
		// The POSITION obligation, where the join declares one. It is the third
		// coordinate of the pair, beside the run the record is filed under and the
		// spelling of the value: what reads this join consults it only for a target at
		// the declared position, so a target at any other is never queried and the
		// record counts for nothing. It is asked only where the target's filename is a
		// bare handle, for the padding leg's reason one block above — what reads the
		// family never opens such a file, so its position decides nothing.
		if join.targetPosition != "" && stemIsHandle {
			posField := target.fields["position"]
			if pos := issueScalar(posField.value); pos != join.targetPosition {
				declares := "declares position '" + pos + "'"
				if isAbsentValue(posField.value) {
					declares = "declares no position"
				}
				out = append(out, Finding{
					File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity,
					Message: join.field + " names '" + ref.String() + "', which " + declares +
						"; what reads this " + r.noun() + " consults it only for a " + target.noun() +
						" at position '" + join.targetPosition + "', so this " + r.noun() +
						" is keyed on a pair nothing ever queries: it counts for nothing, and no line reports " +
						"that an answer was written for the " + target.noun() + " it names",
				})
				continue
			}
		}
		// The bucket obligation, where the join declares one. The target is of the
		// declared family by construction: the spelling leg above refused every value
		// that is not a handle of it, and the index is keyed on the parsed handle.
		// Neither bucket can be empty here either: both sides are then records of a
		// bucketed store, and a record sitting outside every bucket is reported by
		// the walk and never enters the index at all — so it reaches this leg as a
		// target that is not in the corpus, above.
		if join.sameBucketAs == "" || target.bucket == r.bucket {
			continue
		}
		msg := join.field + " names '" + ref.String() + "', which is filed under '" + target.bucket +
			"' while this " + r.noun() + " is filed under '" + r.bucket +
			"'; what reads that family keys it on the pair — the bucket it is filed under and the " +
			target.noun() + " it names — so this " + r.noun() +
			" is keyed on a pair nothing ever queries and counts for nothing"
		// The tail names a REPORT LINE, so it is appended only where the target's
		// filename is a bare handle — the same test the padding leg makes one block
		// above, and for the same reason: what reads the family never opens a file
		// whose name is not one, and emits nothing at all about that target. The
		// leading clause is true of every cross-bucket target, so the finding stands
		// either way; sending the operator to find a line that does not exist is what
		// does not (iss-2608301656193936).
		if stemIsHandle {
			msg += ", and the " + target.noun() +
				" it names goes on being reported as unanswered with no sign that an answer was written"
		}
		out = append(out, Finding{
			File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: cfg.Severity, Message: msg,
		})
	}
	return out
}

// spellsHandleOf reports whether value is VERBATIM one family's handle: the
// family's own prefix, one hyphen, and digits, with nothing around it. It is
// deliberately narrower than anyHandleFullRe, which is case-insensitive because
// it exists to RESOLVE a handle to a record. This asks the other question — is
// this a name the family's own reader could match — and the two answers differed
// on six values (iss-2608301519255871).
//
// It says nothing about PADDING, which is not a property of the value: `rdi-02`
// and `rdi-2` are one id written two ways, and which of them the reader matches
// is decided by the file the target sits in. That comparison needs the resolved
// target and is made against its filename.
func spellsHandleOf(family, value string) bool {
	rest, ok := strings.CutPrefix(value, family+"-")
	if !ok || rest == "" {
		return false
	}
	for i := 0; i < len(rest); i++ {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	return true
}

// spellsHandleOfAny reports whether value is verbatim a handle of one of
// families.
func spellsHandleOfAny(families []string, value string) bool {
	for _, f := range families {
		if spellsHandleOf(f, value) {
			return true
		}
	}
	return false
}

// handleList renders a family set as the handles a message names.
func handleList(families []string) string {
	names := make([]string, 0, len(families))
	for _, f := range families {
		names = append(names, f+"-<N>")
	}
	return strings.Join(names, ", ")
}

// joinFamilyNoun renders the record kind a join's declared family holds, for the
// message that names it. Every declared family is one of recordStores' own
// prefixes, which TestEveryJoinFamilyNamesADeclaredStore pins.
func joinFamilyNoun(family string) string {
	s, _ := storeByPrefix(family)
	return s.noun
}

// checkRecordBucketField asserts that a record whose store states its bucket
// TWICE says the same thing both times.
//
// An admission is filed under the run whose candidate set it joins and carries
// that run as a frontmatter property, so a disagreement is the record
// contradicting itself about which set it joined. The outstanding report keys the
// admitted set on the (run, proposal) pair and can honour neither claim, so the
// admission silently admits nothing — and a record that quietly stops counting is
// the shape this whole rule exists to make loud.
//
// It is scoped to the stores that DECLARE a bucketField, and that scope is the
// whole of it: bucketField is declared by the bucketed admission store alone, so
// a record reaching this check always has a bucket. A second test for an empty
// one would be a branch no fixture can enter, which is how a guard comes to look
// tested (iss-2608301519254240).
//
// The disposition store
// makes the same double claim (`item` beside its item-keyed directory) and is not
// declared here, because that store declares no frontmatter schema at all this
// cycle; adding one field of it would be a schema half-stated in a second place.
func checkRecordBucketField(r schemaRecord, severity string) []Finding {
	if r.store.bucketField == "" {
		return nil
	}
	f, present := r.fields[r.store.bucketField]
	if !present {
		return nil // absence is checkRecordRequiredFields' business, not this one's
	}
	// Absence on the RAW value, one strip, for checkRecordJoins' reason: stripping
	// before the absence test strips twice and parts this leg from the required
	// leg beside it (iss-2608301656192369).
	if isAbsentValue(f.value) {
		return nil
	}
	got := issueScalar(f.value)
	if got == r.bucket {
		return nil
	}
	line := f.line
	if line == 0 {
		line = 1
	}
	return []Finding{{
		File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity,
		Message: r.store.bucketField + " declares '" + got + "' but the " + r.store.noun + " is filed under '" + r.bucket +
			"'; the directory and the field are one value written twice, and a record that contradicts " +
			"itself about which one it belongs to counts under neither",
	}}
}

// checkIssueRecordShape mirrors capture's validateStrict shape checks for the
// ISSUE store: enum membership (severity/category/source) and the kebab-slug
// check. The third of them, the additionalProperties:false unknown-key check, is
// checkRecordUnknownFields, which asks the same question for every store that
// declares a closed schema rather than for this one alone. Each reads the ONE shared
// schema data in core/issueschema — the same allow-list and value sets capture
// validates against — so a record capture would REFUSE (and therefore skip,
// making it invisible to every capture surface) is not lint-green
// (iss-2608261447039180, iss-2608270908342889).
//
// It is scoped to the issue store: the other three stores carry different
// schemas, judged by their own rules. It deliberately does NOT yet mirror
// capture's folder<->field invariants (resolution in resolved/, wontfix_reason in
// wontfix/) or its optional-field type checks — the enum, slug and unknown-key
// checks are the determinate, highest-value half; the folder invariants are a
// follow-up.
//
// An ABSENT required value is the required-fields check's business, not this
// one's, so each value check skips a missing/null value rather than double-report.
func checkIssueRecordShape(r schemaRecord, severity string, judged map[string]bool) []Finding {
	if r.store.prefix != "iss" {
		return nil
	}
	var out []Finding
	add := func(field string, line int, msg string) {
		if line == 0 {
			line = 1
		}
		mark(judged, field)
		out = append(out, Finding{
			File: r.rel, Line: line, RuleID: ruleRecordSchema, Severity: severity, Message: msg,
		})
	}

	// Unknown property: checkRecordUnknownFields asks that question for every
	// store that declares a closed schema, this one included.

	// Enum membership: a present but out-of-enum value is refused by capture.
	enums := []struct {
		field string
		set   []string
	}{
		{"severity", issueschema.Severities},
		{"category", issueschema.Categories},
		{"source", issueschema.Sources},
	}
	//
	// A BLANK is judged here rather than deferred, and that is why the skip is
	// isNull on the trimmed raw value and not isAbsentValue: `severity: ""` is a
	// value capture puts to the enum and refuses, so it belongs to the leg that can
	// say so. Standing down on it left the required-fields leg as the only voice,
	// and that leg can only speak store-wide — so it told the author of a blank
	// severity that their record was being read and rendered as answered, about a
	// record capture refuses and skips (iss-2608301308369559). isNull draws the
	// line at NOT WRITTEN, the same distinction checkRecordFilename already draws.
	for _, e := range enums {
		f, present := r.fields[e.field]
		if !present || isNull(strings.TrimSpace(f.value)) {
			continue
		}
		v := issueScalar(f.value)
		if !inSet(v, e.set) {
			add(e.field, f.line, "invalid "+e.field+" '"+v+"'; capture refuses a value outside {"+strings.Join(e.set, ", ")+"} and skips the record")
		}
	}

	// Kebab-slug: the slug becomes a filename, and capture refuses any other shape
	// — a blank one included, for the reason the enums above are judged blank.
	if f, present := r.fields["slug"]; present && !isNull(strings.TrimSpace(f.value)) {
		v := issueScalar(f.value)
		if !issueschema.SlugRe.MatchString(v) {
			add("slug", f.line, "invalid slug '"+v+"'; a slug is kebab-case (lower-case alphanumerics joined by single hyphens) and capture refuses any other shape")
		}
	}

	// grounds: the record BODY is where they live, as the append-only
	// `## Grounds` section core/grounds holds for both record families. A
	// frontmatter `grounds:` is therefore a MISPLACED value, and the gate blocks
	// it and names the section.
	//
	// The key is still in the reader's allow-list, so a record carrying it reads
	// perfectly well and stays visible to every capture surface — it is the VALUE
	// that goes unread, which is the whole reason this has to be a gate rather
	// than a reader refusal. A refusal would make the reader skip the record, and
	// a record nothing can see is a worse outcome than a value nothing reads.
	//
	// The value's own spelling is deliberately not judged. Nothing parses it any
	// more, so a well-formed one and a malformed one are equally unread, and a
	// second finding about the grammar of a value that has no consumer would send
	// the operator to fix the wrong thing. The remedy is the same either way:
	// move it into the section.
	if groundsField, present := r.fields["grounds"]; present {
		add("grounds", groundsField.line, "grounds is in frontmatter; a frontmatter scalar is SET, so a later triage route overwrites the conjecture an earlier one recorded — grounds are appended as `- <token>: <text>` bullets under a `## Grounds` heading in the record body, and a frontmatter value is read by nothing")
	}

	// lapsed_at: optional on every category, lapse included, and an RFC 3339
	// instant whenever it is present. The absence finding spc-60 raised on a lapse
	// record is parked (iss-2609091009111294) until the rethink of the reading
	// work; the format half stays, and it reads the ONE shared definition in
	// core/issueschema, the same one capture's validateStrict reads, so this gate
	// refuses exactly the record the reader refuses (and therefore skips, making it
	// invisible to every capture surface while it still sits in the ledger).
	// Absence is tested with the frontmatter NULL set alone, not isAbsentValue,
	// which also reads an empty flow collection and an explicit null tag as absent.
	// lapsed_at is a scalar
	// property: capture's reader parses `lapsed_at: []` as a list, refuses the
	// record ("lapsed_at" must be a string) and SKIPS it, so reading the list as
	// absence would leave that record lint-green on every category but lapse —
	// invisible to every capture surface, gate silent (iss-2608300224316569). A
	// list-shaped value falls through as a present value that is no instant, which
	// is exactly what it is.
	//
	// The value is trimmed AFTER issueScalar strips the quotes, because a quoted
	// all-whitespace value (`lapsed_at: "   "`) still carries its padding once the
	// quotes are gone. capture's reader trims before judging, so it reads that as
	// ABSENT: a clean record with an optional property left unset, or — on a lapse
	// — the missing-instant refusal. Reading it here as a present malformed value
	// would report a reader refusal that does not happen, on a record the reader
	// accepts (iss-2608300212513349).
	lapseField, hasLapseField := r.fields["lapsed_at"]
	lapsedAt := ""
	if hasLapseField && !isNull(strings.TrimSpace(lapseField.value)) {
		lapsedAt = strings.TrimSpace(issueScalar(lapseField.value))
	}
	// A key whose own line carries no value may still carry one, on the indented
	// lines below it. The shared scanner is a same-line scanner, so it reports that
	// as empty; capture's reader builds the mapping and refuses the record because
	// lapsed_at must be a string, then skips it. Reading the block-spelled value as
	// ABSENT here would leave that record lint-green on any category but lapse —
	// the sibling of the list case, and the same silent invisibility
	// (iss-2608300234599781). What the block SAYS is not parsed: it is present, and
	// it is no instant, which is the whole of the finding.
	fromBlock := false
	if hasLapseField && lapsedAt == "" && strings.TrimSpace(lapseField.value) == "" {
		if block := r.blocks["lapsed_at"]; block != "" {
			lapsedAt, fromBlock = block, true
		}
	}
	switch {
	case fromBlock:
		// A block-spelled value is refused whatever it spells, so its CONTENT is
		// never put to the format check. capture splits an indented continuation on
		// its first colon and builds a mapping — `lapsed_at:` over an indented
		// 2026-08-28T00:00:00Z becomes map["2026-08-28T00"]="00:00Z" — and then
		// refuses the record because lapsed_at must be a string; an indented line
		// that is no key at all is refused earlier still, at the parse. Handing the
		// joined block to ValidLapsedAt would pass exactly the spelling that reads
		// like an instant and is not one, which is the gap the look-ahead exists to
		// close (iss-2608300244489638).
		add("lapsed_at", lapseField.line, "lapsed_at is spelled as an indented block; capture reads a block-spelled value as a mapping rather than a string, refuses the record and skips it — a lapse time is an RFC 3339 instant on the key's own line")
	case lapsedAt != "" && !issueschema.ValidLapsedAt(lapsedAt):
		add("lapsed_at", lapseField.line, "lapsed_at '"+lapsedAt+"' is not an RFC 3339 instant (want 2026-08-28T00:00:00Z); capture refuses the record and skips it")
	}
	return out
}

// issueScalar reads one frontmatter value as the record's readers read it: the
// surrounding whitespace and quotes stripped, so a quoted enum
// (`severity: "minor"`) compares unquoted.
//
// It is the package's ONE scalar reader. On WHITESPACE it matches capture's parse
// path deliberately: that path trims the raw value before decoding and then strips
// the quotes WITHOUT trimming again, so padding inside the quotes survives. This
// does the same. Trimming it away here would call `severity: "  minor  "` clean while
// capture's enum lookup refuses the record and skips it — lint-green and
// invisible to every ledger surface, which is the one silence this rule exists to
// break. Whether padding is a value is the FIELD's question, and the two fields
// that answer "no" — emptiness in isAbsentValue, and the lapse instant — trim at
// their own call site rather than moving the trim in here for everyone.
//
// Three spellings of this function used to exist — here, in isAbsentValue, and in
// the outstanding report — and they did not agree: the gate called
// `run: " rdg-1 "` a contradiction while the report called it a match, so one
// record got two answers and the more permissive one was the report's.
//
// Exactly ONE surrounding quote pair is stripped, never every quote character at
// either end: a value that is itself two apostrophes inside double quotes is a
// value, and eating it down to nothing puts a missing-property blocker on a
// property the record plainly carries.
//
// On QUOTING the parity is incomplete, and that is a KNOWN GAP rather than a
// claim: this strips a single-quote pair, capture's decodeScalar unquotes only
// double quotes, so `severity: 'minor'` is green here and refused there. The
// divergence is pre-existing and cuts across every shape check that reads a
// scalar, which is why it is recorded as iss-2608300205044566 — naming this
// function as the one place to fix it — rather than closed from inside a change
// about required fields.
func issueScalar(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		v = v[1 : len(v)-1]
	}
	return v
}

// readerScalar decodes a frontmatter scalar EXACTLY as capture's decodeScalar
// does — the same shape test here, then the SAME decoder, frontmatter.Unquote,
// which both call: a matched pair of DOUBLE quotes is removed and its backslash
// escaping reversed; anything else — a single-quoted value included — is the bare token it
// spells, quote characters and all.
//
// It exists beside issueScalar rather than replacing it because the two answer
// different questions. issueScalar compares an enum leniently, where a
// single-quoted `severity: 'minor'` is a spelling nobody needs a finding about.
// A free-text value is different: the quote character survives into the value the
// reader parses, so a gate that strips it judges a string that never existed
// (iss-2608300927577163).
func readerScalar(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		return frontmatter.Unquote(v[1 : len(v)-1])
	}
	return v
}

// inSet reports membership in a small value list.
func inSet(v string, set []string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}

// noun renders the record kind for a message.
func (r schemaRecord) noun() string { return r.store.noun }

// scanRecordStores reads every configured store once, returning its records plus
// the findings the walk itself produces — an undeclared lifecycle directory, a
// record sitting outside every bucket, and a markdown file whose name is not the
// store's id pattern. Those three are the "no lifecycle state escapes" half of the
// rule: they are about where a file IS, so they are found by the walk, not by a
// later pass over frontmatter.
func scanRecordStores(repoRoot string, cfg RuleConfig) ([]schemaRecord, []Finding, error) {
	var records []schemaRecord
	var out []Finding
	add := func(rel string, msg string) {
		out = append(out, Finding{
			File: rel, Line: 0, RuleID: ruleRecordSchema, Severity: cfg.Severity, Message: msg,
		})
	}

	for _, store := range recordStores {
		dir := cfg.RecordStores[store.prefix]
		if dir == "" {
			continue
		}
		storeAbs := filepath.Join(repoRoot, filepath.FromSlash(dir))
		entries, err := os.ReadDir(storeAbs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}

		// A directory that is itself a CONFIGURED store root is not an undeclared
		// bucket of its parent: the reading families live inside the issue store's
		// own root, and without this the day they appear is the day the gate calls
		// each of them an undeclared issue bucket — a blocker over a directory the
		// configuration itself declares. The set is derived from cfg.RecordStores,
		// so the config stays the single declaration of what a store is.
		nestedRoots := nestedStoreRoots(cfg.RecordStores, dir)

		// A flat store holds its records directly; a bucketed store holds only its
		// declared lifecycle directories (plus the store README).
		readBucket := func(bucketAbs, bucketRel, bucket string) error {
			es, err := os.ReadDir(bucketAbs)
			if err != nil {
				return err
			}
			for _, e := range es {
				rel := filepath.Join(bucketRel, e.Name())
				if e.IsDir() {
					// A bucket (and a flat store) holds its records DIRECTLY. A
					// directory inside one is a lifecycle nobody declared, and every
					// check in this rule stops at it — which is the same escape the
					// undeclared-bucket check closes one level up, so it is closed
					// here too rather than only for the store roots. A dot-directory
					// is tooling state, not a lifecycle the record authored.
					if !strings.HasPrefix(e.Name(), ".") {
						add(rel, undeclaredSubdirMessage(store, bucket, e.Name()))
					}
					continue
				}
				if !hasMarkdownExt(e.Name()) || strings.EqualFold(e.Name(), "README.md") {
					continue
				}
				m := store.fileNumRe.FindStringSubmatch(e.Name())
				if m == nil {
					add(rel, "filename is not a well-formed "+store.noun+" filename ("+store.filename+
						"); the filename is the handle every cross-reference resolves through")
					continue
				}
				num, err := strconv.Atoi(m[1])
				if err != nil {
					continue
				}
				content, err := os.ReadFile(filepath.Join(bucketAbs, e.Name()))
				if err != nil {
					return err
				}
				lines := strings.Split(string(content), "\n")
				// A duplicated top-level key is malformed to every record consumer, but the
				// consumers do not agree on WHAT it makes of the record, so the message may
				// not name one answer for all of them: this rule's own scanner keeps the
				// first value — so a second line can hide the value a blocker is armed to
				// reject — while a store whose reader fails closed has the file refused
				// outright, and the disposition reader keeps NEITHER value and calls the
				// record illegible (TestDuplicateTopLevelKeyReaderByReader establishes what
				// each reader does, by running it). The lenient read below is what the rest
				// of this rule needs; the duplicate is reported alongside it so the gate
				// refuses what its consumers refuse (GitHub #357).
				//
				// The refusal half is gated on readerRefusesDuplicateKey rather than on
				// readerFailsClosed, because the two come apart on this malformation
				// alone: the admission reader COUNTS a record carrying a duplicated key,
				// the record dispatcher reads surprise records with the lenient scanner
				// on its first value, and the ADR dispatcher — which
				// does validate the id — reads the frontmatter with the lenient scanner
				// and never sees the second line, so naming a refusal on any of the three
				// sends the author looking for one nobody performs (iss-2608301519254418,
				// iss-2608301656200729). The silenced-blocker half is true of every store,
				// because it is this rule's own scanner that does the silencing.
				//
				// The refusal names the LEDGER reader, not every surface. It once said the
				// file was "skipped by every issue surface", and the release cut reads a
				// resolved record with the lenient scanner and folds its impact into the
				// changelog — so an author told the record was invisible everywhere found it
				// in the generated section (iss-2608301901260678). Both halves are rows in
				// TestDuplicateTopLevelKeyReaderByReader.
				//
				// The RESOLVED qualifier is load-bearing and the correction ran the same way
				// round: the cut walks .abcd/work/issues/resolved and the shipped intents,
				// while this leg fires on every bucket the store declares, so a bare "the
				// release cut reads this record" is false of an open or wontfix issue and
				// would send its author to look in a generated section that cannot hold it —
				// the same defect, inverted, as the clause it replaced.
				for _, dup := range frontmatterDuplicates(lines) {
					msg := "frontmatter has a duplicate top-level key '" + dup.Key +
						"'; this rule's own scanner keeps only the first value, " +
						"so a second line can silence a blocker armed on the value the first hides"
					if store.readerRefusesDuplicateKey {
						msg = "frontmatter has a duplicate top-level key '" + dup.Key +
							"'; the " + store.noun + " ledger reader refuses a duplicated key, so capture skips the file — " +
							"but the release cut still reads a resolved record, on its first value, as this rule's own " +
							"scanner reads it here: a second line can silence a blocker armed on the value the first hides"
					}
					out = append(out, Finding{
						File: rel, Line: dup.Line, RuleID: ruleRecordSchema, Severity: cfg.Severity, Message: msg,
					})
				}
				fields := frontmatterFields(lines)
				records = append(records, schemaRecord{
					rel:    rel,
					store:  store,
					num:    num,
					bucket: bucket,
					title:  recordTitle(lines),
					fields: fields,
					refs:   recordRefsOf(lines, fields),
					blocks: frontmatterBlocksOf(lines, fields),
				})
			}
			return nil
		}

		storeRel := repoRel(repoRoot, storeAbs)
		if !store.bucketed() {
			// A flat store declares NO buckets, so it holds its records directly and
			// readBucket reports any subdirectory of it, exactly as it does for a
			// declared bucket.
			if err := readBucket(storeAbs, storeRel, ""); err != nil {
				return nil, nil, err
			}
			continue
		}

		for _, e := range entries {
			rel := filepath.Join(storeRel, e.Name())
			if e.IsDir() {
				// A dot-directory is tooling state (an editor's, a scanner's), never
				// a lifecycle the record authored — the record's own buckets are all
				// named, so refusing one would be a blocker over somebody's .vscode.
				if strings.HasPrefix(e.Name(), ".") {
					continue
				}
				// The exemption says "something else scans this directory". A bucket
				// the parent ALREADY declares is scanned by the parent, so there is
				// nothing to exempt and granting it can only remove coverage — which
				// is exactly what a config line aiming a real store prefix at
				// another store's bucket did: the parent skipped the bucket, the
				// misdirected store ignored every file not matching its own filename
				// grammar, and a record missing five required properties produced no
				// finding at all.
				if nestedRoots[e.Name()] && !store.declaresBucket(e.Name()) {
					continue
				}
				if !store.declaresBucket(e.Name()) {
					add(rel, "lifecycle directory '"+e.Name()+"' is not a declared "+store.noun+
						" bucket ("+store.bucketDesc()+"); an undeclared bucket is a lifecycle state no rule reads")
					continue
				}
				if err := readBucket(filepath.Join(storeAbs, e.Name()), rel, e.Name()); err != nil {
					return nil, nil, err
				}
				continue
			}
			if !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
				continue
			}
			if store.fileNumRe.MatchString(e.Name()) {
				add(rel, "record sits in the store root rather than a lifecycle bucket ("+
					store.bucketDesc()+"); the directory IS the lifecycle state")
			}
		}
	}

	sort.SliceStable(records, func(i, j int) bool { return records[i].rel < records[j].rel })
	return records, out, nil
}

// nestedStoreRoots names the immediate children of dir that are themselves
// configured record-store roots. Both arguments are repo-relative, slash-joined
// store paths as the configuration spells them.
//
// It walks the CODE's store list and looks each prefix up in the configuration,
// never the configuration's own values, and the difference is the whole point.
// The exemption says "this directory is not an undeclared bucket because
// something else scans it" — so it may only be granted to a directory the scanner
// actually visits. Deriving it from every value in the map would let a committed
// line naming no store at all (a prefix this code has never heard of, pointed
// inside a real store) exempt a directory that nothing scans: a lifecycle state
// no rule reads, which is precisely the escape this rule exists to close. Which
// lifecycle states exist is code, not config, and a config that could add a
// bucket could also hide one.
func nestedStoreRoots(stores map[string]string, dir string) map[string]bool {
	parent := strings.Trim(filepath.ToSlash(dir), "/")
	out := map[string]bool{}
	for _, store := range recordStores {
		other := stores[store.prefix]
		if other == "" {
			continue
		}
		child := strings.Trim(filepath.ToSlash(other), "/")
		if child == parent || !strings.HasPrefix(child, parent+"/") {
			continue
		}
		// Only the IMMEDIATE child is a bucket-shaped name from this store's
		// point of view; a deeper store root is a child of something else.
		if rest := child[len(parent)+1:]; !strings.Contains(rest, "/") {
			out[rest] = true
		}
	}
	return out
}

// undeclaredSubdirMessage names a directory that sits where records should. A
// flat store and a declared bucket both hold records directly, so the defect is
// the same and only the wording differs.
func undeclaredSubdirMessage(store recordStore, bucket, name string) string {
	if bucket == "" {
		return "the " + store.noun + " store is flat, so subdirectory '" + name +
			"' is undeclared; records inside it are read by no rule"
	}
	return "lifecycle bucket '" + bucket + "' holds records directly, so subdirectory '" + name +
		"' is undeclared; records inside it are read by no rule"
}

// isAbsentValue reports whether a frontmatter value says "nothing here".
//
// It is frontmatter.IsEmptyValue, and holds no rule of its own. That is the
// whole point of the indirection: emptiness used to be decided here by comparing
// against a list of literals, so `!!null` was an absence and `!!null null` —
// the SAME YAML node — was a value, and each round closed one spelling and left
// the next one open. Deciding on the node's class rather than on the bytes that
// spell it closes the spellings nobody enumerated at the same time as the ones
// that were, which is adr-56's ruling for the exclusion floor applied one
// workstream over (iss-2608301808198621).
//
// The shared reader is where it belongs rather than beside it, because a gate
// exists to refuse exactly what a reader refuses, and two emptiness rules that
// drift are the split-verdict shape: a record the reader skips going lint-green.
// Every field and every store now asks one question.
//
// isNull stays separate and is still asked directly by the legs that judge
// SCALAR enum fields (severity, category, source, slug): there a collection
// literal is a wrong value rather than an unset one, and should keep saying so.
//
// Two things it does not decide, both recorded rather than hidden. A block
// scalar holding nothing is judged by blockScalarIndicatorRe, because a block
// scalar's content is not on the key's own line at all. And the supersession
// leg's silence on an empty collection is not shared by record.describeADR,
// which renders `[]` and `{}` as a successor link (iss-2608301744300631).
//
// It takes the RAW frontmatter value, never one issueScalar has already read:
// stripping twice empties a value that is two apostrophes inside double quotes,
// so the legs that pre-stripped read it as absent while the leg that did not read
// it as present — and every leg stood down on one admission
// (iss-2608301656192369). The shared reader strips ONE level for the same reason.
//
// A trailing comment can no longer hide any of it. The strip lives in the ONE
// same-line scanner (frontmatter.Fields), so the value reaching here is the
// value and nothing else (iss-2608301744268001).
func isAbsentValue(value string) bool {
	return frontmatter.IsEmptyValue(value)
}

// recordRefsOf reads the handles of every cross-reference field once per record.
//
// YAML spells a list two ways and the record uses both: the inline flow sequence
// (`supersedes: [adr-14, adr-15]`) and the indented block sequence. The shared
// frontmatter scanner reads same-line values only — correct for its own job, and
// the reason a block sequence reaches this rule as an empty string. An empty read
// is not a harmless degradation HERE: the bidirectional check would report that
// some OTHER file omits a link that file plainly carries, which is a confident
// false statement about a record the reader then has to go and disprove. So the
// block form is folded in at the one place the handles are parsed.
func recordRefsOf(lines []string, fields map[string]fmField) map[string][]recordRef {
	refs := make(map[string][]recordRef, len(recordParsedFields))
	for _, field := range recordParsedFields {
		f, ok := fields[field]
		if !ok {
			continue
		}
		value := f.value
		if strings.TrimSpace(value) == "" {
			value = blockSequenceAt(lines, f.line)
		}
		if hs := recordRefsIn(value); len(hs) > 0 {
			refs[field] = hs
		}
	}
	return refs
}

// blockSequenceAt returns the `- item` lines that continue a frontmatter key whose
// own line carries no value, joined so the handle scanner reads them as one value.
// line is the key's 1-based source line, so the scan starts at the line after it
// and stops at the first line that is not a list item — the next key, the closing
// delimiter, or the end of the file. Indentation is not required: YAML nests a
// block sequence under a mapping key at column 0 too, which the record uses.
func blockSequenceAt(lines []string, line int) string {
	var items []string
	for _, trimmed := range frontmatterBlockAt(lines, line) {
		// Only a line that is not a list item at all ends the sequence. The walker
		// also yields an indented non-item continuation (a nested MAPPING), which is
		// not a sequence item and stops the read here exactly as an unindented line
		// would have.
		if !strings.HasPrefix(trimmed, "- ") {
			break
		}
		items = append(items, strings.TrimPrefix(trimmed, "- "))
	}
	return strings.Join(items, ", ")
}

// frontmatterBlockAt is the ONE walker over the lines that continue a
// frontmatter key whose own line carries no value, returning them trimmed and in
// order. The shared frontmatter scanner reads same-line values only — correct for
// its own job — so every reader that must see a block-spelled value walks it
// here rather than growing a scanner of its own.
//
// line is the key's 1-based source line, so the scan starts at the line after it.
// It yields two shapes, because YAML spells a blank-valued key's value two ways
// and the record uses both: a block SEQUENCE item (`- x`, at any indentation —
// `key:\n- item` is the same list as `key:\n  - item`) and an INDENTED
// continuation, which is how a nested mapping is written. The scan stops at the
// first line that is neither: the next key at column 0, the closing delimiter, or
// the end of the file.
//
// A blank line or a comment INTERRUPTS a block without ending it: YAML reads
// `- a`, blank, `# why`, `- b` as one two-item list. Stopping at the interruption
// would drop the tail — and a dropped item is not a quiet under-read for the
// handle reader, it makes the bidirectional check assert that ANOTHER file omits
// a link that file carries, the exact false claim that parse exists to prevent.
// The closing `---` is neither blank nor a comment, so the scan still ends at the
// frontmatter boundary.
func frontmatterBlockAt(lines []string, line int) []string {
	var block []string
	for i := line; i >= 1 && i < len(lines); i++ {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")
		if !strings.HasPrefix(trimmed, "- ") && !indented {
			break
		}
		block = append(block, trimmed)
	}
	return block
}

// frontmatterBlocksOf reads, once per record, the block continuation of every key
// whose same-line value is empty — the shape the same-line scanner reports as no
// value at all. It is held on the record for the same reason refs is: a rule that
// re-read the lines would be a second parse of one spelling, and a rule that
// skipped the look-ahead would read a value that is plainly in the file as absent.
//
// A key carrying a block-scalar HEADER (`grounds: |`) is read on the same terms,
// because its value is on those lines too — the header is not the value, it is
// the announcement of one, and a rule that took the byte for a value would let an
// empty block pass as present.
func frontmatterBlocksOf(lines []string, fields map[string]fmField) map[string]string {
	var blocks map[string]string
	put := func(key, value string) {
		if value == "" {
			return
		}
		if blocks == nil {
			blocks = map[string]string{}
		}
		blocks[key] = value
	}
	for key, f := range fields {
		v := strings.TrimSpace(f.value)
		switch {
		case blockScalarIndicatorRe.MatchString(v):
			put(key, blockScalarAt(lines, f.line))
		case v == "":
			put(key, strings.Join(frontmatterBlockAt(lines, f.line), ", "))
		}
	}
	return blocks
}

// blockScalarAt returns the lines that continue a key carrying a block-scalar
// HEADER (`grounds: |`), joined, or the empty string when the block holds
// nothing.
//
// It is a second walker rather than a reuse of frontmatterBlockAt, and the
// difference is one line of it: that walker skips blank and `#`-led lines,
// because in a block SEQUENCE a `#` line is a comment interrupting the list.
// Inside a literal block scalar a `#` is CONTENT, so reading a scalar with the
// sequence walker makes a legal record's whole value vanish and reports a
// required property missing that the file plainly carries — a confident false
// statement from the rule whose design argument is that it never makes one.
//
// line is the key's 1-based source line, so the scan starts at the line after it
// and ends at the first unindented non-blank line: the next key at column 0, the
// closing delimiter, or the end of the file.
func blockScalarAt(lines []string, line int) string {
	var block []string
	for i := line; i >= 1 && i < len(lines); i++ {
		l := strings.TrimRight(lines[i], " \t\r")
		if strings.TrimSpace(l) == "" {
			// A blank line is inside the block, not the end of it — YAML keeps it as
			// content. It holds nothing on its own, so it never makes an otherwise
			// empty block look full.
			block = append(block, "")
			continue
		}
		if l[0] != ' ' && l[0] != '\t' {
			break
		}
		block = append(block, l)
	}
	joined := strings.Join(block, "\n")
	if strings.TrimSpace(joined) == "" {
		return ""
	}
	return joined
}

// recordRefsIn reads every record handle out of a frontmatter value, tolerating
// the inline-list, quoted, and bare spellings the record uses interchangeably
// (`[adr-14, adr-15]`, `["adr-8", "adr-27"]`, `adr-4`).
func recordRefsIn(value string) []recordRef {
	if isNull(value) {
		return nil
	}
	var out []recordRef
	for _, m := range recordHandleRe.FindAllStringSubmatch(value, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		out = append(out, recordRef{prefix: strings.ToLower(m[1]), num: n})
	}
	return out
}

// recordTitle reads a record's human title out of its lines: the first H1, or —
// for a store whose records carry none, the issue ledger — the first non-blank
// body line, whitespace-collapsed, which is the same one-line summary the
// ledger's own promotion path derives. The frontmatter block (and a leading
// attribution comment before it) is skipped first, so a frontmatter value can
// never be mistaken for a body line. An empty record yields "", and the caller
// substitutes the handle.
func recordTitle(lines []string) string {
	i := recordBodyStart(lines)
	first := ""
	for ; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# ") {
			return strings.TrimSpace(strings.TrimPrefix(lines[i], "# "))
		}
		if first == "" {
			if f := strings.Fields(lines[i]); len(f) > 0 {
				first = strings.Join(f, " ")
			}
		}
	}
	return first
}

// recordBodyStart returns the index of the first BODY line: past a leading
// attribution comment and blank lines, and past the frontmatter block if the
// document opens one. It is the one place that skip is expressed, so recordTitle
// and recordH1 can never disagree about where a document's prose begins.
func recordBodyStart(lines []string) int {
	i := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "<!--") {
			i++
			continue
		}
		break
	}
	if i < len(lines) && strings.TrimSpace(lines[i]) == "---" {
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			i++
		}
		if i < len(lines) {
			i++
		}
	}
	return i
}

// recordH1 returns a document's first H1 and its 1-based line, or ("", 0) when it
// has none.
//
// recordTitle's first-body-line fallback is deliberately NOT applied here. That
// fallback exists for the issue ledger, whose records carry no heading; applied
// to arbitrary markdown it would read an ordinary opening sentence as a title,
// and a sentence that happens to open with a record handle is a mention, not a
// claim on the id.
func recordH1(lines []string) (string, int) {
	for i := recordBodyStart(lines); i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# ") {
			return strings.TrimSpace(strings.TrimPrefix(lines[i], "# ")), i + 1
		}
	}
	return "", 0
}

// refsContain reports whether a parsed handle set names the given handle.
func refsContain(refs []recordRef, want recordRef) bool {
	for _, h := range refs {
		if h == want {
			return true
		}
	}
	return false
}
