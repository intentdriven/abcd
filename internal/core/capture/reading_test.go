package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// readingFixture ingests one run of one detection item into a fresh ledger and
// returns the roots plus the minted item id.
func readingFixture(t *testing.T, position string) (repo, ir, item string) {
	t.Helper()
	repo, ir = ledger(t)
	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: position, Regime: issueschema.ReadingRegime(position),
		Items: []ReadingItem{{Pattern: "a stated constraint", Body: bodyFor(position)}},
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if len(res.Records) != 1 {
		t.Fatalf("IngestReading wrote %d records, want 1", len(res.Records))
	}
	return repo, ir, res.Records[0].ID
}

// bodyFor fills every body field the position declares with a placeholder, so a
// test that is not about the body never fails on it.
func bodyFor(position string) map[string]string {
	body := map[string]string{}
	for _, f := range issueschema.ReadingBodyFields[position] {
		body[f] = "text for " + f
	}
	return body
}

// A reading record is an additionalProperties:false record like every other
// record in this ledger: a key outside the schema is refused at the boundary,
// never accepted and flagged. Accepting it would put a property in the committed
// tree that no reader reads and no gate judges.
func TestReadingRecordRefusesUnknownProperty(t *testing.T) {
	repo, ir := ledger(t)
	_, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{{
			Pattern: "a stated constraint",
			Body:    map[string]string{"tension": "t", "constraint_in_play": "c", "why_a_tension": "w", "vibes": "no"},
		}},
	})
	if !errors.Is(err, ErrMalformedFrontmatter) {
		t.Fatalf("IngestReading with an unknown body property err = %v, want ErrMalformedFrontmatter", err)
	}
	if !strings.Contains(err.Error(), "vibes") {
		t.Fatalf("the refusal must name the offending property; got %v", err)
	}
}

// One record type, four bodies, held as data: an item at a position must carry
// exactly the body that position declares. A missing field is refused and the
// refusal names the position's set, so the caller is told the rule rather than
// left to infer it.
func TestReadingRecordRequiresPositionBodyFields(t *testing.T) {
	for _, p := range issueschema.ReadingPositions {
		t.Run(p.Position, func(t *testing.T) {
			for _, omit := range p.Fields {
				body := bodyFor(p.Position)
				delete(body, omit)
				repo, ir := ledger(t)
				_, err := IngestReading(IngestReadingRequest{
					RepoRoot: repo, IssuesRoot: ir,
					Run: "rdg-2608300000000001", Manifest: "sha256:beef",
					Position: p.Position, Regime: p.Regime,
					Items: []ReadingItem{{Pattern: "a stated constraint", Body: body}},
				})
				if !errors.Is(err, ErrMissingRequiredField) {
					t.Fatalf("omitting %q at %s: err = %v, want ErrMissingRequiredField", omit, p.Position, err)
				}
				if !strings.Contains(err.Error(), omit) || !strings.Contains(err.Error(), p.Position) {
					t.Fatalf("the refusal must name the field and the position; got %v", err)
				}
			}
		})
	}
}

// The grounds are what make a disposition a judgement rather than a vote. They
// are required on every state except `held`, whose exit condition carries the
// same weight.
func TestDispositionRefusesEmptyGrounds(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	for _, state := range []string{
		issueschema.DispositionAccepted, issueschema.DispositionRejected,
	} {
		_, err := Disposition(DispositionRequest{
			RepoRoot: repo, IssuesRoot: ir, Item: item, State: state, Grounds: "   ",
		})
		if !errors.Is(err, ErrMissingRequiredField) {
			t.Fatalf("%s with empty grounds: err = %v, want ErrMissingRequiredField", state, err)
		}
		if !strings.Contains(err.Error(), "disposition_grounds") {
			t.Fatalf("the refusal must name disposition_grounds; got %v", err)
		}
	}
}

// A hold with no exit condition is a parking space, which is the thing `open`
// already is and the thing `held` exists not to be.
func TestHeldDispositionRefusesEmptyExitCondition(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	_, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionHeld, ExitCondition: "",
	})
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Fatalf("held with no exit condition: err = %v, want ErrMissingRequiredField", err)
	}
	if !strings.Contains(err.Error(), "exit_condition") {
		t.Fatalf("the refusal must name exit_condition; got %v", err)
	}
}

