package gonetwork

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

type CTcp struct {
	config      ClientConfig
	connection  net.Conn
	stopChannel chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	once        sync.Once
}

func TCPNewC(config ClientConfig, ctx context.Context, cancel context.CancelFunc) *CTcp {
	return &CTcp{
		config:      config,
		stopChannel: make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
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

func (c *CTcp) readLoop() {
	defer c.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var length uint32
			if err := binary.Read(c.connection, binary.BigEndian, &length); err != nil {
				return
			}
			if length > 1024*1024 {
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

func (c *CTcp) Send(data []byte) error {
	length := uint32(len(data))
	if err := binary.Write(c.connection, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write data length: %v", err)
	}
	if _, err := c.connection.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %v", err)
	}
	return nil
}

func (c *CTcp) Disconnect() {
	c.cancel()
	c.Close()
}

func (c *CTcp) Close() error {
	var err error
	c.once.Do(func() {
		close(c.stopChannel)
		if c.connection != nil {
			err = c.connection.Close()
			c.connection = nil
		}
	})
	return err
}
