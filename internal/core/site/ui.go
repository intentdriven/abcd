package site

// `site-src/ui.json` — the complete list of words the generator may add.
//
// adr-47 decision 2 says every sentence the site renders is a span of a
// repository file, and that the only words the generator may add are the
// interface strings in this file plus numbers, dates, file names and asset
// names. That makes ui.json a CLOSED allowlist, and the way it is kept closed is
// this struct: the file is decoded with unknown fields refused, so a string
// added to the JSON that no field here reads fails the build. Adding a word to
// the site therefore costs a code change and a review, which is the price the
// single-source rule is meant to charge.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxUIBytes bounds the allowlist read.
const maxUIBytes = 64 * 1024

// ErrUIInvalid is returned for an unusable interface-string file.
var ErrUIInvalid = errors.New("site: interface strings are invalid")

// UI is the interface-string allowlist.
type UI struct {
	Purpose       string `json:"_purpose"`
	NavStory      string `json:"nav_story"`
	NavInstall    string `json:"nav_install"`
	NavDocs       string `json:"nav_docs"`
	NavRecord     string `json:"nav_record"`
	NavReferences string `json:"nav_references"`
	CTARoles      string `json:"cta_roles"`
	CTAInstall    string `json:"cta_install"`
	CTADocs       string `json:"cta_docs"`
	RecordLink    string `json:"record_link"`
	FromTheRecord string `json:"from_the_record"`
	LatestRelease string `json:"latest_release"`
	AllReleases   string `json:"all_releases"`
	Copy          string `json:"copy"`
	Copied        string `json:"copied"`
	SearchDocs    string `json:"search_docs"`
	// ForgeNames names each forge host the header and footer links may label —
	// "GitHub" for github.com. A host the map does not name keeps the owner/name
	// handle, so the generic explorer never assumes a forge. The map may be
	// empty; the missing-string check walks only declared strings.
	ForgeNames map[string]string `json:"forge_names"`
	Platform   Platform          `json:"platform"`
	Tiles      Tiles             `json:"tiles"`
	// RecordNav labels the explorer's sub-navigation; each label is also that
	// page's own heading, so a page and the tab that reaches it cannot drift.
	RecordNav RecordNav `json:"record_nav"`
	// Panels captions the dashboard's panels that no count names.
	Panels Panels `json:"panels"`
	// Graph labels the relationship chart's controls.
	Graph GraphUI `json:"graph"`
	// Record labels the per-record page's own sections.
	Record RecordUI `json:"record"`
	// Relations names a typed link read from the far end. The forward name is
	// the relation's own word from the record; only the inverse needs saying.
	Relations Relations `json:"relations"`
	// Contributors labels the attribution page's two rows and two figures.
	Contributors ContributorsUI `json:"contributors"`
	// Health labels each family of finding the record is checked for.
	Health        HealthUI `json:"health"`
	More          string   `json:"more"`
	Standby       string   `json:"standby"`
	CLIGroup      string   `json:"cli_group"`
	MatchesSystem string   `json:"matches_system"`
	// ReadScript labels the link beside the install command that opens the
	// script the command runs. It is an invitation to read before running, so it
	// says what the reader would do, not what the file is.
	ReadScript string `json:"read_script"`
	// Unreleased stands where the version goes on a build of an untagged tree.
	// It is a word rather than a blank because a stamp reading "· · abcdef1"
	// looks like a rendering fault, and one reading "v0.6.1" would be a lie.
	Unreleased string `json:"unreleased"`
	Beta       string `json:"beta"`
}

// RecordNav labels the explorer's sub-navigation.
type RecordNav struct {
	Dashboard   string `json:"dashboard"`
	Graph       string `json:"graph"`
	Timeline    string `json:"timeline"`
	Foundations string `json:"foundations"`
	// Development names the deck of the stores that MOVE — intents, specs and
	// issues — as Foundations names the ones that hold.
	Development string `json:"development"`
	// Health names the page that collects every finding the record can be
	// checked against itself for.
	Health string `json:"health"`
	// Glossary names the page set that publishes the record's own terms — the
	// one place a word the record uses in more than one sense is pinned to a
	// spelling per sense.
	Glossary     string `json:"glossary"`
	Contributors string `json:"contributors"`
}