// The availability rule is a coupling the schema carries: the disposition reads
// `position` off the KEYED reading record and checks the table. `declined` at a
// detection would manufacture a principle never at stake; `rejected` at the
// widening position would assert a purpose nobody proposed.
func TestDispositionRefusesStateUnavailableAtPosition(t *testing.T) {
	for _, tc := range []struct{ position, state string }{
		{"detection", issueschema.DispositionDeclined},
		{"widening", issueschema.DispositionRejected},
	} {
		repo, ir, item := readingFixture(t, tc.position)
		_, err := Disposition(DispositionRequest{
			RepoRoot: repo, IssuesRoot: ir, Item: item, State: tc.state, Grounds: "because",
		})
		if !errors.Is(err, ErrInvariantViolation) {
			t.Fatalf("%s at %s: err = %v, want ErrInvariantViolation", tc.state, tc.position, err)
		}
		if !strings.Contains(err.Error(), tc.state) || !strings.Contains(err.Error(), tc.position) {
			t.Fatalf("the refusal must name the availability rule it enforces; got %v", err)
		}
	}
}

// An orphan disposition — one keyed to an item no run returned — is refused by
// the SAME path that checks availability, because the check needs the item's
// position and cannot invent one.
func TestDispositionRefusesOrphanItem(t *testing.T) {
	repo, ir, _ := readingFixture(t, "detection")
	_, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: "rdi-2608300000009999",
		State: issueschema.DispositionAccepted, Grounds: "because",
	})
	if !errors.Is(err, ErrUnknownIssueID) {
		t.Fatalf("orphan disposition: err = %v, want ErrUnknownIssueID", err)
	}
}

// The two-axis hold field is reserved and DORMANT. A populated value is refused
// until activation is ruled, which is what makes the reservation a behaviour
// rather than a comment: silently accepting one would populate a field whose
// meaning nobody has settled.
func TestPopulatedHoldAxisRefused(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	for _, tc := range []struct{ field, value string }{
		{"hold_frame_location", "the frame's cost element"},
		{"hold_moscow", "must"},
	} {
		req := DispositionRequest{
			RepoRoot: repo, IssuesRoot: ir, Item: item,
			State: issueschema.DispositionHeld, ExitCondition: "the closing run returns it again",
		}
		if tc.field == "hold_frame_location" {
			req.HoldFrameLocation = tc.value
		} else {
			req.HoldMoscow = tc.value
		}
		_, err := Disposition(req)
		if !errors.Is(err, ErrInvariantViolation) {
			t.Fatalf("populated %s: err = %v, want ErrInvariantViolation", tc.field, err)
		}
		if !strings.Contains(err.Error(), tc.field) {
			t.Fatalf("the refusal must name the reserved field; got %v", err)
		}
	}
}

// The reserved surprise key gets the same posture as the hold axes: reserved in
// the family now, populated in Iteration 2, and refused until then.
func TestPopulatedSurpriseKeyRefused(t *testing.T) {
	repo, ir := ledger(t)
	_, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{{
			Pattern: "a stated constraint", Body: bodyFor("detection"),
			OccasionedBy: "iss-1",
		}},
	})
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("populated occasioned_by: err = %v, want ErrInvariantViolation", err)
	}
	if !strings.Contains(err.Error(), "occasioned_by") {
		t.Fatalf("the refusal must name the reserved field; got %v", err)
	}
}

// A disposition lands under a directory keyed by the ITEM, and its own file is
// keyed by its own id — the shape that lets a superseding disposition cite the
// one it replaces.
func TestDispositionLandsInTheItemKeyedDirectory(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	res, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the tension is real and worth acting on",
	})
	if err != nil {
		t.Fatalf("Disposition: %v", err)
	}
	wantDir := filepath.ToSlash(filepath.Join(LedgerRelPath, issueschema.DispositionsDir, item))
	if !strings.HasPrefix(filepath.ToSlash(res.Path), wantDir+"/") {
		t.Fatalf("disposition path = %q, want it under %q", res.Path, wantDir)
	}
	if !strings.HasPrefix(filepath.Base(res.Path), issueschema.DispositionFamily+"-") {
		t.Fatalf("disposition filename = %q, want a dsp-N record", filepath.Base(res.Path))
	}
}

