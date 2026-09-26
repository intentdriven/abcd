package launch

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// versionLocationRelPath is the committed version-location decision artefact.
const versionLocationRelPath = ".abcd/config/version-location.json"

// DryRunRequest is the input to a dry-run.
type DryRunRequest struct {
	RepoRoot string
	// Version is the release version this launch would publish, SUPPLIED by the
	// caller. adr-19 leaves no version key in the source tree, so there is
	// nothing here for the core to read: the version is a fact about the release
	// cut, and the front door that knows the cut injects it. Empty is honest —
	// it means the caller could not name one, and retention says so.
	Version      string
	ExistingTags []Semver // injected; nil → default `git tag -l v*` provider
	// Citations is the citation baseline's state, MEASURED by the caller.
	// Grading it needs internal/core/lint, which imports this package for its
	// semver — so the front door that holds both hands the result in as data
	// rather than this package reaching back. Nil means the repo has not armed
	// the citation gate.
	Citations *CitationPreflight
	// Receipts is the semantic-pass receipts' state, MEASURED by the caller for
	// the same reason as Citations: it needs a git rev-parse and a directory
	// read, and this package does not reach back through the front door. Nil
	// means the measurement was not taken, which the gate reports as such rather
	// than as an absence of receipts.
	Receipts *ReceiptPreflight
	// DocAudit is the documentation audit's result, MEASURED by the caller for
	// the same reason as Citations. Nil means the repository has not armed a
	// docs-lint configuration, which the gate reports as such.
	DocAudit *DocAuditPreflight
	// Parity is the parity diff's input: the previous release's tag and, when
	// the operator asked for it, the release-asset fetcher. Nil runs no diff.
	Parity *ParityInput
	// DeepSmoke is the isolated page runner the installability smoke's deep
	// tier renders every page through. Nil keeps the preview at the light tier:
	// the deep tier is opt-in here and always on in the cut.
	DeepSmoke PageRunner
}

// GateSummary records one gate's disposition.
type GateSummary struct {
	Name string `json:"name"`
	// Status is "ran", "not_implemented", "not_armed" (the repository has not
	// adopted what the gate reads) or "host-run".
	Status string `json:"status"`
	Detail string `json:"detail"`
	// Tier is what a finding in the row does: TierHardFail refuses, TierWarn
	// surfaces. Empty for a row whose refusals are reported elsewhere.
	Tier string `json:"tier,omitempty"`
	// Findings are the row's located concerns, each also carried as a line in
	// would_refuse_on (hard-fail) or warnings (warn).
	Findings []GateFinding `json:"findings,omitempty"`
}

// DryRunReport is the full dry-run preview. No artefact is written.
type DryRunReport struct {
	Version   string             `json:"version"`
	Bundle    Bundle             `json:"bundle"`
	Scan      scanner.ScanResult `json:"scan"`
	Lockstep  LockstepResult     `json:"lockstep"`
	Retention RetentionPlan      `json:"retention"`
	Smoke     SmokeReport        `json:"smoke"`
	// DeepSmoke is the deep installability tier, present when it was asked for.
	DeepSmoke *DeepSmokeReport `json:"deep_smoke,omitempty"`
	// Parity is the file-level diff against the previous release's payload.
	Parity        *ParityReport `json:"parity,omitempty"`
	Gates         []GateSummary `json:"gates"`
	WouldPublish  bool          `json:"would_publish"` // always false in dry-run
	WouldRefuseOn []string      `json:"would_refuse_on,omitempty"`
	// Warnings are the warn-tier concerns: surfaced, refusing nothing unless
	// the repository configures the suite strict.
	Warnings []string `json:"warnings,omitempty"`
	// ReportPath is the repo-relative directory the front door wrote this
	// preview's pre-flight report into; ReportError says why it could not.
	ReportPath  string `json:"report_path,omitempty"`
	ReportError string `json:"report_error,omitempty"`
}

