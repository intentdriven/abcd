---
id: itd-2611012048006057
slug: macos-ran-out-of-application-memory-on-a
spec_id: spc-2611020732419632
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_issues: [iss-2611011737317454]
origin: extracted-from-record
production_mode: hand-written
---

# abcd's load gate waits for memory as it waits for load

## Press Release

> A preflight never starts on a machine that is out of memory. When several agent sessions build, review and test on one machine at once, abcd's load gate already waits for the processor load to fall and refuses when the disk is nearly full; now it also reads how much memory pressure the machine is under. When the system reports memory pressure at or above the floor the person set (macOS's own pressure level, or on Linux the share of recent time tasks stalled waiting for memory), a preflight or a gated test run waits at the gate, bounded as the load wait is, instead of starting a test stage that would push the machine past its memory and stall every session on it. The wait says what it is waiting for, and a reading abcd cannot take lets the work through and says so, so the floor never wedges a run.

## Why This Matters

Graduated from `iss-2611011737317454`. On 2026-11-01, at about 17:32, a 128 GB, 16-core development machine ran out of application memory while several agent sessions drained, built and reviewed in parallel, with only this person's sessions active. The machine's counters showed about 155 GiB written to swap over its five days of uptime, so it had run past its memory before. abcd's load gate reads the processor load, the process table and free disk, but never memory, so nothing in abcd held a test stage back while memory was short. The person ruled on 2026-11-02 that the gate waits rather than refuses below the floor, that the floor is set by macOS's own memory-pressure level, that Linux is guarded from the start through the kernel's pressure stall information, that waiters admitted after the pressure falls start one at a time with a settle period between them, and that attributing a spike to a process belongs with the intent that records how each preflight behaved (itd-2610262140595217).

## Mechanism

We expect a gate that waits while the system reports memory pressure at or above its floor (macOS's warning level, or Linux's stalled share) to stop a new test stage from starting into a memory shortage, because the pressure level is the signal the operating system itself acts on before it runs out, and the stages that use the most memory (race-enabled test binaries, several at once) are the ones the gate already admits one by one; waiting costs a stage minutes where starting it costs every session on the machine.

## Scope Conditions

- The person's own macOS or Linux machine, where the system reports memory pressure (macOS's pressure level; on Linux, kernel 4.20 or later, `/proc/pressure/memory`); on any other platform, or where the reading fails, the floor is reported as not checked and the gate admits as it does today. <!-- cond: cond-2611020732414692 -->
- Runs that pass abcd's load gate: `make preflight` and the command form `abcd implement load --wait -- <command>`; a test an agent runs outside the gate is not held, as iss-2610241130157455 settled. <!-- cond: cond-2611020732413735 -->
- One person's machine shared by that person's sessions and accounts; the floor is a machine setting in `~/.abcd.noindex/config.json`, never read from a repository file. <!-- cond: cond-2611020732419293 -->

## Acceptance Criteria