// The last guard before an irreversible write. The mint now proves every id free
// across the whole ledger under the same lock, so this can only fire against a
// writer that did not take that lock — a hand-edit, or another tool. That race is
// not reachable in-process, which is exactly why the guard is tested directly
// rather than through an ingest that cannot reach it: an atomic write would
// otherwise replace a file nobody knows about, leaving no trace that it existed.
func TestRefuseExistingRecordGuardsAnIrreversibleWrite(t *testing.T) {
	dir := t.TempDir()
	free := filepath.Join(dir, "rdi-2608300000000001.md")
	if err := refuseExistingRecord(free, "rdi-2608300000000001"); err != nil {
		t.Fatalf("a free path must not be refused: %v", err)
	}
	taken := filepath.Join(dir, "rdi-2608300000000002.md")
	if err := os.WriteFile(taken, []byte("a file abcd did not write\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := refuseExistingRecord(taken, "rdi-2608300000000002")
	if !errors.Is(err, ErrDuplicateIssueID) {
		t.Fatalf("a taken path: err = %v, want ErrDuplicateIssueID", err)
	}
	content, readErr := os.ReadFile(taken)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), "a file abcd did not write") {
		t.Fatal("the refused write must leave the existing file untouched")
	}
}

// The supply regime is resolved from the definition THROUGH the position, so no
// operator input can set it (ruling (4), ruling (18)). Validating it as a free
// string would leave the one field whose whole purpose is to be underivable by
// the caller derivable by the caller: the record must say which regime it was
// read under, and a regime that disagrees with its own position says nothing.
func TestIngestRefusesARegimeThePositionDoesNotImply(t *testing.T) {
	repo, ir := ledger(t)
	_, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: "sha256:beef",
		Position: "detection", Regime: "generative",
		Items: []ReadingItem{{Pattern: "a stated constraint", Body: bodyFor("detection")}},
	})
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("a regime the position does not imply: err = %v, want ErrInvariantViolation", err)
	}
	if !strings.Contains(err.Error(), "generative") || !strings.Contains(err.Error(), "registrative") {
		t.Fatalf("the refusal must name the regime given and the one the position implies; got %v", err)
	}
}

// Every position implies exactly one regime, and an ingest that states it
// correctly is accepted — the gate above must not be a gate on all four.
func TestIngestAcceptsThePositionsOwnRegime(t *testing.T) {
	for _, p := range issueschema.ReadingPositions {
		t.Run(p.Position, func(t *testing.T) {
			repo, ir := ledger(t)
			if _, err := IngestReading(IngestReadingRequest{
				RepoRoot: repo, IssuesRoot: ir,
				Run: "rdg-2608300000000001", Manifest: "sha256:beef",
				Position: p.Position, Regime: issueschema.ReadingRegime(p.Position),
				Items: []ReadingItem{{Pattern: "a stated constraint", Body: bodyFor(p.Position)}},
			}); err != nil {
				t.Fatalf("IngestReading at %s: %v", p.Position, err)
			}
		})
	}
}

// The manifest reference is free text an instrument hands over, and it lands in
// the committed ledger like every other free-text value here. Writing it verbatim
// is the same leak the ledger redactor exists to close — nothing upstream
// constrains what it holds.
func TestIngestRedactsTheManifestReference(t *testing.T) {
	repo, ir := ledger(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	res, err := IngestReading(IngestReadingRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Run: "rdg-2608300000000001", Manifest: filepath.Join(home, "runs", "manifest.json"),
		Position: "detection", Regime: "registrative",
		Items: []ReadingItem{{Pattern: "a stated constraint", Body: bodyFor("detection")}},
	})
	if err != nil {
		t.Fatalf("IngestReading: %v", err)
	}
	if res.Redacted == 0 {
		t.Fatal("Redacted = 0, want the home path in the manifest reference counted")
	}
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.Records[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), home) {
		t.Fatalf("the committed reading record carries the caller's home root:\n%s", content)
	}
}

