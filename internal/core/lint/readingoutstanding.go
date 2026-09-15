package lint

// The outstanding-readings report (reading_outstanding): every reading item that
// carries no disposition, and every open hold with its exit condition.
//
// It is a REPORT, not a gate, and the difference is enforced in code rather than
// left to configuration. Its severity is pinned to `info` here and never read
// from RuleConfig, because a rule whose severity a config could raise to blocker
// is a gate waiting to happen — and a reading must never block a push that has
// nothing to do with it. What it protects against is quieter than a gate and
// harder to notice: nothing in this design means "already covered", so an
// unanswered item has no state to sit in, and without this report it would
// simply not appear anywhere.
//
// The status signal it reads is the presence of the KEYED disposition directory,
// never folder membership — one probe, and the reason the reading families are
// deliberately absent from issueschema.StatusDirs.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const ruleReadingOutstanding = "reading_outstanding"

// severityInfo is the report tier: printed by every surface, counted by no
// gate. record-lint exits non-zero on `blocker` alone, so an info finding is
// visible and inert by construction.
const severityInfo = "info"

var (
	readingRunDirRe   = regexp.MustCompile(`^` + issueschema.ReadingRunFamily + `-[0-9]+$`)
	readingItemFileRe = regexp.MustCompile(`^(` + issueschema.ReadingItemFamily + `-[0-9]+)\.md$`)
	// The admission filename grammar is the RESOLVER's, the same value
	// record_schema holds the store to — never a local copy. A stricter one here
	// would pass a record through the gate and then report the proposal it admits
	// as unadmitted: a confident false statement about a file the gate has just
	// accepted, and these records are hand-written this cycle, which is exactly
	// when `adm-N-<slug>.md` arrives.
	admissionFileRe = recordid.FilenameNumRe(issueschema.AdmissionFamily)
)

// admissionKey is the pair an admission actually answers: the RUN whose candidate
// set it joins, and the PROPOSAL it admits.
//
// The proposal alone is not a key. Keying on it alone made a proposal id a GLOBAL
// silencer: an admission filed under one run naming an id that belongs to another
// silenced the other run's item — the report going quiet about a proposal nobody
// had admitted, which is the one answer this leg exists to prevent
// (iss-2608300935215868). Reading ids do NOT collide across runs —
// mintUnusedItemID probes the whole ledger under its lock and redraws on a hit
// (iss-2608300227228575) — so the pair is what identifies the admission, never
// what disambiguates the id.
type admissionKey struct {
	run      string
	proposal string
}

// admissionTree is what one walk of the admission store learned: which pairs it
// admits, and — separately for each run — whether it read enough to say a
// proposal was NOT admitted.
//
// The readability verdict is per RUN because the claim it supports is per run.
// An admission counts only under the run it is filed in — this walk keys the
// admitted set on the (run, proposal) pair and reads the run from the DIRECTORY,
// so nothing an unreadable run holds can bear on another run's items. (What the
// walk does not establish is that the proposal named is one of that run's own:
// record_schema's sameBucketAs join is what refuses an admission reaching into
// another run, and until it landed such a record admitted nothing, silently —
// iss-2608301327013320.) So one run's fault says nothing about another's, and a
// store-wide verdict let a single symlinked or
// oversized file committed under one run empty the widening leg of the board for
// every run in the repository — a far wider silence than the one standing down
// exists to prevent. The disposition side already keeps this discipline: its
// whole-tree stand-down is a ROOT probe, and a leaf it cannot read withholds that
// item alone.
type admissionTree struct {
	admitted map[admissionKey]bool
	// rootUnreadable is the whole-tree verdict: the store root itself could not be
	// probed or listed, so nothing is known about any run.
	rootUnreadable bool
	// unreadableRuns names the run buckets the walk could not read in full.
	unreadableRuns map[string]bool
}

// unknown reports whether the walk read too little to say whether run admitted
// anything. An ABSENT tree, and an absent run inside a readable one, are both
// readable and empty: a repository that has admitted nothing is in a state, not a
// fault.
func (t admissionTree) unknown(run string) bool {
	return t.rootUnreadable || t.unreadableRuns[run]
}

// admits reports whether the store admits proposal within run.
func (t admissionTree) admits(run, proposal string) bool {
	return t.admitted[admissionKey{run: run, proposal: proposal}]
}

