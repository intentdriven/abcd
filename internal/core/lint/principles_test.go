package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// The principles family as a declared record store, and the four rules over its
// typed entries (spc-2609020626042471, adr-2609021016270132).

const (
	prnDir      = "rec/principles"
	prnShipped  = "rec/intents/shipped"
	prnCondA    = "cond-2609020626047525"
	prnCondB    = "cond-2609020626048283"
	prnOccasion = "rdi-2609011200000001"
)

// principleStores is schemaStores plus the principles family.
func principleStores() map[string]string {
	s := schemaStores()
	s["prn"] = prnDir
	return s
}

// principleConfig arms record_schema and the four principle rules at the
// severities the committed configuration declares.
func principleConfig() Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema:         {Enabled: true, Severity: severityBlocker, RecordStores: principleStores()},
			rulePrincipleUntyped:     {Enabled: true, Severity: severityWarn},
			rulePrincipleClaims:      {Enabled: true, Severity: severityBlocker},
			rulePrincipleInheritance: {Enabled: true, Severity: severityWarn},
			rulePrincipleFalsified:   {Enabled: true, Severity: severityBlocker},
		},
	}
}

// typedPrinciple renders a typed entry with the four keys given verbatim (an
// empty argument leaves the key out) and a statement paragraph.
func typedPrinciple(stem, claimType, reference, comparison, evidence, rule string) string {
	var b strings.Builder
	b.WriteString("---\nid: prn-" + stem + "\n")
	for _, kv := range [][2]string{{"claim_type", claimType}, {"reference", reference}, {"comparison", comparison}, {"evidence", evidence}} {
		if kv[1] == "" {
			continue
		}
		b.WriteString(kv[0] + ": " + kv[1] + "\n")
	}
	b.WriteString("---\n\n# A principle\n\n**The rule.** " + rule + "\n\n**Why.** Because adr-1 said so.\n")
	return b.String()
}

// shippedIntent renders a shipped intent carrying one scope condition and the
// given Audit Notes blocks.
func shippedIntent(id string, conds []string, audit string) string {
	var b strings.Builder
	b.WriteString("---\nid: " + id + "\n---\n\n# A shipped intent\n\n## Scope Conditions\n\n")
	for _, c := range conds {
		b.WriteString("- A condition held. <!-- cond: " + c + " -->\n")
	}
	b.WriteString("\n## Audit Notes\n\n" + audit + "\n")
	return b.String()
}

func conditionBlock(id, value, rationale, narrowing string) string {
	s := "<!-- abcd-condition: " + id + " occasion=" + prnOccasion + " -->\n" +
		"Condition disposition — 2026-09-02, occasioned by " + prnOccasion + ".\n" +
		"- " + id + " — " + value + ": " + rationale + "\n"
	if narrowing != "" {
		s += "  narrowing: " + narrowing + "\n"
	}
	return s
}

