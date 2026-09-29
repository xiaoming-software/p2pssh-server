//go:build windows && amd64

package webrpc

import _ "embed"

//go:embed runtimelib/windows_amd64/libwebrpc.dll
var embeddedRuntimeLib []byte

func embeddedLibName() string { return "libwebrpc.dll" }
