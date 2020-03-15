package mqtt

import (
	"io"
)

type PingReq struct {
	Version version
}

type PingResp struct {
	Version version
}

func (p *PingReq) Marshal() (b []byte, err error) {
	return []byte{cmdPingReq, 0}, nil
}

func (p *PingReq) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PingReq) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	N, err := r.Read(buf[:])
	n = int64(N)
	if err != nil {
		return n, err
	} else if N > 0 {
		return n, ErrPacketLong
	}
	return n, nil
}

func (p *PingResp) Marshal() (b []byte, err error) {
	return []byte{cmdPingResp, 0}, nil
}

func (p *PingResp) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PingResp) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	N, err := r.Read(buf[:])
	n = int64(N)
	if err != nil {
		return n, err
	} else if buf[0] > 0 {
		return n, ErrPacketLong
	}
	return n, nil
}
