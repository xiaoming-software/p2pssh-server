package tunnel

import (
	"encoding/binary"
	"errors"
	"io"
)

// Wire format (little-endian), magic-4 channel protocol (frozen):
//
//	magic[4]=0x00 0x00 0x00 0x04
//	op u8
//	channel u32
//	seq u32
//	payload_len u32
//	payload[payload_len]
//
// OpOpen payload:
//
//	empty / "tcp"  → dial local sshd (port-forward for external ssh clients)
//	"shell"        → spawn a login PTY shell (in-app terminal; no sshd required)
//
// Channel lifecycle (MUST hold on every peer; do not change without a new magic):
//
//  1. Client allocates channel id, registers local receive state, THEN sends OpOpen.
//  2. Server prepares resource (Dial / PTY). On failure → OpOpenErr, stop.
//  3. Server registers channel, sends OpOpenAck, THEN may send OpData / start local read.
//  4. OpData uses per-direction seq starting at 0; receiver reorders by seq.
//  5. Either side may OpClose; after close, further frames for that id are ignored.
//
// Why (1)+(3): sshd/PTY often emit bytes immediately after Dial. If the client only
// registers after OpOpenAck, early OpData is dropped and external ssh hangs on
// banner exchange. Server must never OpData before OpOpenAck for that channel.

const MagicByte = byte(4)

const (
	OpOpen    = 1
	OpOpenAck = 2
	OpOpenErr = 3
	OpData    = 4
	OpClose   = 5
	OpResize  = 6
)

var Magic = [4]byte{0, 0, 0, MagicByte}

var ErrBadMagic = errors.New("bad magic")

type Frame struct {
	Op      byte
	Channel uint32
	Seq     uint32
	Payload []byte
}

func Encode(f Frame) []byte {
	n := 4 + 1 + 4 + 4 + 4 + len(f.Payload)
	b := make([]byte, n)
	copy(b[0:4], Magic[:])
	b[4] = f.Op
	binary.LittleEndian.PutUint32(b[5:9], f.Channel)
	binary.LittleEndian.PutUint32(b[9:13], f.Seq)
	binary.LittleEndian.PutUint32(b[13:17], uint32(len(f.Payload)))
	copy(b[17:], f.Payload)
	return b
}

func Decode(b []byte) (Frame, error) {
	if len(b) < 17 {
		return Frame{}, io.ErrUnexpectedEOF
	}
	if b[0] != Magic[0] || b[1] != Magic[1] || b[2] != Magic[2] || b[3] != Magic[3] {
		return Frame{}, ErrBadMagic
	}
	plen := binary.LittleEndian.Uint32(b[13:17])
	if int(plen) > len(b)-17 {
		return Frame{}, io.ErrUnexpectedEOF
	}
	payload := make([]byte, plen)
	copy(payload, b[17:17+plen])
	return Frame{
		Op:      b[4],
		Channel: binary.LittleEndian.Uint32(b[5:9]),
		Seq:     binary.LittleEndian.Uint32(b[9:13]),
		Payload: payload,
	}, nil
}

func IsShellOpen(payload []byte) bool {
	return string(payload) == "shell"
}
