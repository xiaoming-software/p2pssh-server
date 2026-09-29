package webrpc

import (
	"fmt"
	"sync"
	"time"
)

// Client wraps the webrpc SDK. API names match the official C headers.
type Client struct {
	h handle
}

var (
	loadOnce sync.Once
	loadErr  error
)

func Open(token, passwd, permission string) (*Client, error) {
	loadOnce.Do(func() {
		loadErr = openEmbedded()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	h := cNew(token, passwd, permission)
	if h == 0 {
		return nil, fmt.Errorf("WebrpcClient_New 返回空句柄")
	}
	return &Client{h: h}, nil
}

func (c *Client) Close() {
	if c.h != 0 {
		cFree(c.h)
		c.h = 0
	}
}

func (c *Client) LoginStatus() int {
	return cLoginStatus(c.h)
}

// WaitLogin polls LoginStatus until it is non-zero or timeout.
func (c *Client) WaitLogin(poll, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for c.LoginStatus() == 0 {
		if time.Now().After(deadline) {
			return fmt.Errorf("webrpc 登录超时（%s），请检查 token/密码/网络", timeout)
		}
		time.Sleep(poll)
	}
	return nil
}

func (c *Client) ReceivePort() int {
	return cReceivePort(c.h)
}

func (c *Client) OpenSession(toToken, permission string) uint32 {
	return cOpenSession(c.h, toToken, permission)
}

func (c *Client) CloseSession(sessionID uint32) int {
	return cCloseSession(c.h, sessionID)
}

func (c *Client) SessionSize() int {
	return cSessionSize(c.h)
}

func (c *Client) PeerToken(sessionID uint32) string {
	return cTarTokenBySession(c.h, sessionID)
}

func (c *Client) SendData(sessionID uint32, data []byte, timeoutMs int64) int {
	return cSendData(c.h, sessionID, data, timeoutMs)
}

func (c *Client) SendFile(sessionID uint32, filePath string) int {
	return cSendFile(c.h, sessionID, filePath)
}