// OutstandingItem is one reading item nobody has answered.
type OutstandingItem struct {
	Item string `json:"item"`
	Run  string `json:"run"`
	Path string `json:"path"`
}

// OpenHold is one standing `held` disposition, rendered with the exit condition
// that is the only thing distinguishing a hold from a parking space.
type OpenHold struct {
	Item          string `json:"item"`
	Disposition   string `json:"disposition"`
	ExitCondition string `json:"exit_condition"`
	Path          string `json:"path"`
}

// OutstandingReadings is the whole report, ordered deterministically.
type OutstandingReadings struct {
	Undispositioned []OutstandingItem `json:"undispositioned"`
	// Unadmitted names widening PROPOSALS that are neither admitted nor declined
	// (spc-67). The widening position's answer set is wider than a disposition —
	// an admission record carrying its grounds answers a proposal, and so does a
	// decline — so the same question needs one more branch here rather than a
	// parallel rule beside it: two rules asking one question diverge, and the
	// first divergence between them is silent.
	//
	// Acceptance at the widening position IS admission, so an accepted proposal
	// with no admission record is an admission whose grounds were never written.
	// That is the case this leg exists for: uniform adoption of everything a
	// reading proposes is equally consistent with careful judgement and with
	// abdication, and only the grounds tell the two apart.
	Unadmitted []OutstandingItem `json:"unadmitted,omitempty"`
	OpenHolds  []OpenHold        `json:"open_holds"`
	// Unsafe names the repo-relative directories the walk declined to enter
	// because they are not real directories. core/capture REFUSES these outright,
	// because its read is followed by a write; this walk is genuinely read-only,
	// so it declines and SAYS SO instead. Going quiet is the thing it must not do:
	// a tree nobody walked looks exactly like a tree with nothing in it, and "no
	// outstanding items" is the one answer this report must never give by
	// accident.
	Unsafe []UnsafePath `json:"unsafe,omitempty"`
	// Cyclic names items whose dispositions supersede one another, so every
	// answer is retired and none stands. It is a ledger fault, not an unanswered
	// item: reporting it as outstanding would be a confident wrong statement about
	// an item that has been answered twice, and would invite exactly the fresh
	// uncited answer the write path refuses.
	Cyclic []OutstandingItem `json:"cyclic,omitempty"`
	// Contested names items on which more than one disposition stands. It is not
	// an exotic state: two branches each answer an item, both merge without
	// conflict, and neither cites the other. Picking the first by id resolved that
	// silently — an `accepted` sorting first hid a `held` and its exit condition,
	// and no line said two answers stood. The report names every standing id and
	// resolves nothing, because which answer is in force is the researcher's
	// judgement and there is nothing here to make it from.
	Contested []ContestedItem `json:"contested,omitempty"`
	// Unreadable names items whose SOLE standing answer is a record no reader can
	// read. Such a record has no state, so it matched none of the report's cases
	// and the item simply vanished from the board — and an item whose only answer
	// is unreadable is the case most in need of a line, not least.
	Unreadable []UnreadableAnswer `json:"unreadable,omitempty"`
}

// UnsafePath is one path the walk declined to read, with the reason it declined.
// The reason travels with the path because one line served every failure and was
// false for most of them: an oversized or permission-refused REGULAR FILE was
// described as "not a real directory", which sends the reader to look for a
// symlink that is not there.
type UnsafePath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// UnreadableAnswer is one item whose standing disposition cannot be read.
type UnreadableAnswer struct {
	Item        string `json:"item"`
	Run         string `json:"run"`
	Path        string `json:"path"`
	Disposition string `json:"disposition"`
}

// ContestedItem is one item with more than one standing answer.
type ContestedItem struct {
	Item string `json:"item"`
	Run  string `json:"run"`
	Path string `json:"path"`
	// Standing is every standing id, sorted — the whole fault, not a sample.
	Standing []string `json:"standing"`
}

// Empty reports whether there is nothing outstanding — the ordinary state of a
// repository that has commissioned no reading, and the state a surface renders
// as silence rather than as a heading with nothing under it.
func (r OutstandingReadings) Empty() bool {
	return len(r.Undispositioned) == 0 && len(r.Unadmitted) == 0 && len(r.OpenHolds) == 0 &&
		len(r.Unsafe) == 0 && len(r.Cyclic) == 0 && len(r.Contested) == 0 &&
		len(r.Unreadable) == 0
}

