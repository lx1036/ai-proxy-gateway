package xds

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"time"
)

// ConnectionContext is used by the RPC event loop to respond to requests and pushes.
type ConnectionContext struct {
	Connection

	// Original node metadata, to avoid unmarshal/marshal.
	// This is included in internal events.
	node *core.Node

	// proxy is the client to which this connection is established.
	proxy *model.Proxy

	// deltaStream is used for Delta XDS. Only one of deltaStream or stream will be set
	deltaStream DeltaDiscoveryStream

	deltaReqChan chan *discovery.DeltaDiscoveryRequest

	s   *DiscoveryServer
	ids []string
}

func newConnectionContext(peerAddr string, stream DiscoveryStream) *ConnectionContext {
	return &ConnectionContext{
		Connection: NewConnection(peerAddr, stream),
	}
}

func (conn *ConnectionContext) XdsConnection() *Connection {
	return &conn.Connection
}

func (conn *ConnectionContext) Watcher() Watcher {

}

// Initialize checks the first request.
func (conn *ConnectionContext) Initialize(node *core.Node) error {
	return conn.s.initConnection(node, conn, conn.ids)
}

// Close discards the connection.
func (conn *ConnectionContext) Close() {
	conn.s.closeConnection(conn)
}

// Process responds to a discovery request.
func (conn *ConnectionContext) Process(req *discovery.DiscoveryRequest) error {
	return conn.s.processRequest(req, conn)
}

// Event represents a config or registry event that results in a push.
type Event struct {
	// pushRequest PushRequest to use for the push.
	pushRequest *model.PushRequest

	// function to call once a push is finished. This must be called or future changes may be blocked.
	done func()
}

// Push responds to a push event queue
func (conn *ConnectionContext) Push(ev any) error {
	pushEv := ev.(*Event)
	err := conn.s.pushConnection(conn, pushEv)
	pushEv.done()
	return err
}

// Connection holds information about an xDS client connection. There may be more than one connection to the same client.
type Connection struct {

	// peerAddr is the address of the client, from network layer.
	peerAddr string
	// conID is the connection conID, used as a key in the connection table.
	// Currently based on the node name and a counter.
	conID string

	// initialized channel will be closed when proxy is initialized. Pushes, or anything accessing
	// the proxy, should not be started until this channel is closed.
	initialized chan struct{}

	// Time of connection, for debugging
	connectedAt time.Time

	// Both ADS and SDS streams implement this interface
	stream DiscoveryStream
	// Sending on this channel results in a push.
	pushChannel chan any

	// reqChan is used to receive discovery requests for this connection.
	reqChan chan *discovery.DiscoveryRequest

	// errorChan is used to process error during discovery request processing.
	errorChan chan error

	stop chan struct{}
}

func NewConnection(peerAddr string, stream DiscoveryStream) Connection {
	return Connection{
		pushChannel: make(chan any),
		initialized: make(chan struct{}),
		stop:        make(chan struct{}),
		reqChan:     make(chan *discovery.DiscoveryRequest, 1),
		errorChan:   make(chan error, 1),
		peerAddr:    peerAddr,
		connectedAt: time.Now(),
		stream:      stream,
	}
}
