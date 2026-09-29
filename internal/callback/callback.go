package callback

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/xiaoming-software/p2pssh-server/internal/tunnel"
)

const (
	streamTypeFile = 1
	streamTypeData = 2
)

type SendFunc func(sessionID uint32, data []byte) int

// Manager keeps one tunnel.Session per webrpc session id.
type Manager struct {
	sshHost string
	sshPort int
	send    SendFunc

	mu       sync.Mutex
	sessions map[uint32]*tunnel.Session
}

func NewManager(sshHost string, sshPort int, send SendFunc) *Manager {
	return &Manager{
		sshHost:  sshHost,
		sshPort:  sshPort,
		send:     send,
		sessions: make(map[uint32]*tunnel.Session),
	}
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		s.Close()
		delete(m.sessions, id)
	}
}

func (m *Manager) getOrCreate(sessionID uint32) *tunnel.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		return s
	}
	s := tunnel.NewSession(m.sshHost, m.sshPort, func(data []byte) error {
		// WebrpcClient_SendData returns 1 on success (same as File2File / mywebdisk).
		ret := m.send(sessionID, data)
		if ret != 1 {
			return fmt.Errorf("SendData ret=%d", ret)
		}
		return nil
	})
	m.sessions[sessionID] = s
	log.Printf("新建 SSH 隧道会话 session=%d -> %s:%d", sessionID, m.sshHost, m.sshPort)
	return s
}

func (m *Manager) drop(sessionID uint32) {
	m.mu.Lock()
	s := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if s != nil {
		s.Close()
		log.Printf("关闭 SSH 隧道会话 session=%d", sessionID)
	}
}

// Listen connects to the local webrpc callback port and dispatches magic-4 frames.
func Listen(tcpPort int, mgr *Manager) {
	addr := fmt.Sprintf("127.0.0.1:%d", tcpPort)
	backoff := time.Second
	for {
		err := drain(addr, mgr)
		log.Printf("webrpc 回调连接结束: %v，%s 后重连", err, backoff)
		time.Sleep(backoff)
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func drain(addr string, mgr *Manager) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("已连接 webrpc 回调端口 %s", addr)

	for {
		var hdr [5]byte
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			if err == io.EOF {
				return fmt.Errorf("连接关闭")
			}
			return fmt.Errorf("读帧头: %w", err)
		}
		sessionID := binary.BigEndian.Uint32(hdr[:4])
		dataType := hdr[4]

		switch dataType {
		case streamTypeData:
			n, err := readUint32(conn)
			if err != nil {
				return fmt.Errorf("读数据长度: %w", err)
			}
			payload := make([]byte, n)
			if _, err := io.ReadFull(conn, payload); err != nil {
				return fmt.Errorf("读数据体: %w", err)
			}
			handleData(sessionID, payload, mgr)
		case streamTypeFile:
			// p2pssh does not use file streams; drain and ignore.
			nameLen, err := readUint32(conn)
			if err != nil {
				return fmt.Errorf("读文件名长度: %w", err)
			}
			if _, err := io.CopyN(io.Discard, conn, int64(nameLen)); err != nil {
				return fmt.Errorf("丢弃文件名: %w", err)
			}
			dataLen, err := readUint32(conn)
			if err != nil {
				return fmt.Errorf("读文件长度: %w", err)
			}
			if _, err := io.CopyN(io.Discard, conn, int64(dataLen)); err != nil {
				return fmt.Errorf("丢弃文件体: %w", err)
			}
		default:
			log.Printf("未知流类型 session=%d type=%d", sessionID, dataType)
		}
	}
}

func handleData(sessionID uint32, payload []byte, mgr *Manager) {
	if len(payload) < 4 || payload[0] != 0 || payload[1] != 0 || payload[2] != 0 || payload[3] != tunnel.MagicByte {
		return
	}
	s := mgr.getOrCreate(sessionID)
	s.HandleFrame(payload)
}

func readUint32(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
