package gonetwork

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type ServerConfig struct {
	Address        string
	Port           string
	Domain         string
	TCPPort        string
	WSPort         string
	Secure         bool
	MaxWorkers     int
	OnConnected    func(connectionID string)
	OnData         func(connectionID string, data []byte)
	OnDisconnected func(connectionID string)
	CertFile       string
	KeyFile        string
	AllowedOrigins []string
}

type ClientConfig struct {
	Address        string
	Port           string
	Secure         bool
	Type           string
	OnConnected    func()
	OnData         func(data []byte)
	OnDisconnected func()
	CertFile       string
}

func TLSCert(certFile, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

type ConnPool struct {
	mu       sync.Mutex
	idle     []net.Conn
	active   int
	maxConns int
	timeout  time.Duration
}

func NewConnPool(maxConns int, timeout time.Duration) *ConnPool {
	return &ConnPool{maxConns: maxConns, timeout: timeout}
}

func (p *ConnPool) Get() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.idle) > 0 {
		conn := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		return conn, nil
	}
	if p.active < p.maxConns {
		conn, err := net.DialTimeout("tcp", "server_address", p.timeout)
		if err != nil {
			return nil, fmt.Errorf("connection error: %v", err)
		}
		p.active++
		return conn, nil
	}
	return nil, errors.New("no available connections")
}

type ClientConnection interface {
	Connect() error
	Send(data []byte) error
	Disconnect()
}

type ServerConnection interface {
	Listen(ctx context.Context) error
	Broadcast(dataHandler, dataType string, data proto.Message)
	Send(connectionID string, dataHandler, dataType string, data proto.Message) error
	Disconnect(connectionID string)
	Shutdown()
}

func EncodeMessage(dataHandler, dataType string, dataProto proto.Message) ([]byte, error) {
	protoBytes, err := proto.Marshal(dataProto)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}
	networkData := &SMessage{
		Handler: dataHandler,
		Type:    dataType,
		Proto:   protoBytes,
	}
	return proto.Marshal(networkData)
}
