---
id: spc-2611020732419632
slug: macos-ran-out-of-application-memory-on-a
intent: itd-2611012048006057
origin: researcher-authored
production_mode: hand-written
---
# abcd's load gate waits for memory as it waits for load

## Summary

spc-2611020732419632 delivers itd-2611012048006057. The load gate gets a
memory condition. While the system reports memory pressure at or above the
person's floor, the gate stays shut and the waiter keeps waiting. The load
check reports the same reading as a warning. The reading comes from the
operating system's own pressure signal:

- on macOS, the `kern.memorystatus_vm_pressure_level` sysctl, which reads 1
  for normal, 2 for warning, 3 for urgent and 4 for critical, and which an ordinary account
  can read;
- on Linux, the `some avg10` share of `/proc/pressure/memory`, the kernel's
  pressure stall information, available from 4.20.

A platform that has neither reading, or a read that fails, admits the run as
today and says once that memory was not checked.

The disk floor (iss-2610281502581386, `load_disk.go`) is the precedent this
follows in every respect but one. It is read only from the person's machine
configuration, it is read again just before the gate opens, and the load
check reports it without ever failing. The exception: the disk floor
refuses, since nothing frees disk while a waiter sleeps, but memory pressure
falls as other work finishes, so this condition waits, bounded by the gate's
own timeout.

## Footprint

- packages: internal/core/machineload, internal/core/implement, internal/surface/cli, internal/core/rules, internal/core/ahoy, commands, docs, .abcd/development/brief
- tests: internal/core/machineload, internal/core/implement, internal/surface/cli

## Design

### The reading (`internal/core/machineload`)

`ReadMemory() (Memory, error)` is one reading. `Memory` has these fields:

- `Platform`: `darwin` or `linux`.
- `Level`: on macOS, `normal`, `warn`, `urgent` or `critical`, mapped from
  the sysctl values 1, 2, 3 and 4. Any value other than 1, 2, 3 or 4 reads as `critical`,
  the highest level, so a level abcd does not know shuts the gate at either floor rather
  than opening it; only a failed read is unreadable.
- `StallPct`: on Linux, `avg10` of the `some` line, a percentage.
- `Raw`: the raw value, for the report.

Each platform's reader has its own file, as `read_*.go` and `disk_*.go` do:

- `memory_darwin.go` makes one `syscall.SysctlUint32`;
- `memory_linux.go` makes one bounded read of `/proc/pressure/memory`. A
  missing file is an `ErrUnsupported`-shaped error naming the kernel
  requirement;
- `memory_other.go` returns `ErrUnsupported`.

The parse is a pure function with its own test, as `parse.go` is for the
load averages. The reader is injected through `WaitRequest.ReadMemory` and
`LoadRequest.ReadMemory`, with nil meaning the real reader, so every test
drives it without the machine.

### The floor (`internal/core/implement/load_memory.go`)

Two keys in the person's `~/.abcd.noindex/config.json`. Each is read through
the same layered machine stack that `PreflightSlots` reads:

- `load.memory_floor_macos`: `warn` (the default) or `critical`;
- `load.memory_floor_linux`: a whole percentage from 1 to 100, default 10.

A value outside its set stops the gate with a message naming the key, the
file and the values it takes, as a bad `load.preflights` does. The gate does
not guess a floor. Each key gets a `stack.Claim`, so an unknown key under
`load` is still named.

A repository cannot set the floor. `memoryFloorAt` also looks at the
checkout's `.abcd/config.json` for either key. When one is present, the
result carries a `MemoryNote`: the key is ignored, the floor is a machine
setting, and here is the file that sets it. The note is printed once on
stderr. The repository's value is never used.

`memoryShut(m Memory, f Floor) bool` is the one judgement, shared by the
waiter and the check:

- macOS: `Level` is at or above the floor's level;
- Linux: `StallPct` is at or above the floor's percentage.

### The waiter (`load_wait.go`)

- A new shut condition, `WaitOnMemory = "memory"`, joins `WaitOnLoad` and the
  others in `waitLook`. It holds at both gates (the race gate's own stage is
  the memory-heaviest), and at the race gate too once the one-slot shortcut
  applies. With one slot today the race gate opens without a look, and in
  that case memory is read once more before it opens.
- Memory is read at every look and once more at the re-check before opening,
  after the jitter, so a reading that has gone stale cannot open the gate
  (criterion 1).
- `WaitResult` gains `Memory` (the last reading: level or stall share, and
  the floor) and `MemoryNote` (not checked and why, or a repository key
  ignored). The wait line names the level and the floor, for example
  `memory pressure warn (floor warn)` or
  `memory stalled 14.2% over 10s (floor 10%)`.
- When the timeout passes with memory still shut, the result is `WaitTimedOut`,
  as for load, and `Still` names `memory` (criterion 3).
- A reading that cannot be taken sets `MemoryNote` and shuts nothing
  (criterion 4). This deliberately differs from `WaitOnUnreadable`, which
  shuts the gate on an unreadable load or process table. An unread memory
  floor must never wedge a run, and the person ruled the gate fails open on
  it.

### One at a time after the pressure falls (criterion 7)

