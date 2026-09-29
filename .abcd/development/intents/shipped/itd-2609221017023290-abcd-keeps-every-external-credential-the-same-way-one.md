---
id: itd-2609221017023290
slug: abcd-keeps-every-external-credential-the-same-way-one
spec_id: spc-2609221017544877
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-63]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609081951381895, itd-2609061543533170, itd-6]
related_adrs: [adr-2609221017021499]
---

# abcd keeps every external credential the same way

## Press Release

> **One credential store, three homes the person chooses once, and every adapter reads through it; no secret ever enters the harness or the repository.**
>
> "OpenRouter today, a hosting provider next month, my transcript cloud after that, and each one wanted a key somewhere," said a product thinker. "Now abcd asks me once per service where the secret should live, explains what it unlocks first, and every part of abcd reads it the same way. Nothing lands in the harness's settings. Nothing lands in the repo."

## Why This Matters

The first adapter to need a secret was given its own walkthrough and its own file; the second was about to be. The record's rule is one canonical primitive, and a secret is the last thing to have three copies of. Ruled 2026-09-22 (adr-2609221017021499): one store, three homes, one reader, scanned before written.

## Mechanism

We expect one store with one reader to make every secret abcd holds auditable in one place and impossible to commit by accident, because the only write path runs the scanner and the only read path is named; shown wrong if a secret is found in a harness file or a tracked path after this ships, or if an adapter is found reading one another way.

## Scope Conditions

- Holds on macOS with the Keychain and on Linux with a secret service; a platform with neither offers the two other homes and says why. <!-- cond: cond-2609221017547155 -->

## What's In Scope

- **The store** (`internal/core/credential`): `Resolve(name)` for every adapter; `Set(name, home, value)` used only by the walkthrough; homes `external` (a pointer to an existing tool's configuration or an environment variable name), `abcd` (`~/.abcd/credentials.json`, mode 0600), `keychain` (the platform keychain under abcd's service name).
- **The walkthrough** at `ahoy`, through itd-63's explain-then-install mode: what the credential unlocks, what works without it, then the three homes with the keychain recommended in prose, then a verification call the calling adapter supplies.
- **Refusals**: a name that resolves to nothing refuses naming the walkthrough; a write that would land in a tracked path is refused; the write path runs the secret scanner.
- **The readers**: the API adapter, the site setup and any later hook resolve by name; a review finding of any other read is a defect.
- **The record**: the run record names which credential names a run used, never a value.

## What's Out of Scope

- Rotating or expiring credentials.
- Sharing a credential between machines.
- Any credential the host itself holds for its own model.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221017021499 records the vocabulary rulings it rests on):

1. One store, three homes, one reader, scanned before written (adr-2609221017021499).
2. The keychain is recommended in prose; the choice is the person's.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** an adapter needing a credential that is not set, **when** it resolves the name, **then** it refuses naming the walkthrough, and no unauthenticated call is made.
- **Given** the walkthrough accepted for a service, **when** it runs, **then** it explains what the credential unlocks and what works without it, offers the three homes with the keychain recommended in the prose above the choice, stores the value in the chosen home, and verifies with the adapter's own call.
- **Given** any home, **when** the tree and the harness's settings are inspected afterwards, **then** neither carries the value; a write that would land in a tracked path is refused and the write path runs the scanner.
- **Given** the API adapter and the site setup, **when** they read their credentials, **then** both call the store by name, and a test walks the adapters for any other read.
- **Given** a run, **when** its record is read, **then** it names the credential names used and no value.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-ebf7d171b544 -->
Fidelity review — receipt rcp-ebf7d171b544 (verifier intent-auditor (autonomous run A, lane audits12) claude-fable-5-1).

