package reading

import "github.com/intentdriven/abcd/internal/core/recordid"

// RunIDFamily is the readings family's id prefix. It satisfies the mint's
// ^[a-z]+$ bound, and the mint reads no maximum, so two checkouts assembling in
// the same window cannot converge on one run id (adr-45).
const RunIDFamily = "rdg"

// minter is the package's record-id mint. The zero value is the production
// configuration; tests swap it for an injected clock and entropy.
var minter recordid.Minter

// mintRunID allocates a run identifier of the form rdg-<yymmddHHMMSS><rrrr>.
func mintRunID() (string, error) { return minter.Mint(RunIDFamily) }
