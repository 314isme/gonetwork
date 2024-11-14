package gonetwork

import (
	"context"
	"fmt"
	"net/url"

	"github.com/coder/websocket"
)

type CWs struct {
	config ClientConfig
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func CWSNew(config ClientConfig, ctx context.Context, cancel context.CancelFunc) *CWs {
	return &CWs{config: config, ctx: ctx, cancel: cancel}
}

func (c *CWs) Connect() error {
	u := url.URL{Scheme: "wss", Host: c.config.Address + ":" + c.config.Port, Path: "/"}
	if !c.config.Secure {
		u.Scheme = "ws"
	}
	conn, _, err := websocket.Dial(c.ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	c.conn = conn
	if c.config.OnConnected != nil {
		c.config.OnConnected()
	}
	go c.readLoop()
	return nil
}

func (c *CWs) Send(data []byte) error {
	if err := c.conn.Write(c.ctx, websocket.MessageBinary, data); err != nil {
		c.Disconnect()
		return fmt.Errorf("failed to send data: %v", err)
	}
	return nil
}

func (c *CWs) Disconnect() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
}

func (c *CWs) readLoop() {
	defer c.Disconnect()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, data, err := c.conn.Read(c.ctx)
			if err != nil {
				return
			}
			if c.config.OnData != nil {
				c.config.OnData(data)
			}
		}
	}
}
