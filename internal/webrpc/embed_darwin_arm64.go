//go:build darwin && arm64

package webrpc

import _ "embed"

//go:embed runtimelib/darwin_arm64/libwebrpc.dylib
var embeddedRuntimeLib []byte

func embeddedLibName() string { return "libwebrpc.dylib" }
