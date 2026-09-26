package lab

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Minted is what mint laid down.
type Minted struct {
	Entry
	Home     string   `json:"home"`
	Snapshot string   `json:"snapshot"`
	Written  []string `json:"written"`
	Next     []string `json:"next"`
}

// Mint lays down a new lab for question, pinned at pin (a revision of the
// repository at repoRoot; empty means HEAD): the lab home under this
// repository's lane of the store, a standalone clone of the repository detached
// at the pin, the lifecycle's documents, and one registry line. Nothing is
// written into the repository.
//
// The clone runs under the isolated git environment, so no hook the operator's
// global configuration names fires while it is made, and its origin remote is
// removed, so nothing done in the lab world can be pushed back into the checkout
// it came from.
func Mint(repoRoot, question, pin string) (Minted, error) {
	q := strings.TrimSpace(question)
	if q == "" {
		return Minted{}, fmt.Errorf("%w: a lab is minted for a question; give it as one quoted line", ErrRefused)
	}
	if strings.ContainsAny(q, "\r\n") || termsafe.Sanitize(q) != q {
		return Minted{}, fmt.Errorf("%w: the question must be one line of printable text", ErrRefused)
	}
	if len(q) > maxQuestionBytes {
		return Minted{}, fmt.Errorf("%w: the question is %d bytes; a lab answers one question of at most %d", ErrRefused, len(q), maxQuestionBytes)
	}
	if pin == "" {
		pin = "HEAD"
	}
	if strings.HasPrefix(pin, "-") {
		return Minted{}, fmt.Errorf("%w: %q is not a revision", ErrRefused, clip(pin))
	}
	s, err := resolveStore(repoRoot)
	if err != nil {
		return Minted{}, err
	}
	sha, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "--end-of-options", pin+"^{commit}")
	if err != nil || !gitutil.IsFullSHA(sha) {
		return Minted{}, fmt.Errorf("%w: %q names no commit in this repository", ErrRefused, clip(pin))
	}
	if err := s.ensure(); err != nil {
		return Minted{}, err
	}

	e := Entry{
		ID:       "lab-" + now().UTC().Format("060102150405") + "-" + sha[:7],
		RootSHA:  s.rootSHA,
		Pin:      sha,
		Created:  stamp(),
		Question: q,
	}
	sr, err := os.OpenRoot(s.dir)
	if err != nil {
		return Minted{}, fmt.Errorf("cannot open the lab store: %v", redact(err, s.home))
	}
	defer sr.Close()
	// The exclusive mkdir is the id's uniqueness: two mints of one pin in the
	// same second collide here, and the second is refused rather than merged.
	if err := sr.Mkdir(e.ID, storeDirPerm); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Minted{}, fmt.Errorf("%w: a lab %s already exists; mint again in a second", ErrRefused, e.ID)
		}
		return Minted{}, fmt.Errorf("cannot create the lab home: %v", redact(err, s.home))
	}
	dir := filepath.Join(s.dir, e.ID)
	m, err := layDown(s, sr, dir, repoRoot, e)
	if err != nil {
		// Everything under dir was made by this call: the id is fresh and the
		// mkdir was exclusive. Removing it leaves no half-minted lab behind.
		_ = os.RemoveAll(dir)
		return Minted{}, err
	}
	return m, nil
}

// layDown fills a freshly made lab home and registers it.
func layDown(s store, sr *os.Root, dir, repoRoot string, e Entry) (Minted, error) {
	lr, err := sr.OpenRoot(e.ID)
	if err != nil {
		return Minted{}, fmt.Errorf("cannot open the lab home: %v", redact(err, s.home))
	}
	defer lr.Close()

	snap := filepath.Join(dir, snapshotDir)
	if _, err := gitutil.Run(repoRoot, "clone", "--quiet", "--no-checkout", "--", repoRoot, snap); err != nil {
		return Minted{}, fmt.Errorf("cannot clone the snapshot: %s", redact(err, s.home))
	}
	if _, err := gitutil.Run(snap, "checkout", "--quiet", "--detach", e.Pin); err != nil {
		return Minted{}, fmt.Errorf("cannot detach the snapshot at the pin: %s", redact(err, s.home))
	}
	if _, err := gitutil.Run(snap, "remote", "remove", "origin"); err != nil {
		return Minted{}, fmt.Errorf("cannot cut the snapshot's remote: %s", redact(err, s.home))
	}

	for _, d := range []string{labHomeDir, binDir, stateDir, probesDir, harvestDir} {
		if err := lr.Mkdir(filepath.FromSlash(d), storeDirPerm); err != nil {
			return Minted{}, fmt.Errorf("cannot scaffold %s: %v", d, redact(err, s.home))
		}
	}
	docs := []struct{ rel, body string }{
		{intentionName, intentionDoc(e)},
		{findingsName, findingsDoc(e)},
		{correctionsName, correctionsDoc(e)},
		{amendmentsName, amendmentsDoc(e)},
	}
	written := []string{}
	for _, d := range docs {
		if err := fsutil.CreateExclusiveIn(lr, d.rel, []byte(d.body), fileMode); err != nil {
			return Minted{}, fmt.Errorf("cannot write %s: %v", d.rel, redact(err, s.home))
		}
		written = append(written, s.display(e.ID, d.rel))
	}

	line, err := json.Marshal(e)
	if err != nil {
		return Minted{}, err
	}
	if err := fsutil.AppendLineIn(sr, indexName, line, fileMode); err != nil {
		return Minted{}, fmt.Errorf("cannot register the lab: %v", redact(err, s.home))
	}
	return Minted{
		Entry:    e,
		Home:     s.display(e.ID),
		Snapshot: s.display(e.ID, snapshotDir),
		Written:  written,
		Next: []string{
			"write the hypothesis, measures and STOP conditions into INTENTION.md before anything mutates",
			"build the work binary once from the pristine snapshot into bin/abcd, and never rebuild it",
			"run the preflight: abcd lab preflight " + e.ID,
		},
	}, nil
}