// The ledger's status directories refuse a symlinked leaf, because a link is a
// way to make a write land outside the tree that is supposed to contain it. The
// readings tree needed the same refusal and did not have it on every path: only
// the ingest provisioned its directories (and so met safeMkdirLeaf), while the
// disposition and promote paths merely READ through whatever was there — so a
// symlinked readings root let promote stamp a file outside the ledger.
func TestReadingPathsRefuseASymlinkedTree(t *testing.T) {
	t.Run("the readings root", func(t *testing.T) {
		repo, ir := ledger(t)
		if err := ensureLedgerDirs(repo, ir); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(ir, issueschema.ReadingsDir)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		_, err := findReadingItem(ir, "rdi-2608300000000002")
		if !errors.Is(err, ErrPathUnsafe) {
			t.Fatalf("a symlinked readings root: err = %v, want ErrPathUnsafe", err)
		}
		_, err = Disposition(DispositionRequest{
			RepoRoot: repo, IssuesRoot: ir, Item: "rdi-2608300000000002",
			State: issueschema.DispositionAccepted, Grounds: "because",
		})
		if !errors.Is(err, ErrPathUnsafe) {
			t.Fatalf("Disposition through a symlinked readings root: err = %v, want ErrPathUnsafe", err)
		}
	})

	t.Run("a run directory", func(t *testing.T) {
		repo, ir, item := readingFixture(t, "detection")
		runDir := filepath.Join(ir, issueschema.ReadingsDir, "rdg-2608300000000009")
		outside := t.TempDir()
		if err := os.Symlink(outside, runDir); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		_, err := findReadingItem(ir, item)
		if !errors.Is(err, ErrPathUnsafe) {
			t.Fatalf("a symlinked run directory: err = %v, want ErrPathUnsafe", err)
		}
		_ = repo
	})
}

// The dispositions FAMILY root was not guarded — only the item leaf below it —
// so promote read an item's standing state through a symlinked dispositions/,
// and the answer that licenses the stamp came from outside the ledger.
func TestDispositionsFamilyRootSymlinkRefused(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(ir, issueschema.DispositionsDir)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := standingDispositions(filepath.Join(ir, issueschema.DispositionsDir, item)); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("standingDispositions through a symlinked family root: err = %v, want ErrPathUnsafe", err)
	}
	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item}); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("Promote through a symlinked dispositions root: err = %v, want ErrPathUnsafe", err)
	}
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "because",
	}); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("Disposition through a symlinked dispositions root: err = %v, want ErrPathUnsafe", err)
	}
}

