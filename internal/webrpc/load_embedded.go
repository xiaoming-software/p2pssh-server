package webrpc

import (
	"fmt"
	"os"
	"path/filepath"
)

func loadEmbeddedLibrary() (string, error) {
	if len(embeddedRuntimeLib) == 0 {
		return "", fmt.Errorf("未嵌入 webrpc 运行库，请使用 ./build.sh 编译")
	}
	dir, err := os.MkdirTemp("", "p2pssh-webrpc-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, embeddedLibName())
	if err := os.WriteFile(path, embeddedRuntimeLib, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return path, nil
}
