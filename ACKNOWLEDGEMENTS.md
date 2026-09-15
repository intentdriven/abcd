# Acknowledgements

abcd stands on ideas, tools, and writing from many sources. This file records them
in three parts: the **development** that built abcd, the **inspirations** that
shaped its design, and the **references** it draws on. Each entry is added in the
same change that lands what it records — the pull request that adopts a pattern,
cites a source in an ADR, or integrates a tool — so the list grows with the work
rather than being reconstructed later. Runtime dependencies are not listed here;
they live in `go.mod` and the licence notices they carry.

## Development

Development of abcd has been assisted by Claude Code (Anthropic). Per-commit
disclosure uses an `Assisted-by:` trailer; the human contributor is the author of
record and is responsible for all AI-assisted output — its correctness, licensing,
and fit for the project. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

External reports sharpen the record, and fix commits credit their reporters with
a `Reported-by:` trailer. [Andy Woods (@andytwoods)](https://github.com/andytwoods)
reported the README's undocumented git requirement and proposed the route-scoped
shape of the Requirements section the README carries
([#496](https://github.com/intentdriven/abcd/issues/496)), and surfaced the
failed-provisioning fallback whose guidance framed Go as a requirement it is not
([#494](https://github.com/intentdriven/abcd/issues/494)).

## Inspirations

Ideas and methodologies that shaped the design — not code abcd depends on.

- **The International Code of Signals (International Maritime Organization)** —
  the maritime flag alphabet whose Alfa, Bravo, Charlie and Delta are encoded
  pixel-for-pixel as abcd's logo (itd-133, `internal/livery`): the halved
  white-and-blue swallowtail, the all-red swallowtail, the five-stripe and the
  three-band flags. The full-size logo is held to the standard's geometry by
  test; the compact variant declares itself an approximation rather than
  claiming a fidelity three rows cannot carry.
- **The NO_COLOR convention (<https://no-color.org>)** — the environment
  variable that asks a program to emit no colour, and specifically its rule
  that the variable counts when *present and not empty*, whatever its value.
  That rule is what abcd's colour ladder implements (itd-112,
  `internal/term`), and the precedence test pins the empty-string case that a
  presence-only reading would get wrong.
- **"20 Must-Know Agentic AI Terms" (Andreas Horn, LinkedIn, 2026)** — the
  practitioner term list whose assessment seeded the terminology crosswalk
  (itd-100, `docs/reference/terminology.md`). Credited as the prompt, not a
  source: every crosswalk citation is to a primary anchor, and the crosswalk's
  admission rule (no single-author coinages, no aggregators) was formulated in
  reaction to it. The post's URL was not recoverable when this entry was
  completed; the credit stands on title, author, platform and year.
- **"AI puts people in a moral crumple zone" (Sylvain Bangma, LinkedIn, 2026,
  <https://www.linkedin.com/posts/sylvain-bangma_ai-puts-people-in-a-moral-crumple-zone-share-7498728574032494592-CnjG>)**
  — the three conditions that hollow out a human in the loop: no transparency
  into what the automation did, no authority to change anything, and a drive
  for efficiency. abcd adopts them as a standing test rather than a metaphor
  (rfc-3), which is what turns Elish's diagnosis into something a design can be
  measured against. The post arrives independently at tiered routing, where
  higher-stakes work goes to people and lower-stakes work to automation, which
  is the shape of abcd's verification ladder (itd-173).
- **Agentic Context Engineering (ACE)** — the append-only-delta model of a
  self-improving instruction record, and the two failure modes it names —
  *brevity bias* and *context collapse* — which itd-81 cites to strike itd-5's
  "shorter by >10%" prompt tiebreak.
- **Amazon "Working Backwards"** — the press-release format of abcd's intents.
- **Architecture Decision Records (MADR)** — the shape of the decision record.
- **CARL (Context Augmentation & Reinforcement Layer, Christopher Kahler,
  MIT)** — the just-in-time rule-injection mechanism (a prompt hook, a JSON
  rule source, recall keywords, dedup, down to the `*<DOMAIN>` star-command)
  that abcd's rules loader re-implements natively with plugin-bundled
  defaults (itd-3, `internal/core/rules`). The record's own comparison is
  "structurally identical": the pattern is adopted, the tool never depended
  on. <https://github.com/ChristopherKahler/carl>
- **ccpm (Claude Code PM, Automaze)** — the markdown spec/task conventions
  (PRD → epic → issue, directory-as-store) that abcd's native spec layer is
  convention-compatible with, and the designated deeper backend of the spec
  seam (ADR-24, ADR-26). <https://github.com/automazeio/ccpm>
- **Citation Style Language (CSL-JSON)** — the bibliography format of the
  confidential-sources design (itd-76), whose reserved `custom` field carries
  the confidentiality metadata.
- **Claude Code's permission and sandboxing model** — the posture adr-42 adopts
  for `abcd guard`: an argument-constraining command pattern is documented as
  fragile agent steering rather than a boundary, an unsound pattern is refused
  outright instead of shipped, and the enforcing control sits at the execution
  layer.
  <https://code.claude.com/docs/en/permissions>
- **Conftest (Open Policy Agent)** — the severity→exit-code convention (`0`
  clean / `1` warnings / `2` any error) the `abcd lint` verb adopts for its
  tri-state exit, taken as vocabulary without adopting the Rego engine (itd-85).
- **CriticGPT (OpenAI)** — the injected-bug construction behind itd-81's
  calibration corpus: natural defects are unlabelled, so ground truth is
  manufactured by reintroducing defects whose class is already known.
- **Cursor's terminal command controls** — the decisive negative precedent
  behind adr-42: a shipped command denylist bypassed via `bash -c`, subshells
  and base64, replaced by an allowlist, which was then bypassed by poisoning an
  allowed command's environment, published as GHSA-82wg-qcm4-fp2w /
  CVE-2026-22708 (identifiers as recorded during the iss-272 investigation; the
  advisory host was unreachable when this entry was written). abcd takes the
  lesson, not the mechanism.
  <https://cursor.com/security>
- **DITA subject scheme maps** — the controlled-vocabulary pattern behind the
  persona registry: a field's legal values live in a dedicated registry file
  and a processor flags unbound values (the `persona_registry` lint rule).
- **Diátaxis (Daniele Procida, CC-BY-SA 4.0)** — the four-type model behind
  the user documentation, and the orientation phrasing the docs folder pages
  adapt in their one-line descriptions — which is why this entry carries the
  author and the licence, not only the idea. <https://diataxis.fr>
- **Dieter Rams's "Weniger, aber besser"** — the less-but-better maxim
  adopted as a guiding principle
  (`.abcd/development/principles/less-but-better.md`): reach for the
  subtraction first.
- **Domain-Driven Design (bounded contexts)** — the surface boundaries.
- **Doorstop** — the suspect-link fingerprint mechanism adopted for intent
  dependency edges (itd-78), and the store-one-direction/derive-the-reverse
  link model the edge schema follows (shared with OpenFastTrace and
  Sphinx-Needs).
- **GEPA (reflective prompt evolution)** — the score → reflect-on-failing-traces
  → minimal-delta → re-score loop that itd-81 adopts as a human-approved manual
  procedure rather than as a library dependency.
- **Go's embedded build info (`runtime/debug.BuildInfo` VCS stamping)** — the
  source of the running binary's vintage that itd-111's staleness detection
  reads (build revision and the `vcs.modified` dirty flag), with its documented
  stamping holes driving the first-class *unknown* outcome.
- **GTFOBins** — the `shell` / `command` function taxonomy, which names the
  class `abcd guard` is exposed to (a binary that spawns a shell or runs a
  command) and, by its privilege-escalation inclusion criterion, demonstrates
  that no curated list covers abcd's threat: `nice`, `setsid`, `stdbuf` grant no
  privilege and are perfect bypasses (adr-42). <https://gtfobins.github.io>
- **gitleaks (Zachary Rice, MIT)** — the canonical aws-access-token prefix
  family the launch scanner's AWS rule deliberately narrows (self-declared
  at `internal/adapter/scanner/patterns.go`), and the full-history secret
  scan CI runs as the authoritative backstop behind abcd's own fast
  pre-push pass. <https://github.com/gitleaks/gitleaks>
- **Homebrew's auto-update-on-use and the `update-notifier` pattern (npm)** — the
  UX grammar itd-111 keeps (cached comparison, a gentle nudge, a one-command
  fix) while rejecting their implicit background network check: abcd implements
  the same grammar over disk-only sources, and the network answers only an
  explicit `--check` (adr-38).
- **git's "behind upstream" notice** — the disk-only precedent itd-111 follows:
  a comparison against locally cached refs, refreshed only by an explicit fetch,
  never a background poll.
- **Karpathy's LLM-wiki gist (Andrej Karpathy, 2026)** — the three-layer
  raw-sources → wiki → schema pattern the `.abcd/memory/` substrate is
  structured on (itd-36, brief `05-internals/07-memory`), and the
  no-accumulation critique of query-time retrieval it answers. The gist
  declares itself "designed to be copy pasted to your own LLM Agent".
  <https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f>
- **The Linux kernel's coding-assistants policy** — the `Assisted-by:` attribution
  model abcd adopts for AI-assisted commits.
- **mattpocock/skills (Matt Pocock, MIT)** — four adaptations: the
  glossary-file format (each term a frontmattered Markdown file with aliases
  and forbidden synonyms) behind the brief's terminology glossary; the
  "grill me" frontier-questioning pattern — an interview that advances by
  asking only what the answers so far cannot settle — adapted as the frontier
  rounds of the brief-creation interview (itd-142); the three-clause ADR
  admission test (hard-to-reverse, surprising-without-context, real
  trade-off) adopted as adr-7's decision gate; and the `to-prd` PRD section
  shape itd-27's silent synthesis phase follows.
  <https://github.com/mattpocock/skills>
- **GitLab's documentation-testing severity policy** — the warn-then-promote
  discipline the writing-style guide's preamble adopts: deterministic prose
  checks may block, heuristics stay advisory, and a rule is promoted to
  blocking only after every existing occurrence in the corpus is fixed.
- **kapa.ai's "Writing documentation for AI" guidance** — the position that
  documentation which works for AI is well-structured human documentation
  (analogised to screen-reader accessibility), which seeded the
  audience-by-placement ratification (adr-53) and the guide's
  self-contained-sections rule.
  <https://docs.kapa.ai/improving/writing-best-practices>
- **OpenAI Codex's sandbox/approval split** — the vocabulary adr-42 borrows for
  naming what a parse layer is: the OS-enforced sandbox is the boundary, the
  approval policy is "a workflow choice layered on top of" it, and the pattern
  engine carries no threat model.
  <https://github.com/openai/codex>
- **PAUL (Plan-Apply-Unify Loop, Christopher Kahler, MIT)** — the
  mandatory-closure loop discipline whose four escalation states itd-1
  lifts into the intent lifecycle, alongside acceptance-criteria-first
  development. The patterns are adopted into abcd's own primitives; the
  tool is never depended on, since two scaffolders fighting over a
  project's structure would be the wrong outcome.
  <https://github.com/ChristopherKahler/paul>
- **Priority inheritance (real-time scheduling)** — the derived-priority rule
  of the intent dependency graph (itd-78): a minor blocker of a major intent
  computes to major.
- **repolinter** — the declarative rule-object schema (`id` / `severity` /
  `where` / `fix` / `policyInfo`) the `abcd lint` rule model adapts as data,
  separate from the evaluator (itd-85). The tool itself is archived and is not a
  dependency.
- **The Rust RFC process** — the required "Prior Art" section on intents.
- **SpecStory** — the hosted session-transcript capture the native history
  store replaced (adr-29), and the named opt-in cloud backend of the history
  seam; the `specstory-import` provenance kind keeps transcripts it captured
  importable. The relationship mirrors ccpm's on the spec seam: a designated
  deeper backend, never a dependency. <https://specstory.com>
- **sudo's `NOEXEC` tag and the sudoers(5) shell-escape statement** — thirty
  years of the same job, and the normative form of adr-42's conclusion:
  restricting users to programs that offer no shell escape "is often
  unworkable", so the answer is an execution-layer control that revokes the
  capability, not a list of the programs that hold it.
  <https://manpages.ubuntu.com/manpages/noble/en/man5/sudoers.5.html>
- **TruffleHog (Truffle Security)** — the optional deeper secret scanner the
  `scan.deep` recommendation keys on when the binary is present
  (`internal/core/ahoy`); integrated as an opt-in engine, never bundled.
  <https://github.com/trufflesecurity/trufflehog>

## References & sources

CSL-JSON: [`.abcd/development/research/references.csl.json`](.abcd/development/research/references.csl.json)

1. Lisanne Bainbridge. 1983. Ironies of automation. *Automatica* 19, 6 (1983),
   775–779. [doi:10.1016/0005-1098(83)90046-8](https://doi.org/10.1016/0005-1098%2883%2990046-8)
2. Shraddha Barke, Michael B. James, and Nadia Polikarpova. 2023. Grounded
   Copilot: How programmers interact with code-generating models. *Proceedings
   of the ACM on Programming Languages* 7, OOPSLA1 (2023), 85–111.
   [doi:10.1145/3586030](https://doi.org/10.1145/3586030)
3. Joel Becker, Nate Rush, Elizabeth Barnes, and David Rein. 2025. Measuring
   the impact of early-2025 AI on experienced open-source developer
   productivity. METR. [arXiv:2507.09089](https://arxiv.org/abs/2507.09089)
4. Colin Bryar and Bill Carr. 2021. *Working Backwards: Insights, Stories, and
   Secrets from Inside Amazon*. St. Martin's Press, New York.
   <https://openlibrary.org/works/OL20924654W>
5. Parmit K. Chilana, Rishabh Singh, and Philip J. Guo. 2016. Understanding
   conversational programmers: A perspective from the software industry. In
   *Proceedings of the 2016 CHI Conference on Human Factors in Computing
   Systems (CHI '16)*, 1462–1472. [doi:10.1145/2858036.2858323](https://doi.org/10.1145/2858036.2858323)
6. Fabrizio Dell'Acqua, Edward McFowland III, Ethan R. Mollick, Hila
   Lifshitz-Assaf, Katherine Kellogg, Saran Rajendran, Lisa Krayer, François
   Candelon, and Karim R. Lakhani. 2023. Navigating the jagged technological
   frontier: Field experimental evidence of the effects of AI on knowledge
   worker productivity and quality. Harvard Business School Working Paper
   24-013. [doi:10.2139/ssrn.4573321](https://doi.org/10.2139/ssrn.4573321)
7. Eric Evans. 2003. *Domain-Driven Design: Tackling Complexity in the Heart
   of Software*. Addison-Wesley, Boston.
   <https://openlibrary.org/works/OL4464385W>
8. Ahmed Fawzy, Amjed Tahir, and Kelly Blincoe. 2026. Vibe coding in practice:
   Motivations, challenges, and a future outlook — a grey literature review.
   In *Proceedings of the 48th International Conference on Software
   Engineering: Software Engineering in Practice (ICSE-SEIP 2026)*, 212–223.
   [doi:10.1145/3786583.3786866](https://doi.org/10.1145/3786583.3786866)
9. Alan R. Hevner, Salvatore T. March, Jinsoo Park, and Sudha Ram. 2004.
   Design science in information systems research. *MIS Quarterly* 28, 1
   (2004), 75–105. [doi:10.2307/25148625](https://doi.org/10.2307/25148625)
10. Ken Huang. 2025. *Secure Vibe Coding Guide*. Cloud Security Alliance.
   <https://cloudsecurityalliance.org/blog/2025/04/09/secure-vibe-coding-guide>
11. Andrej Karpathy. 2023. The hottest new programming language is English.
    Post on X (24 January 2023).
    <https://x.com/karpathy/status/1617979122625712128>
12. Andrej Karpathy. 2025. Post coining the term "vibe coding". X (February
    2025). <https://x.com/karpathy/status/1886192184808149383>
13. Andrej Karpathy. 2026. Post proposing the term "agentic engineering". X
    (February 2026). <https://x.com/karpathy/status/2019137879310836075>
14. Amy J. Ko et al. 2011. The state of the art in end-user software
    engineering. *ACM Computing Surveys* 43, 3 (2011), 21:1–21:44.
    [doi:10.1145/1922649.1922658](https://doi.org/10.1145/1922649.1922658)
15. Oliver Kopp, Anita Armbruster, and Olaf Zimmermann. 2018. Markdown
    architectural decision records: Format and tool support. In *Proceedings
    of the 10th ZEUS Workshop* (CEUR-WS Vol. 2072).
    <https://ceur-ws.org/Vol-2072/paper9.pdf>
16. Alistair Mavin, Philip Wilkinson, Adrian Harwood, and Mark Novak. 2009.
    Easy approach to requirements syntax (EARS). In *Proceedings of the 17th
    IEEE International Requirements Engineering Conference (RE '09)*, 317–322.
    [doi:10.1109/RE.2009.9](https://doi.org/10.1109/RE.2009.9)
17. Bonnie A. Nardi. 1993. *A Small Matter of Programming: Perspectives on End
    User Computing*. MIT Press, Cambridge, MA.
    <https://openlibrary.org/works/OL1923390W>
18. Peter Naur. 1985. Programming as theory building. *Microprocessing and
    Microprogramming* 15, 5 (1985), 253–261.
    [doi:10.1016/0165-6074(85)90032-8](https://doi.org/10.1016/0165-6074%2885%2990032-8)
19. Michael Nygard. 2011. Documenting architecture decisions. Cognitect blog
    (15 November 2011).
    <https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions>
20. Hammond Pearce, Baleegh Ahmad, Benjamin Tan, Brendan Dolan-Gavitt, and
    Ramesh Karri. 2022. Asleep at the keyboard? Assessing the security of
    GitHub Copilot's code contributions. In *Proceedings of the 43rd IEEE
    Symposium on Security and Privacy (S&P 2022)*, 754–768.
    [arXiv:2108.09293](https://arxiv.org/abs/2108.09293)
21. Neil Perry, Megha Srivastava, Deepak Kumar, and Dan Boneh. 2023. Do users
    write more insecure code with AI assistants? In *Proceedings of the 2023
    ACM SIGSAC Conference on Computer and Communications Security (CCS '23)*,
    2785–2799. [doi:10.1145/3576915.3623157](https://doi.org/10.1145/3576915.3623157)
22. Ranjan Sapkota, Konstantinos I. Roumeliotis, and Manoj Karkee. 2025. Vibe
    coding vs. agentic coding: Fundamentals and practical implications of
    agentic AI. [arXiv:2505.19443](https://arxiv.org/abs/2505.19443)
23. Advait Sarkar, Andrew D. Gordon, Carina Negreanu, Christian Poelitz, Sruti
    Srinivasa Ragavan, and Ben Zorn. 2022. What is it like to program with
    artificial intelligence? In *Proceedings of the 33rd Annual Conference of
    the Psychology of Programming Interest Group (PPIG 2022)*. [arXiv:2208.06213](https://arxiv.org/abs/2208.06213)
24. Christopher Scaffidi, Mary Shaw, and Brad A. Myers. 2005. Estimating the
    numbers of end users and end user programmers. In *Proceedings of the 2005
    IEEE Symposium on Visual Languages and Human-Centric Computing (VL/HCC
    '05)*, 207–214. [doi:10.1109/VLHCC.2005.34](https://doi.org/10.1109/VLHCC.2005.34)
25. Donald A. Schön. 1983. *The Reflective Practitioner: How Professionals
    Think in Action*. Basic Books, New York.
    <https://openlibrary.org/works/OL3466056W>
26. Shivani Shukla, Himanshu Joshi, and Romilla Syed. 2025. Security
    degradation in iterative
    AI code generation — a systematic analysis of the paradox. In *Proceedings
    of the 2025 IEEE International Symposium on Technology and Society (ISTAS
    2025)*. [arXiv:2506.11022](https://arxiv.org/abs/2506.11022)
27. U.S. Copyright Office. 2025. *Copyright and Artificial Intelligence,
    Part 2: Copyrightability*. <https://www.copyright.gov/ai/>
28. Gerald M. Weinberg. 1971. *The Psychology of Computer Programming*. Van
    Nostrand Reinhold, New York.
    <https://openlibrary.org/works/OL1958820W>
29. Songwen Zhao, Danqing Wang, Kexun Zhang, Jiaxuan Luo, Zhuo Li, and Lei Li.
    2025. Is vibe coding safe? Benchmarking vulnerability of agent-generated
    code in real-world tasks. [arXiv:2512.03262](https://arxiv.org/abs/2512.03262)
