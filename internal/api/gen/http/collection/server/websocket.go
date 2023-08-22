// Code generated by goa v3.12.4, DO NOT EDIT.
//
// collection WebSocket server streaming
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	collection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/gorilla/websocket"
	goahttp "goa.design/goa/v3/http"
)

// ConnConfigurer holds the websocket connection configurer functions for the
// streaming endpoints in "collection" service.
type ConnConfigurer struct {
	MonitorFn goahttp.ConnConfigureFunc
}

// MonitorServerStream implements the collection.MonitorServerStream interface.
type MonitorServerStream struct {
	once sync.Once
	// upgrader is the websocket connection upgrader.
	upgrader goahttp.Upgrader
	// configurer is the websocket connection configurer.
	configurer goahttp.ConnConfigureFunc
	// cancel is the context cancellation function which cancels the request
	// context when invoked.
	cancel context.CancelFunc
	// w is the HTTP response writer used in upgrading the connection.
	w http.ResponseWriter
	// r is the HTTP request.
	r *http.Request
	// conn is the underlying websocket connection.
	conn *websocket.Conn
}

// NewConnConfigurer initializes the websocket connection configurer function
// with fn for all the streaming endpoints in "collection" service.
func NewConnConfigurer(fn goahttp.ConnConfigureFunc) *ConnConfigurer {
	return &ConnConfigurer{
		MonitorFn: fn,
	}
}

// Send streams instances of "collection.EnduroMonitorUpdate" to the "monitor"
// endpoint websocket connection.
func (s *MonitorServerStream) Send(v *collection.EnduroMonitorUpdate) error {
	var err error
	// Upgrade the HTTP connection to a websocket connection only once. Connection
	// upgrade is done here so that authorization logic in the endpoint is executed
	// before calling the actual service method which may call Send().
	s.once.Do(func() {
		var conn *websocket.Conn
		conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
		if err != nil {
			return
		}
		if s.configurer != nil {
			conn = s.configurer(conn, s.cancel)
		}
		s.conn = conn
	})
	if err != nil {
		return err
	}
	res := collection.NewViewedEnduroMonitorUpdate(v, "default")
	body := NewMonitorResponseBody(res.Projected)
	return s.conn.WriteJSON(body)
}

// Close closes the "monitor" endpoint websocket connection.
func (s *MonitorServerStream) Close() error {
	var err error
	if s.conn == nil {
		return nil
	}
	if err = s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server closing connection"),
		time.Now().Add(time.Second),
	); err != nil {
		return err
	}
	return s.conn.Close()
}
