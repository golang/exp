// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binscan

// This file is a somewhat modified version of cmd/go/internal/version/exe.go
// that adds functionality for extracting the PCLN table.

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"

	// "internal/xcoff"
	"io"
	"os"
)

// An exe is a generic interface to an OS executable (ELF, Mach-O, PE, XCOFF).
type exe interface {
	// Close closes the underlying file.
	Close() error

	// ReadData reads and returns up to size byte starting at virtual address addr.
	ReadData(addr, size uint64) ([]byte, error)

	// DataStart returns the writable data segment start address.
	DataStart() uint64

	PCLNTab() ([]byte, uint64)
}

// openExe opens file and returns it as an exe.
func openExe(file string) (exe, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	data := make([]byte, 16)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, err
	}
	f.Seek(0, 0)
	if bytes.HasPrefix(data, []byte("\x7FELF")) {
		e, err := elf.NewFile(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &elfExe{f, e}, nil
	}
	if bytes.HasPrefix(data, []byte("MZ")) {
		e, err := pe.NewFile(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &peExe{f, e}, nil
	}
	if bytes.HasPrefix(data, []byte("\xFE\xED\xFA")) || bytes.HasPrefix(data[1:], []byte("\xFA\xED\xFE")) {
		e, err := macho.NewFile(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &machoExe{f, e}, nil
	}
	// TODO(rolandshoemaker): we cannot support XCOFF files due to the usage of internal/xcoff.
	// Once this code is moved into the stdlib, this support can be re-enabled.
	// if bytes.HasPrefix(data, []byte{0x01, 0xDF}) || bytes.HasPrefix(data, []byte{0x01, 0xF7}) {
	// 	e, err := xcoff.NewFile(f)
	// 	if err != nil {
	// 		f.Close()
	// 		return nil, err
	// 	}
	// 	return &xcoffExe{f, e}, nil

	// }
	return nil, fmt.Errorf("unrecognized executable format")
}

// elfExe is the ELF implementation of the exe interface.
type elfExe struct {
	os *os.File
	f  *elf.File
}

func (x *elfExe) Close() error {
	return x.os.Close()
}

func (x *elfExe) ReadData(addr, size uint64) ([]byte, error) {
	for _, prog := range x.f.Progs {
		if prog.Vaddr <= addr && addr <= prog.Vaddr+prog.Filesz-1 {
			n := prog.Vaddr + prog.Filesz - addr
			if n > size {
				n = size
			}
			data := make([]byte, n)
			_, err := prog.ReadAt(data, int64(addr-prog.Vaddr))
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not mapped")
}

func (x *elfExe) DataStart() uint64 {
	for _, s := range x.f.Sections {
		if s.Name == ".go.buildinfo" {
			return s.Addr
		}
	}
	for _, p := range x.f.Progs {
		if p.Type == elf.PT_LOAD && p.Flags&(elf.PF_X|elf.PF_W) == elf.PF_W {
			return p.Vaddr
		}
	}
	return 0
}

func (x *elfExe) PCLNTab() ([]byte, uint64) {
	var offset uint64
	text := x.f.Section(".text")
	if text != nil {
		offset = text.Offset
	}
	pclntab := x.f.Section(".gopclntab")
	if pclntab == nil {
		pclntab = x.f.Section(".data.rel.ro.gopclntab")
		if pclntab == nil {
			panic("no pclntab")
		}
	}
	b, err := pclntab.Data()
	if err != nil {
		panic(err)
	}
	return b, offset
}

// peExe is the PE (Windows Portable Executable) implementation of the exe interface.
type peExe struct {
	os *os.File
	f  *pe.File
}

func (x *peExe) Close() error {
	return x.os.Close()
}

func (x *peExe) imageBase() uint64 {
	switch oh := x.f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return uint64(oh.ImageBase)
	case *pe.OptionalHeader64:
		return oh.ImageBase
	}
	return 0
}

func (x *peExe) ReadData(addr, size uint64) ([]byte, error) {
	addr -= x.imageBase()
	for _, sect := range x.f.Sections {
		if uint64(sect.VirtualAddress) <= addr && addr <= uint64(sect.VirtualAddress+sect.Size-1) {
			n := uint64(sect.VirtualAddress+sect.Size) - addr
			if n > size {
				n = size
			}
			data := make([]byte, n)
			_, err := sect.ReadAt(data, int64(addr-uint64(sect.VirtualAddress)))
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not mapped")
}

func (x *peExe) DataStart() uint64 {
	// Assume data is first writable section.
	const (
		IMAGE_SCN_CNT_CODE               = 0x00000020
		IMAGE_SCN_CNT_INITIALIZED_DATA   = 0x00000040
		IMAGE_SCN_CNT_UNINITIALIZED_DATA = 0x00000080
		IMAGE_SCN_MEM_EXECUTE            = 0x20000000
		IMAGE_SCN_MEM_READ               = 0x40000000
		IMAGE_SCN_MEM_WRITE              = 0x80000000
		IMAGE_SCN_MEM_DISCARDABLE        = 0x2000000
		IMAGE_SCN_LNK_NRELOC_OVFL        = 0x1000000
		IMAGE_SCN_ALIGN_32BYTES          = 0x600000
	)
	for _, sect := range x.f.Sections {
		if sect.VirtualAddress != 0 && sect.Size != 0 &&
			sect.Characteristics&^IMAGE_SCN_ALIGN_32BYTES == IMAGE_SCN_CNT_INITIALIZED_DATA|IMAGE_SCN_MEM_READ|IMAGE_SCN_MEM_WRITE {
			return uint64(sect.VirtualAddress) + x.imageBase()
		}
	}
	return 0
}

func (x *peExe) PCLNTab() ([]byte, uint64) {
	var textOffset uint64
	for _, section := range x.f.Sections {
		if section.Name == ".text" {
			textOffset = uint64(section.Offset)
			break
		}
	}
	var start, end int64
	var section int
	for _, symbol := range x.f.Symbols {
		if symbol.Name == "runtime.pclntab" {
			start = int64(symbol.Value)
			section = int(symbol.SectionNumber - 1)
		} else if symbol.Name == "runtime.epclntab" {
			end = int64(symbol.Value)
			break
		}
	}
	if start == 0 || end == 0 {
		panic("didn't find both start and enc")
	}
	offset := int64(x.f.Sections[section].Offset) + start
	size := end - start

	pclntab := make([]byte, size)
	if _, err := x.os.ReadAt(pclntab, offset); err != nil {
		panic(err)
	}

	return pclntab, textOffset
}

// machoExe is the Mach-O (Apple macOS/iOS) implementation of the exe interface.
type machoExe struct {
	os *os.File
	f  *macho.File
}

func (x *machoExe) Close() error {
	return x.os.Close()
}

func (x *machoExe) ReadData(addr, size uint64) ([]byte, error) {
	for _, load := range x.f.Loads {
		seg, ok := load.(*macho.Segment)
		if !ok {
			continue
		}
		if seg.Addr <= addr && addr <= seg.Addr+seg.Filesz-1 {
			if seg.Name == "__PAGEZERO" {
				continue
			}
			n := seg.Addr + seg.Filesz - addr
			if n > size {
				n = size
			}
			data := make([]byte, n)
			_, err := seg.ReadAt(data, int64(addr-seg.Addr))
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not mapped")
}

func (x *machoExe) DataStart() uint64 {
	// Look for section named "__go_buildinfo".
	for _, sec := range x.f.Sections {
		if sec.Name == "__go_buildinfo" {
			return sec.Addr
		}
	}
	// Try the first non-empty writable segment.
	const RW = 3
	for _, load := range x.f.Loads {
		seg, ok := load.(*macho.Segment)
		if ok && seg.Addr != 0 && seg.Filesz != 0 && seg.Prot == RW && seg.Maxprot == RW {
			return seg.Addr
		}
	}
	return 0
}

func (x *machoExe) PCLNTab() ([]byte, uint64) {
	var textOffset uint64
	text := x.f.Section("__text")
	if text != nil {
		textOffset = uint64(text.Offset)
	}
	pclntab := x.f.Section("__gopclntab")
	if pclntab == nil {
		panic("no pclntab")
	}
	b, err := pclntab.Data()
	if err != nil {
		panic("err")
	}
	return b, textOffset
}

// TODO(rolandshoemaker): we cannot support XCOFF files due to the usage of internal/xcoff.
// Once this code is moved into the stdlib, this support can be re-enabled.

// // xcoffExe is the XCOFF (AIX eXtended COFF) implementation of the exe interface.
// type xcoffExe struct {
// 	os *os.File
// 	f  *xcoff.File
// }
//
// func (x *xcoffExe) Close() error {
// 	return x.os.Close()
// }
//
// func (x *xcoffExe) ReadData(addr, size uint64) ([]byte, error) {
// 	for _, sect := range x.f.Sections {
// 		if uint64(sect.VirtualAddress) <= addr && addr <= uint64(sect.VirtualAddress+sect.Size-1) {
// 			n := uint64(sect.VirtualAddress+sect.Size) - addr
// 			if n > size {
// 				n = size
// 			}
// 			data := make([]byte, n)
// 			_, err := sect.ReadAt(data, int64(addr-uint64(sect.VirtualAddress)))
// 			if err != nil {
// 				return nil, err
// 			}
// 			return data, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("address not mapped")
// }
//
// func (x *xcoffExe) DataStart() uint64 {
// 	return x.f.SectionByType(xcoff.STYP_DATA).VirtualAddress
// }
