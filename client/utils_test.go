package client

import (
	"io"
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

type FakeConn struct {
	mock.Mock

	// Used to make read block for input
	ReadChan chan []byte
	Buf      []byte
	i        int

	closeChan chan struct{}
}

func NewFakeConn(bufSize int) *FakeConn {
	return &FakeConn{
		ReadChan:  make(chan []byte, bufSize),
		closeChan: make(chan struct{}, 1),
	}
}

func (f *FakeConn) Write(b []byte) (int, error) {
	args := f.Called(b)

	var r0 int
	if rf, ok := args.Get(0).(func([]byte) int); ok {
		r0 = rf(b)
	} else {
		r0 = args.Int(0)
	}

	var r1 error
	if rf, ok := args.Get(1).(func([]byte) error); ok {
		r1 = rf(b)
	} else {
		r1 = args.Error(1)
	}
	return r0, r1
}

func (f *FakeConn) Read(b []byte) (int, error) {
	if f.Buf == nil {
		select {
		case f.Buf = <-f.ReadChan:
		case <-f.closeChan:
			return 0, io.EOF
		}
	}
	i := copy(b, f.Buf)
	if i < len(f.Buf) {
		f.Buf = f.Buf[i:]
	} else {
		f.Buf = nil
	}
	args := f.Called(b)

	var r0 int
	if rf, ok := args.Get(0).(func([]byte) int); ok {
		r0 = rf(b)
	} else {
		r0 = args.Int(0)
	}

	var r1 error
	if rf, ok := args.Get(1).(func([]byte) error); ok {
		r1 = rf(b)
	} else {
		r1 = args.Error(1)
	}

	return r0, r1
}

func (f *FakeConn) Close() error {
	args := f.Called()
	f.closeChan <- struct{}{}

	var r0 error
	if rf, ok := args.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = args.Error(0)
	}
	return r0
}

func (f *FakeConn) LocalAddr() net.Addr {
	args := f.Called()

	var r0 net.Addr
	if rf, ok := args.Get(0).(func() net.Addr); ok {
		r0 = rf()
	} else {
		r0 = args.Get(0).(net.Addr)
	}
	return r0
}

func (f *FakeConn) RemoteAddr() net.Addr {
	args := f.Called()

	var r0 net.Addr
	if rf, ok := args.Get(0).(func() net.Addr); ok {
		r0 = rf()
	} else {
		r0 = args.Get(0).(net.Addr)
	}
	return r0
}

func (f *FakeConn) SetDeadline(t time.Time) error {
	args := f.Called()

	var r0 error
	if rf, ok := args.Get(0).(func(time.Time) error); ok {
		r0 = rf(t)
	} else {
		r0 = args.Error(0)
	}
	return r0
}

func (f *FakeConn) SetReadDeadline(t time.Time) error {
	args := f.Called()

	var r0 error
	if rf, ok := args.Get(0).(func(time.Time) error); ok {
		r0 = rf(t)
	} else {
		r0 = args.Error(0)
	}
	return r0
}

func (f *FakeConn) SetWriteDeadline(t time.Time) error {
	args := f.Called()

	var r0 error
	if rf, ok := args.Get(0).(func(time.Time) error); ok {
		r0 = rf(t)
	} else {
		r0 = args.Error(0)
	}
	return r0
}
