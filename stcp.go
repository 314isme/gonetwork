package gonetwork

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"
)

type TCPServer struct {
	config      ServerConfig
	listener    net.Listener
	connections sync.Map
	wg          sync.WaitGroup
}

func STCPNew(config ServerConfig) *TCPServer {
	return &TCPServer{config: config}
}

func (s *TCPServer) Listen(ctx context.Context) error {
	address := s.config.Address + ":" + s.config.TCPPort
	var err error
	if s.config.Secure {
		cert, _ := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		s.listener, err = tls.Listen("tcp", address, tlsConfig)
	} else {
		s.listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %v", err)
	}

	s.wg.Add(1)
	go s.acceptConnections(ctx)
	return nil
}

func (s *TCPServer) acceptConnections(_ context.Context) {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		connID := conn.RemoteAddr().String()
		s.connections.Store(connID, conn)
		if s.config.OnConnected != nil {
			s.config.OnConnected(connID)
		}
		go s.handleClient(connID, conn)
	}
}

func (s *TCPServer) handleClient(connID string, conn net.Conn) {
	defer func() {
		conn.Close()
		s.connections.Delete(connID)
		if s.config.OnDisconnected != nil {
			s.config.OnDisconnected(connID)
		}
	}()

	for {
		var length uint32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			return
		}
		data := make([]byte, length)
		if _, err := conn.Read(data); err != nil {
			return
		}
		if s.config.OnData != nil {
			s.config.OnData(connID, data)
		}
	}
}

func (s *TCPServer) Send(connectionID, dataHandler, dataType string, data proto.Message) error {
	dataByte, err := EncodeMessage(dataHandler, dataType, data)
	if err != nil {
		return err
	}
	if conn, ok := s.connections.Load(connectionID); ok {
		length := uint32(len(dataByte))
		binary.Write(conn.(net.Conn), binary.BigEndian, length)
		conn.(net.Conn).Write(dataByte)
		return nil
	}
	return fmt.Errorf("connection ID not found")
}

func (s *TCPServer) Broadcast(dataHandler, dataType string, data proto.Message) {
	dataByte, err := EncodeMessage(dataHandler, dataType, data)
	if err != nil {
		return
	}
	s.connections.Range(func(_, conn interface{}) bool {
		length := uint32(len(dataByte))
		binary.Write(conn.(net.Conn), binary.BigEndian, length)
		conn.(net.Conn).Write(dataByte)
		return true
	})
}

func (s *TCPServer) Disconnect(connectionID string) {
	if conn, ok := s.connections.Load(connectionID); ok {
		conn.(net.Conn).Close()
		s.connections.Delete(connectionID)
	}
}

func (s *TCPServer) Shutdown() {
	s.listener.Close()
	s.wg.Wait()
	s.connections.Range(func(_, conn interface{}) bool {
		conn.(net.Conn).Close()
		return true
	})
}