// Panels captions the dashboard panels.
type Panels struct {
	Latest string `json:"latest"`
	Health string `json:"health"`
	// Unresolved, Baseline and Isolated label the health summary's three
	// numbers; unlabelled they read as a rendering fault.
	Unresolved string `json:"unresolved"`
	Baseline   string `json:"baseline"`
	Isolated   string `json:"isolated"`
}

// GraphUI labels the relationship chart's controls.
type GraphUI struct {
	Arrange        string `json:"arrange"`
	ByDate         string `json:"by_date"`
	ByLinks        string `json:"by_links"`
	Filters        string `json:"filters"`
	Find           string `json:"find"`
	Mentions       string `json:"mentions"`
	BrowseList     string `json:"browse_list"`
	ZoomIn         string `json:"zoom_in"`
	ZoomOut        string `json:"zoom_out"`
	ResetView      string `json:"reset_view"`
	FullScreen     string `json:"full_screen"`
	ExitFullScreen string `json:"exit_full_screen"`
	Close          string `json:"close"`
	Back           string `json:"back"`
	Forward        string `json:"forward"`
	History        string `json:"history"`
	Linked         string `json:"linked"`
	NoLinks        string `json:"no_links"`
}

// RecordUI labels a per-record page's sections.
type RecordUI struct {
	Frontmatter   string `json:"frontmatter"`
	Inbound       string `json:"inbound"`
	Outbound      string `json:"outbound"`
	Mentions      string `json:"mentions"`
	NotInTree     string `json:"not_in_tree"`
	OpenOnForge   string `json:"open_on_forge"`
	CommitHistory string `json:"commit_history"`
}

// Relations names each directed link read from its target's side.
type Relations struct {
	BlockedBy  string `json:"blocked_by"`
	Supersedes string `json:"supersedes"`
	Implements string `json:"implements"`
	BuildsOn   string `json:"builds_on"`
}

// HealthUI labels the health page's finding families. Every one of them is a
// check the record can be run against ITSELF — nothing here is a judgement, an
// opinion, or a number a human has to interpret before acting on it.
type HealthUI struct {
	// Unresolved is a typed reference whose target no file answers to.
	Unresolved string `json:"unresolved"`
	// Isolated is a record nothing links to and which links to nothing.
	Isolated string `json:"isolated"`
	// SameAuthor is a candidate duplicate identity: two author names that the
	// mailmap has not folded but which the evidence says are one person.
	SameAuthor string `json:"same_author"`
	// Undeclared is an authored commit carrying no `Assisted-by:` trailer.
	Undeclared string `json:"undeclared"`
	// MultiTrailer is a commit declaring more than one assisting model — not a
	// fault, but the reason a trailer tally and a commit count differ.
	MultiTrailer string `json:"multi_trailer"`
	// NotADefect is what the multi-trailer panel says about itself. The family
	// is the one on the page that reports no fault at all, and a panel sitting
	// among findings without saying so is read as a fifth finding.
	NotADefect string `json:"not_a_defect"`
	// SupersedesLead says what a supersession row means, and that the family
	// reports no fault: the left record replaced the right one.
	SupersedesLead string `json:"supersedes_lead"`
	// Clean is what the page says when a family has nothing to report.
	Clean string `json:"clean"`
	// Suggestion prefixes the line a finding proposes a human confirm.
	Suggestion string `json:"suggestion"`
}

