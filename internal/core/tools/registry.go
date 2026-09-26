// Package tools is abcd's explain-then-install mode (itd-63): the curated
// registry of the external tools abcd knows, the plain-language explanation a
// verb gives when it finds one missing, and the one place a confirmed install
// step is run.
//
// It is a MODE other verbs call, never a surface of its own (the product
// thinker's decision 3, 2026-09-21): a verb that finds a tool missing asks
// Explain for what to say and, where it offers the install, hands Install a
// Confirm function its front door supplies. The package has no transport
// knowledge — it never reads a terminal and never writes to stdout — so the CLI
// asks on the terminal, the plugin page asks through the host's question tool,
// and this code is the same under both (criterion 5).
//
// The trust boundary is Install. What it runs is fixed data in this file,
// compiled into the binary: an argv per platform, never composed from a flag, a
// repository file or an environment value, never a shell string. It runs only
// after the caller's Confirm returns yes, never in CI, and never a program that
// resolves inside the repository it was asked from. See install.go.
package tools

import "sort"

// Requirement is whether a capability needs a tool or merely works better with
// it (adr-22: most dependencies are optional adapters over a native default).
type Requirement string

const (
	// Optional: the capability runs on its native default without the tool.
	Optional Requirement = "optional"
	// Required: the capability cannot run without the tool.
	Required Requirement = "required"
)

// Capability names what a verb is doing when it finds a tool missing. The same
// tool is optional for one capability and required for another, so an
// explanation is always for a (tool, capability) pair.
type Capability string

const (
	// TranscriptScan is the secret scan of captured session transcripts in a
	// repository that has NOT armed gitleaks: the native scanner covers it.
	TranscriptScan Capability = "transcript-scan"
	// TranscriptScanArmed is the same scan in a repository that armed gitleaks
	// in .abcd/config/gitleaks.json: the history store refuses to store a
	// transcript with less coverage than the repository asked for.
	TranscriptScanArmed Capability = "transcript-scan-armed"
	// RemoteSettings is `ahoy remote`: reading and changing the repository's
	// GitHub secret-scanning settings, which abcd does only through gh.
	RemoteSettings Capability = "remote-settings"
)

// Use is what one capability does with a tool, in the words a person reads.
type Use struct {
	// Capability is the capability's human name.
	Capability string
	// Requirement is optional or required, for this capability.
	Requirement Requirement
	// Does is what abcd uses the tool for here.
	Does string
	// WithoutIt is what works without the tool (for an optional use, the
	// native default; for a required one, what fails and the way back).
	WithoutIt string
	// NativeDefault is what the capability continues on after a no; empty when
	// there is none.
	NativeDefault string
	// OnDecline is the loud line's tail after a no or a failed install:
	// "continuing on <native default>" for an optional use, the standing
	// refusal for a required one.
	OnDecline string
}

// Step is one platform's install step: the package manager it uses, by name,
// and the exact argv run. Argv[0] is a bare program name resolved on PATH at
// run time; the rest are literal arguments.
type Step struct {
	Manager string
	Argv    []string
}

// Tool is one registry entry.
type Tool struct {
	Name     string
	What     string // what the tool is, plainly
	Homepage string // where to read what it is
	Uses     map[Capability]Use
	Install  map[string]Step // keyed by runtime.GOOS
	// Effects is what the install does to the machine, and what it does not.
	Effects string
	// Verify is the argv that proves the tool runs once installed.
	Verify []string
}

// homebrew builds the Homebrew step for a formula. Homebrew is the one package
// manager with a single fixed command on both supported platforms (macOS, and
// Linux through Homebrew on Linux); a machine without it is told so, never
// offered a step composed for some other manager.
func homebrew(formula string) Step {
	return Step{Manager: "Homebrew", Argv: []string{"brew", "install", formula}}
}

// registry is the curated set (the product thinker's decision 2). An entry is
// added by a change to this file, never at run time: a gap met at run time is
// named as abcd's own and captured, not composed (explain.go).
var registry = map[string]Tool{
	"gitleaks": {
		Name: "gitleaks",
		What: "an open-source secret scanner: it reads text and reports the credentials it recognises " +
			"(API keys, access tokens, private keys)",
		Homepage: "https://github.com/gitleaks/gitleaks",
		Uses: map[Capability]Use{
			TranscriptScan: {
				Capability:  "the secret scan of captured session transcripts",
				Requirement: Optional,
				Does: "when a repository opts in with .abcd/config/gitleaks.json, abcd runs gitleaks over each " +
					"transcript before storing it and masks what it finds, on top of the native scanner; it reaches " +
					"labelled, high-entropy keys in prose that the native patterns miss",
				WithoutIt: "the native secret scanner built into abcd still scans and masks every transcript and " +
					"every launch payload",
				NativeDefault: "the native secret scanner",
				OnDecline:     "continuing on the native secret scanner",
			},
			TranscriptScanArmed: {
				Capability:  "the secret scan of captured session transcripts, which this repository armed in .abcd/config/gitleaks.json",
				Requirement: Required,
				Does: "abcd runs gitleaks over each transcript before storing it and masks what it finds, on top of " +
					"the native scanner, because this repository asked for that coverage",
				WithoutIt: "nothing is stored: abcd refuses to store this repository's transcripts with less coverage " +
					"than the repository asked for; setting enabled to false in .abcd/config/gitleaks.json returns it " +
					"to the native secret scanner",
				OnDecline: "transcript capture stays refused for this repository until gitleaks is installed or " +
					".abcd/config/gitleaks.json sets enabled to false",
			},
		},
		Install: map[string]Step{"darwin": homebrew("gitleaks"), "linux": homebrew("gitleaks")},
		Effects: "Homebrew downloads the gitleaks program and puts it on your PATH. It changes no repository, " +
			"starts no background service, and sends nothing anywhere; abcd runs it only in a repository that opts in.",
		Verify: []string{"gitleaks", "version"},
	},
	"gh": {
		Name:     "gh",
		What:     "GitHub's own command-line program, signed in as you",
		Homepage: "https://cli.github.com",
		Uses: map[Capability]Use{
			RemoteSettings: {
				Capability:  "reading and changing this repository's GitHub secret-scanning settings (ahoy remote)",
				Requirement: Required,
				Does: "abcd speaks to GitHub through gh, so a remote change is made by your own signed-in identity " +
					"and abcd never holds a token",
				WithoutIt: "ahoy remote cannot read or change the settings; nothing else in abcd needs gh",
				OnDecline: "the remote settings stay unread and unchanged",
			},
		},
		Install: map[string]Step{"darwin": homebrew("gh"), "linux": homebrew("gh")},
		Effects: "Homebrew downloads the gh program and puts it on your PATH. It does not sign you in: run " +
			"gh auth login afterwards, and abcd never sees the token.",
		Verify: []string{"gh", "--version"},
	},
}

// Names returns the registered tool names, sorted.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Known reports whether the registry holds an entry for name.
func Known(name string) bool {
	_, ok := registry[name]
	return ok
}
