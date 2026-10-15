//go:build darwin || linux

package ptyutil

import (
	"bytes"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestStartWithSize(t *testing.T) {
	shell := "/bin/bash"
	if runtime.GOOS == "darwin" {
		shell = "/bin/zsh"
	}
	cmd := exec.Command(shell, "-c", "printf HELLO_PTY")
	f, err := StartWithSize(cmd, 24, 80)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var got bytes.Buffer
	buf := make([]byte, 1024)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = f.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, err := f.Read(buf)
		if n > 0 {
			got.Write(buf[:n])
			if bytes.Contains(got.Bytes(), []byte("HELLO_PTY")) {
				break
			}
		}
		if err != nil && n == 0 {
			break
		}
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if !bytes.Contains(got.Bytes(), []byte("HELLO_PTY")) {
		t.Fatalf("output %q", got.Bytes())
	}
}
