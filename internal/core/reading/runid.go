package reading

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// RunIDFamily is the readings family's id prefix. It satisfies the mint's
// ^[a-z]+$ bound, and the mint reads no maximum, so two checkouts assembling in
// the same window cannot converge on one run id (adr-45).
const RunIDFamily = "rdg"

// minter is the package's record-id mint. The zero value is the production
// configuration; tests swap it for an injected clock and entropy.
var minter recordid.Minter

// mintRunID allocates a run identifier of the form rdg-<yymmddHHMMSS><rrrr>.
func mintRunID() (string, error) { return minter.Mint(RunIDFamily) }

// runIDDraws bounds the redraws freeRunID makes before it refuses. The suffix is
// uniform over ten thousand values, so a second collision in one second is a
// one-in-ten-thousand event on top of the first; eight draws is a bound the
// refusal can be reached by in a test and never by chance.
const runIDDraws = 8

// freeRunID mints a run identifier whose default directory under repoRoot does
// not exist yet. The mint reads no maximum (adr-45), so two assemblies in one
// second can draw one suffix, and the id names the run's directory: without
// this, the second run met the empty-directory refusal that guards an
// operator's occupied directory, though nothing about the run was wrong. The
// coincidence is absorbed here, the one site that can tell a collision from an
// occupied directory, by redrawing while the drawn directory exists. Entropy
// that never leaves a taken suffix is refused after runIDDraws, naming the id.
func freeRunID(repoRoot string) (string, error) {
	var last string
	for range runIDDraws {
		id, err := mintRunID()
		if err != nil {
			return "", err
		}
		_, err = os.Stat(filepath.Join(repoRoot, filepath.FromSlash(DefaultRunDir), id))
		if os.IsNotExist(err) {
			return id, nil
		}
		if err != nil {
			return "", fmt.Errorf("reading: checking the run directory for %s: %w", id, err)
		}
		last = id
	}
	return "", fmt.Errorf("reading: %d draws in a row named a run directory that already exists (last %s); "+
		"the default run directory is not free, so name one with --out or clear %s", runIDDraws, last, DefaultRunDir)
}
