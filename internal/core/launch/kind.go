package launch

// kind.go — what the declared artefact kind changes about a launch run
// (itd-2609150819432059). The gate's inputs — the ledger, the anchor tag, the
// version location — are kind-independent; only the reads that bind a run to a
// plugin are chosen here: the bundle a run scans, the lockstep table, and the
// rows that judge a plugin payload.

import (
	"errors"
	"os"
	"path/filepath"
)

// PayloadTreeDescription names the tree a plugin's preview scans.
const PayloadTreeDescription = "the plugin payload (the include set in .abcd/config/launch-payload.json)"

// ErrNotAPlugin reports a plugin-only operation — staging or archiving a plugin
// payload — asked of a repository that declares another artefact kind.
var ErrNotAPlugin = errors.New("the repository does not declare a plugin")

// LoadArtefactOrPlugin reads the declaration for the verbs that predate it and
// stay lenient about its absence: an absent file is the plugin shape they have
// always assumed, while a present file is held to the one reader like anywhere
// else, so an unknown kind refuses every verb.
func LoadArtefactOrPlugin(repoRoot string) (Artefact, error) {
	art, err := LoadArtefact(repoRoot)
	if errors.Is(err, ErrNoArtefact) {
		return Artefact{Kind: KindPlugin}, nil
	}
	return art, err
}

// kindBundle resolves the bundle a declared kind previews and names the tree it
// is. A plugin resolves its payload include set. Any other kind resolves that
// same set when it declares one, and otherwise the tree the release tag would
// archive (decision 8): an empty include set is not a refusal for a kind that
// ships no plugin payload.
func kindBundle(repoRoot string, art Artefact) (Bundle, string, error) {
	if !art.IsPlugin() {
		if _, err := os.Lstat(filepath.Join(repoRoot, includeConfigRelPath)); os.IsNotExist(err) {
			b, err := ResolveArchiveBundle(repoRoot)
			return b, ArchiveTreeDescription, err
		}
	}
	b, err := ResolveBundle(repoRoot, nil)
	return b, PayloadTreeDescription, err
}

// kindGatePolicy reads the suite's policy. A non-plugin kind with no payload
// include config has nowhere to configure the suite, and runs it at the default.
func kindGatePolicy(repoRoot string, art Artefact) (GatePolicy, error) {
	policy, err := LoadGatePolicy(repoRoot)
	if err != nil && !art.IsPlugin() && errors.Is(err, ErrNoLaunchPayload) {
		return GatePolicy{}, nil
	}
	return policy, err
}

// kindLockstep checks the source tree's lockstep for the declared kind: the
// pinned plugin-manifest table for a plugin, the declared list for any other.
func kindLockstep(tree LockstepTree, repoRoot string, art Artefact) LockstepResult {
	vl := filepath.Join(repoRoot, versionLocationRelPath)
	if art.IsPlugin() {
		return CheckLockstep(tree, repoRoot, vl)
	}
	return CheckDeclaredLockstep(tree, repoRoot, vl, art.Lockstep)
}

// pluginOnlyRow is the row a plugin-only gate reports for another kind: not
// armed, and why, so a reader sees the row was considered rather than missed.
func pluginOnlyRow(name string, kind ArtefactKind) GateSummary {
	return GateSummary{Name: name, Status: "not_armed",
		Detail: "the declared artefact kind is " + string(kind) + " (" + ArtefactRelPath + "), and this row judges a plugin payload"}
}
