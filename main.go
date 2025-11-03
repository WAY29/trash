//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	foDelete          = 0x0003
	fofAllowUndo      = 0x0040
	fofNoConfirmation = 0x0010
	fofSilent         = 0x0004
	fofNoErrorUI      = 0x0400
)

type shfileopstruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var (
	shell32              = syscall.NewLazyDLL("shell32.dll")
	procSHFileOperationW = shell32.NewProc("SHFileOperationW")
)

func toDoubleNullUTF16(paths []string) (*uint16, error) {
	if len(paths) == 0 {
		return nil, errors.New("empty path list")
	}
	joined := strings.Join(paths, "\x00") + "\x00\x00"
	runes := []rune(joined)
	utf16s := utf16.Encode(runes)
	return &utf16s[0], nil
}

func sendToRecycleBin(paths []string) error {
	pFrom, err := toDoubleNullUTF16(paths)
	if err != nil {
		return err
	}
	// Combine flags: allow undo sends to Recycle Bin, other flags avoid UI prompts
	f := uint16(fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI)
	ops := shfileopstruct{
		wFunc:  foDelete,
		pFrom:  pFrom,
		fFlags: f,
	}
	r1, _, e1 := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&ops)))
	// SHFileOperation returns zero on success; non-zero is an error code
	if r1 != 0 {
		if e1 != syscall.Errno(0) {
			return e1
		}
		return fmt.Errorf("SHFileOperationW failed with code %d", r1)
	}
	return nil
}

func isGlob(s string) bool {
	return strings.ContainsAny(s, "*?")
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: trash <file-or-dir> [more paths...]")
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	var toDelete []string
	var errorsList []string

	for _, a := range args {
		var matches []string
		if isGlob(a) {
			ms, _ := filepath.Glob(a)
			matches = ms
		} else {
			matches = []string{a}
		}
		if len(matches) == 0 {
			errorsList = append(errorsList, fmt.Sprintf("%s: not found", a))
			continue
		}
		for _, m := range matches {
			abs, err := filepath.Abs(m)
			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("%s: %v", m, err))
				continue
			}
			if _, statErr := os.Stat(abs); statErr != nil {
				errorsList = append(errorsList, fmt.Sprintf("%s: %v", abs, statErr))
				continue
			}
			toDelete = append(toDelete, abs)
		}
	}

	// Process one-by-one to get per-path error handling
	for _, p := range toDelete {
		if err := sendToRecycleBin([]string{p}); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: %v", p, err))
		}
	}

	if len(errorsList) > 0 {
		for _, msg := range errorsList {
			fmt.Fprintln(os.Stderr, "trash:", msg)
		}
		os.Exit(1)
	}
}
