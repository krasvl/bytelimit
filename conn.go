package bytelimit

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
)

// Conn is a net.Conn that wraps a net.Conn and limits the rate of requests.
type Conn struct {
	bufConn
	key     string
	limiter *Limiter
}

// NewConn creates a new Conn with the given net.Conn, limiter, and keyBuilder.
// The limiter is used to limit the rate of requests.
// The keyBuilder is used to build the key for the rate limiter.
func NewConn(conn net.Conn, limiter *Limiter, keyBuilder *KeyBuilder) (net.Conn, error) {
	// read request to build limiter key
	tap := &bytes.Buffer{}
	tee := io.TeeReader(conn, tap)
	br := bufio.NewReader(tee)

	req, err := http.ReadRequest(br)
	if err != nil {
		return nil, err
	}

	// create buffered net.Conn to read all request data
	bufConn := bufConn{Conn: conn}
	bufConn.Unread(tap.Bytes())

	// if the request is not allowed, return the buffered net.Conn
	if !keyBuilder.Allow(req) {
		return &bufConn, nil
	}

	// build limiter key
	key := keyBuilder.Key(req)

	// return wrapped net.Conn
	return &Conn{
		bufConn: bufConn,
		key:     key,
		limiter: limiter,
	}, nil
}

// Read reads data from the Conn.
// If the rate limit is exceeded, a 429 Too Many Requests response is written to the Conn and the Conn is closed.
// The number of bytes read is returned.
// The error is returned.
func (c *Conn) Read(p []byte) (n int, err error) {
	n, err = c.bufConn.Read(p)
	if err == nil && !c.limiter.AllowN(c.key, Unit(n)) {
		resp := &http.Response{
			Status:     "429 Too Many Requests",
			StatusCode: http.StatusTooManyRequests,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
		}

		body := "Rate limit exceeded. Please try again later."
		resp.Body = io.NopCloser(strings.NewReader(body))
		resp.ContentLength = int64(len(body))

		if err := resp.Write(c); err != nil {
			return 0, err
		}
		if err := c.Close(); err != nil {
			return 0, err
		}
		return 0, io.EOF
	}

	return n, err
}

// bufConn is a net.Conn that wraps a net.Conn and buffers the data read from the Conn.
type bufConn struct {
	net.Conn
	buffer bytes.Buffer
}

// Read reads data from the bufConn.
// If the buffer has data, it is read from the buffer.
// Otherwise, the data is read from the Conn.
// The number of bytes read is returned.
// The error is returned.
func (rc *bufConn) Read(b []byte) (int, error) {
	if rc.buffer.Len() > 0 {
		return rc.buffer.Read(b)
	}

	return rc.Conn.Read(b)
}

// Unread unreads data to the bufConn.
// The data is added to the buffer.
// The buffer is then read from.
func (rc *bufConn) Unread(data []byte) {
	newBuffer := bytes.Buffer{}
	newBuffer.Write(data)

	if rc.buffer.Len() > 0 {
		oldData := rc.buffer.Bytes()
		newBuffer.Write(oldData)
	}

	rc.buffer = newBuffer
}