// ReadReadingOutstanding builds the report from the ledger at issuesDir
// (repo-relative). It is exported because the report has two surfaces — the lint
// findings and the bare capture status board — and two implementations of one
// question would be two answers to it. An absent readings tree is not an error:
// a repository that has commissioned no reading is in a state, not a fault.
func ReadReadingOutstanding(repoRoot, issuesDir string) (OutstandingReadings, error) {
	var report OutstandingReadings
	issuesRoot := filepath.Join(repoRoot, filepath.FromSlash(issuesDir))
	readingsRoot := filepath.Join(issuesRoot, issueschema.ReadingsDir)
	if !realDir(readingsRoot) {
		report.Unsafe = append(report.Unsafe, UnsafePath{
			Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.ReadingsDir)),
			Reason: notARealDirectory,
		})
		return report, nil
	}
	runs, err := os.ReadDir(readingsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	// The dispositions family root answers for every item below, so a link there
	// silently empties the standing set of ALL of them — every item would read as
	// unanswered. It is checked once, before any item is judged.
	dispositionsRoot := filepath.Join(issuesRoot, issueschema.DispositionsDir)
	dispositionsReadable := realDir(dispositionsRoot)
	if !dispositionsReadable {
		report.Unsafe = append(report.Unsafe, UnsafePath{
			Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.DispositionsDir)),
			Reason: notARealDirectory,
		})
	}
	// The admission side of the widening position's answer set, read once for the
	// same reason: a proposal's admission is a record in a different family, and
	// re-walking that tree per item would ask one question many times.
	admissions, admissionUnsafe := admittedProposals(issuesRoot, issuesDir)
	report.Unsafe = append(report.Unsafe, admissionUnsafe...)

	for _, run := range runs {
		if !readingRunDirRe.MatchString(run.Name()) {
			continue
		}
		runDir := filepath.Join(readingsRoot, run.Name())
		// A symlink is a directory to ReadDir, so the check is on the entry
		// itself, not on whether the walk would descend into it.
		if !realDir(runDir) {
			report.Unsafe = append(report.Unsafe, UnsafePath{
				Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.ReadingsDir, run.Name())),
				Reason: notARealDirectory,
			})
			continue
		}
		entries, err := os.ReadDir(runDir)
		if err != nil {
			return OutstandingReadings{}, err
		}
		for _, e := range entries {
			m := readingItemFileRe.FindStringSubmatch(e.Name())
			if e.IsDir() || m == nil {
				continue
			}
			item := m[1]
			rel := filepath.Join(issuesDir, issueschema.ReadingsDir, run.Name(), e.Name())
			// The item file itself, on the same terms as everything below it. A
			// symlinked rdi-N.md was admitted as a real item, so the board reported
			// it outstanding and the verb it named then refused to touch it.
			//
			// It is READ rather than stat-ed because the item's own POSITION decides
			// which answers count for it: a widening proposal is answered by an
			// admission or a decline, and every other position by a disposition. The
			// guarded read is one open with O_NOFOLLOW validated on the same
			// descriptor, so no symlink swap fits between the check and the read.
			content, rerr := fsutil.ReadGuarded(filepath.Join(runDir, e.Name()), issueschema.RecordReadLimit)
			if rerr != nil {
				report.Unsafe = append(report.Unsafe, UnsafePath{
					Path: filepath.ToSlash(rel), Reason: unreadableReason(rerr),
				})
				continue
			}
			position := readingPosition(string(content))
			if !dispositionsReadable {
				// The item's answer is unreadable, which is not the same fact as
				// "unanswered" — reporting it outstanding would be a confident
				// wrong statement, and the Unsafe line above already says why.
				continue
			}
			answer, err := standingDisposition(issuesRoot, issuesDir, item)
			if err != nil {
				return OutstandingReadings{}, err
			}
			switch {
			case len(answer.unsafe) > 0:
				// A path the walk could not read supports no claim about whether the
				// item has been answered. Saying "nobody has answered it" here would
				// invite the answer to be written a second time.
				report.Unsafe = append(report.Unsafe, answer.unsafe...)
			case answer.cyclic:
				report.Cyclic = append(report.Cyclic, OutstandingItem{
					Item: item, Run: run.Name(), Path: filepath.ToSlash(rel),
				})
			case len(answer.contested) > 1:
				report.Contested = append(report.Contested, ContestedItem{
					Item: item, Run: run.Name(), Path: filepath.ToSlash(rel),
					Standing: answer.contested,
				})
			case answer.standing == nil:
				// A widening proposal carrying an admission is answered: the
				// admission records the grounds it was admitted on, and telling the
				// researcher to answer a proposal they have already admitted would
				// be the report saying something the ledger contradicts.
				//
				// Readability supports ONE direction, and both halves are used
				// below: a record the walk actually READ is a fact whatever else in
				// that tree it could not read, so an admission it holds answers its
				// proposal even beside an unreadable sibling — while the claim that
				// a proposal is UNADMITTED needs this run's admissions behind it,
				// and stands down when they are not there to stand on.
				if position == issueschema.PositionWidening {
					if admissions.admits(run.Name(), item) {
						break
					}
					// And admissions nobody could read support no claim that this
					// proposal was NOT admitted — the direction that needs the tree
					// behind it. Reporting it outstanding here would tell the
					// researcher to write a DISPOSITION, which is the wrong record
					// for a proposal that may already carry an admission, and would
					// contradict the invariant the disposition branch above keeps
					// for its own tree. The stand-down is scoped to THIS run,
					// because an admission for this item could only have lived
					// there; the Unsafe line names the path that could not be read.
					if admissions.unknown(run.Name()) {
						break
					}
				}
				report.Undispositioned = append(report.Undispositioned, OutstandingItem{
					Item: item, Run: run.Name(), Path: filepath.ToSlash(rel),
				})
			case !answer.standing.wellFormed:
				report.Unreadable = append(report.Unreadable, UnreadableAnswer{
					Item: item, Run: run.Name(), Path: filepath.ToSlash(answer.standing.rel),
					Disposition: answer.standing.id,
				})
			}
			// The admission leg. It sits outside the switch for the same reason the
			// holds do: it is an ADDITIONAL fact about an item that already has a
			// standing answer, not an alternative to it.
			//
			// A `declined` proposal is answered on the ledger side and owes nothing
			// further. A `held` one is already published with its exit condition
			// below, and naming it twice would trade one silence for a duplicate —
			// whether `held` is even available at this position is deferred to the
			// first widening run's dispositions, so this leg does not decide it.
			// Everything else standing on a widening proposal — acceptance above
			// all, which at this position IS admission — needs an admission record,
			// because that is where the grounds live.
			if position == issueschema.PositionWidening && !admissions.unknown(run.Name()) &&
				!admissions.admits(run.Name(), item) &&
				answer.standing != nil && answer.standing.wellFormed &&
				answer.standing.state != issueschema.DispositionDeclined &&
				answer.standing.state != issueschema.DispositionHeld {
				report.Unadmitted = append(report.Unadmitted, OutstandingItem{
					Item: item, Run: run.Name(), Path: filepath.ToSlash(rel),
				})
			}
			// Every standing hold is rendered, whatever else is true of the item —
			// including an item the walk has just called contested. Naming the
			// contest and then withholding the one thing a hold exists to publish
			// would trade one silence for another. It sits outside the switch
			// because a hold is not an alternative to the facts above; it is an
			// additional one.
			report.OpenHolds = append(report.OpenHolds, answer.holds...)
		}
	}

	sort.Slice(report.Undispositioned, func(i, j int) bool {
		return report.Undispositioned[i].Item < report.Undispositioned[j].Item
	})
	sort.Slice(report.Unadmitted, func(i, j int) bool {
		return report.Unadmitted[i].Item < report.Unadmitted[j].Item
	})
	sort.Slice(report.OpenHolds, func(i, j int) bool {
		return report.OpenHolds[i].Item < report.OpenHolds[j].Item
	})
	sort.Slice(report.Cyclic, func(i, j int) bool { return report.Cyclic[i].Item < report.Cyclic[j].Item })
	sort.Slice(report.Contested, func(i, j int) bool { return report.Contested[i].Item < report.Contested[j].Item })
	sort.Slice(report.Unreadable, func(i, j int) bool { return report.Unreadable[i].Item < report.Unreadable[j].Item })
	sort.Slice(report.Unsafe, func(i, j int) bool { return report.Unsafe[i].Path < report.Unsafe[j].Path })
	return report, nil
}

