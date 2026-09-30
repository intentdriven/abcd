package reading

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// DefinitionsDir is where the four reading definitions live. The assembler never
// reads one — a definition is the reader's, and this package denies the whole
// directory to every assembly — but the status render reports which are present,
// because a missing definition is the difference between an instrument that can
// be dispatched and one that cannot.
const DefinitionsDir = "agents"

// definitionPrefix is the filename prefix a cold-reading definition carries.
const definitionPrefix = "cold-reading-"

// Status is the read-only render behind the bare verb: what this assembler is,
// what it would admit, which definitions are present, which runs an assembly
// has parked in the local tier, and which ingests were interrupted.
type Status struct {
	AssemblerVersion string     `json:"assembler_version"`
	SchemaVersion    int        `json:"schema_version"`
	CharterPath      string     `json:"charter_path"`
	Positions        []Position `json:"positions"`
	IncludeRows      int        `json:"include_rows"`
	ExclusionRows    int        `json:"exclusion_rows"`
	Definitions      []string   `json:"definitions"`
	// StagedRuns is what an ASSEMBLY parked and no ingest has yet given an
	// outcome: the runs still awaiting a reading. Nothing removes an assembly's
	// directory after its run is ingested, so the parking area alone lists every
	// run ever assembled, committed and refused alike; a parked run whose id
	// already has a commit marker or a refusal record is therefore left out, by
	// the probe the rerun refusal makes (runOutcome, iss-2608311621412224).
	StagedRuns []string `json:"staged_runs"`
	// OrphanedIngests names the runs whose ingest reached the ledger and never
	// reached its commit marker.
	//
	// It is reported because the ingest verb's sweep rides with the COMMIT: an
	// orphan therefore survives until the next invocation that validates, and
	// until then its reading records sit in the committed ledger for a run that
	// never happened. That is a state an operator has to be able to see, and no
	// other surface shows it — StagedRuns reads the assembly parking area, which
	// is a different directory.
	OrphanedIngests []string `json:"orphaned_ingests"`
	// LeftoverStages names the runs whose stage is still present although their
	// commit marker landed: the commit path's RemoveAll failed after run.json
	// was written. Those runs are complete. The next ingest that validates
	// clears the stage and leaves the records alone — the sweep probes the same
	// marker (rollbackRun) — so reporting one as an orphan would promise a
	// rollback that never happens (iss-2609012043437282).
	LeftoverStages []string `json:"leftover_stages"`
}

// Describe reports the assembler's state over a repository. It writes nothing.
func Describe(repoRoot string) (Status, error) {
	s := Status{
		AssemblerVersion: AssemblerVersion(),
		SchemaVersion:    SchemaVersion,
		CharterPath:      CharterPath,
		Positions:        Positions(),
		IncludeRows:      len(Table),
		ExclusionRows:    len(Exclusions),
		Definitions:      []string{},
		StagedRuns:       []string{},
		OrphanedIngests:  []string{},
		LeftoverStages:   []string{},
	}
	if repoRoot == "" {
		return s, nil
	}

	// Resolved, not listed. The render reports the definitions the ingest verb
	// would actually resolve: a file the locator refuses — silent about its
	// position or its regime — is a fault reported here rather than an
	// instrument reported present, and a `cold-reading-*.md` naming no position
	// is not an instrument at all, because the position set is closed.
	defs, err := loadDefinitions(repoRoot)
	if err != nil {
		return Status{}, err
	}
	for _, d := range defs {
		s.Definitions = append(s.Definitions, definitionPrefix+string(d.Position))
	}
	sort.Strings(s.Definitions)

	// Every read of the local tier and every probe of the durable tier goes
	// through ONE root over the repository. The listings are included: a
	// parking area or a stage reached through a link out of the checkout would
	// otherwise have its run-id-shaped names echoed into the render, while the
	// sweep that deletes from the same stage lists it through the root and
	// refuses a linked directory (readDirIn). A parked run and a stage agree on
	// a symlink too: a record directory that escapes the checkout refuses the
	// render for both, rather than refusing it for one and classifying the other
	// by a marker read outside the repository (iss-2609261905354450,
	// iss-2609012043432648).
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Status{}, fmt.Errorf("reading: opening the repository to probe the staged runs: %w", err)
	}
	defer root.Close()
	runs, err := readDirIn(root, DefaultRunDir)
	if err != nil && !os.IsNotExist(err) {
		return Status{}, fmt.Errorf("reading: listing the staged runs: %w", err)
	}
	stages, err := readDirIn(root, IngestStageDir)
	if err != nil && !os.IsNotExist(err) {
		return Status{}, fmt.Errorf("reading: listing the ingest stage: %w", err)
	}
	if len(runs) == 0 && len(stages) == 0 {
		return s, nil
	}

	if s.StagedRuns, err = awaitingOutcome(root, runs); err != nil {
		return Status{}, err
	}

	// A stage directory named by a run id is left in one of two states, and the
	// commit marker is what tells them apart — the same probe the sweep's
	// rollbackRun makes. No marker: the ingest never finished, the run never
	// happened, and its records will be rolled back. Marker present: the run
	// committed and only the stage failed to clear, so the records stay and
	// only the stage goes. Calling both an orphan would tell an operator that a
	// committed run's records are about to be deleted.
	for _, e := range stages {
		if !e.IsDir() || !recordid.ValidReadingRunID(e.Name()) {
			continue
		}
		switch _, err := root.Lstat(ReadingsRecordDir + "/" + e.Name() + "/" + RunFileName); {
		case err == nil:
			s.LeftoverStages = append(s.LeftoverStages, e.Name())
		case os.IsNotExist(err):
			s.OrphanedIngests = append(s.OrphanedIngests, e.Name())
		default:
			return Status{}, fmt.Errorf("reading: probing the commit marker of run %s: %w", e.Name(), err)
		}
	}
	sort.Strings(s.OrphanedIngests)
	sort.Strings(s.LeftoverStages)
	return s, nil
}

// awaitingOutcome returns the parked runs no ingest has given an outcome, sorted.
//
// The parking directory is kept after an ingest — it is the run's local
// evidence, and removing it is not this read-only render's to do — so what
// tells an outstanding run from an ingested one is the record: an ingested run
// has a commit marker or a refusal record under its id in the durable tier, the
// same probe refuseARerun makes before an ingest writes.
func awaitingOutcome(root *os.Root, parked []os.DirEntry) ([]string, error) {
	out := []string{}
	for _, e := range parked {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), RunIDFamily+"-") {
			continue
		}
		// A parked name the run-id shape refuses cannot have an outcome, since
		// ingest refuses to name a record directory after it; it stays listed,
		// as everything parked did, rather than be probed as a path.
		if recordid.ValidReadingRunID(e.Name()) {
			rel, err := runOutcome(root, e.Name())
			if err != nil {
				return nil, fmt.Errorf("reading: probing the outcome of staged run %s: %w", e.Name(), err)
			}
			if rel != "" {
				continue
			}
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}
