package gonetwork

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type Server struct {
	config  ServerConfig
	tcpServ ServerConnection
	wsServ  ServerConnection
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewServer(config ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	var tcpServ, wsServ ServerConnection
	if config.TCPPort != "" {
		tcpServ = STCPNew(config)
	}
	if config.WSPort != "" {
		wsServ = SWSNew(config)
	}
	return &Server{
		config:  config,
		tcpServ: tcpServ,
		wsServ:  wsServ,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *Server) Listen() {
	if s.tcpServ != nil {
		go s.tcpServ.Listen(s.ctx)
	}
	if s.wsServ != nil {
		go s.wsServ.Listen(s.ctx)
	}
}

func (s *Server) Broadcast(dataHandler, dataType string, data proto.Message) {
	if s.tcpServ != nil {
		s.tcpServ.Broadcast(dataHandler, dataType, data)
	}
	if s.wsServ != nil {
		s.wsServ.Broadcast(dataHandler, dataType, data)
	}
}

func (s *Server) Send(connectionID, dataHandler, dataType string, data proto.Message) {
	if s.tcpServ != nil {
		s.tcpServ.Send(connectionID, dataHandler, dataType, data)
	}
	if s.wsServ != nil {
		s.wsServ.Send(connectionID, dataHandler, dataType, data)
	}
}

func (s *Server) Disconnect(connectionID string) {
	if s.tcpServ != nil {
		s.tcpServ.Disconnect(connectionID)
	}
	if s.wsServ != nil {
		s.wsServ.Disconnect(connectionID)
	}
}

func (s *Server) Shutdown() {
	s.cancel()
	if s.tcpServ != nil {
		s.tcpServ.Shutdown()
	}
	if s.wsServ != nil {
		s.wsServ.Shutdown()
	}
}
