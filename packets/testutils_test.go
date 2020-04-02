package packets

import (
	"bytes"
	"net"
	"time"
)

type BufferConn struct {
	*bytes.Buffer
	writeErr error
	readErr  error
}

func NewBufferConn(buf *bytes.Buffer) *BufferConn {
	return &BufferConn{Buffer: buf}
}

func (f *BufferConn) Write(b []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.Buffer.Write(b)
}

func (f *BufferConn) Read(b []byte) (int, error) {
	if f.readErr != nil {
		return 0, f.writeErr
	}
	return f.Buffer.Read(b)
}

func (f *BufferConn) LocalAddr() net.Addr {
	return nil
}

func (f *BufferConn) RemoteAddr() net.Addr {
	return nil
}

func (f *BufferConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *BufferConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *BufferConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (f *BufferConn) Close() error {
	return nil
}