// DryRun assembles the bundle, scans it, checks lockstep, previews retention and
// runs the pre-flight gate suite, then reports what a real ship WOULD refuse on.
// It ALWAYS returns exit-0 semantics: an error is returned only for a preflight
// fault (bad include config) that makes a report impossible — never on a
// finding. It writes nothing; the front door writes the pre-flight report.
func DryRun(req DryRunRequest) (DryRunReport, error) {
	var report DryRunReport

	bundle, err := ResolveBundle(req.RepoRoot, nil)
	if err != nil {
		return DryRunReport{}, err // preflight fault only
	}
	report.Bundle = bundle
	policy, err := LoadGatePolicy(req.RepoRoot)
	if err != nil {
		return DryRunReport{}, err // preflight fault only
	}

	scan := scanBundle(req.RepoRoot, bundle)
	report.Scan = scan

	// The DEV polarity is the one the SOURCE TREE must satisfy: adr-19 keeps the
	// version keys out of the committed manifests, so a public check here would
	// accuse a correct repository of drift and prescribe the exact key the ADR
	// forbids. The public polarity belongs over the rendered payload, where
	// RenderPayload applies it to its own output.
	vlPath := filepath.Join(req.RepoRoot, versionLocationRelPath)
	lockstep := CheckLockstep(TreeDev, req.RepoRoot, vlPath)
	report.Lockstep = lockstep

	report.Version = req.Version
	report.Retention = computeRetentionForReport(req.Version, req)

	// The smoke reads the RESOLVED BUNDLE, not the working tree: a file present
	// in the tree but excluded from the payload is exactly the break it exists
	// to catch. It subsumes itd-65's placeholder `plugin.json-parse` gate, which
	// asserted a strict subset of what the light tier asserts.
	smoke := SmokeLight(NewBundleTree(bundle))
	report.Smoke = smoke

	// The preview has no override flag, so the dirty-tree gate runs at its
	// refusing default: a dirty tree is reported as what a cut would refuse on.
	suite := runGateSuite(suiteRequest{
		RepoRoot: req.RepoRoot, Bundle: bundle, Dirty: DirtyRefuse,
		DocAudit: req.DocAudit, Policy: policy,
	})

	report.Gates = append([]GateSummary{
		{Name: "secret+pii-scan", Status: "ran", Detail: scanDetail(scan)},
		{Name: "installability-smoke", Status: "ran", Detail: smokeDetail(smoke)},
	}, suite.Gates...)
	report.Gates = append(report.Gates,
		citationGate(req.Citations),
		// Reporting-only: the semantic passes are host-run and release.yml owns
		// the required-gates list, so this row never feeds WouldRefuseOn.
		receiptGate(req.Receipts),
	)

	if req.DeepSmoke != nil {
		deep := smokeDeepOverBundle(bundle, req.DeepSmoke)
		report.DeepSmoke = &deep
		report.Gates = append(report.Gates, deepSmokeGate(&deep))
	}
	if req.Parity != nil {
		parity := PayloadParity(req.RepoRoot, bundle, *req.Parity)
		report.Parity = &parity
		report.Gates = append(report.Gates, parityGate(&parity))
	}

	report.WouldRefuseOn = wouldRefuseOn(bundle, scan, lockstep, report.Retention, smoke)
	report.WouldRefuseOn = append(report.WouldRefuseOn, deepSmokeRefusals(report.DeepSmoke)...)
	report.WouldRefuseOn = append(report.WouldRefuseOn, parityRefusals(report.Parity)...)
	report.WouldRefuseOn = append(report.WouldRefuseOn, suite.Refusals...)
	report.WouldRefuseOn = append(report.WouldRefuseOn, citationRefusals(req.Citations)...)
	report.Warnings = suite.Warnings
	report.WouldPublish = false
	return report, nil
}

// scanBundle adapts the bundle's Included files to the scanner and runs it.
func scanBundle(repoRoot string, bundle Bundle) scanner.ScanResult {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return scanner.ScanResult{Unavailable: true, UnavailableReason: err.Error()}
	}
	files := make([]scanner.BundleFile, 0, len(bundle.Included))
	for _, f := range bundle.Included {
		files = append(files, scanner.BundleFile{LogicalPath: f.LogicalPath, ResolvedPath: f.ResolvedPath})
	}
	res, _ := sc.ScanBundle(files)
	return res
}

// computeRetentionForReport builds the retention preview, resolving the existing
// tag list from the injected slice or the default git provider.
func computeRetentionForReport(version string, req DryRunRequest) RetentionPlan {
	pub, err := ParseSemver(version)
	if err != nil {
		return RetentionPlan{
			Published: "v" + version, Refused: true,
			RefusalReason: "published version is not strict SemVer: " + version,
		}
	}
	existing := req.ExistingTags
	if existing == nil {
		// A listing that FAILED is not an empty release set: read as one, it
		// previews "nothing to prune" for a repository whose tags were never
		// seen, indistinguishable from a genuine nothing-to-prune (iss-194).
		tags, err := GitExistingTags(req.RepoRoot)
		if err != nil {
			return RetentionPlan{
				Published: pub.Tag(), Line: pub.Line(), Refused: true,
				RefusalReason: "the existing release tags could not be listed, so the plan cannot say what the release would prune: " + err.Error(),
			}
		}
		// A shallow checkout's listing SUCCEEDS but holds only the tags that
		// were fetched, so it is not the release set either
		// (iss-2609251238184553).
		if shallow, err := ShallowCheckout(req.RepoRoot); err != nil || shallow {
			reason := "the checkout is shallow, so its tag listing may hold only the tags that were fetched"
			if err != nil {
				reason = "whether the checkout is shallow could not be read: " + err.Error()
			}
			return RetentionPlan{
				Published: pub.Tag(), Line: pub.Line(), Refused: true,
				RefusalReason: reason + ", and the plan cannot say what the release would prune — fetch the full history and tags first",
			}
		}
		existing = tags
	}
	return ComputeRetention(pub, existing)
}

