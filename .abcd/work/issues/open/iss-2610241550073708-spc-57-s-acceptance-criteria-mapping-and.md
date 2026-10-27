---
schema_version: 1
id: "iss-2610241550073708"
slug: "spc-57-s-acceptance-criteria-mapping-and"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b09 reading iss-2608301620341037"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/closed/spc-57-the-reasoning-behind-what-was-pursued-no.md"
remedy: "Waits on the itd-2609150819439571 planning interview (M25): in the correction iss-2608301620341037 adds to spc-57, also say which named tests no longer exist and that the promote and resolve refusals are parked by dcn-2609090000000001, pointing at the test files rather than listing names, without editing the closed text; proven by record-lint green."
---

spc-57's acceptance-criteria mapping and Tests section name six tests that no longer exist: TestCaptureReaderAcceptsGroundsKey and TestRecordSchemaFlagsOutOfVocabularyGrounds, replaced by 8a97c281e the day before the spec closed, and TestPromoteRefusesWithoutGrounds, TestPromoteWithoutGroundsWritesNothing, TestResolveRefusesWithoutGrounds and TestCapturePromoteMissingGroundsExit2, removed by 667a57f4f when dcn-2609090000000001 parked the grounds refusals; the mapping still says promote refuses without grounds.