// readingPosition reads the `position` off a reading record's frontmatter. It is
// the lenient scanner the record-schema rule already reads every record in this
// tree with, not a second parser: what the position decides here is which
// ANSWERS count for an item, and a record whose frontmatter no reader can read
// carries no position, so it takes the ordinary disposition-only path.
func readingPosition(content string) string {
	f := frontmatterFields(strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n"))["position"]
	return issueScalar(f.value)
}

// admittedProposals reads the admission store once and returns the set of
// proposals it admits, whether the store was readable at all, and the paths the
// walk declined to read.
//
// The readability verdict travels with the set because an unreadable admission
// tree is not an empty one: reporting every widening proposal as unadmitted
// because nobody could read the admissions would be the same confident wrong
// statement the disposition side already refuses to make. It is recorded PER RUN,
// so one run's fault withholds only that run's proposals — see admissionTree. An
// ABSENT tree, and an absent run inside a readable one, are readable and empty: a
// repository that has admitted nothing is in a state, not a fault.
func admittedProposals(issuesRoot, issuesDir string) (admissionTree, []UnsafePath) {
	tree := admissionTree{
		admitted:       map[admissionKey]bool{},
		unreadableRuns: map[string]bool{},
	}
	var unsafe []UnsafePath
	admissionsRoot := filepath.Join(issuesRoot, issueschema.AdmissionsDir)
	if !realDir(admissionsRoot) {
		tree.rootUnreadable = true
		return tree, []UnsafePath{{
			Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.AdmissionsDir)),
			Reason: notARealDirectory,
		}}
	}
	runs, err := os.ReadDir(admissionsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return tree, nil
		}
		// A root that exists and cannot be listed supports no claim about what has
		// been admitted in ANY run, which is the same fact an unsafe root carries.
		tree.rootUnreadable = true
		return tree, []UnsafePath{{
			Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.AdmissionsDir)),
			Reason: unreadableReason(err),
		}}
	}
	for _, run := range runs {
		if !readingRunDirRe.MatchString(run.Name()) {
			continue
		}
		runRel := filepath.Join(issuesDir, issueschema.AdmissionsDir, run.Name())
		runDir := filepath.Join(admissionsRoot, run.Name())
		if !realDir(runDir) {
			unsafe = append(unsafe, UnsafePath{Path: filepath.ToSlash(runRel), Reason: notARealDirectory})
			tree.unreadableRuns[run.Name()] = true
			continue
		}
		entries, err := os.ReadDir(runDir)
		if err != nil {
			unsafe = append(unsafe, UnsafePath{Path: filepath.ToSlash(runRel), Reason: unreadableReason(err)})
			tree.unreadableRuns[run.Name()] = true
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !admissionFileRe.MatchString(e.Name()) {
				continue
			}
			content, err := fsutil.ReadGuarded(filepath.Join(runDir, e.Name()), issueschema.RecordReadLimit)
			if err != nil {
				unsafe = append(unsafe, UnsafePath{
					Path:   filepath.ToSlash(filepath.Join(runRel, e.Name())),
					Reason: unreadableReason(err),
				})
				tree.unreadableRuns[run.Name()] = true
				continue
			}
			fields := frontmatterFields(strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n"))
			proposal := issueScalar(fields["proposal"].value)
			// The record states its run twice — the directory it sits in and its own
			// `run` — and a disagreement is the record contradicting itself about
			// which candidate set it joined. Honouring the bucket alone would let the
			// field lie; honouring the field alone would make the bucket decorative.
			// So it admits under neither, and record_schema names the contradiction.
			//
			// The `proposal == ""` half is the SET'S DEFINITION, not defence: this
			// returns the proposals the store admits, and an admission naming nothing
			// admits nothing, so it contributes no key. Its effect is invisible through
			// admits(), because every query is an item name captured out of a filename
			// matching readingItemFileRe and so is never empty — but that is a property
			// of the caller, one call site away, and it is not what makes this set
			// correct. It is pinned on the set itself
			// (TestAnAdmissionNamingNoProposalIsKeyedOnNothing) rather than deleted for
			// being unobservable (iss-2608301656202623).
			if proposal == "" || issueScalar(fields["run"].value) != run.Name() {
				continue
			}
			tree.admitted[admissionKey{run: run.Name(), proposal: proposal}] = true
		}
	}
	return tree, unsafe
}

