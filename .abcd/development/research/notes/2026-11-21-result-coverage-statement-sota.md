# A result that says how far it looked: a state-of-the-art check

Date: 2026-11-21. Made as the evidence for itd-2611211024360819. Recency was
judged against this date. Evidence tiers: **primary-verified** means the
source text was read directly; **primary-quoted** means the primary source was
opened and its wording extracted by a fetch, so small transcription slips are
possible; **abstract-only** means only a paper's abstract was readable. No
blog or forum was used. Every source in the table was opened on that date by a
research agent in a session of its own; the session that filed this note did
not open them again. What could not be verified is listed at the end.

## The question

A command-line tool runs many unlike commands. Some are deterministic: they
list leftover branches, lint records, check gates. Some hand the judging to a
language model: a code review, an audit. Should every command's result carry
one short statement, in the same shape, of what this run checked, what the
command never checks and what it could not check this time, so that nobody
reads "nothing found" as "nothing is there"? And if so, what shape, where, and
how is it kept from turning into noise?

## What the evidence says

1. **Build a small native shape with a real seam to SARIF; do not make SARIF
   the shape** (primary-quoted). SARIF 2.1.0 covers the could-not-check part:
   `invocations[].toolExecutionNotifications`,
   `toolConfigurationNotifications` and `executionSuccessful`. It has no field
   for standing exclusions or skipped inputs: `run.artifacts` only "MAY
   contain elements representing all artifacts examined by the tool". The
   checked and never-checks parts would sit in property bags, and the largest
   consumer of the format "will only use the following supported properties".
   For the seam, a Go library covers 2.1.0 and 2.2, and govulncheck already
   offers `-format sarif`. [S1, S2, S3, S10, S20]
2. **Let a check that could not be made change the wording of the verdict**
   (primary-verified for the auditing standard; primary-quoted for SARIF).
   ISA 705 says the auditor "shall modify the opinion" when "unable to obtain
   sufficient appropriate audit evidence" (para 6(b)). The qualified wording
   is "except for the possible effects of the matter(s)" (para 17), with a
   disclaimer of opinion when the effects could be "both material and
   pervasive" (para 9), and the report gives "the reasons for that inability"
   (para 24). SARIF agrees: an error-level notification "SHALL mean that the
   run failed", a consumer "SHALL NOT assume that a failed run contains a
   complete set of analysis results", and `results` is `null`, not an empty
   list, when analysis never began. That gives three outcomes: clear within
   what was checked; clear except for named gaps; no verdict. [S1, S4]
3. **Default to one line of counts per reason, detail on request, nothing
   when empty, and capped lists** (primary-quoted). pytest's skip detail "is
   not shown by default to avoid cluttering the output": the report flag
   defaults to `fE`, and deselection shows only as a count. Semgrep prints
   "Scan skipped:" with a count per reason and points at its verbose flag for
   the list, which another flag caps at 100 entries. CodeQL's status page
   reports the share of files scanned per language. [S5, S6, S7, S25]
4. **Keep the part that never changes out of the repeated text** (primary-
   quoted; contested). Identical text habituates: one study reports "a
   dramatic drop in the visual processing centers of the brain after only the
   second exposure to a warning". Drug-safety alerts are overridden "in 49% to
   96% of cases", and their authors advise that low-severity alerts "can be
   absent or shown nonintrusively". Design matters too: users continued
   through 70.2% of one browser's certificate warnings and a third of
   another's. govulncheck keeps its exclusions as a "Limitations" section in
   its manual. The inference drawn from these, that standing exclusions
   belong in the machine-readable result always and in the text a person
   reads only as a short pointer, expanded when one bears on this run, is the
   research agent's own and is not a tested finding. [S8, S9, S10, S26]
