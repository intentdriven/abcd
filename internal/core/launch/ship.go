package launch

import (
	"errors"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// ShipRequest is the input to a ship run.
type ShipRequest struct {
	RepoRoot string
	// Version is the release version, supplied by the caller for the reason
	// DryRunRequest.Version states: adr-19 leaves nothing in the tree to read.
	Version string
	// AllowDirty is --allow-dirty: an uncommitted change is carried rather than
	// refused, and the report names every path it carried. It waives the
	// dirty-tree gate and nothing else — never lockstep (adr-20).
	AllowDirty   bool
	ExistingTags []Semver
	// DocAudit is the documentation audit, measured by the caller
	// (DryRunRequest.DocAudit states why).
	DocAudit *DocAuditPreflight
}

// ShipReport is the outcome of a ship run. It stops at WouldPublish before any
// network/publish (no GitHub Release, SLSA, tag push, or retention execution).
type ShipReport struct {
	Version   string             `json:"version"`
	Bundle    Bundle             `json:"bundle"`
	Scan      scanner.ScanResult `json:"scan"`
	Lockstep  LockstepResult     `json:"lockstep"`
	Retention RetentionPlan      `json:"retention"`
	Smoke     SmokeReport        `json:"smoke"`
	Gates     []GateSummary      `json:"gates"`
	Warnings  []string           `json:"warnings,omitempty"`
	// AllowedDirty is every uncommitted path AllowDirty carried into the cut.
	AllowedDirty []string `json:"allowed_dirty,omitempty"`
	Blocked      bool     `json:"blocked"`
	BlockReasons []string `json:"block_reasons,omitempty"`
	WouldPublish bool     `json:"would_publish"` // true iff all gates pass
}

// ErrShipBlocked is returned when any gate hard-fails.
var ErrShipBlocked = errors.New("ship blocked by a launch gate")

// Ship runs the SAME gates as DryRun but HARD-FAILS: a scanner hard-fail, any
// bundle rejected[] entry, a lockstep drift/unreadable contract, a retention
// refusal, or a hard-fail in the pre-flight gate suite sets Blocked=true and
// returns ErrShipBlocked. If every gate passes it stops HERE and returns
// WouldPublish=true with NO network call — the real GitHub Release + SLSA + tag
// push + retention prune are a later phase (itd-72, itd-70).
//
// AllowDirty waives the dirty-tree gate only; it must NOT bypass lockstep
// (adr-20), which never consults it.
//
// Ship has no production caller: the shipped cut is `abcd launch ship`, whose
// render path runs the same suite through PrecheckPayload. It is kept as the
// suite's whole-verdict form, the shape itd-72's publishing step will call.
func Ship(req ShipRequest) (ShipReport, error) {
	var report ShipReport

	bundle, err := ResolveBundle(req.RepoRoot, nil)
	if err != nil {
		return ShipReport{}, err // preflight fault
	}
	report.Bundle = bundle
	policy, err := LoadGatePolicy(req.RepoRoot)
	if err != nil {
		return ShipReport{}, err // preflight fault
	}

	scan := scanBundle(req.RepoRoot, bundle)
	report.Scan = scan

	// The source tree is checked under the DEV polarity, for the reason DryRun
	// states: adr-19 keeps the version out of the committed manifests, and the
	// public polarity is proved over the rendered payload instead.
	vlPath := filepath.Join(req.RepoRoot, versionLocationRelPath)
	lockstep := CheckLockstep(TreeDev, req.RepoRoot, vlPath)
	report.Lockstep = lockstep

	report.Version = req.Version
	report.Retention = computeRetentionForReport(req.Version, DryRunRequest{
		RepoRoot: req.RepoRoot, Version: req.Version, ExistingTags: req.ExistingTags,
	})

	report.Smoke = SmokeLight(NewBundleTree(bundle))
	dirty := DirtyRefuse
	if req.AllowDirty {
		dirty = DirtyAllow
	}
	suite := runGateSuite(suiteRequest{
		RepoRoot: req.RepoRoot, Bundle: bundle, Dirty: dirty,
		DocAudit: req.DocAudit, Policy: policy,
	})
	report.Gates = suite.Gates
	report.Warnings = suite.Warnings
	if req.AllowDirty {
		report.AllowedDirty = suite.Dirty
	}
	report.BlockReasons = wouldRefuseOn(bundle, scan, lockstep, report.Retention, report.Smoke)
	report.BlockReasons = append(report.BlockReasons, suite.Refusals...)
	if len(report.BlockReasons) > 0 {
		report.Blocked = true
		report.WouldPublish = false
		return report, ErrShipBlocked
	}
	report.WouldPublish = true
	return report, nil
}