func lintPrinciples(t *testing.T, root string) []Finding {
	t.Helper()
	fs, err := Lint(principleConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func rulesOf(fs []Finding, file string) []string {
	var out []string
	for _, f := range fs {
		if f.File == file {
			out = append(out, f.RuleID+": "+f.Message)
		}
	}
	return out
}

// TestPrincipleStoreIsSlugKeyed: the handle is prn-<filename stem>, the id a
// typed entry carries is compared against it, and the record graph carries the
// entry as a principle node under that handle.
func TestPrincipleStoreIsSlugKeyed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/fix-the-detector.md",
		typedPrinciple("fix-the-detector", "causal", `"abcd lint"`, `"Hand fixes against a detector."`, "[adr-1]", "Fix the class."))
	writeFile(t, root, prnDir+"/wrong-id.md",
		strings.Replace(typedPrinciple("wrong-id", "causal", `"abcd lint"`, `"One against another."`, "[adr-1]", "Fix."), "id: prn-wrong-id", "id: prn-something-else", 1))
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")

	fs := lintPrinciples(t, root)
	if got := rulesOf(fs, filepath.Join(prnDir, "fix-the-detector.md")); len(got) != 0 {
		t.Errorf("a well-typed principle drew findings: %v", got)
	}
	if !findingWith(fs, filepath.Join(prnDir, "wrong-id.md"), ruleRecordSchema, "'prn-wrong-id'") {
		t.Errorf("an id disagreeing with prn-<stem> is not reported by record_schema: %+v", fs)
	}

	g, err := LoadRecordGraph(principleConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range g.Nodes {
		if n.ID == "prn-fix-the-detector" {
			found = true
			if n.Type != "principle" || n.Title != "A principle" {
				t.Errorf("the principle node is %+v, want type principle titled by its H1", n)
			}
		}
	}
	if !found {
		t.Errorf("the record graph carries no prn-fix-the-detector node: %+v", g.Nodes)
	}
}

// TestSlugKeyedStoreCollidesOnTheRenderedHandle: uniqueness keys on the rendered
// handle, so thirty untyped principles are thirty handles and not thirty claims
// on prn-0.
func TestSlugKeyedStoreCollidesOnTheRenderedHandle(t *testing.T) {
	root := t.TempDir()
	for _, stem := range []string{"one", "two", "three"} {
		writeFile(t, root, prnDir+"/"+stem+".md", "# "+stem+"\n\n**The rule.** A rule.\n")
	}
	fs := lintPrinciples(t, root)
	for _, f := range fs {
		if f.RuleID == ruleRecordSchema {
			t.Errorf("a clean slug-keyed store drew a record_schema finding: %+v", f)
		}
	}
}

// TestUntypedPrincipleHasNoSchemaFinding: an entry with no frontmatter is a
// prose file keyed by its filename, which is what every entry is today.
func TestUntypedPrincipleHasNoSchemaFinding(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/less-but-better.md", "# Less, but better\n\n**The rule.** Fewer things, done well.\n")
	writeFile(t, root, prnDir+"/README.md", "# principles/\n\nThe store's own index.\n")
	fs := lintPrinciples(t, root)
	if n := countRule(fs, ruleRecordSchema); n != 0 {
		t.Errorf("an untyped principle drew %d record_schema finding(s): %+v", n, fs)
	}
}

// TestUntypedPrincipleIsAWarnAndNothingElse is ac-2.
func TestUntypedPrincipleIsAWarnAndNothingElse(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join(prnDir, "less-but-better.md")
	writeFile(t, root, prnDir+"/less-but-better.md", "# Less, but better\n\n**The rule.** See [adr-9](x.md).\n")
	fs := lintPrinciples(t, root)
	got := rulesOf(fs, rel)
	if len(got) != 1 {
		t.Fatalf("an untyped principle drew %d findings, want exactly the one untyped warning: %v", len(got), got)
	}
	f := fs[0]
	for _, x := range fs {
		if x.File == rel {
			f = x
		}
	}
	if f.RuleID != rulePrincipleUntyped || f.Severity != severityWarn {
		t.Errorf("the finding is %s at %s, want %s at warn", f.RuleID, f.Severity, rulePrincipleUntyped)
	}
	want := "principle prn-less-but-better is untyped (carries none of claim_type, reference, comparison, evidence)"
	if f.Message != want {
		t.Errorf("message = %q, want %q", f.Message, want)
	}
}

// TestPrincipleClaimsNamesTheMissingKey is ac-1.
func TestPrincipleClaimsNamesTheMissingKey(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/partial.md", typedPrinciple("partial", "causal", `"abcd lint"`, "", "[adr-1]", "A rule."))
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs := lintPrinciples(t, root)
	rel := filepath.Join(prnDir, "partial.md")
	if !findingWith(fs, rel, rulePrincipleClaims, "'comparison'") {
		t.Fatalf("a typed principle missing comparison is not reported naming the key: %v", rulesOf(fs, rel))
	}
	for _, f := range fs {
		if f.File == rel && f.RuleID == rulePrincipleClaims && f.Severity != severityBlocker {
			t.Errorf("principle_claims fired at %s, want blocker", f.Severity)
		}
		if f.File == rel && f.RuleID == rulePrincipleUntyped {
			t.Errorf("a partially typed principle is reported as untyped: %s", f.Message)
		}
	}
}

// TestPrincipleClaimsRefusesEmptyValue: a blank value is the byte shape of a key
// someone forgot, and only the literal null declines a claim.
func TestPrincipleClaimsRefusesEmptyValue(t *testing.T) {
	for name, tc := range map[string]struct{ key, value string }{
		"blank claim_type":  {"claim_type", ""},
		"tilde reference":   {"reference", "~"},
		"NULL comparison":   {"comparison", "NULL"},
		"empty evidence":    {"evidence", "[]"},
		"quoted empty":      {"comparison", `""`},
		"blank id":          {"id", ""},
		"capitalised null":  {"claim_type", "Null"},
		"empty flow string": {"reference", `''`},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			body := typedPrinciple("p", "causal", `"abcd lint"`, `"One against another."`, "[adr-1]", "A rule.")
			lines := strings.Split(body, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, tc.key+":") {
					lines[i] = strings.TrimRight(tc.key+": "+tc.value, " ")
				}
			}
			writeFile(t, root, prnDir+"/p.md", strings.Join(lines, "\n"))
			writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
			fs := lintPrinciples(t, root)
			if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, "'"+tc.key+"'") {
				t.Errorf("%s: %s: %q is not refused: %v", name, tc.key, tc.value, rulesOf(fs, filepath.Join(prnDir, "p.md")))
			}
		})
	}

	// The explicit null alone is a declined claim, not a fault.
	root := t.TempDir()
	writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "null", "null", "null", "[adr-1]", "A rule."))
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs := lintPrinciples(t, root)
	if got := rulesOf(fs, filepath.Join(prnDir, "p.md")); len(got) != 0 {
		t.Errorf("a principle declining three claims with null drew findings: %v", got)
	}
}