// A disposition FILE that is a symlink is read out of the ledger just as surely
// as a directory that is one: the standing computation, promote's state read and
// the board's exit-condition line all take their answer from whatever it points
// at. It is refused rather than followed.
func TestSymlinkedDispositionFileRefused(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "the tension is real",
	}); err != nil {
		t.Fatalf("Disposition: %v", err)
	}
	itemDir := filepath.Join(ir, issueschema.DispositionsDir, item)

	outside := filepath.Join(t.TempDir(), "elsewhere.md")
	if err := os.WriteFile(outside, []byte("---\nid: \"dsp-2608300000000009\"\nstate: \"accepted\"\n---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(itemDir, "dsp-2608300000000009.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := standingDispositions(itemDir); !errors.Is(err, ErrPathUnsafe) {
		t.Fatalf("a symlinked disposition file: err = %v, want ErrPathUnsafe", err)
	}
}

// writeRawDisposition drops a disposition into an item's directory by hand,
// which is the only way the two faults below can arise: the verb refuses a second
// answer that does not supersede a standing one, and --supersedes must name a
// standing id, so neither a self-citation nor a cycle can be written through it.
func writeRawDisposition(t *testing.T, ir, item, id, state, supersedes string) {
	t.Helper()
	dir := filepath.Join(ir, issueschema.DispositionsDir, item)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nschema_version: 1\nid: \"" + id + "\"\nitem: \"" + item + "\"\nstate: \"" + state + "\"\n" +
		"disposition_grounds: \"by hand\"\n"
	if supersedes != "" {
		body += "supersedes_disposition: \"" + supersedes + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(body+"---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A disposition citing ITSELF retires nothing. Honouring the citation would make
// the record supersede itself out of the standing set, leaving the item reading
// as undispositioned — so the verb would accept a fresh uncited answer and the
// board would report an item that plainly carries one as unanswered.
func TestSelfSupersedingDispositionRetiresNothing(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	writeRawDisposition(t, ir, item, "dsp-2608300000000001", "accepted", "dsp-2608300000000001")

	standing, err := standingDispositions(filepath.Join(ir, issueschema.DispositionsDir, item))
	if err != nil {
		t.Fatalf("standingDispositions: %v", err)
	}
	if len(standing) != 1 || standing[0] != "dsp-2608300000000001" {
		t.Fatalf("standing = %v, want the self-citing record to stand", standing)
	}
	_, err = Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "a second answer",
	})
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("a second answer beside a standing one: err = %v, want ErrInvariantViolation", err)
	}
}

// Two dispositions superseding each other leave NOTHING standing over a
// non-empty set — every answer retired, and an item that carries two answers
// reading as though it carries none. It is a ledger fault, not an unanswered
// item, and the difference matters: the verb must not accept a fresh uncited
// answer on top of it, and the board must not report it as outstanding.
func TestSupersessionCycleIsALedgerFault(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	writeRawDisposition(t, ir, item, "dsp-2608300000000001", "accepted", "dsp-2608300000000002")
	writeRawDisposition(t, ir, item, "dsp-2608300000000002", "rejected", "dsp-2608300000000001")

	_, err := standingDispositions(filepath.Join(ir, issueschema.DispositionsDir, item))
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("a supersession cycle: err = %v, want ErrInvariantViolation", err)
	}
	if !strings.Contains(err.Error(), item) {
		t.Fatalf("the refusal must name the item; got %v", err)
	}

	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: "a fresh answer",
	}); !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("Disposition over a cycle: err = %v, want ErrInvariantViolation", err)
	}
	if _, err := Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: item}); !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("Promote over a cycle: err = %v, want ErrInvariantViolation", err)
	}
}

// contestedFixture leaves two independent standing answers on one item, the
// state two branches produce by each answering it and merging cleanly.
func contestedFixture(t *testing.T) (repo, ir, item string) {
	t.Helper()
	repo, ir, item = readingFixture(t, "detection")
	writeRawDisposition(t, ir, item, "dsp-2608300000000001", "accepted", "")
	writeRawDisposition(t, ir, item, "dsp-2608300000000002", "held", "")
	return repo, ir, item
}

// RULED: the disposition verb refuses under contest, exactly as promote does.
//
// The remedy the board used to offer was `--supersedes`, which retires one id
// and adds one, so a contested set never shrinks — the reader would have been
// sent round a loop. Untangling it means writing supersedes_disposition into the
// surplus records by hand, which is what the refusal now says.
func TestDispositionRefusesUnderContest(t *testing.T) {
	repo, ir, item := contestedFixture(t)

	for _, tc := range []struct {
		name       string
		supersedes string
	}{
		{"a fresh answer", ""},
		{"an answer citing one of them", "dsp-2608300000000001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Disposition(DispositionRequest{
				RepoRoot: repo, IssuesRoot: ir, Item: item,
				State: issueschema.DispositionAccepted, Grounds: "another answer",
				Supersedes: tc.supersedes,
			})
			if !errors.Is(err, ErrInvariantViolation) {
				t.Fatalf("a disposition under contest: err = %v, want ErrInvariantViolation", err)
			}
			if !strings.Contains(err.Error(), "dsp-2608300000000001") ||
				!strings.Contains(err.Error(), "dsp-2608300000000002") {
				t.Fatalf("the refusal must name every standing answer; got %v", err)
			}
			if !strings.Contains(err.Error(), "supersedes_disposition") {
				t.Fatalf("the refusal must name the hand repair; got %v", err)
			}
		})
	}
}
