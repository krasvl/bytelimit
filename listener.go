package bytelimit

import (
	"net"
)

// Listener is a net.Listener that wraps a net.Listener and limits the rate of requests.
type Listener struct {
	net.Listener
	limiter    *Limiter
	keyBuilder *KeyBuilder
}

// NewListener creates a new Listener with the given net.Listener, limiter, and keyBuilder.
// The limiter is used to limit the rate of requests.
// The keyBuilder is used to build the key for the rate limiter.
func NewListener(listener net.Listener, limiter *Limiter, keyBuilder *KeyBuilder) *Listener {
	return &Listener{
		Listener:   listener,
		limiter:    limiter,
		keyBuilder: keyBuilder,
	}
}

// Accept accepts a new connection and wraps it with a Conn.
// The Conn is used to limit the rate of requests.
// The Conn is returned.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	wrapConn, err := NewConn(conn, l.limiter, l.keyBuilder)
	if err != nil {
		if err := conn.Close(); err != nil {
			return nil, err
		}
		return nil, err
	}

	return wrapConn, nil
}
