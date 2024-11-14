package gonetwork

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
)

type WebSocketServer struct {
	config      ServerConfig
	httpServer  *http.Server
	connections sync.Map
}

func SWSNew(config ServerConfig) *WebSocketServer {
	mux := http.NewServeMux()
	server := &WebSocketServer{
		config: config,
		httpServer: &http.Server{
			Addr:    config.Address + ":" + config.WSPort,
			Handler: mux,
		},
	}
	mux.HandleFunc("/", server.handleConnection)
	return server
}

func (s *WebSocketServer) Listen(ctx context.Context) error {
	go func() {
		if s.config.Secure {
			cert, _ := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
			s.httpServer.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
			s.httpServer.ListenAndServeTLS("", "")
		} else {
			s.httpServer.ListenAndServe()
		}
	}()
	return nil
}

func (s *WebSocketServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.config.AllowedOrigins})
	if err != nil {
		log.Printf("failed to accept WebSocket connection: %v", err)
		return
	}
	connID := r.RemoteAddr
	s.connections.Store(connID, conn)
	if s.config.OnConnected != nil {
		s.config.OnConnected(connID)
	}
	go s.handleClient(connID, conn)
}

func (s *WebSocketServer) handleClient(connID string, conn *websocket.Conn) {
	defer func() {
		conn.Close(websocket.StatusNormalClosure, "")
		s.connections.Delete(connID)
		if s.config.OnDisconnected != nil {
			s.config.OnDisconnected(connID)
		}
	}()

	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}
		if s.config.OnData != nil {
			s.config.OnData(connID, data)
		}
	}
}

func (s *WebSocketServer) Send(connectionID, dataHandler, dataType string, data proto.Message) error {
	dataByte, err := EncodeMessage(dataHandler, dataType, data)
	if err != nil {
		return err
	}
	if conn, ok := s.connections.Load(connectionID); ok {
		conn.(*websocket.Conn).Write(context.Background(), websocket.MessageBinary, dataByte)
		return nil
	}
	return fmt.Errorf("connection ID not found")
}

func (s *WebSocketServer) Broadcast(dataHandler, dataType string, data proto.Message) {
	dataByte, err := EncodeMessage(dataHandler, dataType, data)
	if err != nil {
		return
	}
	s.connections.Range(func(_, conn interface{}) bool {
		conn.(*websocket.Conn).Write(context.Background(), websocket.MessageBinary, dataByte)
		return true
	})
}

func (s *WebSocketServer) Disconnect(connectionID string) {
	if conn, ok := s.connections.Load(connectionID); ok {
		conn.(*websocket.Conn).Close(websocket.StatusNormalClosure, "")
		s.connections.Delete(connectionID)
	}
}

func (s *WebSocketServer) Shutdown() {
	s.httpServer.Shutdown(context.Background())
	s.connections.Range(func(_, conn interface{}) bool {
		conn.(*websocket.Conn).Close(websocket.StatusNormalClosure, "")
		return true
	})
}
