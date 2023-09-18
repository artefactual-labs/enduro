// Code generated by goa v3.13.0, DO NOT EDIT.
//
// collection WebSocket client streaming
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"io"

	collection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionviews "github.com/artefactual-labs/enduro/internal/api/gen/collection/views"
	"github.com/gorilla/websocket"
	goahttp "goa.design/goa/v3/http"
)

// ConnConfigurer holds the websocket connection configurer functions for the
// streaming endpoints in "collection" service.
type ConnConfigurer struct {
	MonitorFn goahttp.ConnConfigureFunc
}

// MonitorClientStream implements the collection.MonitorClientStream interface.
type MonitorClientStream struct {
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

// Recv reads instances of "collection.EnduroMonitorUpdate" from the "monitor"
// endpoint websocket connection.
func (s *MonitorClientStream) Recv() (*collection.EnduroMonitorUpdate, error) {
	var (
		rv   *collection.EnduroMonitorUpdate
		body MonitorResponseBody
		err  error
	)
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewMonitorEnduroMonitorUpdateOK(&body)
	vres := &collectionviews.EnduroMonitorUpdate{res, "default"}
	if err := collectionviews.ValidateEnduroMonitorUpdate(vres); err != nil {
		return rv, goahttp.ErrValidationError("collection", "monitor", err)
	}
	return collection.NewEnduroMonitorUpdate(vres), nil
}