5. **Use closed reason and state codes that keep not applicable, not checked
   and inconclusive apart** (primary-quoted). SARIF's `kind` has
   `notApplicable` and `open` ("insufficient information to decide whether a
   problem exists"), which suits a model's unsure judgement. Semgrep's reason
   codes include `insufficient_permissions`, `exceeded_size_limit` and
   `analysis_failed_parser_or_internal_error`. ASVS says requirements that do
   not apply "must be noted in the report". Test formats carry one reason
   string per skip: TAP's `# SKIP`, the JUnit-style `skipped` element's
   `message`, Go's `skip` action. [S1, S6, S11, S12, S13, S14]
6. **Keep narrow flags, document each as what the default checks, what it
   does not, what the flag adds and what that costs, and let the gap line
   name the flag** (primary-quoted). The manuals of restic, git fsck and rsync
   take that form: restic's default check "does not verify that the actual
   pack files on disk in the repository are unmodified" and one flag adds
   that; git fsck's connectivity-only mode says "Corruption in blob objects
   will not be detected at all"; rsync's checksum flag "can slow things down
   significantly". Semgrep's verbose output names the flag that closes a gap.
   ASVS asks for the level attempted, stated "from the perspective of what
   was included rather than what was not included", and a level can state its
   own weakness: SLSA's first build level is "trivial to bypass or forge".
   [S6, S11, S15, S16, S17, S18]
7. **Treat the effect on over-trust as unproven, and test it with readers**
   (primary-quoted for the review; abstract-only otherwise; contested).
   Over-trust is measured: wrong advice from a decision aid raised wrong
   decisions (risk ratio 1.26 across four pooled healthcare studies), and it
   "cannot be prevented by training or instructions". People assisted by a
   language model "wrote significantly less secure code" yet were "more
   likely to believe they wrote secure code". Continually updated confidence
   information helped 30 instructor pilots, who had "significantly fewer
   icing-related stalls". Against that, explanations "increased the chance
   that humans will accept the AI's recommendation, regardless of its
   correctness", so a statement that looks rigorous may raise trust in a
   clear result. [S19, S21, S22, S23, S24]

## Not worth adopting

- **OSCAL assessment results.** Built for compliance programmes, far beyond
  this tool's scale. [S27]
- **The in-toto test-result predicate (v0.1).** It has no field for tests
  that were not run. [S27]
- **CTRF.** Shaped for tests, with no reason field for a skip on its status
  page. [S27]
- **JUnit-style XML as the native shape.** Its `skipped` element carries only
  a message. [S13]
- **A bespoke shape with no seam.** Rejected because the mapping in point 1
  costs little.

## Sources

| # | Address | What | Quote or field relied on |
| --- | --- | --- | --- |
| S1 | <https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/sarif-v2.1.0-os.html> | SARIF 2.1.0 standard | sections 3.14.15, 3.14.23, 3.20.14, 3.20.21, 3.20.22 and 3.27.9, as quoted |
| S2 | <https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/schemas/sarif-schema-2.1.0.json> | SARIF schema | a notification is "a condition relevant to the tool itself"; `level` of none, note, warning or error; `run.properties` |
| S3 | <https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning> | A consumer's SARIF support | as quoted |
| S4 | <https://www.ifac.org/_flysystem/azure-private/publications/files/ISA-705-Revised_0.pdf> | ISA 705 (Revised) | paragraphs 6(b), 7(b), 9, 17 and 24 |
| S5 | <https://docs.pytest.org/en/stable/how-to/skipping.html>, <https://docs.pytest.org/en/stable/how-to/output.html>, <https://docs.pytest.org/en/stable/example/markers.html> | pytest manual | the skip report flag; "defaults to `fE`"; "1 passed, 3 deselected" |
| S6 | <https://raw.githubusercontent.com/semgrep/semgrep/develop/cli/src/semgrep/target_manager.py>, <https://raw.githubusercontent.com/semgrep/semgrep/develop/cli/src/semgrep/output.py> | Semgrep source | "Scan skipped:"; "Files without read access:"; "Partially analyzed due to parsing or internal Semgrep errors" |
| S7 | <https://docs.semgrep.dev/cli-reference> | Semgrep manual | the flag that caps listed entries; the timeout threshold |
| S8 | <https://scholarsarchive.byu.edu/facpub/9306> | Anderson et al., CHI 2015 (abstract) | as quoted |
| S9 | <https://pmc.ncbi.nlm.nih.gov/articles/PMC1447540/> | van der Sijs et al., JAMIA 2006 | as quoted |
| S10 | <https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck> | govulncheck manual | "Calls to functions made using package reflect are not visible to static analysis." |
| S11 | <https://raw.githubusercontent.com/OWASP/ASVS/master/5.0/en/0x04-Assessment_and_Certification.md> | OWASP ASVS 5.0 | "which Level the organization is attempting to achieve"; "from the perspective of what was included rather than what was not included"; "a summary of all requirements checked" |
| S12 | <https://testanything.org/tap-version-14-specification.html> | TAP 14 | `# SKIP`; "1..0 # skip because English-to-French translator isn't installed" |
| S13 | <https://maven.apache.org/surefire/maven-surefire-plugin/xsd/surefire-test-report.xsd> | JUnit-style schema | the suite's `skipped` count; the `skipped` element's `message` |
| S14 | <https://pkg.go.dev/cmd/test2json>, <https://pkg.go.dev/testing> | Go test documentation | `skip`: "the test was skipped or the package contained no tests" |
| S15 | <https://restic.readthedocs.io/en/stable/045_working_with_repos.html> | restic manual | "does not verify that the actual pack files on disk in the repository are unmodified"; the read-data flag |
| S16 | <https://git-scm.com/docs/git-fsck> | git manual | connectivity-only: "Corruption in blob objects will not be detected at all." |
| S17 | <https://download.samba.org/pub/rsync/rsync.1> | rsync manual | checksum: "can slow things down significantly" |
| S18 | <https://slsa.dev/spec/v1.0/levels>, <https://slsa.dev/spec/v1.0/about>, <https://slsa.dev/spec/v1.2/about> | SLSA (v1.0 retired; v1.2 approved, same section) | "What SLSA doesn't cover"; "SLSA is only one part of a thorough approach to supply chain security." |
| S19 | <https://pmc.ncbi.nlm.nih.gov/articles/PMC3240751/> | Goddard et al., JAMIA 2012 | "risk ratio was 1.26 (95% CI 1.11 to 1.44)"; "position of advice on the screen" |
| S20 | <https://pkg.go.dev/github.com/owenrumney/go-sarif/v3> | A Go SARIF library | v3.3.2, October 2026, Unlicense |
| S21 | <https://www.ebi.ac.uk/europepmc/webservices/rest/search?query=ext_id:21077562%20AND%20src:med&resultType=core&format=json> | Parasuraman and Manzey, Human Factors 2010 (abstract) | as quoted |
| S22 | <https://www.ebi.ac.uk/europepmc/webservices/rest/search?query=ext_id:17240714%20AND%20src:med&resultType=core&format=json> | McGuirl and Sarter, Human Factors 2006 (abstract) | as quoted |
| S23 | <https://arxiv.org/abs/2211.03622> | Perry et al., CCS 2023 (abstract) | as quoted |
| S24 | <https://arxiv.org/abs/2006.14779> | Bansal et al., CHI 2021 (abstract) | as quoted |
| S25 | <https://docs.github.com/en/code-security/code-scanning/managing-your-code-scanning-configuration/about-the-tool-status-page> | CodeQL status page | "Whether the file was successfully extracted" |
| S26 | <https://www.usenix.org/conference/usenixsecurity13/technical-sessions/presentation/akhawe> | Akhawe and Felt, USENIX Security 2013 (abstract) | "over 25 million warning impressions"; "70.2%" |
| S27 | <https://pages.nist.gov/OSCAL/learn/concepts/layer/assessment/assessment-results/>, <https://github.com/in-toto/attestation/blob/main/spec/predicates/test-result.md>, <https://ctrf.io/docs/specification/status> | Formats not adopted | "Identifies the controls actually reviewed by this assessment."; `passedTests`, `warnedTests`, `failedTests`; `skipped`, `pending`, `other` |

## Could not verify

- **SARIF's notification section (3.58) and appendices.** The specification
  page was cut short; the notification field names come from the official
  schema instead.
- **Semgrep's summary in its manual.** The wording comes from source on the
  develop branch. The strings "Partially scanned" and "Some files were skipped
  or only partially analyzed" were not found there.
- **CodeQL.** The page explaining low file-coverage percentages was not
  opened.
- **PCAOB AS 3105.** The fetch was refused; ISA 705 stands in.
- **A static-analysis article in CACM (2018).** The body could not be
  fetched, so it is not used.
- **The abstract-only papers** (S8, S21 to S24, S26). Effect sizes beyond the
  quotes are unchecked, and one paper's participant count is not in its
  abstract.
- **No primary source was found for** a standard set of fields for the effort
  of model-judged work (model class, runs); any study measuring whether a
  list of what was not checked changes how developers read a clear result; or
  any study of one report shape failing across unlike tools (S3 is the
  nearest evidence).
- **The Go SARIF library.** Its importer count is not shown, and its page
  says v3.3.2 is not the module's latest.
- **ASVS.** Fetched from the master branch; the page carries no version
  string.

## Corrections from the fit-challenge

An independent pass challenged the choice of path on 2026-11-21, opening the
sources itself. It agreed that no maintained format carries all three parts,
and corrected five things above.

- **SARIF carries more than finding 1 says.** A result of kind `pass` means a
  rule "was evaluated, and no problem was found", and `notApplicable` means it
  was not evaluated because it does not apply, so what a run looked at has a
  native place, as finding 5 already implies. Only what a tool is known not to
  look at, and how hard a model looked, need a property bag.
- **The seam is for interchange only.** The forge's code scanning lists the
  SARIF properties it uses, and the notifications, the flag for whether a run
  succeeded, the result kind and the run's property bag are not among them
  (S3). An export shown there would carry none of the three parts.
- **OSCAL is the closest existing model of the first two parts.** Its
  assessment results list the controls a review includes and excludes, and the
  subjects a task includes and excludes
  (<https://pages.nist.gov/OSCAL-Reference/models/v1.1.3/assessment-results/json-outline/>).
  "Not worth adopting" dismisses it on scale alone; the reason that stands is
  that nothing was found in it for what could not be checked in one run.
- **Fields for model-judged work are partial, not absent.** OpenTelemetry's
  conventions for generative models name the model asked for and are marked as
  under development; no field for effort was found. The repository already
  records the model and effort asked for on each call, with a closed set of
  reasons a field could not be pinned (itd-166).
- **Finding 6 reads more evenly than its sources.** Of the three manuals,
  rsync's gives the cost, git fsck's a flag that narrows the check, and
  restic's what the default does not verify and the flag that adds it. Between
  them they cover the form; no one of them covers all of it.

The pass could not open SARIF sections 3.8, 3.50 and 3.58, OSCAL's finding
states, or the full list of OpenTelemetry's attributes in their new
repository, and did not find any viewer other than the forge's that shows
SARIF notifications to a person.

## Corrections from the plan review

A plan review of the intent this note serves reported on 2026-11-21, after
the fit-challenge. Two of its findings bear on what this note says.

- The statement cannot be written through the serializer interface the lint
  command has. That interface takes the lint's own result, and its package
  imports the cores of three commands that would need to import it. The
  seam the intent claims is therefore narrower: identifiers that do not
  change, and a mapping onto SARIF written down in the spec for a later
  writer.
- The depth of model-judged work is read from a run's review set (the class
  of each reviewer, how many there are, the effort asked for), and not from
  the record of each call, which stays with the technical facilitator.

## Corrections from the final review

A final review before landing reported the same day. Two of its findings
correct the section above.

- The depth of model-judged work is read from what a run's state records
  for each judging run, and not from the review set. The set is fixed when a
  run starts: a reviewer whose class holds the model that wrote the work is
  moved to another class, a fix round runs the reviewers again, and the
  judge of a round and the drain's judge are not in the set.
- The serializer's package imports the docs lint's core and the outbound
  scanner directly, and reaches the cores of the launch preview and the
  readiness gate through them. The conclusion stands for all four.
