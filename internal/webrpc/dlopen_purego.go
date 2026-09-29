package webrpc

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

type handle uintptr

var (
	wNew   func(a, b, c *byte) uintptr
	wFree  func(h uintptr)
	wLogin func(h uintptr) int32
	wPort  func(h uintptr) int32
	wOpen  func(h uintptr, t, p *byte) uint32
	wClose func(h uintptr, sid uint32) int32
	wSess  func(h uintptr) uint16
	wSend  func(h uintptr, sid uint32, d *byte, n int32, timeout int64) int32
	wFile  func(h uintptr, sid uint32, p *byte) int32
	wToken func(h uintptr, sid uint32) uintptr
)

func openEmbedded() error {
	path, err := loadEmbeddedLibrary()
	if err != nil {
		return err
	}
	lib, err := openLibrary(path)
	if err != nil {
		return fmt.Errorf("加载 webrpc 运行库失败: %w", err)
	}
	purego.RegisterLibFunc(&wNew, lib, "WebrpcClient_New")
	purego.RegisterLibFunc(&wFree, lib, "WebrpcClient_Free")
	purego.RegisterLibFunc(&wLogin, lib, "WebrpcClient_LoginStatus")
	purego.RegisterLibFunc(&wPort, lib, "WebrpcClient_GetReceivePort")
	purego.RegisterLibFunc(&wOpen, lib, "WebrpcClient_OpenSession")
	purego.RegisterLibFunc(&wClose, lib, "WebrpcClient_CloseSession")
	purego.RegisterLibFunc(&wSess, lib, "WebrpcClient_SessionSize")
	purego.RegisterLibFunc(&wSend, lib, "WebrpcClient_SendData")
	purego.RegisterLibFunc(&wFile, lib, "WebrpcClient_SendFile")
	purego.RegisterLibFunc(&wToken, lib, "WebrpcClient_TarTokenBySession")
	if wNew == nil || wFree == nil || wLogin == nil || wPort == nil || wOpen == nil ||
		wClose == nil || wSess == nil || wSend == nil || wFile == nil || wToken == nil {
		return fmt.Errorf("webrpc 符号解析失败: %s", path)
	}
	return nil
}

func cstr(s string) (*byte, error) {
	return syscall.BytePtrFromString(s)
}

func cNew(token, passwd, permission string) handle {
	ct, _ := cstr(token)
	cp, _ := cstr(passwd)
	cperm, _ := cstr(permission)
	return handle(wNew(ct, cp, cperm))
}

func cFree(h handle) { wFree(uintptr(h)) }

func cLoginStatus(h handle) int { return int(wLogin(uintptr(h))) }

func cReceivePort(h handle) int { return int(wPort(uintptr(h))) }

func cOpenSession(h handle, toToken, permission string) uint32 {
	ct, _ := cstr(toToken)
	cp, _ := cstr(permission)
	return wOpen(uintptr(h), ct, cp)
}

func cCloseSession(h handle, sessionID uint32) int {
	return int(wClose(uintptr(h), sessionID))
}

func cSessionSize(h handle) int { return int(wSess(uintptr(h))) }

func cTarTokenBySession(h handle, sessionID uint32) string {
	p := wToken(uintptr(h), sessionID)
	if p == 0 {
		return ""
	}
	return cString(p)
}

func cString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for {
		if *(*byte)(unsafe.Pointer(p + uintptr(n))) == 0 {
			break
		}
		n++
		if n > 4096 {
			break
		}
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(p)), n)
	return string(b)
}

func cSendData(h handle, sessionID uint32, data []byte, timeoutMs int64) int {
	var p *byte
	if len(data) > 0 {
		p = &data[0]
	}
	return int(wSend(uintptr(h), sessionID, p, int32(len(data)), timeoutMs))
}

func cSendFile(h handle, sessionID uint32, filePath string) int {
	cs, _ := cstr(filePath)
	return int(wFile(uintptr(h), sessionID, cs))
}
