package reading

// parked.go reads back one run an assembly parked, for a front door that sends
// the reading to a provider itself instead of handing it to the host
// (itd-2609081951381895, spc-2609251028149555 AC 3). A cold-reading position
// is self-contained under ruling DR5 of 2026-09-29: its whole input is the
// parked bundle, so the step a provider is sent carries every byte the reader
// may see, and the reader reads no file.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Parked is one staged run as a dispatch reads it: the run, its position, the
// manifest's content hash an output must cite, and the bundle bytes as the
// assembly wrote them.
type Parked struct {
	RunID            string
	Position         Position
	AssemblerVersion string
	ManifestSHA256   string
	Bundle           []byte
	// Items is how many items the manifest lists, and Unscanned how many of
	// them it marks `unscanned`: the items the exclusion floor never examined,
	// which travel whole. A front door that sends the run to a provider names
	// the count before the send, so a paid send never carries it silently.
	Items     int
	Unscanned int
}

// ReadParked reads the run runID parked under DefaultRunDir in repoRoot,
// through the repository root, so a symlinked ancestor cannot serve another
// tree's run. The run id must match the run-id grammar before any path is
// built from it.
func ReadParked(repoRoot, runID string) (Parked, error) {
	if !recordid.ValidReadingRunID(runID) {
		return Parked{}, fmt.Errorf("reading: %q is not a reading run id (rdg-<digits>)", echo(runID))
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Parked{}, fmt.Errorf("reading: opening the repository root: %w", err)
	}
	defer root.Close()
	dir := DefaultRunDir + "/" + runID + "/"
	raw, err := fsutil.ReadGuardedInRoot(root, dir+ManifestFileName, MaxFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Parked{}, fmt.Errorf("reading: run %s is not parked at %s; assemble the run first", runID, dir)
		}
		return Parked{}, fmt.Errorf("reading: the manifest of run %s: %w", runID, err)
	}
	m, err := DecodeManifest(raw)
	if err != nil {
		return Parked{}, fmt.Errorf("reading: the manifest of run %s: %s", runID, redactRefused(repoRoot, err.Error()))
	}
	if m.RunID != runID {
		return Parked{}, fmt.Errorf("reading: the manifest parked for run %s names run %s", runID, echo(m.RunID))
	}
	if _, err := ParsePosition(string(m.Position)); err != nil {
		return Parked{}, fmt.Errorf("reading: the manifest of run %s: %w", runID, err)
	}
	bundle, err := fsutil.ReadGuardedInRoot(root, dir+BundleFileName, MaxFileBytes)
	if err != nil {
		return Parked{}, fmt.Errorf("reading: the bundle of run %s: %w", runID, err)
	}
	var head struct {
		Position Position `json:"position"`
	}
	if err := json.Unmarshal(bundle, &head); err != nil || head.Position != m.Position {
		return Parked{}, fmt.Errorf("reading: the bundle parked for run %s does not state the manifest's position %s", runID, m.Position)
	}
	unscanned := 0
	for _, it := range m.Items {
		if it.Scan == ScanUnscanned {
			unscanned++
		}
	}
	return Parked{RunID: runID, Position: m.Position, AssemblerVersion: m.AssemblerVersion,
		ManifestSHA256: sha256Hex(raw), Bundle: bundle, Items: len(m.Items), Unscanned: unscanned}, nil
}

// DispatchInput is the input a provider is sent for p under def: the facts
// the output's envelope must cite, then the bundle. The reader is handed the
// bundle's items and nothing else from the repository; the envelope facts are
// the run's own identifiers, which the output must echo for the ingest to
// prove the run.
func (p Parked) DispatchInput(def Definition, model string) (string, error) {
	in := struct {
		Type             string          `json:"_type"`
		RunID            string          `json:"run_id"`
		Position         Position        `json:"position"`
		Regime           string          `json:"regime"`
		ManifestSHA256   string          `json:"manifest_sha256"`
		DefinitionSHA256 string          `json:"definition_sha256"`
		AssemblerVersion string          `json:"assembler_version"`
		Model            string          `json:"model"`
		Bundle           json.RawMessage `json:"bundle"`
	}{
		Type: "abcd.reading.dispatch/1", RunID: p.RunID, Position: p.Position, Regime: def.Regime,
		ManifestSHA256: p.ManifestSHA256, DefinitionSHA256: def.SHA256, AssemblerVersion: p.AssemblerVersion,
		Model: model, Bundle: json.RawMessage(p.Bundle),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("reading: composing the dispatch input for run %s: %w", p.RunID, err)
	}
	return "Return one JSON document of type " + OutputType + " for this run. Copy run_id, position, regime and " +
		"manifest_sha256 into the envelope, and definition_sha256, assembler_version and model into its instrument, " +
		"exactly as given; the bundle's items are the only material you read.\n\n" + string(raw), nil
}