// intentionDoc is the INTENTION scaffold: the pre-mutation contract the
// harvest is scored against, and the map of the lifecycle onto the lab home.
func intentionDoc(e Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nlab_id: %s\nroot_sha: %s\nsnapshot_pin: %s\ncreated: %s\n---\n\n", e.ID, e.RootSHA, e.Pin, e.Created)
	fmt.Fprintf(&b, "# INTENTION — %s\n\n", e.ID)
	b.WriteString("> Written before any mutation. The harvest is scored against it; a constraint\n")
	b.WriteString("> refined mid-lab is recorded under state/, and this file stays as it was.\n\n")
	fmt.Fprintf(&b, "## Question\n\n%s\n\n", e.Question)
	b.WriteString("## Hypothesis\n\n_What you expect, and what would show it wrong._\n\n")
	b.WriteString("## Measures\n\n_What the lab counts, and from which probe records._\n\n")
	b.WriteString("## STOP conditions\n\n_A hit condition halts the lab. It is recorded as a finding, never adapted around._\n\n")
	b.WriteString("## Sources\n\n_The records and sources this lab rests on, and the issues it must reproduce first._\n\n")
	b.WriteString("## Amendments chain read\n\n_Every earlier lab's amendments, read before this one mutates anything._\n\n")
	b.WriteString("## Lifecycle\n\n")
	b.WriteString("| Stage | Where |\n| --- | --- |\n")
	b.WriteString("| INTENTION | this file |\n")
	b.WriteString("| SNAPSHOT | `snapshot/` (a standalone clone at the pin), `bin/abcd`, `state/preflight.md` |\n")
	b.WriteString("| MUTATE | `state/probes/`, `findings.md`, `corrections.md` |\n")
	b.WriteString("| HARVEST | `harvest/harvest.md`, after `state/sweep.md` passes |\n")
	b.WriteString("| RECORD | the capture candidates the harvest lists, filed through capture |\n")
	b.WriteString("| DISCARD | archive the snapshot as a bundle and delete the live tree; the home stays |\n")
	return b.String()
}

// findingsDoc is the findings log scaffold, with the shape the harvest reads.
func findingsDoc(e Entry) string {
	return "# Findings — " + e.ID + "\n\n" +
		"Each finding is a level-two heading `## F-<n> <title>` followed by field lines:\n\n" +
		"- `kind:` product (a defect or gap in the world studied: a capture candidate),\n" +
		"  procedure (an amendment to the lab procedure), result (an answer to the\n" +
		"  question), or gate (a refusal that halted the lab)\n" +
		"- `status:` open or worked\n" +
		"- `probes:` the probe records it rests on, comma-separated\n" +
		"- `evidence:` any other lab file it rests on, comma-separated\n" +
		"- `refutation:` the source or record read to try to kill it\n\n" +
		"A finding cites at least one probe record or evidence file, or the harvest\n" +
		"refuses it.\n"
}

// correctionsDoc is the corrections scaffold the sweep reads.
func correctionsDoc(e Entry) string {
	return "# Corrections — " + e.ID + "\n\n" +
		"A retracted or corrected claim is swept, not edited: each line below names the\n" +
		"literal text that must be absent from the lab's own documents, and the sweep\n" +
		"fails while any instance remains. One per line:\n\n" +
		"    - retract: `<the literal text>` <why, optional>\n\n" +
		"The sweep reads only such lines starting at the margin, so the indented\n" +
		"example above is never a correction.\n"
}

// amendmentsDoc is the amendments scaffold: the procedure deltas this lab
// contributes to the next one.
func amendmentsDoc(e Entry) string {
	return "# Amendments — " + e.ID + "\n\n" +
		"The procedure deltas this lab contributes. Each names what forced it; the next\n" +
		"lab's intention reads every earlier lab's amendments before it mutates anything.\n"
}
