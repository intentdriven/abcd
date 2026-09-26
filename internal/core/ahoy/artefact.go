package ahoy

// artefact.go — the artefact declaration's adoption home
// (itd-2609150819432059, decision 3).
//
// A repository abcd manages declares what it ships in
// .abcd/config/artefact.json, and the launch verbs read that declaration. Its
// absence is an ahoy gap: `ahoy install` asks for the kind and writes the file,
// and a repository carrying a plugin manifest adopts kind plugin without being
// asked, so the shipped shape adopts silently. The file is validated by the one
// reader in internal/core/launch that every launch verb goes through, so a kind
// ahoy writes is a kind launch accepts.

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// ArtefactMissingGapID is the gap a repository with no declaration raises.
	ArtefactMissingGapID = "artefact.missing"
	// ArtefactInvalidGapID is the diagnostic for a declaration that is present
	// and wrong. It is the user's data, so install reports it and never
	// overwrites it.
	ArtefactInvalidGapID = "artefact.invalid"
	// artefactKindKey is the prompt key the kind is asked under.
	artefactKindKey = "artefact_kind"
	// artefactKindDefault is the answer an unanswered prompt takes: the kind
	// that assumes least about the build, gate plumbing and an empty build job.
	artefactKindDefault = string(launch.KindApplication)
	// pluginManifestRelPath is the manifest whose presence adopts kind plugin.
	pluginManifestRelPath = ".claude-plugin/plugin.json"
)

// detectArtefact reports the declaration's state as a gap: none when it is
// present and valid, artefact.missing when absent, artefact.invalid when the one
// reader refuses it.
func detectArtefact(cwd string) []Gap {
	_, err := launch.LoadArtefact(cwd)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, launch.ErrNoArtefact):
		detail := "this repository has not declared what it ships, so the launch verbs cannot choose what to preview, check and scaffold"
		if hasPluginManifest(cwd) {
			detail += "; it carries a plugin manifest, so install declares kind plugin without asking"
		}
		return []Gap{{
			ID: ArtefactMissingGapID, Category: ConfigChange, Scope: "repo",
			Title:    "artefact kind not declared (" + launch.ArtefactRelPath + ")",
			Detail:   detail,
			FixHint:  "abcd ahoy install asks for the kind (" + kindChoiceList() + ") and writes " + launch.ArtefactRelPath,
			Required: true, Resolvable: true,
		}}
	default:
		return []Gap{{
			ID: ArtefactInvalidGapID, Category: ConfigChange, Scope: "repo",
			Title:    "artefact declaration refused (" + launch.ArtefactRelPath + ")",
			Detail:   err.Error(),
			FixHint:  "repair " + launch.ArtefactRelPath + " by hand; install never overwrites a declaration it did not write",
			Required: true, Resolvable: false,
		}}
	}
}

// stepArtefact writes the declaration when the missing gap is present and
// config changes are approved: kind plugin without a question for a repository
// carrying a plugin manifest, otherwise the kind artefactKind settles.
func (a *applyCtx) stepArtefact() {
	if !a.approved[ConfigChange] || !a.has(ArtefactMissingGapID) {
		return
	}
	kind, note := a.artefactKind()
	data := launch.MarshalArtefact(launch.Artefact{Kind: kind})
	// Proved through the one reader before it is written, so ahoy can never
	// write a declaration launch refuses.
	if _, err := launch.ParseArtefact(data); err != nil {
		a.refuse("refused to write " + launch.ArtefactRelPath + ": " + err.Error())
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		a.refuse("could not write " + launch.ArtefactRelPath + ": " + errText(err))
		return
	}
	defer root.Close()
	// Contained under an os.Root opened at the repository, as the identity pin
	// is: a committed .abcd ancestor symlink is refused rather than followed.
	if err := fsutil.WriteFileAtomicInRoot(root, launch.ArtefactRelPath, data, 0o644); err != nil {
		a.refuse("could not write " + launch.ArtefactRelPath + ": " + errText(err))
		return
	}
	a.note(launch.ArtefactRelPath)
	if note != "" {
		a.refuse(note)
	}
}

// artefactKind settles the kind to declare, and the note that says so when abcd
// chose rather than the operator. It follows the house-style question's rule
// (emDashSeverity): an unattended --yes install is not asked, and an answer
// naming no kind — the "y" a `yes |` pipe sends to every question — is not
// refused, because withholding the declaration would leave every launch verb
// refusing the repository. Both take the default, application, the kind that
// assumes least about the build, and the note says what was heard and where the
// kind lives. A bare Enter or end of input takes the default the question
// displays, which is an answer and draws no note.
func (a *applyCtx) artefactKind() (launch.ArtefactKind, string) {
	if hasPluginManifest(a.cwd) {
		return launch.KindPlugin, ""
	}
	fix := "; edit " + launch.ArtefactRelPath + " to declare another (" + kindChoiceList() + ")"
	if a.autoYes {
		return launch.ArtefactKind(artefactKindDefault), "the artefact kind was not asked (an unattended --yes install): " +
			artefactKindDefault + " is declared" + fix
	}
	choices := make([]string, len(launch.ArtefactKinds))
	for i, k := range launch.ArtefactKinds {
		choices[i] = string(k)
	}
	answer := strings.TrimSpace(a.prompter.Prompt(artefactKindKey, choices, artefactKindDefault))
	switch {
	case launch.ValidArtefactKind(answer):
		return launch.ArtefactKind(answer), ""
	case answer == "":
		return launch.ArtefactKind(artefactKindDefault), ""
	}
	return launch.ArtefactKind(artefactKindDefault), "the answer " + termsafe.Sanitize(strconv.Quote(answer)) + " to " +
		artefactKindKey + " names none of " + kindChoiceList() + ", so " + artefactKindDefault + " is declared" + fix
}

// hasPluginManifest reports whether the repository carries a plugin manifest.
func hasPluginManifest(cwd string) bool {
	info, err := os.Lstat(filepath.Join(cwd, filepath.FromSlash(pluginManifestRelPath)))
	return err == nil && info.Mode().IsRegular()
}

func kindChoiceList() string {
	names := make([]string, len(launch.ArtefactKinds))
	for i, k := range launch.ArtefactKinds {
		names[i] = string(k)
	}
	return strings.Join(names, ", ")
}
