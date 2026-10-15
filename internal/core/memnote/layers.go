package memnote

import (
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// RepoRelPath is the repository layer, beside the repository's rules.json: the
// notes a person promoted into the project.
const RepoRelPath = ".abcd/memory.json"

// MachineRelPath is the machine layer, beside ~/.abcd.noindex/rules.json: the
// person's own notes, written by agents through Add.
var MachineRelPath = abcdhome.Rel("memory.json")

// MachineDisplayPath is how the machine layer is named in a message: the tilde
// form, never the expanded home.
var MachineDisplayPath = abcdhome.Display("memory.json")

// maxFileBytes caps each memory.json, as the rules layers are capped.
const maxFileBytes = 256 * 1024

// ReadMachine reads the machine layer under home through the rules user
// layer's guarded read (rules.ReadUserLayerFile, over
// fsutil.ReadHomeDeclaration): a regular file this account owns that no one
// else can write, within 256 KiB, never behind a symlinked ~/.abcd.noindex.
// present is false when there is no file, or no absolute home; nothing is
// created by a read. A file failing a check or failing to parse is an error
// naming the file in tilde form.
func ReadMachine(home string) (f File, present bool, err error) {
	data, ok, err := rules.ReadUserLayerFile(home, MachineRelPath, "memory", "notes", maxFileBytes)
	return machineResult(data, ok, err)
}

// readMachineIn is ReadMachine through dir, ~/.abcd.noindex as the write's walk
// opened it (rules.ReadUserLayerFileIn): the write re-reads the file under its
// lock through the directory it then writes through, never by walking home
// again.
func readMachineIn(dir *os.Root, home string) (f File, present bool, err error) {
	data, ok, err := rules.ReadUserLayerFileIn(dir, home, MachineRelPath, "memory", "notes", maxFileBytes)
	return machineResult(data, ok, err)
}

// machineResult parses one guarded read of the machine layer.
func machineResult(data []byte, ok bool, err error) (f File, present bool, _ error) {
	if err != nil || !ok {
		return File{}, false, err
	}
	f, err = parseFile(data, machineLayer)
	if err != nil {
		return File{}, false, fmt.Errorf("memory: %s: %w", MachineDisplayPath, err)
	}
	return f, true, nil
}

// ReadRepo reads the repository layer under repoRoot — the root the rules
// loader resolved (rules.ResolveRoot) — through the repository layer's guarded
// read (rules.ReadRepoLayerFile): a symlinked .abcd or leaf refused, a regular
// file within 256 KiB. present is false when the file is absent.
func ReadRepo(repoRoot string) (f File, present bool, err error) {
	data, ok, err := rules.ReadRepoLayerFile(repoRoot, RepoRelPath, "memory", maxFileBytes)
	if err != nil || !ok {
		return File{}, false, err
	}
	f, err = parseFile(data, repoLayer)
	if err != nil {
		return File{}, false, fmt.Errorf("memory: %s: %w", RepoRelPath, err)
	}
	return f, true, nil
}
