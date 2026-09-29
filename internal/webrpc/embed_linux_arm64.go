//go:build linux && arm64

package webrpc

import _ "embed"

//go:embed runtimelib/linux_arm64/libwebrpc.so
var embeddedRuntimeLib []byte

func embeddedLibName() string { return "libwebrpc.so" }