The queue already admits waiters in start order (`machineload.QueueAhead`,
`WaitOnWaiter`). With one slot the first gate's hold keeps them one at a
time. With more slots, several waiting on memory could all open at the one
look that sees the pressure fall. So, for a bounded time after the pressure
was last seen, the gate admits one run per settle:

- The record is `~/.abcd.noindex/load-memory-admit.json`, written through
  the same `os.Root` the holds use. It carries `pressure_at`, the last time
  any waiter read memory at or above the floor, and `admitted_at` with the
  pid and start of the last waiter admitted while the episode lasted.
- A waiter that reads memory at or above the floor writes `pressure_at`. So
  an episode starts only when a waiter sees pressure, and a machine that
  never comes under pressure has no record and behaves exactly as today.
- The episode lasts while `pressure_at` is younger than `WaitMemoryEpisode`
  (10 minutes). Outside it the record holds nothing back, and every reading
  below the floor admits as it would with no floor (criterion 2).
- Inside the episode every waiter reads the record, whether or not it saw
  memory shut itself: while `admitted_at` is younger than `WaitMemorySettle`
  (2 minutes, the same as `WaitSettle`) and its holder is not this waiter,
  `memory` stays shut. A waiter that opens inside the episode writes
  `admitted_at` first, before it opens and before its queue entry or hold
  lets any later waiter look, so no two runs start within one settle of
  each other while the episode lasts. The queue keeps them in start order.
- After each settle, memory is read again before the next waiter opens. So
  the stage just admitted has had time to show its own memory use, and a
  pressure that returns renews the episode.
- So the ramp after an episode is bounded: at most one admission per two
  minutes, for at most ten minutes after the pressure was last seen; then
  the gate admits as it does without a floor.
- A record it cannot read or write is named in `MemoryNote`, and the waiter
  judges as if there were none, so the record never wedges a run.

### The check (`load.go`)

`CheckLoad` reads memory beside the disk. `LoadResult.Memory` carries the
reading and whether it is at or above the floor. The text report prints a
warning line when it is. The check's exit never changes (criterion 6).

### Surfaces

- `internal/surface/cli/implement_load.go` renders the wait line, the
  timeout's cause, the not-checked note and the repository-ignored note, in
  both text and JSON. The help names the two keys and their defaults.
- `commands/implement.md`, `docs/reference/cli/commands.md` and
  `.abcd/development/brief/04-surfaces/27-implement.md` describe the memory
  condition beside the disk floor.
- The `LOAD` domain in `internal/core/rules/defaults/rules.json` adds memory
  pressure to what the load gate waits on. `internal/core/ahoy/defaults/
  claude-md-marker-block.md` carries the same line where it describes the
  gate.

## Out of scope

- Attributing a spike to a process, and sampling each preflight's peak
  memory. Both belong to itd-2610262140595217, by the person's ruling.
- Holding a test an agent runs outside the gate (iss-2610241130157455
  settled that the gate holds only what passes it).
- Lowering a running stage's parallelism under pressure. The gate decides
  only whether a stage starts.

## Acceptance-criteria satisfaction

Each criterion has a test, watched to fail before the change.

1. **Waits at or above the floor, re-reads, names the level.**
   `TestWaitMemoryShutsAtFloor` (an injected `warn` reading against the
   `warn` floor keeps the gate shut; the wait line names both).
   `TestWaitMemoryRereadBeforeOpen` (a reading that rises during the jitter
   keeps it shut).
2. **Admits when the pressure falls.** `TestWaitMemoryOpensWhenItFalls`
   (`warn`, then `normal`, opens as with no floor).
3. **Times out naming memory.** `TestWaitMemoryTimeoutNamesMemory`
   (`Status` `timeout`, `Still` `[memory]`).
4. **Fails open with one not-checked line.**
   `TestWaitMemoryUnreadableAdmits` (the reader errors; the gate opens;
   `MemoryNote` is set once). `TestReadMemoryUnsupported` on the other-platform
   reader.
5. **Default floor; a repository key is ignored and named.**
   `TestMemoryFloorDefaults` (`warn`, and 10 on Linux, with no key).
   `TestMemoryFloorMachineKey` (`critical` and 25 honoured from the machine
   file; out-of-range values refused naming the key).
   `TestMemoryFloorRepoKeyIgnored` (a repository's key changes nothing, and
   the note names it).
6. **The check warns only.** `TestCheckLoadMemoryWarns` (exit unchanged; the
   finding present). A CLI test renders the warning line.
7. **One at a time with a settle.** `TestWaitMemorySettleSerialises`: two
   waiters on memory at four slots; the pressure falls; the first opens and
   writes the record; the second stays shut on `memory` until the settle
   passes, then re-reads. `TestWaitMemorySettleHoldsALatecomer`: a
   waiter that arrives after the pressure falls, inside the settle, waits
   for it too. `TestWaitMemoryEpisodeEnds`: ten minutes after the pressure
   was last seen, arrivals open without a settle between them. `TestWaitMemoryNoRecordWithoutPressure`.
8. **Linux.** `TestParsePressureMemory` (the `some`/`full` lines, malformed
   and missing fields). `TestWaitMemoryLinuxStall` (14.2 against 10 waits).
   `TestReadMemoryLinuxNoFile` (a missing file reads as not checked, naming
   the kernel requirement).

## Steps

_No steps listed: the spec is built as one step._