- Given the system reports memory pressure at or above the floor when a preflight reaches the load gate, when the gate decides, then it waits rather than admitting the test stage, re-reads the pressure before it opens, and names the pressure level and the floor in its wait line.
- Given the pressure falls below the floor while the gate waits, when the gate next checks, then it admits the stage as it would with no floor.
- Given the pressure stays at or above the floor until the gate's own time limit, when the limit passes, then the gate ends as its load wait ends today, naming memory pressure as what it waited for.
- Given the pressure level cannot be read (another platform, or the read fails), when the gate decides, then it admits the stage and says once that the memory floor was not checked, and why.
- Given no floor is set in the person's machine configuration, when the gate decides, then it uses abcd's default floor (macOS's warning level; on Linux, 10% of time stalled over the last 10 seconds), and given a floor set in a repository's `.abcd/config.json`, then that setting is ignored and named as ignored.
- Given `abcd implement load` without `--wait` (the load check), when memory pressure is at or above the floor, then the check reports it as a warning and never fails.
- Given several runs wait at the gate for memory pressure to fall, when it falls, then they are admitted one at a time in queue order, and after each admission the gate waits a settle period before reading memory again for the next waiter, so a stage's own memory use shows before another starts.
- Given a Linux machine whose `/proc/pressure/memory` reports a stalled share at or above the floor, when a preflight reaches the gate, then it waits as on macOS, and given a kernel without that file, then the floor is reported as not checked and the stage is admitted.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-ca6448963203 -->
Fidelity review — receipt rcp-ca6448963203 (verifier intent-auditor claude-agent-sdk (model id not stated in session)).

Provenance: intent-auditor@claude-agent-sdk (model id not stated in session) · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:14855f908d3648453d71afd0afefa6ee9f1dd81111c012545a4bfa84872bd8b1
Input attestations: diff:4b0daa18ee376a47763a41ced0e80b7baa318e8a..694682652b586cde8375f7f128af6de967e75d38 (build/run-2611030127507347-lane-1)@sha256:b8494e7099a9c73a5f227d58a6ae140f565ccccf56e7f171f3ccb70656419a4e; intent:.abcd/development/intents/planned/itd-2611012048006057-macos-ran-out-of-application-memory-on-a.md@sha256:b177dea27e51d193d9488f0083927d8cdd48fdf450621a4ec72e464356bbdcc0; spec:.abcd/development/specs/open/spc-2611020732419632-macos-ran-out-of-application-memory-on-a.md@sha256:517c2f8f5aadde705ac3219083281dd9d848cf964c7b9566fe01950277da5a82; implementer-report:.abcd/.work.local/run/run-2611030127507347/lane-1/report.md@sha256:63319ddf1208c6d754b5480a448cde246a00deed56a8f34ae1a8f6e298d3939b; fix-report:.abcd/.work.local/run/run-2611030127507347/lane-1/validate/round-1/fix/report.md@sha256:0ea03bccd8f10589ffb8413e1c2568516610c171105c989b0415cd47e536b971; request:.abcd/.work.local/run/run-2611030127507347/lane-1/validate/round-2/intent-auditor/request.md@sha256:56ddcd4c1ac180d41577c5a513a9b200c37cd516df22bac14b69d6826f3cade1;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: waitLook reads memory at every look, and a reading at or above the floor adds WaitOnMemory to the shut conditions, so the gate waits. Memory is read again at the re-check after the jitter (take=true), and memory.admit runs under the hold flock before the hold is taken. The one-slot race shortcut, which takes no look, now reads memory and judges admission before it opens. The wait line comes from MemoryCheck.WaitText, which names both the level and the floor. make preflight passes through this waiter (Makefile:396 and :424). The tests for these paths passed when I re-ran them on HEAD 694682652.
  evidence: internal/core/implement/load_wait.go:993 — "if mem.look(root, res, t, self, ancestor) { shut = append(shut, WaitOnMemory) }"
  evidence: internal/core/implement/load_wait.go:612 — "if mem.look(root, &res, t, self, ancestor) || mem.admit(root, &res, t, self, ancestor) {"
  evidence: internal/core/implement/load_memory_test.go:206 (TestWaitMemoryRereadBeforeOpen, PASS) — "want memory, 4 and 4 (free, risen at the re-check, free, open)"
  evidence: internal/core/implement/load_memory_test.go:179 (TestWaitMemoryShutsAtFloor, PASS) — "said.Memory.WaitText() != "memory pressure warn (floor warn)""
  evidence: internal/surface/cli/implement_load_memory_test.go:47 (TestImplementLoadWaitNamesMemoryPressure, PASS) — "waiting for capacity: memory pressure warn (floor warn)"
  evidence: Makefile:396 — "go run ./cmd/abcd implement load --wait --timeout 12h -- $(PREFLIGHT_STAGES_MAKE)"
- ac-2 — MET: When a reading falls below the floor, memory.look returns false. If no settle episode is in force, nothing else is added, and the gate opens through its usual path: it takes the hold and reports Still empty, exactly as it would with no floor. A machine that never comes under pressure writes no record and behaves exactly as before. The ten-minute episode is the ac-7 settle, which the intent itself asks for after pressure falls.
  evidence: internal/core/implement/load_memory_test.go:221 (TestWaitMemoryOpensWhenItFalls, PASS) — "res.Status != WaitOpen || res.Hold == nil || len(res.Still) != 0"
  evidence: internal/core/implement/load_memory_test.go:391 (TestWaitMemoryNoRecordWithoutPressure, PASS) — "a wait that never saw pressure wrote the admission record"
  evidence: internal/core/implement/load_memory_test.go:376 (TestWaitMemoryEpisodeEnds, PASS) — "ten minutes after the pressure was last seen, an arrival opens without a settle, and writes nothing."
- ac-3 — MET: Memory is one of the shut conditions, so a deadline that passes while memory still holds the gate gives WaitTimedOut with res.Still = shut, the same status a load wait ends with. The CLI's timeout line lists the conditions through waitConditions, which renders memory by its WaitText. The core test and the CLI test (text and JSON) pass.
  evidence: internal/core/implement/load_memory_test.go:236 (TestWaitMemoryTimeoutNamesMemory, PASS) — "res.Status != WaitTimedOut || !slices.Equal(res.Still, []string{WaitOnMemory}) || res.WaitedS < 120"
  evidence: internal/surface/cli/implement_load.go:796 — "case implement.WaitOnMemory: if m := res.Memory; m != nil { parts = append(parts, m.WaitText())"
  evidence: internal/surface/cli/implement_load_memory_test.go:77 (TestImplementLoadWaitTimesOutOnMemory, PASS) — "func TestImplementLoadWaitTimesOutOnMemory(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: When the memory read fails, memory.look notes memoryNotChecked(err), which gives the reason, and shuts nothing. The note is deduplicated (memoryGate.note) and printed once on stderr. The tests prove this on macOS and Linux, where the load can be read. Concern: on another platform (for example FreeBSD), the load reader also returns ErrUnsupported, so at the first gate waitLook returns WaitUnchecked before it reaches mem.look. The gate does admit, as the criterion requires. But what it prints is the gate's general reason ('abcd reads load and processes on macOS and Linux only ... opening without judging'), not a line saying the memory floor was not checked. Only the load check (checkMemory runs before the load read) and the one-slot race shortcut give the memory-specific line there. TestWaitMemoryUnreadableAdmits fakes a plan9 memory reader beside a readable load, so it does not exercise the real other-platform path.
  evidence: internal/core/implement/load_memory.go:476 (memoryGate.look) — "m, err := g.read() if err != nil { res.Memory = nil g.note(res, memoryNotChecked(err)) return false }"
  evidence: internal/core/implement/load_memory_test.go:250 (TestWaitMemoryUnreadableAdmits, PASS) — "want := "the memory floor was not checked: abcd reads memory pressure on macOS and Linux only, and this is plan9" if strings.Count(res.MemoryNote, want) != 1 {"
  evidence: internal/surface/cli/implement_load_memory_test.go:106 (TestImplementLoadWaitSaysMemoryWasNotChecked, PASS) — "abcd implement load --wait: the memory floor was not checked: /proc/pressure/memory is absent"
  evidence: internal/core/implement/load_wait.go:876-880 — "if errors.Is(readErr, machineload.ErrUnsupported) { res.Status = WaitUnchecked res.Reason = "abcd reads load and processes on macOS and Linux only, and this is " + termsafe.Sanitize(runtime.GOOS) + "; opening without judging" return nil, true, crossedLook{}, nil"
- ac-5 — MET: When no key is set, the defaults apply: warn on macOS and 10 on Linux. The floor is read only from the machine layer (layered.MachineRoots), and a machine key such as critical or 25 is honoured. repoMemoryKeys reads the checkout's .abcd/config.json only to find either key. When it finds one, the note names the key as ignored, and that value is never used. The note appears once on the waiter's stderr and on the check's 'memory check' line.
  evidence: internal/core/implement/load_memory.go:67-68 — "DefaultMemoryFloorMacOS = machineload.MemoryWarn DefaultMemoryFloorLinux = 10"
  evidence: internal/core/implement/load_memory.go:153 — "notes = append(notes, fmt.Sprintf("%s in %s %s ignored: the memory floor is a machine setting, read only from %s, never from a repository","
  evidence: internal/core/implement/load_memory_test.go:87 (TestMemoryFloorDefaults, PASS) — "floor = %+v, want warn and 10 from the defaults"
  evidence: internal/core/implement/load_memory_test.go:134 (TestMemoryFloorRepoKeyIgnored, PASS) — "memoryFloorAt = %+v, %v; a repository's key moved the floor"
- ac-6 — MET: CheckLoad sets LoadResult.Memory and MemoryNote beside the disk finding and leaves Status alone. The text report prints a 'MEMORY PRESSURE WARNING' line when the reading is at the floor. The CLI test checks that the check exits 0 with memory at the floor.
  evidence: internal/core/implement/load.go:218 — "res.Memory, res.MemoryNote = checkMemory(req.ReadMemory, req.Home, req.RepoRoot)"
  evidence: internal/core/implement/load_memory_test.go:437 (TestCheckLoadMemoryWarns, PASS) — "if res.Status != LoadOK || res.Memory == nil || !res.Memory.AtFloor {"
  evidence: internal/surface/cli/implement_load_memory_test.go:151 (TestImplementLoadCheckWarnsOnMemoryAndExitsZero, PASS) — "func TestImplementLoadCheckWarnsOnMemoryAndExitsZero(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: A waiter that sees pressure writes pressure_at to ~/.abcd.noindex/load-memory-admit.json. For the next ten minutes, any other waiter's admission younger than WaitMemorySettle (2 minutes) keeps a waiter shut on memory. memory.admit writes admitted_at under the record's lock before the hold is taken, so no two waiters can open within one settle. Every look reads memory again, so the next waiter is judged on a fresh reading once the settle has passed. Queue order comes from the existing queue: at the first gate, Ahead.Shut shuts on any queued waiter ahead, and at the race gate RaceAhead does the same. TestWaitMemorySettleSerialises and TestWaitMemorySettleHoldsALatecomer pass. Concern: the record is per account, because it lives under each account's ~/.abcd.noindex. The intent's machine is shared by the person's sessions and accounts, but waiters in different accounts are not held by each other's settle. When the pressure falls, runs from two accounts can therefore start at the same look. The help and the report say this openly ('any waiter of your account'), and round 1's fix report declined to change it, as the spec's design.
  evidence: internal/core/implement/load_memory.go:512 (settling) — "if !r.inEpisode(t) || !within(r.AdmittedAt, t, WaitMemorySettle) || r.is(self) || (ancestor != nil && ancestor(r.AdmittedPID)) {"
  evidence: internal/core/implement/load_wait.go:1052-1058 — "// The memory admission is written first, before the hold, so no later // waiter can look past this one's open inside an episode without // seeing it. if mem.admit(root, res, t, self, ancestor) {"
  evidence: internal/core/implement/load_memory_test.go:317 (TestWaitMemorySettleSerialises, PASS) — "said.Memory.WaitText() != "memory settling after pid 60000 opened 10s ago (pressure normal, floor warn)""
  evidence: internal/core/machineload/queue.go:285 — "return full && len(a.Preflights) > 0, len(a.Queued) > 0 || (full && running)"
  evidence: internal/surface/cli/implement_load.go (help, memory paragraph) — "After any waiter of your account saw the pressure at the floor, for " + implement.WaitMemoryEpisode.String() + " the gates\n" + "admit one run per " + implement.WaitMemorySettle.String() + ", in queue order"
  evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/report.md — "The record is per account (it lives under `~/.abcd.noindex`). Across accounts there is no settle, as with the holds"
- ac-8 — MET_WITH_CONCERNS: On Linux, ReadMemory calls readPressureFS(os.DirFS("/proc")). parsePressureMemory reads 'some avg10'. memoryShut compares StallPct with the floor percentage, and 14.2 against 10 waits with the wait line 'memory stalled 14.2% over 10s (floor 10%)'. A missing pressure/memory gives a MemoryUnsupportedError naming the 4.20/PSI requirement, which the gate turns into the not-checked note and admits. Concern: all of this was proven with fakes (an fstest filesystem and an injected reader) on macOS. The real Linux build was only vetted (GOOS=linux go vet in the DoD) and never run on a Linux kernel, with or without the file.
  evidence: internal/core/machineload/memory_linux.go — "func ReadMemory() (Memory, error) { return readPressureFS(os.DirFS("/proc")) }"
  evidence: internal/core/machineload/memory.go:145 — "return Memory{}, &MemoryUnsupportedError{Why: "/proc/pressure/memory is absent: the kernel's pressure stall information needs Linux 4.20 or later, built with it (CONFIG_PSI) and not turned off (psi=0)"}"
  evidence: internal/core/implement/load_memory_test.go:271 (TestWaitMemoryLinuxStall, PASS) — "said.Memory.WaitText() != "memory stalled 14.2% over 10s (floor 10%)""
  evidence: internal/core/machineload/memory_test.go (TestParsePressureMemory, TestReadMemoryLinuxNoFile, PASS) — "--- PASS: TestReadMemoryLinuxNoFile (0.00s)"
  evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/validate/round-1/fix/report.md — "the real Linux kernel path was only vetted, not run. Testing it needs a Linux host"

Gap audit:
- honoured:
  - The gate waits rather than refuses while memory pressure is at or above the floor, bounded by the gate's own timeout, as the intent's ruling requires.
    evidence: internal/core/implement/load_memory.go (package comment) — "It waits rather than refuses, as the disk floor does (load_disk.go): memory pressure falls as other work finishes, so the wait is bounded by the gate's own timeout"
  - The floor follows macOS's own pressure level, and a level abcd does not know reads as critical, so an unknown level shuts the gate rather than opening it.
    evidence: internal/core/machineload/memory.go:90 — "level := MemoryCritical"
    evidence: internal/core/implement/load_memory_test.go:195-201 — "// A level abcd does not know reads as critical, so it shuts even the // critical floor."
  - An unreadable memory floor, or an admission record that cannot be read or written, never wedges a run.
    evidence: internal/core/implement/load_memory_test.go:420 (TestWaitMemoryUnusableRecordAdmits, PASS) — "!strings.Contains(res.MemoryNote, "could not be read") || !strings.Contains(res.MemoryNote, "judged as if there were none")"
  - The surfaces (help, commands/implement.md, the brief, generated CLI reference, rules.json LOAD domain, ahoy marker block) describe the memory condition, and the race-gate wording no longer says 'at once' after a wait.
    evidence: internal/core/ahoy/defaults/claude-md-marker-block.md — "and no test stage started while the system reports memory pressure at or above the person's own floor (`load.memory_floor_macos` or `load.memory_floor_linux` in the same file, never set by a repository)."
    evidence: internal/surface/cli/implement_load.go:731 — "fmt.Fprintf(w, "race gate open after %s: %s\n", waited, res.Reason)"
  - Spike attribution and per-preflight memory sampling are left to itd-2610262140595217, as the person ruled.
    evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/report.md — "Attributing a spike to a process, and sampling memory per preflight (itd-2610262140595217)."
- diverged:
  - The serialisation after pressure falls is per account: waiters in different accounts on the same machine do not hold each other through the settle. The intent's scope covers one person's sessions and accounts.
    evidence: internal/core/implement/load_memory.go:97 — "WaitMemoryAdmitFileName = "load-memory-admit.json""
    evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/report.md — "Across accounts there is no settle, as with the holds; the pressure reading itself still holds every account's gates."
  - At the first gate on a platform where abcd cannot read the load (neither macOS nor Linux), the waiter opens on its general 'unchecked' reason and never says that the memory floor in particular was not checked.
    evidence: internal/core/implement/load_wait.go:879 — "res.Reason = "abcd reads load and processes on macOS and Linux only, and this is " + termsafe.Sanitize(runtime.GOOS) + "; opening without judging""
- missing:
  - Nothing shows the real Linux /proc/pressure/memory reader running on a Linux kernel, with the file or without it. Only fakes and a GOOS=linux vet cover it.
    evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/validate/round-1/fix/report.md — "the real Linux kernel path was only vetted, not run. Testing it needs a Linux host, and the reader's parsing and missing-file handling are covered by fakes."

Scope-condition dispositions:
- cond-2611020732414692 — survived: The delivery reads macOS's pressure level and Linux's /proc/pressure/memory. On any other platform, or when the read fails, the gate admits as it does today and nothing is held. On macOS and Linux a failed read gives the 'memory floor was not checked' note, and the load check gives that note on every platform. One caveat, which belongs under ac-4: on another platform the first gate's report is the gate-wide 'opening without judging' reason rather than the memory-specific line. The admission behaviour the condition expects still holds.
  evidence: internal/core/machineload/memory_other.go — "func ReadMemory() (Memory, error) { return memoryUnsupported(runtime.GOOS) }"
  evidence: internal/core/implement/load_memory.go:309 — "return "the memory floor was not checked: " + termsafe.Sanitize(err.Error()) + "; the gate does not wait on a memory reading it cannot take""
- cond-2611020732413735 — survived: The memory condition lives inside WaitLoad, which serves `make preflight` (first gate and race gate) and the command form `abcd implement load --wait -- <command>`. Nothing in the delivery holds tests run outside the gate, which the report lists as out of scope.
  evidence: Makefile:396 — "go run ./cmd/abcd implement load --wait --timeout 12h -- $(PREFLIGHT_STAGES_MAKE)"
  evidence: Makefile:424 — "go run ./cmd/abcd implement load --race-gate --timeout 2h"
  evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/report.md — "Holding tests that run outside the gate."
- cond-2611020732419293 — narrowed: The floor is read only from ~/.abcd.noindex/config.json, and a repository's key is ignored and named, so that part holds. The pressure reading itself is machine-wide, so every account's gates wait while it is at the floor. But the one-at-a-time settle after the pressure falls is recorded under each account's own ~/.abcd.noindex, so it does not reach across the accounts that share the machine.
  narrowing: On a machine shared by several of the person's accounts, the pressure wait holds in every account, but the one-run-per-settle admission after the pressure falls holds only among one account's waiters. Waiters in different accounts can be admitted at the same look.
  evidence: internal/core/implement/load_memory.go:149 (memoryFloorAt) — "from the machine configuration alone (layered.MachineRoots), as // PreflightSlots reads load.preflights."
  evidence: internal/surface/cli/implement_load.go (help, memory paragraph) — "After any waiter of your account saw the pressure at the floor"
  evidence: .abcd/.work.local/run/run-2611030127507347/lane-1/report.md — "The record is per account (it lives under `~/.abcd.noindex`). Across accounts there is no settle, as with the holds"
<!-- abcd-review-end receipt=rcp-ca6448963203 -->

## Grounds

- pursued: no test stage starts while the system reports memory pressure at or above its floor, so a machine running several agent sessions no longer runs out of memory; shown wrong if the machine runs out of application memory again with the floor in force and every stage admitted through the gate.