// TestPrincipleClaimsRefusesFourthClaimType: the vocabulary is the three claim
// kinds, and a fourth is a ruling this rule does not make.
func TestPrincipleClaimsRefusesFourthClaimType(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "normative", `"abcd lint"`, `"One against another."`, "[adr-1]", "A rule."))
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs := lintPrinciples(t, root)
	if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, "criterion, causal, context") {
		t.Errorf("a fourth claim type is not refused naming the three: %v", rulesOf(fs, filepath.Join(prnDir, "p.md")))
	}
}

// TestPrincipleClaimsReadsMechanismAsCausal: the shipped intent token is an
// alias on read, never refused with a rename.
func TestPrincipleClaimsReadsMechanismAsCausal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "mechanism", `"abcd lint"`, `"One against another."`, "[adr-1]", "A rule."))
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs := lintPrinciples(t, root)
	if got := rulesOf(fs, filepath.Join(prnDir, "p.md")); len(got) != 0 {
		t.Errorf("claim_type: mechanism drew findings: %v", got)
	}
}

// TestPrincipleClaimsJudgesTheGrammar covers the three remaining grammars: a
// reference that is neither a handle nor a quoted surface, an evidence member
// that is neither a handle nor a condition identity, and a duplicated member.
func TestPrincipleClaimsJudgesTheGrammar(t *testing.T) {
	for name, tc := range map[string]struct{ reference, evidence, want string }{
		"bare prose reference":  {"abcd lint", "[adr-1]", "'reference'"},
		"prose evidence member": {`"abcd lint"`, "[adr-1, the ADR]", "'the ADR'"},
		"duplicate member":      {`"abcd lint"`, "[adr-1, adr-1]", "twice"},
		"block sequence":        {`"abcd lint"`, "", "'evidence'"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			ev := tc.evidence
			body := typedPrinciple("p", "context", tc.reference, `"One against another."`, ev, "A rule.")
			if ev == "" {
				body = strings.Replace(body, "---\n\n# A", "evidence:\n  - adr-1\n---\n\n# A", 1)
			}
			writeFile(t, root, prnDir+"/p.md", body)
			writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
			fs := lintPrinciples(t, root)
			if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, tc.want) {
				t.Errorf("%s: no principle_claims finding quoting %s: %v", name, tc.want, rulesOf(fs, filepath.Join(prnDir, "p.md")))
			}
		})
	}
}

// TestPrincipleStatementMayNotCite: the projection promises a statement free of
// genealogy, so a typed statement carrying a record handle or a link is refused
// before the assembler has to.
func TestPrincipleStatementMayNotCite(t *testing.T) {
	for name, rule := range map[string]string{
		"record handle": "Fix the class, as adr-1 ruled.",
		"markdown link": "Fix the class, as [the ruling](../decisions/adrs/0001-a.md) says.",
		"condition":     "Fix the class while " + prnCondA + " holds.",
		// Every link shape, not the inline one alone (iss-2609261039139464): a
		// URL is a citation whatever markup carries it, and a reference-style
		// link is a link whose target sits elsewhere.
		"bare URL":            "Fix the class, as https://example.com/LEAKURL says.",
		"bare www URL":        "Fix the class, as www.example.com/LEAKURL says.",
		"autolink":            "Fix the class, as <https://example.com/LEAKAUTO> says.",
		"email autolink":      "Fix the class, as <someone@example.com> says.",
		"reference link":      "Fix the class, as [LEAKREFLABEL][ruling] says.",
		"collapsed reference": "Fix the class, as [the ruling][] says.",
		"nested-bracket link": "Fix the class, as [the [first] ruling](../decisions/adrs/0001-a.md) says.",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "causal", `"abcd lint"`, `"One against another."`, "[adr-1]", rule))
			writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
			fs := lintPrinciples(t, root)
			if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, "The rule.") {
				t.Errorf("%s in the statement is not refused: %v", name, rulesOf(fs, filepath.Join(prnDir, "p.md")))
			}
		})
	}
}