// ContributorsUI labels the attribution page.
type ContributorsUI struct {
	Authors  string `json:"authors"`
	Tools    string `json:"tools"`
	Assisted string `json:"assisted"`
	Trailers string `json:"trailers"`
	// MergesExcluded names what the disclosure rate leaves out, so the
	// denominator is never a silent choice.
	MergesExcluded string `json:"merges_excluded"`
	// DeclaredNone and Undeclared label the two commit-level facts stated
	// beneath the occurrence chart rather than drawn inside it.
	DeclaredNone string `json:"declared_none"`
	Undeclared   string `json:"undeclared"`
}

// Inverse names a directed relation read from the record it points at. An
// undirected relation reads the same from both ends and is returned unchanged.
func (r Relations) Inverse(rel string) string {
	switch rel {
	case "blocked_by":
		return r.BlockedBy
	case "supersedes":
		return r.Supersedes
	case "implements":
		return r.Implements
	case "builds_on":
		return r.BuildsOn
	}
	return relationWord(rel)
}

// relationWord is a relation's own name as prose: the record's field spelling
// with its underscore opened out. It is derived rather than declared, so a new
// relation never needs a word written for it.
func relationWord(rel string) string { return strings.ReplaceAll(rel, "_", " ") }

// Platform names each released binary in the reader's own words.
type Platform struct {
	DarwinARM64 string `json:"darwin-arm64"`
	DarwinAMD64 string `json:"darwin-amd64"`
	LinuxARM64  string `json:"linux-arm64"`
	LinuxAMD64  string `json:"linux-amd64"`
}

// Tiles captions the record dashboard's counts.
type Tiles struct {
	Releases   string `json:"releases"`
	ADR        string `json:"adr"`
	Intent     string `json:"intent"`
	Spec       string `json:"spec"`
	Issue      string `json:"issue"`
	Principle  string `json:"principle"`
	Discipline string `json:"discipline"`
	Commits    string `json:"commits"`
}

// ForType captions one record store's count. A store with no caption is named
// by the store itself, which is a fact about the record rather than a word the
// generator wrote.
func (t Tiles) ForType(typ string) string {
	switch typ {
	case "adr":
		return t.ADR
	case "intent":
		return t.Intent
	case "spec":
		return t.Spec
	case "issue":
		return t.Issue
	case "principle":
		return t.Principle
	}
	return typ
}

// LoadUI reads the interface-string allowlist named by the manifest.
func LoadUI(repoRoot, rel string) (UI, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return UI{}, err
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxUIBytes)
	if err != nil {
		return UI{}, err
	}
	var ui UI
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ui); err != nil {
		return UI{}, fmt.Errorf("%w: %s: %v", ErrUIInvalid, rel, err)
	}
	if missing := ui.missing(); len(missing) > 0 {
		return UI{}, fmt.Errorf("%w: %s: no text for %s — every interface string the site renders is declared here",
			ErrUIInvalid, rel, strings.Join(missing, ", "))
	}
	return ui, nil
}

// missing names every interface string the file leaves empty. An empty string
// renders as a blank button or an unlabelled tab, which reads as a rendering
// bug rather than as the missing declaration it is.
func (ui UI) missing() []string {
	var out []string
	var walk func(v reflect.Value, t reflect.Type)
	walk = func(v reflect.Value, t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			switch f.Type.Kind() {
			case reflect.Struct:
				walk(v.Field(i), f.Type)
			case reflect.Map:
				// An ABSENT key is a declaration the repository chose not to
				// make, and the renderer degrades by absence. A key declared
				// BLANK is a mistake of the same kind an empty field is, and
				// the walk names it rather than letting it read as a choice.
				iter := v.Field(i).MapRange()
				for iter.Next() {
					if iter.Value().Kind() == reflect.String && strings.TrimSpace(iter.Value().String()) == "" {
						out = append(out, name+"."+iter.Key().String())
					}
				}
			case reflect.String:
				// `_purpose` documents the file for its human readers and is
				// never rendered, so it is the one field with nothing to say.
				if name != "_purpose" && strings.TrimSpace(v.Field(i).String()) == "" {
					out = append(out, name)
				}
			}
		}
	}
	walk(reflect.ValueOf(ui), reflect.TypeOf(ui))
	return out
}
