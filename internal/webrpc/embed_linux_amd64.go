//go:build linux && amd64

package webrpc

import _ "embed"

//go:embed runtimelib/linux_amd64/libwebrpc.so
var embeddedRuntimeLib []byte

func embeddedLibName() string { return "libwebrpc.so" }
