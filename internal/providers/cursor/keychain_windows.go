//go:build windows

package cursor

import (
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	keychainService = "cursor-access-token"
	keychainAccount = "cursor-user"
	credTypeGeneric = 1
)

var (
	advapi32     = windows.NewLazySystemDLL("advapi32.dll")
	procCredRead = advapi32.NewProc("CredReadW")
	procCredFree = advapi32.NewProc("CredFree")
)

type credentialW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

func loadOSStore() (*Credential, error) {
	for _, target := range []string{
		keychainService,
		keychainService + "/" + keychainAccount,
		"LegacyGeneric:target=" + keychainService,
		keychainAccount + "@" + keychainService,
	} {
		if token, ok := readCredentialTarget(target); ok {
			return &Credential{AccessToken: token, Source: SourceOSStore}, nil
		}
	}
	return nil, nil
}

func readCredentialTarget(target string) (string, bool) {
	name, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return "", false
	}
	var cred *credentialW
	ret, _, _ := procCredRead.Call(
		uintptr(unsafe.Pointer(name)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&cred)))
	if ret == 0 || cred == nil {
		return "", false
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))
	if cred.CredentialBlob == nil || cred.CredentialBlobSize == 0 {
		return "", false
	}
	blob := unsafe.Slice(cred.CredentialBlob, cred.CredentialBlobSize)
	token := decodeBlob(blob)
	return token, token != ""
}

func decodeBlob(blob []byte) string {
	if len(blob)%2 == 0 && looksUTF16(blob) {
		units := make([]uint16, 0, len(blob)/2)
		for i := 0; i+1 < len(blob); i += 2 {
			units = append(units, uint16(blob[i])|uint16(blob[i+1])<<8)
		}
		return trimNul(string(utf16.Decode(units)))
	}
	return trimNul(string(blob))
}

func looksUTF16(blob []byte) bool {
	if len(blob) < 2 {
		return false
	}
	for i := 1; i < len(blob); i += 2 {
		if blob[i] != 0 {
			return false
		}
	}
	return true
}

func trimNul(s string) string {
	s = strings.TrimRight(s, "\x00\r\n")
	return strings.TrimSpace(s)
}