// The reasons an Unsafe entry can carry. They are prose because they are read by
// a person deciding what to go and look at, and each names a DIFFERENT thing to
// look for.
const (
	notARealDirectory = "not a real directory (a symlink, or not a directory at all)"
	notARegularFile   = "not a regular file (a symlink, a directory, or a device)"
)

// unreadableReason turns a guarded-read failure into the thing a reader should
// go and look for. It exists because one sentence used to serve every failure and
// was false for most of them.
func unreadableReason(err error) string {
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return "larger than the " + strconv.Itoa(issueschema.RecordReadLimit) + "-byte record cap"
	case errors.Is(err, fsutil.ErrNotRegular), errors.Is(err, syscall.ELOOP):
		return notARegularFile
	case errors.Is(err, fs.ErrPermission):
		return "unreadable: permission denied"
	case errors.Is(err, fs.ErrNotExist):
		return "gone between the directory listing and the read"
	}
	return "unreadable"
}

// realDir reports whether path is a directory the walk may enter — present, and
// not a symlink. An ABSENT path is not unsafe: an unpopulated tree is a state,
// and the caller distinguishes the two by probing separately.
func realDir(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return fi.IsDir() && fi.Mode()&os.ModeSymlink == 0
}

// standingRecord is the disposition currently in force for one item.
type standingRecord struct {
	id            string
	state         string
	exitCondition string
	rel           string
	// wellFormed carries issueschema's verdict forward, so a sole standing record
	// nobody can read is reported as unreadable rather than falling through every
	// case on the strength of its empty state.
	wellFormed bool
}