// ShallowCheckout reports whether the checkout at repoRoot is a shallow clone.
// A shallow clone's tag listing succeeds while holding only the tags that were
// fetched, so neither the retention plan nor the parity diff's baseline may
// read that listing as the release set. An error is a checkout whose shallowness
// git could not report, which a caller treats as shallow.
func ShallowCheckout(repoRoot string) (bool, error) {
	out, err := gitutil.Run(repoRoot, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return true, err
	}
	return out != "false", nil
}

// scanRefusals collects the secret/PII scan gate's refusals: scanner
// unavailability, secret/PII hard-fails, and fail-closed coverage gaps. It is
// the scan slice of wouldRefuseOn, extracted so the render's MATERIALISING path
// (PrecheckPayload) fails closed on exactly the same scan verdict the dry-run and
// ship gates report — one scanner, one notion of "what the scan refuses"
// (gh-328).
func scanRefusals(scan scanner.ScanResult) []string {
	var reasons []string
	if scan.Unavailable {
		reasons = append(reasons, "scanner unavailable: "+scan.UnavailableReason)
	}
	if scan.HardFails > 0 {
		reasons = append(reasons, hardFailReason(scan))
	}
	// Fail closed on the coverage gap: any include-selected file the scanner
	// could not cover (unreadable, over the byte-scan cap, a non-regular leaf,
	// or binary without a reviewed skip) is refused rather than shipped raw,
	// and the reason travels with the path. "Some other file scanned" is not
	// coverage of the unscanned one, so a single scanned README cannot disarm
	// this (GHSA-5mmm-3whv-3rqp). Skip-listed files (scan.ScannedBinary) were
	// byte-scanned with the secret, harness-leak and long-literal identity
	// rules, no format decoded, so their plaintext regions are covered and
	// their findings arrive through HardFails like any other
	// (GHSA-9wv7-88w3-f77m). A container format abcd decodes
	// (scan.ContentDecoded) was opened as well and its entries scanned with
	// the same rules, so a token inside a gzip member, a zip entry or a PNG
	// zTXt chunk arrives through HardFails too. What remains
	// (scan.ContentUnverified) is what the decoder could not account for: a
	// format it cannot read (a JPEG entropy stream, a PDF, an mp4 box), a
	// stream a decode bound refused, or a file whose STRUCTURE did not add up
	// — an archive whose entries do not tile it, a tar entry padded with
	// something other than zeros, a header field carrying a member nothing
	// read. Those got the byte scan alone and, per iss-2608291832160371, do
	// not refuse on their own. The gate row counts that tier apart from the decoded
	// one rather than folding the two into a single green, and the scan
	// result carries each unverified path's reason and detected format.
	for _, p := range scan.Unscanned {
		reason := "unscanned payload file (fail-closed coverage gap): " + p
		if why := scan.UnscannedWhy[p]; why != "" {
			reason += " (" + why + ")"
		}
		reasons = append(reasons, reason)
	}
	return reasons
}

// wouldRefuseOn collects everything a real ship WOULD block on: scan hard-fails,
// bundle rejections, lockstep drift/unreadable, retention refusal, and an
// uninstallable declared surface.
func wouldRefuseOn(bundle Bundle, scan scanner.ScanResult, lockstep LockstepResult, retention RetentionPlan, smoke SmokeReport) []string {
	reasons := scanRefusals(scan)
	for _, r := range bundle.Rejected {
		reasons = append(reasons, "bundle rejected: "+r.LogicalPath+" ("+string(r.Reason)+")")
	}
	if lockstep.Unreadable {
		reasons = append(reasons, "lockstep contract unreadable: "+lockstep.Detail)
	}
	for _, d := range lockstep.Drifts {
		reasons = append(reasons, "lockstep drift: "+d)
	}
	if retention.Refused {
		reasons = append(reasons, "retention refused: "+retention.RefusalReason)
	}
	reasons = append(reasons, smokeRefusals(smoke)...)
	return reasons
}

func hardFailReason(scan scanner.ScanResult) string {
	n := 0
	for _, f := range scan.Findings {
		if f.Severity == scanner.SeverityHardFail {
			n++
		}
	}
	return "secret/PII hard-fail findings: " + itoa(n)
}

func scanDetail(scan scanner.ScanResult) string {
	if scan.Unavailable {
		return "unavailable: " + scan.UnavailableReason
	}
	return "scanned " + itoa(scan.FilesScanned) + " files with the full rule set, " +
		itoa(len(scan.ScannedBinary)) + " binary (byte rules only), " +
		itoa(len(scan.ContentDecoded)) + " decoded (entries scanned), " +
		itoa(len(scan.ContentUnverified)) + " compressed (not content-verified), " +
		itoa(scan.HardFails) + " hard-fails"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