Provenance: intent-auditor (autonomous run A, lane audits12)@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:6e9160f5189342c4d26eb9c54cbfc77bd3949c40c312c4c13af6576db25d1eb2
Input attestations: diff:38301e724..5923c49a3 (feat/credential-store, merged into integ/land-14) plus 278e266d8, f16941a3b, 89e77b7e6, 03374c1b2@-; tree:baf6f8443 (origin/main, the audit's BASE; go test ./internal/core/credential/ -count=1: ok, 41 tests)@-;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Store.Resolve returns a notSetError that names the walkthrough (store.go:104-110, tested at store_test.go:182); the API adapter resolves the key before complete() and refuses on ErrNotSet with no call (call.go:63-67, 83-87; connect_test.go:173 asserts zero calls); the site setup's host stage stops at no_credential without contacting the provider (setup.go:706-710; setup_test.go:257 asserts an empty call log).
  evidence: internal/core/credential/store.go:109 — "is not set on this machine%s; `%s` explains what it unlocks and stores it"
  evidence: internal/core/credential/store_test.go:182 — "func TestAnUnsetNameRefusesNamingTheWalkthrough(t *testing.T) {"
  evidence: internal/core/oracle/call.go:86 — "which is not set on this machine, so no call is made"
  evidence: internal/core/oracle/connect_test.go:173 — "if p.calls.Load() != 0 {"
  evidence: internal/core/site/setup.go:709 — "credential on this machine, so the host was not contacted"
  evidence: internal/core/site/setup_test.go:257 — "if n := len(h.host.CallLog()); n != 0 {"
- ac-2 — MET_WITH_CONCERNS: Service.Explain gives what it unlocks, what works without it, HomesProse (the keychain recommended in prose, never marked) and the three homes (walk.go:22-52; store_test.go:484); Walk verifies with the adapter's own call and only then calls Set (walk.go:114-117; store_test.go:444); the CLI and the plugin page are wired (ahoy_credential.go:189, 116; commands/ahoy.md:462). CONCERN: the person's choice of home is a --home flag on the CLI and the host's question tool on the plugin page (ahoy_credential.go:132; commands/ahoy.md:483-484); the CLI never asks, though spc-2609221017544877 scope 2 says 'the CLI asks on the terminal' and the press release says abcd 'asks me once per service'.
  evidence: internal/core/credential/walk.go:22 — "const HomesProse = "Where the credential lives is your choice of three, made once. The platform keychain is " +"
  evidence: internal/core/credential/walk.go:43 — "func (s Service) Explain() []string {"
  evidence: internal/core/credential/walk.go:114 — "if err := s.Verify(ctx, value); err != nil {"
  evidence: internal/core/credential/walk.go:117 — "changed, err := Set(home, s.Name, c)"
  evidence: internal/core/credential/store_test.go:444 — "func TestTheWalkthroughVerifiesBeforeItStores(t *testing.T) {"
  evidence: internal/core/credential/store_test.go:484 — "func TestTheWalkthroughExplainsFirst(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_credential.go:132 — "cmd.Flags().StringVar(&home, "home", "", "where the credential lives: external"
  evidence: commands/ahoy.md:483 — "ask the technical facilitator for the home through"
  evidence: .abcd/development/specs/closed/spc-2609221017544877-abcd-keeps-every-external-credential-the-same-way-one.md:17 — "the CLI asks on the terminal, the plugin page through the host's question tool (criterion 2)"
- ac-3 — MET_WITH_CONCERNS: After a Set in each home no file under the home, .claude/settings.json included, carries the value except the owner-only abcd file (store_test.go:332); the abcd home's write is refused when ~/.abcd lies inside a git working tree and a pointer at a file inside one is refused (store.go:223; store_test.go:201); the index write runs scanner.ScanText and a finding refuses it (store.go:464, 481; store_test.go:312). CONCERN: the scanner runs over the index alone; credentials.json, the file that holds the value, is written unscanned by construction (credential.go:237), and only the abcd home's value write is refused inside a working tree: the index (names, homes, pointers) is written there after its scan, a narrowing ruled at review (278e266d8) and stated on the plugin page (commands/ahoy.md:488-489), so 'the write path runs the scanner' and 'a write that would land in a tracked path is refused' hold for the value, not for every write.
  evidence: internal/core/credential/store.go:223 — "if c.Home == HomeABCD && workingTreeAbove(home, ".abcd") != "" {"
  evidence: internal/core/credential/store.go:481 — "findings := scanner.ScanText(string(body), scanner.Identity{}, scanner.DefaultPatterns(), nil, IndexFileName)"
  evidence: internal/core/credential/credential.go:237 — "if err := fsutil.WriteFileAtomicInRoot(dir, StoreFileName, append(body, '\n'), 0o600); err != nil {"
  evidence: internal/core/credential/store_test.go:201 — "func TestAValueIntoAWorkingTreeIsRefused(t *testing.T) {"
  evidence: internal/core/credential/store_test.go:237 — "func TestAHomeThatIsAWorkingTreeKeepsTheOtherHomes(t *testing.T) {"
  evidence: internal/core/credential/store_test.go:312 — "func TestTheIndexWriteRunsTheScanner(t *testing.T) {"
  evidence: internal/core/credential/store_test.go:332 — "func TestNeitherTheTreeNorTheHarnessCarriesTheValue(t *testing.T) {"
  evidence: commands/ahoy.md:488 — "The abcd home is refused when `~/.abcd` lies inside a git"
- ac-4 — MET_WITH_CONCERNS: The API adapter resolves p.Key through a credential.Source that defaults to credential.UserStore() (call.go:76-91) and the site setup's host stage resolves adapter.CredentialName() the same way (setup.go:702-706); TestEveryReaderGoesThroughTheStore walks every non-test .go file under cmd/ and internal/ for a direct store, keychain or secret-shaped-environment read and passes at BASE (readers_test.go:34-83, run: ok). CONCERNS: APIConfig.Call has no production caller at BASE, declared in its own header as awaiting spc-2609251028149555 (call.go:10-13; route.go:39-42), so the API adapter's store read runs only under test while the site setup's is live; and the walk is a line-regex drift grep that its own comment says is 'not an evasion gate' (readers_test.go:30), so an aliased import or a name held in a variable passes it.
  evidence: internal/core/oracle/call.go:83 — "key, err := creds.Resolve(p.Key)"
  evidence: internal/core/oracle/call.go:81 — "creds = credential.UserStore()"
  evidence: internal/core/site/setup.go:706 — "token, err := src.Resolve(adapter.CredentialName())"
  evidence: internal/core/credential/readers_test.go:34 — "func TestEveryReaderGoesThroughTheStore(t *testing.T) {"
  evidence: internal/core/credential/readers_test.go:30 — "// evasion gate: a new reader written the obvious way fails here, naming the"
  evidence: internal/core/oracle/call.go:10 — "// No delegating verb dispatches through it yet: sending a step whose route"
  evidence: internal/surface/cli/route.go:41 — "// handed to the verbs yet: a route resolved to a provider would name a leg no"
- ac-5 — MET_WITH_CONCERNS: CallRecord carries Credential, the name of the credential the call used and never a key (call.go:33-36, set at call.go:69), the route receipt carries it as provider_call (receipt.go:79-86), and HostOutcome.Credential names the credential the host stage resolved (setup.go:158-160, 701); the tests assert the name is present and the value absent from the marshalled record (call_test.go:199-221, 112-113; credential_service_test.go:22-44). CONCERN: at BASE nothing in production calls APIConfig.Call or ReceiptRoute.WithCall (call.go:10-13), so a route receipt's provider_call is always null in a real run and the site setup's HostOutcome is the only run record that names a credential today.
  evidence: internal/core/oracle/call.go:36 — "Credential string `json:"credential,omitempty"`"
  evidence: internal/core/oracle/call.go:69 — "rec.Credential = p.Key"
  evidence: internal/core/oracle/receipt.go:79 — "ProviderCall *CallRecord `json:"provider_call"`"
  evidence: internal/core/site/setup.go:701 — "Credential: adapter.CredentialName()}"
  evidence: internal/core/oracle/call_test.go:199 — "func TestTheReceiptCarriesTheProviderCall(t *testing.T) {"
  evidence: internal/core/oracle/call_test.go:113 — "t.Fatal("the record carries the key")"
  evidence: internal/core/site/credential_service_test.go:41 — "t.Fatal("the result carries the credential's value")"
  evidence: internal/core/oracle/call.go:12 — "// is spc-2609251028149555's (AC 3). Until then the setup's verification call"

Gap audit:
- honoured:
  - One store, three homes, one reader by name: Store(home).Resolve routes the abcd, keychain and external homes through one function
    evidence: internal/core/credential/store.go:88 — "func Store(home string) Source { return store{home: home} }"
    evidence: internal/core/credential/store_test.go:140 — "func TestStoreResolvesEveryHome(t *testing.T) {"
  - A name that resolves to nothing refuses naming the walkthrough, and no unauthenticated call is made
    evidence: internal/core/credential/store.go:109 — "explains what it unlocks and stores it"
    evidence: internal/core/oracle/connect_test.go:173 — "if p.calls.Load() != 0 {"
  - The keychain is recommended in the prose above the choice and never as a marked option
    evidence: internal/core/credential/walk.go:22 — "The platform keychain is"
    evidence: internal/core/credential/store_test.go:484 — "func TestTheWalkthroughExplainsFirst(t *testing.T) {"
  - The value is verified with the adapter's own call before it is stored, and the walkthrough's result never carries it
    evidence: internal/core/credential/walk.go:114 — "if err := s.Verify(ctx, value); err != nil {"
    evidence: internal/core/credential/walk.go:54 — "// WalkResult is what a walkthrough did. It never carries the value."
  - Nothing lands in the harness's settings or the repository: no file under the home but the owner-only abcd file carries the value after a Set in any home
    evidence: internal/core/credential/store_test.go:337 — "for _, p := range []string{".claude/settings.json", "work/repo/.claude/settings.json", "work/repo/README.md"} {"
  - The keychain value never reaches an argv, and the tool runs from a fixed system path
    evidence: internal/core/credential/keychain.go:9 — "// The value never reaches an argv, which a process listing shows: security"
    evidence: internal/core/credential/store_test.go:361 — "func TestTheKeychainValueNeverReachesAnArgv(t *testing.T) {"
  - The record names the credential name and no value
    evidence: internal/core/oracle/call.go:33 — "// Credential is the name of the credential the call used, never its"
    evidence: internal/core/site/setup.go:158 — "// Credential is the name of the credential the stage resolved, never"
  - Wired on both front doors: the CLI sub-verb and the plugin page
    evidence: internal/surface/cli/ahoy_credential.go:68 — "func newAhoyCredentialCommand(asJSON *bool) *cobra.Command {"
    evidence: commands/ahoy.md:462 — "## `credential` — the credential store's walkthrough"
- diverged:
  - 'The write path runs the scanner' (ac-3; adr-2609221017021499 ruling 4 'on any file it touches'): delivered over the index alone; credentials.json, which holds the value, is written unscanned by construction
    evidence: internal/core/credential/store.go:476 — "// scanIndex runs the secret scanner over the index's bytes before they are"
    evidence: internal/core/credential/credential.go:237 — "fsutil.WriteFileAtomicInRoot(dir, StoreFileName, append(body, '\n'), 0o600)"
  - 'A write that would land in a tracked path is refused' (ac-3): delivered for the abcd home's value only; the index is written inside a working tree after its scan, a narrowing ruled at review (278e266d8) and documented
    evidence: internal/core/credential/store.go:218 — "// The abcd home is the one home that writes a value under ~/.abcd, so it"
    evidence: commands/ahoy.md:489 — "working tree (the keychain and an external home stay open"
  - 'abcd asks me once per service where the secret should live' and spec scope 2 'the CLI asks on the terminal' (ac-2): the CLI takes the home as a --home flag and never asks; only the plugin page asks, through the host's question tool
    evidence: internal/surface/cli/ahoy_credential.go:132 — "cmd.Flags().StringVar(&home, "home", "","
    evidence: commands/ahoy.md:484 — "your question tool; never ask for the value, and never pass it yourself: give"
    evidence: .abcd/development/specs/closed/spc-2609221017544877-abcd-keeps-every-external-credential-the-same-way-one.md:17 — "the CLI asks on the terminal"
  - 'The API adapter ... resolve by name' and 'the run record names which credential names a run used' (ac-4, ac-5): the provider call and its receipt entry have no production producer at BASE; both are reachable only from tests until provider dispatch (spc-2609251028149555) lands
    evidence: internal/core/oracle/call.go:10 — "// No delegating verb dispatches through it yet: sending a step whose route"
    evidence: internal/surface/cli/route.go:46 — "var machineConnections = func() oracle.Connections { return oracle.NoConnections{} }"
- missing:
  - A real keychain round trip: the keychain home's write and read have only run against the test binary's fake, on macOS and Linux alike; the real security/secret-tool round trip is owed (iss-2609281654467661, open, deferred past v0.11.0)
    evidence: internal/core/credential/store_test.go:24 — "// No test here touches the real keychain: the keychain home runs a fake,"
    evidence: .abcd/work/issues/open/iss-2609281654467661-the-macos-keychain-home-s-write-and-read-have-never-run.md:12 — "deferred_after: "v0.11.0""

Scope-condition dispositions:
- cond-2609221017547155 — narrowed: The no-keychain branch holds and is tested: locateKeychain refuses on a GOOS with no tool or a missing binary with errKeychainAbsent naming the external and abcd homes (keychain.go:44-69; store_test.go:380). The 'holds on macOS with the Keychain and on Linux with a secret service' half is exercised only through the test binary's fake of security and secret-tool; the real tools have never been run, which the lane captured and deferred (iss-2609281654467661).
  narrowing: Holds for a platform with neither tool (the two other homes are offered and the refusal says why), and for macOS and Linux only as far as the fake keychain's emulation of the security and secret-tool argv/stdin contract goes; a real round trip on either platform is unexercised (iss-2609281654467661, open, deferred past v0.11.0).
  evidence: internal/core/credential/keychain.go:45 — "var errKeychainAbsent = errors.New("credential: the keychain home needs the platform keychain's tool " +"
  evidence: internal/core/credential/keychain.go:57 — "switch runtime.GOOS {"
  evidence: internal/core/credential/store_test.go:380 — "func TestAPlatformWithoutAKeychainOffersTheOtherHomes(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2609281654467661-the-macos-keychain-home-s-write-and-read-have-never-run.md:16 — "The macOS keychain home's write and read have never run against the real security tool."
<!-- abcd-review-end receipt=rcp-ebf7d171b544 -->

## Grounds

- pursued: three adapters are about to be built and each would otherwise invent its own store; we expect one store to be the one place a secret is audited; shown wrong if a secret is found in a harness file or a tracked path, or an adapter reads one another way
