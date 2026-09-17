package history

// The per-repository history configuration.
//
// One file, `.abcd/config/history.json`, holding the three things a repository
// can say about ingesting transcripts it did not capture: where to look, which
// harness project directories it claims when the repository a transcript
// recorded no longer exists, and what to do about the ones it does not claim.
//
// It is read under the same discipline as the scanner's per-repo override: a
// size cap, a refusal of anything that is not a regular file, and containment
// inside the repository through os.Root — because the file sits two directories
// down, where O_NOFOLLOW on the leaf alone would still follow a symlinked
// `.abcd` or `.abcd/config` out of the tree.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ConfigRelPath is the per-repo history configuration, relative to the repo root.
const ConfigRelPath = ".abcd/config/history.json"

// maxHistoryConfigBytes caps the configuration read. It holds a schema version
// and three short string lists; anything larger is not this file.
const maxHistoryConfigBytes = 64 << 10

// The accepted on_orphan policies. Neither of them lets core prompt: `prompt`
// means core still ingests nothing and returns the orphan list, and the front
// door is what asks a human and re-invokes with the chosen names. An
// interactive question is a transport concern.
const (
	OnOrphanIgnore = "ignore"
	OnOrphanPrompt = "prompt"
)

// Config is `.abcd/config/history.json`.
type Config struct {
	SchemaVersion int `json:"schema_version"`
	// IngestRoots are the directories `history ingest` walks when the operator
	// names no source. This is the ONLY place a transcript-source path lives: a
	// vendor's directory layout baked into code would be the on-disk dependency
	// this design exists to avoid.
	IngestRoots []string `json:"ingest_roots"`
	// AdoptProjects lists the harness project directory names this repository
	// claims. A transcript under a claimed name is ingested here, under THIS
	// repository's redaction configuration, when its own repository can no
	// longer be found on disk.
	AdoptProjects []string `json:"adopt_projects"`
	// OnOrphan is ignore (the default) or prompt.
	OnOrphan string `json:"on_orphan"`
}

// DefaultConfig is what a repository with no configuration file gets: no
// declared roots, no claimed projects, and orphans ignored and reported.
func DefaultConfig() Config {
	return Config{SchemaVersion: 1, OnOrphan: OnOrphanIgnore}
}

// LoadConfig reads a repository's history configuration.
//
// An absent file is not an error — it is the default. Everything else is: a
// configuration that exists but cannot be read is a statement the operator made
// and this run cannot honour, and quietly falling back to the default would
// ingest under a policy nobody chose.
func LoadConfig(repoRoot string) (Config, error) {
	cfg := DefaultConfig()
	if repoRoot == "" {
		return cfg, errors.New("history: config needs a repository root")
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return cfg, fmt.Errorf("history: cannot open %s for contained reads: %w", ConfigRelPath, err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, ConfigRelPath, maxHistoryConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		switch {
		case errors.Is(err, fsutil.ErrNotRegular):
			return cfg, fmt.Errorf("history: %s is not a regular file (a symlinked leaf is refused)", ConfigRelPath)
		case errors.Is(err, fsutil.ErrTooBig):
			return cfg, fmt.Errorf("history: %s exceeds the %d-byte cap", ConfigRelPath, maxHistoryConfigBytes)
		default:
			return cfg, fmt.Errorf("history: %s is unreadable: %w", ConfigRelPath, err)
		}
	}
	var onDisk Config
	if err := json.Unmarshal(data, &onDisk); err != nil {
		return cfg, fmt.Errorf("history: %s is not valid JSON: %w", ConfigRelPath, err)
	}
	if onDisk.OnOrphan == "" {
		onDisk.OnOrphan = OnOrphanIgnore
	}
	if onDisk.OnOrphan != OnOrphanIgnore && onDisk.OnOrphan != OnOrphanPrompt {
		return cfg, fmt.Errorf("history: %s on_orphan is %q, which is not one of %s, %s",
			ConfigRelPath, onDisk.OnOrphan, OnOrphanIgnore, OnOrphanPrompt)
	}
	if onDisk.SchemaVersion == 0 {
		onDisk.SchemaVersion = 1
	}
	return onDisk, nil
}