// TestPrincipleTitleMayNotCite: the statement a reading receives is the H1
// title AND the paragraph, so the lint judges the title the projection sends
// above the paragraph, and a citation there is refused by record-lint rather
// than only by the assembler (iss-2609261039134673).
func TestPrincipleTitleMayNotCite(t *testing.T) {
	for name, title := range map[string]string{
		"record handle": "A principle citing itd-79",
		"markdown link": "A principle after [the ruling](../decisions/adrs/0001-a.md)",
		"bare URL":      "A principle after https://example.com/LEAKURL",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			doc := strings.Replace(typedPrinciple("p", "causal", `"abcd lint"`, `"One against another."`, "[adr-1]",
				"Fix the class, not the instance."), "# A principle\n", "# "+title+"\n", 1)
			writeFile(t, root, prnDir+"/p.md", doc)
			writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
			fs := lintPrinciples(t, root)
			if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, "title") {
				t.Errorf("%s in the title is not refused: %v", name, rulesOf(fs, filepath.Join(prnDir, "p.md")))
			}
		})
	}
}

// TestHeadingShapedPrincipleHasNoStatement: the statement is the labelled
// paragraph and nothing else, in both readers (iss-2609261039132350). A typed
// principle that writes it as a `## The rule` heading carries no statement the
// projection can send, and principle_claims says so rather than passing a
// principle no reading will ever receive.
func TestHeadingShapedPrincipleHasNoStatement(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(typedPrinciple("p", "causal", `"abcd lint"`, `"One against another."`, "[adr-1]",
		"Fix the class, not the instance."), "**The rule.** ", "## The rule\n\n", 1)
	writeFile(t, root, prnDir+"/p.md", doc)
	writeFile(t, root, "rec/decisions/adrs/0001-a.md", "---\nid: adr-1\n---\n# ADR-1\n")
	fs := lintPrinciples(t, root)
	if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleClaims, "no **The rule.** paragraph") {
		t.Errorf("a heading-shaped statement is not reported: %v", rulesOf(fs, filepath.Join(prnDir, "p.md")))
	}
}

// inheritanceCorpus writes one typed principle resting on prnCondA, and one
// shipped intent carrying that condition with the given audit block.
func inheritanceCorpus(t *testing.T, audit string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "causal", "itd-181", `"One against another."`, "[itd-181, "+prnCondA+"]", "A rule."))
	writeFile(t, root, prnShipped+"/itd-181-a.md", shippedIntent("itd-181", []string{prnCondA}, audit))
	return root
}

// TestFalsifiedConditionIsReported is ac-3.
func TestFalsifiedConditionIsReported(t *testing.T) {
	root := inheritanceCorpus(t, conditionBlock(prnCondA, "falsified", "delivery refuted it", ""))
	fs := lintPrinciples(t, root)
	rel := filepath.Join(prnDir, "p.md")
	if !findingWith(fs, rel, rulePrincipleFalsified, prnCondA) || !findingWith(fs, rel, rulePrincipleFalsified, "prn-p") {
		t.Fatalf("a principle resting on a falsified condition is not reported naming both: %v", rulesOf(fs, rel))
	}
	for _, f := range fs {
		if f.RuleID == rulePrincipleFalsified && f.Severity != severityBlocker {
			t.Errorf("principle_falsified fired at %s, want blocker", f.Severity)
		}
	}
}

// TestNarrowedConditionCarriesTheNarrowing is ac-4.
func TestNarrowedConditionCarriesTheNarrowing(t *testing.T) {
	root := inheritanceCorpus(t, conditionBlock(prnCondA, "narrowed", "only half holds", "holds for one repository only"))
	fs := lintPrinciples(t, root)
	rel := filepath.Join(prnDir, "p.md")
	if !findingWith(fs, rel, rulePrincipleInheritance, "holds for one repository only") {
		t.Fatalf("a principle resting on a narrowed condition does not carry the narrowing: %v", rulesOf(fs, rel))
	}
	if countRule(fs, rulePrincipleFalsified) != 0 {
		t.Errorf("a narrowed condition drew a principle_falsified finding: %v", rulesOf(fs, rel))
	}
}