// itemAnswer is everything the walk learned about one item's dispositions. It is
// a struct rather than a tuple because the facts are not alternatives to be
// squeezed into one return value: an item can be contested AND carry a hold, and
// collapsing that pair is exactly how the hold went missing.
type itemAnswer struct {
	// standing is the single record in force, or nil when none is.
	standing *standingRecord
	// contested is every standing id when more than one stands.
	contested []string
	// cyclic reports records present with none standing — a supersession cycle.
	cyclic bool
	// holds is every standing record that is a hold, so an exit condition is
	// published whatever else is true of the item.
	holds []OpenHold
	// unsafe names the repo-relative paths the walk declined to read. Any one of
	// them means the item's answer is unknown, not absent.
	unsafe []UnsafePath
}

// standingDisposition reads one item's dispositions and says what stands.
func standingDisposition(issuesRoot, issuesDir, item string) (itemAnswer, error) {
	var answer itemAnswer
	itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, item)
	if !realDir(itemDir) {
		answer.unsafe = append(answer.unsafe, UnsafePath{
			Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.DispositionsDir, item)),
			Reason: notARealDirectory,
		})
		return answer, nil
	}
	entries, err := os.ReadDir(itemDir)
	if err != nil {
		if os.IsNotExist(err) {
			return answer, nil
		}
		return answer, err
	}
	var records []issueschema.DispositionRecord
	byID := map[string]issueschema.DispositionRecord{}
	rel := map[string]string{}
	for _, e := range entries {
		id, ok := issueschema.DispositionFileID(e.Name())
		if !ok {
			continue
		}
		// ReadGuarded rather than Lstat-then-ReadFile: it opens once with
		// O_NOFOLLOW and validates on the SAME descriptor, so no symlink swap fits
		// between the check and the read. A record it refuses makes the item's
		// answer UNKNOWN, which is not the same fact as unanswered.
		content, err := fsutil.ReadGuarded(filepath.Join(itemDir, e.Name()), issueschema.RecordReadLimit)
		if err != nil {
			answer.unsafe = append(answer.unsafe, UnsafePath{
				Path:   filepath.ToSlash(filepath.Join(issuesDir, issueschema.DispositionsDir, item, e.Name())),
				Reason: unreadableReason(err),
			})
			continue
		}
		r := issueschema.ParseDisposition(id, string(content))
		records = append(records, r)
		byID[id] = r
		rel[id] = filepath.Join(issuesDir, issueschema.DispositionsDir, item, e.Name())
	}

	// The WALK is here; the JUDGEMENT is not. Which records stand is decided by
	// issueschema.StandingDispositionIDs, the one reader core/capture calls too:
	// a record neither can read retires nothing and does not vanish either, so
	// the board and the verb cannot disagree about what is in force.
	standing := issueschema.StandingDispositionIDs(records)
	if len(standing) == 0 {
		// Records present with nothing standing is a supersession CYCLE; no
		// records at all is simply an item nobody has answered. Rendering the two
		// the same way would announce an item answered twice as unanswered.
		answer.cyclic = len(records) > 0
		return answer, nil
	}

	for _, id := range standing {
		r := byID[id]
		if r.State != issueschema.DispositionHeld {
			continue
		}
		answer.holds = append(answer.holds, OpenHold{
			Item: item, Disposition: id, ExitCondition: r.ExitCondition,
			Path: filepath.ToSlash(rel[id]),
		})
	}
	if len(standing) > 1 {
		// No first-by-id tie-break. Which answer is in force is the researcher's
		// judgement, and there is nothing here to make it from; choosing one would
		// publish a verdict the ledger does not contain.
		answer.contested = standing
		return answer, nil
	}

	r := byID[standing[0]]
	answer.standing = &standingRecord{
		id: r.ID, state: r.State, exitCondition: r.ExitCondition, rel: rel[r.ID],
		wellFormed: r.WellFormed,
	}
	return answer, nil
}

