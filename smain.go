package gonetwork

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type Server struct {
	config    ServerConfig
	tcpServer *STcp
	wsServer  *SWs
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewServer(config ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config:    config,
		tcpServer: TCPNewS(config),
		wsServer:  WSNewS(config),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *Server) Listen() {
	if s.config.TCPPort != "" {
		go s.tcpServer.Listen(s.ctx)
	}
	if s.config.WSPort != "" {
		go s.wsServer.Listen(s.ctx)
	}
}

func (s *Server) Broadcast(dataHandler, dataType string, data proto.Message, except ...map[string]bool) {
	dataByte, err := Encode(dataHandler, dataType, data)
	if err != nil {
		return
	}
	if s.config.TCPPort != "" {
		s.tcpServer.Broadcast(dataByte, except...)
	}
	if s.config.WSPort != "" {
		s.wsServer.Broadcast(dataByte, except...)
	}
}

func (s *Server) Send(connectionID string, dataHandler, dataType string, data proto.Message) {
	dataByte, err := Encode(dataHandler, dataType, data)
	if err != nil {
		return
	}
	if s.config.TCPPort != "" && s.tcpServer.IsConnection(connectionID) {
		s.tcpServer.Send(connectionID, dataByte)
	}
	if s.config.WSPort != "" && s.wsServer.IsConnection(connectionID) {
		s.wsServer.Send(connectionID, dataByte)
	}
}

func (s *Server) GetConnection(connectionID string) interface{} {
	if s.config.TCPPort != "" && s.tcpServer.IsConnection(connectionID) {
		return s.tcpServer.GetConnection(connectionID)
	}
	if s.config.WSPort != "" && s.wsServer.IsConnection(connectionID) {
		return s.wsServer.GetConnection(connectionID)
	}
	return nil
}

func (s *Server) Shutdown() {
	s.cancel()
	s.tcpServer.Shutdown()
	s.wsServer.Shutdown()
}
