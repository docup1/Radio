package infra

import (
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSProxy bridges a client WebSocket to an upstream WebSocket server.
type WSProxy struct {
	upstream url.URL
	dialer   websocket.Dialer
}

// NewWSProxy creates a proxy that will dial upstream (e.g. ws://sender-service:8081).
func NewWSProxy(upstreamURL string) (*WSProxy, error) {
	u, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}
	return &WSProxy{
		upstream: *u,
		dialer:   websocket.Dialer{HandshakeTimeout: 10 * time.Second},
	}, nil
}

// ServeWS upgrades the client connection, dials upstream, and bridges both directions.
// path is the sub-path to append to the upstream URL (e.g. "/stream/abc123").
func (p *WSProxy) ServeWS(w http.ResponseWriter, r *http.Request, path string) {
	// 1. Upgrade client → WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws-proxy] upgrade: %v", err)
		return
	}

	// 2. Dial upstream
	upstreamURL := p.upstream.String() + path
	upstreamConn, _, err := p.dialer.Dial(upstreamURL, nil)
	if err != nil {
		log.Printf("[ws-proxy] dial %s: %v", upstreamURL, err)
		clientConn.Close()
		return
	}

	// 3. Bridge bidirectionally
	var once sync.Once
	close := func() {
		once.Do(func() {
			clientConn.Close()
			upstreamConn.Close()
		})
	}
	defer close()

	done := make(chan struct{}, 2)

	// client → upstream
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			if err := upstreamConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	// upstream → client
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, msg, err := upstreamConn.ReadMessage()
			if err != nil {
				return
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	<-done
}