// checkReadingOutstanding renders the report as findings, every one of them at
// severityInfo whatever the configuration says.
func checkReadingOutstanding(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	report, err := ReadReadingOutstanding(repoRoot, issuesDirOf(cfg))
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, o := range report.Undispositioned {
		out = append(out, Finding{
			File: o.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: o.Item + " (run " + o.Run + ") carries no disposition — outstanding. " +
				"An item nobody has answered has no state to sit in, because nothing in this " +
				"vocabulary means \"already covered\"; answer it with `abcd capture disposition " +
				o.Item + " --state <state> ...`",
		})
	}
	for _, o := range report.Unadmitted {
		out = append(out, Finding{
			File: o.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: o.Item + " (run " + o.Run + ") is a widening proposal with neither an admission nor a decline — outstanding. " +
				"At the widening position acceptance IS admission, and the grounds an admission was made on live in an " +
				"admission record (`" + issueschema.AdmissionFamily + "-N` under " + issueschema.AdmissionsDir + "/" + o.Run +
				"/), because uniform adoption of everything a reading proposes is equally consistent with judgement and with abdication. " +
				"Declining costs nothing epistemically and is recorded as a disposition in the `" + issueschema.DispositionDeclined + "` state",
		})
	}
	for _, u := range report.Unreadable {
		out = append(out, Finding{
			File: u.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: u.Item + " (run " + u.Run + ") stands on " + u.Disposition +
				", which no reader of this ledger can read — its frontmatter is malformed, so the record carries no state. " +
				"The item is answered and the answer is illegible; repair the record rather than writing a second one",
		})
	}
	for _, u := range report.Unsafe {
		out = append(out, Finding{
			File: u.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: "the reading walk did not read this — " + u.Reason + ". " +
				"What it holds is neither reported outstanding nor confirmed answered, because a path nobody read " +
				"supports no claim either way. `abcd capture` refuses the same paths outright, because its read is followed by a write",
		})
	}
	for _, c := range report.Contested {
		out = append(out, Finding{
			File: c.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: c.Item + " (run " + c.Run + ") has " + strconv.Itoa(len(c.Standing)) +
				" standing answers, none superseding another: " + strings.Join(c.Standing, ", ") +
				". Which one is in force is a judgement the ledger does not contain, so nothing here picks one. " +
				"`abcd capture disposition " + c.Item + "` refuses until exactly one stands: write " +
				"`supersedes_disposition` into the records that are no longer meant to stand, by hand — a new " +
				"disposition would retire one id and add its own, so the contest would never shrink",
		})
	}
	for _, c := range report.Cyclic {
		out = append(out, Finding{
			File: c.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: c.Item + " (run " + c.Run + ") carries dispositions that supersede one another, so none stands — " +
				"a ledger fault, not an unanswered item. No write path can produce it and only a hand edit can repair it; " +
				"`abcd capture disposition " + c.Item + "` refuses until it is untied",
		})
	}
	for _, h := range report.OpenHolds {
		out = append(out, Finding{
			File: h.Path, Line: 1, RuleID: ruleReadingOutstanding, Severity: severityInfo,
			Message: h.Item + " is held (" + h.Disposition + "), exit condition: " + h.ExitCondition +
				". A hold exits only through a superseding disposition that cites it — never by expiry, and never silently",
		})
	}
	return out, nil
}
