package gonetwork

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
)

type CTcp struct {
	config     ClientConfig
	connection net.Conn
	ctx        context.Context
	cancel     context.CancelFunc
}

func CTCPNew(config ClientConfig, ctx context.Context, cancel context.CancelFunc) *CTcp {
	return &CTcp{config: config, ctx: ctx, cancel: cancel}
}

func (c *CTcp) Connect() error {
	address := c.config.Address + ":" + c.config.Port
	var conn net.Conn
	var err error
	if c.config.Secure {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		conn, err = tls.Dial("tcp", address, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	c.connection = conn
	if c.config.OnConnected != nil {
		c.config.OnConnected()
	}
	go c.readLoop()
	return nil
}

func (c *CTcp) Send(data []byte) error {
	length := uint32(len(data))
	if err := binary.Write(c.connection, binary.BigEndian, length); err != nil {
		c.Disconnect()
		return fmt.Errorf("failed to write data length: %v", err)
	}
	if _, err := c.connection.Write(data); err != nil {
		c.Disconnect()
		return fmt.Errorf("failed to write data: %v", err)
	}
	return nil
}

func (c *CTcp) Disconnect() {
	c.cancel()
	if c.connection != nil {
		c.connection.Close()
		c.connection = nil
	}
}

func (c *CTcp) readLoop() {
	defer c.Disconnect()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var length uint32
			if err := binary.Read(c.connection, binary.BigEndian, &length); err != nil {
				return
			}
			data := make([]byte, length)
			if _, err := c.connection.Read(data); err != nil {
				return
			}
			if c.config.OnData != nil {
				c.config.OnData(data)
			}
		}
	}
}
