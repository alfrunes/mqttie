package packets

import (
	"io"

	"github.com/alfrunes/mqttie/mqtt"
)

const (
	cmdPingReq  uint8 = 0xC0
	cmdPingResp uint8 = 0xD0
)

type PingReq struct {
	Version mqtt.Version
}

type PingResp struct {
	Version mqtt.Version
}

func (p *PingReq) MarshalBinary() (b []byte, err error) {
	return []byte{cmdPingReq, 0}, nil
}

func (p *PingReq) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.MarshalBinary()
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
		return n, mqtt.ErrPacketLong
	}
	return n, nil
}

func (p *PingResp) MarshalBinary() (b []byte, err error) {
	return []byte{cmdPingResp, 0}, nil
}

func (p *PingResp) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.MarshalBinary()
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
		return n, mqtt.ErrPacketLong
	}
	return n, nil
}