// TestUndispositionedConditionIsUntested is ac-5, with the two spellings of no
// judgement: no disposition at all, and one recorded as untested.
func TestUndispositionedConditionIsUntested(t *testing.T) {
	for name, audit := range map[string]string{
		"no disposition":     "_Empty._",
		"untested recorded":  conditionBlock(prnCondA, "untested", "no reading reached it", ""),
		"another condition":  conditionBlock(prnCondB, "survived", "held", ""),
		"survived elsewhere": "",
	} {
		t.Run(name, func(t *testing.T) {
			root := inheritanceCorpus(t, audit)
			fs := lintPrinciples(t, root)
			if !findingWith(fs, filepath.Join(prnDir, "p.md"), rulePrincipleInheritance, "untested condition "+prnCondA) {
				t.Errorf("%s: the principle is not reported as resting on an untested condition: %v",
					name, rulesOf(fs, filepath.Join(prnDir, "p.md")))
			}
		})
	}

	// A survived condition is what held, and is silent.
	root := inheritanceCorpus(t, conditionBlock(prnCondA, "survived", "held", ""))
	fs := lintPrinciples(t, root)
	if got := rulesOf(fs, filepath.Join(prnDir, "p.md")); len(got) != 0 {
		t.Errorf("a principle resting on a survived condition drew findings: %v", got)
	}
}

// TestUnresolvableEvidenceIsReportedNotAbsent: a principle distilled from a
// lifeboat cites packed ids, which resolve only in the source repository, and a
// condition identity no shipped intent carries is in the same position.
func TestUnresolvableEvidenceIsReportedNotAbsent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, prnDir+"/p.md", typedPrinciple("p", "context", "adr-9", `"One against another."`, "[adr-9, "+prnCondB+"]", "A rule."))
	fs := lintPrinciples(t, root)
	rel := filepath.Join(prnDir, "p.md")
	for _, id := range []string{"adr-9", prnCondB} {
		if !findingWith(fs, rel, rulePrincipleInheritance, "'"+id+"', which is unresolvable") {
			t.Errorf("%s is not reported as unresolvable: %v", id, rulesOf(fs, rel))
		}
	}
	for _, f := range fs {
		if f.File == rel && strings.Contains(f.Message, "absent") {
			t.Errorf("unresolvable evidence is reported as absent: %s", f.Message)
		}
	}

	// Carried by two shipped intents: ambiguous.
	writeFile(t, root, prnShipped+"/itd-1-a.md", shippedIntent("itd-1", []string{prnCondB}, ""))
	writeFile(t, root, prnShipped+"/itd-2-b.md", shippedIntent("itd-2", []string{prnCondB}, ""))
	fs = lintPrinciples(t, root)
	if !findingWith(fs, rel, rulePrincipleInheritance, "ambiguous") {
		t.Errorf("a condition two shipped intents carry is not reported as ambiguous: %v", rulesOf(fs, rel))
	}
}

// TestWarnRuleDoesNotFailPreflight: the committed configuration arms the four
// rules at the severities the spec declares, and a tree whose only principle
// findings are the untyped count carries no blocker — record-lint fails on
// blockers alone, so the count is reported and never a wall.
func TestWarnRuleDoesNotFailPreflight(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(repoRootFromPackage, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	for rule, want := range map[string]string{
		rulePrincipleUntyped:     severityWarn,
		rulePrincipleClaims:      severityBlocker,
		rulePrincipleInheritance: severityWarn,
		rulePrincipleFalsified:   severityBlocker,
	} {
		rc, ok := cfg.Rules[rule]
		if !ok || !rc.Enabled || rc.Severity != want {
			t.Errorf("the committed config arms %s as %+v, want enabled at %s", rule, rc, want)
		}
	}
	if got := cfg.Rules[ruleRecordSchema].RecordStores["prn"]; got != ".abcd/development/principles" {
		t.Errorf("record_schema declares the prn store at %q", got)
	}

	root := t.TempDir()
	writeFile(t, root, prnDir+"/a.md", "# A\n\n**The rule.** A.\n")
	writeFile(t, root, prnDir+"/b.md", "# B\n\n**The rule.** B.\n")
	fs := lintPrinciples(t, root)
	if n := countRule(fs, rulePrincipleUntyped); n != 2 {
		t.Errorf("two untyped principles drew %d untyped finding(s), want 2", n)
	}
	for _, f := range fs {
		if f.Severity == severityBlocker {
			t.Errorf("an untyped-only store drew a blocker: %+v", f)
		}
	}
}
