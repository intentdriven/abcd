package ahoy

// docsTargetRetiredGapID is the one gap a saved retired docs.target raises:
// required, because setup refuses until the setting changes, and not
// resolvable, because only the person changes it (itd-2610030814013772).
const docsTargetRetiredGapID = "config.docs_target_retired"

// RetiredDocsTarget reports whether v is a docs.target value setup reads but
// no longer writes (claude_md, both), and the one explanation every front door
// shows for it: the detection gap, the install refusal and the --docs-target
// flag all carry these words, so none restates them (adr-2610030814023326:
// AGENTS.md is the one conventions file abcd writes). A writable value, and a
// value abcd never accepted, is not retired and has no explanation.
func RetiredDocsTarget(v string) (explanation string, retired bool) {
	if !inSet(v, docsTargetChoices) || inSet(v, docsTargetWritable) {
		return "", false
	}
	return "This project's saved setup choice `docs.target` in `.abcd/config.json` is `" + v + "`. " +
		"abcd now writes its block into AGENTS.md alone, which Claude Code and most other agent tools read directly. " +
		"Change that one setting with `abcd ahoy install --docs-target agents_md` (or `skip` for no block); " +
		"the block already in CLAUDE.md is then taken out of it, and nothing else of yours changes.", true
}

// docsTargetRetiredGap is the detection half: the saved value is read, never
// reported missing, and named with the explanation as the gap's detail, so the
// dry run and the doctor say what install will refuse.
func docsTargetRetiredGap(v string) Gap {
	why, _ := RetiredDocsTarget(v)
	return Gap{
		ID: docsTargetRetiredGapID, Category: ConfigChange, Scope: "repo",
		Title:    "docs.target names a conventions file abcd no longer writes",
		Detail:   why,
		FixHint:  "Change the one setting the detail names; ahoy install is refused until then.",
		Required: true, Resolvable: false,
	}
}

// retiredDocsTargetRefusal is the install half: the explanation when this run
// would write under a retired docs.target, or "" when it may go on. An override
// naming a writable value is the one setting changed, so it goes on whatever is
// saved; an override naming a retired value is refused whatever is saved; with
// no override the saved value decides. A config that cannot be parsed is left
// to the malformed-config refusal, which names its own cause.
func retiredDocsTargetRefusal(cwd string, overrides map[string]string) string {
	if v := overrides["docs_target"]; v != "" {
		if inSet(v, docsTargetWritable) {
			return ""
		}
		if why, retired := RetiredDocsTarget(v); retired {
			return why
		}
	}
	cfg, err := readConfig(cwd)
	if err != nil {
		return ""
	}
	saved, _ := stringVal(subMap(cfg, "docs"), "target")
	why, _ := RetiredDocsTarget(saved)
	return why
}

// writableMarkerTargets is markerTargets for the write side: the files a
// docs.target plants abcd's block into, which is none for a retired value.
// markerTargets itself still maps the retired values, because the retraction
// (markerFilesDropped) must know which files they named.
func writableMarkerTargets(docsTarget string) []string {
	if _, retired := RetiredDocsTarget(docsTarget); retired {
		return nil
	}
	return markerTargets(docsTarget)
}
