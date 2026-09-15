package reading

import (
	"fmt"
	"os"
	"path/filepath"
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
	// StagedRuns is what an ASSEMBLY parked. It is not filtered by whether the
	// run was ingested: nothing removes an assembly's directory afterwards, so a
	// committed run and an unread one appear alike (iss captured separately).
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
	defs, err := LoadDefinitions(repoRoot)
	if err != nil {
		return Status{}, err
	}
	for _, d := range defs {
		s.Definitions = append(s.Definitions, definitionPrefix+string(d.Position))
	}
	sort.Strings(s.Definitions)

	runs, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(DefaultRunDir)))
	if err != nil && !os.IsNotExist(err) {
		return Status{}, fmt.Errorf("reading: listing the staged runs: %w", err)
	}
	for _, e := range runs {
		if e.IsDir() && strings.HasPrefix(e.Name(), RunIDFamily+"-") {
			s.StagedRuns = append(s.StagedRuns, e.Name())
		}
	}
	sort.Strings(s.StagedRuns)

	// A stage directory named by a run id is left in one of two states, and the
	// commit marker is what tells them apart — the same probe the sweep's
	// rollbackRun makes. No marker: the ingest never finished, the run never
	// happened, and its records will be rolled back. Marker present: the run
	// committed and only the stage failed to clear, so the records stay and
	// only the stage goes. Calling both an orphan would tell an operator that a
	// committed run's records are about to be deleted.
	stages, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(IngestStageDir)))
	if err != nil && !os.IsNotExist(err) {
		return Status{}, fmt.Errorf("reading: listing the ingest stage: %w", err)
	}
	for _, e := range stages {
		if !e.IsDir() || !recordid.ValidReadingRunID(e.Name()) {
			continue
		}
		marker := filepath.Join(repoRoot, filepath.FromSlash(ReadingsRecordDir), e.Name(), RunFileName)
		switch _, err := os.Lstat(marker); {
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
