package implement

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// How many preflights the first gate admits to their test stages at once
// (iss-2610161639025702), one race stage able to run beside them
// (iss-2610282016250805), is a setting of the person's own machine, never of
// a checkout: a repository that could raise it could raise the machine's load
// for every account on it.
// So it is read from two places only, the first that holds it winning:
//
//   - ABCD_LOAD_PREFLIGHTS in the waiter's own environment, so a measured
//     trial can raise it without editing a file;
//   - load.preflights in ~/.abcd.noindex/config.json, the machine layer of the
//     layered configuration, loaded alone (layered.MachineRoots), so the
//     guarded read refuses a file that is not a regular file this account
//     owns and nobody else can write, and a home inside a git working tree
//     that is not the account's own is not read at all, exactly as for every
//     machine setting.
//
// Absent from both it is one, and every gate reads as it did before the
// setting existed. A value from either place that is not a whole number from
// one to MaxPreflightSlots is refused naming where it was set, never replaced
// by the default: a setting that silently did less than it says is the
// failure the layered reader exists to close.
//
// A machine config.json that cannot be read for the count at all (writable by
// others, not valid JSON, a load that is not an object) is read past instead,
// with a note naming the file: the count is the default, one. The gate never
// read that file before the setting existed, so refusing there would stop
// every make preflight of a person who never set the count, for a file that
// holds other settings. A key under load that nothing reads is named in the
// note too, and a usable load.preflights beside it still counts. A later step
// derives the count from the machine's resources instead
// (iss-2610161756384596).

// PreflightSlotsEnv is the environment variable a trial sets the count with.
const PreflightSlotsEnv = "ABCD_LOAD_PREFLIGHTS"

// PreflightSlotsKey is the machine configuration key.
const PreflightSlotsKey = "load.preflights"

// MaxPreflightSlots bounds the count. The product thinker approved a measured
// trial of two preflights at once (dcn-2610161814279204), and on 2026-10-28
// asked for the machine's capacity to be used, so four is the most either
// place may set: one race run at a time (the race gate) and the load limit
// stay the bounds, each stage's -p is the cores divided among the slots
// (machineload.GoTestParallelism), and the count is the person's own setting.
// A larger value is refused saying so (slotsRangeError).
const MaxPreflightSlots = 4

// slotsRangeError is the refusal of a count outside one to MaxPreflightSlots.
func slotsRangeError(n int) error {
	if n > MaxPreflightSlots {
		return fmt.Errorf("the first gate admits at most %d preflights to their test stages at once (want a whole number from 1 to %d)",
			MaxPreflightSlots, MaxPreflightSlots)
	}
	return fmt.Errorf("want a whole number from 1 to %d", MaxPreflightSlots)
}

// Where the count came from.
const (
	SlotsDefault = "default"
	SlotsMachine = "machine"
	SlotsEnv     = "env"
)

// The two gates a waiter waits at.
const (
	// GateTests is the first gate, before a preflight's test stages.
	GateTests = "tests"
	// GateRace is the race gate, before the race stage (--race-gate).
	GateRace = "race"
)

// PreflightSlots resolves how many preflights the first gate admits to their
// test stages at once, where that came from, and a note when the machine's
// config.json was read past (machineConfigNote), for the account whose home
// is home. It reads getenv and the machine's config.json, and no repository
// file. It is the one reader of the count: WaitLoad calls it at every look,
// its race-gate shortcut included, and cmd/test-parallelism calls it before it
// divides the cores by the count (PreflightParallelism,
// machineload.GoTestParallelism), so the gate and a stage's -p never judge by
// different values.
func PreflightSlots(getenv func(string) string, home string) (int, string, string, error) {
	if v := strings.TrimSpace(getenv(PreflightSlotsEnv)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			n = 0
		}
		if n < 1 || n > MaxPreflightSlots {
			return 0, "", "", fmt.Errorf("%s is %q: %w", PreflightSlotsEnv, layered.BoundKey(v), slotsRangeError(n))
		}
		return n, SlotsEnv, "", nil
	}
	roots, _ := layered.MachineRoots(home)
	if roots.HomeRefused {
		// Its own note, not MachineRoots': the wait refuses such a home
		// before reading this (WaitLoad), and test-parallelism reads it too.
		return 1, SlotsDefault, fmt.Sprintf("HOME lies inside a git working tree and is not this account's home, so %s is not read for %s and one preflight at a time applies",
			layered.Config.MachineOrigin(), PreflightSlotsKey), nil
	}
	stack, err := layered.Load(layered.Config, roots)
	if err != nil {
		return 1, SlotsDefault, machineConfigNote(err), nil
	}
	if _, err := stack.Lookup(PreflightSlotsKey); err != nil {
		return 1, SlotsDefault, machineConfigNote(err), nil
	}
	claimErr := stack.Claim("load", "preflights")
	v, err := layered.Get(stack, PreflightSlotsKey, 1, func(n int) error {
		if n < 1 || n > MaxPreflightSlots {
			return slotsRangeError(n)
		}
		return nil
	})
	if err != nil {
		return 0, "", "", fmt.Errorf("the load gate stops rather than guess how many preflights the first gate admits to their test stages at once: %w", err)
	}
	if v.Layer == layered.Machine {
		note := ""
		if claimErr != nil {
			note = fmt.Sprintf("the machine configuration %s holds a key under load that nothing reads (%v); %s is %d as set there",
				layered.Config.MachineOrigin(), claimErr, PreflightSlotsKey, v.V)
		}
		return v.V, SlotsMachine, note, nil
	}
	if claimErr != nil {
		return v.V, SlotsDefault, machineConfigNote(claimErr), nil
	}
	return v.V, SlotsDefault, "", nil
}

// machineConfigNote says why the machine config.json was read past for the
// count, naming the file, the cause and how to fix it.
func machineConfigNote(err error) string {
	file := layered.Config.MachineOrigin()
	return fmt.Sprintf("the machine configuration %s cannot be read for %s (%v), so one preflight at a time applies; "+
		"to set it, keep %s a regular file you own that nobody else can write (chmod go-w %s), holding one valid JSON object whose \"load\" names no key but \"preflights\", a whole number from 1 to %d",
		file, PreflightSlotsKey, err, file, file, MaxPreflightSlots)
}
