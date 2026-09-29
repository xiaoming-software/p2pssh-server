package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

// Sender sends a complete encoded frame over webrpc.
type Sender func(data []byte) error

// Session multiplexes many channels over one webrpc session (server side).
type Session struct {
	sshHost string
	sshPort int
	send    Sender

	mu       sync.Mutex
	channels map[uint32]*channel
}

type channel struct {
	id     uint32
	rw     io.ReadWriteCloser
	closed atomic.Bool

	resize  func(cols, rows uint16) error
	cleanup func()

	// outbound seq (server → client)
	nextOut atomic.Uint32

	// inbound reorder (client → server)
	mu      sync.Mutex
	nextIn  uint32
	pending map[uint32][]byte
}

func NewSession(sshHost string, sshPort int, send Sender) *Session {
	return &Session{
		sshHost:  sshHost,
		sshPort:  sshPort,
		send:     send,
		channels: make(map[uint32]*channel),
	}
}

func (s *Session) Close() {
	s.mu.Lock()
	chs := make([]*channel, 0, len(s.channels))
	for _, ch := range s.channels {
		chs = append(chs, ch)
	}
	s.channels = make(map[uint32]*channel)
	s.mu.Unlock()
	for _, ch := range chs {
		ch.forceClose()
	}
}

func (s *Session) HandleFrame(raw []byte) {
	f, err := Decode(raw)
	if err != nil {
		return
	}
	switch f.Op {
	case OpOpen:
		s.onOpen(f.Channel, f.Payload)
	case OpData:
		s.onData(f.Channel, f.Seq, f.Payload)
	case OpClose:
		s.onClose(f.Channel)
	case OpResize:
		s.onResize(f.Channel, f.Payload)
	}
}

func (s *Session) onOpen(id uint32, payload []byte) {
	s.mu.Lock()
	if _, ok := s.channels[id]; ok {
		s.mu.Unlock()
		_ = s.send(Encode(Frame{Op: OpOpenErr, Channel: id, Payload: []byte("channel exists")}))
		return
	}
	s.mu.Unlock()

	var (
		rw      io.ReadWriteCloser
		resize  func(cols, rows uint16) error
		cleanup func()
		mode    string
		err     error
	)

	if IsShellOpen(payload) {
		mode = "shell"
		rw, resize, cleanup, err = startShell(80, 24)
		if err != nil {
			log.Printf("shell open channel=%d failed: %v", id, err)
			_ = s.send(Encode(Frame{Op: OpOpenErr, Channel: id, Payload: []byte(err.Error())}))
			return
		}
	} else {
		mode = "tcp"
		addr := fmt.Sprintf("%s:%d", s.sshHost, s.sshPort)
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr != nil {
			log.Printf("tcp open channel=%d dial %s failed: %v", id, addr, dialErr)
			_ = s.send(Encode(Frame{Op: OpOpenErr, Channel: id, Payload: []byte(dialErr.Error())}))
			return
		}
		rw = conn
		cleanup = func() { _ = conn.Close() }
	}

	ch := &channel{
		id:      id,
		rw:      rw,
		resize:  resize,
		cleanup: cleanup,
		pending: make(map[uint32][]byte),
	}
	s.mu.Lock()
	s.channels[id] = ch
	s.mu.Unlock()

	if err := s.send(Encode(Frame{Op: OpOpenAck, Channel: id})); err != nil {
		log.Printf("open ack channel=%d mode=%s send failed: %v", id, mode, err)
		s.removeChannel(id)
		return
	}
	log.Printf("channel=%d opened mode=%s", id, mode)
	go s.pipeLocal(ch)
}

func (s *Session) onResize(id uint32, payload []byte) {
	if len(payload) < 4 {
		return
	}
	cols := binary.LittleEndian.Uint16(payload[0:2])
	rows := binary.LittleEndian.Uint16(payload[2:4])
	s.mu.Lock()
	ch := s.channels[id]
	s.mu.Unlock()
	if ch == nil || ch.closed.Load() || ch.resize == nil {
		return
	}
	if err := ch.resize(cols, rows); err != nil {
		log.Printf("resize channel=%d %dx%d: %v", id, cols, rows, err)
	}
}

func (s *Session) onData(id uint32, seq uint32, payload []byte) {
	s.mu.Lock()
	ch := s.channels[id]
	s.mu.Unlock()
	if ch == nil || ch.closed.Load() {
		return
	}
	ch.mu.Lock()
	ch.pending[seq] = payload
	for {
		p, ok := ch.pending[ch.nextIn]
		if !ok {
			break
		}
		delete(ch.pending, ch.nextIn)
		ch.nextIn++
		ch.mu.Unlock()
		if len(p) > 0 {
			if _, err := ch.rw.Write(p); err != nil {
				s.closeChannel(ch, true)
				return
			}
		}
		ch.mu.Lock()
	}
	ch.mu.Unlock()
}

func (s *Session) onClose(id uint32) {
	s.mu.Lock()
	ch := s.channels[id]
	delete(s.channels, id)
	s.mu.Unlock()
	if ch != nil {
		ch.forceClose()
	}
}

func (s *Session) removeChannel(id uint32) {
	s.mu.Lock()
	ch := s.channels[id]
	delete(s.channels, id)
	s.mu.Unlock()
	if ch != nil {
		ch.forceClose()
	}
}

func (s *Session) closeChannel(ch *channel, notify bool) {
	if !ch.closed.CompareAndSwap(false, true) {
		return
	}
	if ch.cleanup != nil {
		ch.cleanup()
	} else {
		_ = ch.rw.Close()
	}
	s.mu.Lock()
	delete(s.channels, ch.id)
	s.mu.Unlock()
	if notify {
		_ = s.send(Encode(Frame{Op: OpClose, Channel: ch.id}))
	}
}

func (s *Session) pipeLocal(ch *channel) {
	buf := make([]byte, 32*1024)
	for {
		n, err := ch.rw.Read(buf)
		if n > 0 {
			seq := ch.nextOut.Add(1) - 1
			payload := make([]byte, n)
			copy(payload, buf[:n])
			if err := s.send(Encode(Frame{Op: OpData, Channel: ch.id, Seq: seq, Payload: payload})); err != nil {
				s.closeChannel(ch, false)
				return
			}
		}
		if err != nil {
			s.closeChannel(ch, true)
			return
		}
	}
}

func (ch *channel) forceClose() {
	if ch.closed.CompareAndSwap(false, true) {
		if ch.cleanup != nil {
			ch.cleanup()
		} else if ch.rw != nil {
			_ = ch.rw.Close()
		}
	}
}
