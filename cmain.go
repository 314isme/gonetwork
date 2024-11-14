package gonetwork

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

type Client struct {
	config ClientConfig
	conn   ClientConnection
	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(config ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	var conn ClientConnection
	switch config.Type {
	case "TCP":
		conn = CTCPNew(config, ctx, cancel)
	case "WS":
		conn = CWSNew(config, ctx, cancel)
	default:
		cancel()
		return nil
	}
	return &Client{
		config: config,
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Client) Send(dataHandler, dataType string, dataProto proto.Message) error {
	dataByte, err := EncodeMessage(dataHandler, dataType, dataProto)
	if err != nil {
		return fmt.Errorf("send error: %v", err)
	}
	return c.conn.Send(dataByte)
}

func (c *Client) Connect() error {
	return c.conn.Connect()
}

func (c *Client) Disconnect() {
	c.conn.Disconnect()
}
