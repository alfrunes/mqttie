package mqtt

import (
	"fmt"
	"io"
)

const (
	QoS0 QoS = 0
	QoS1 QoS = 1
	QoS2 QoS = 2

	// Version definitions
	MQTTv311 Version = 0x04
	// TODO: MOAR VERSIONS!
)

var (
	ErrPacketShort = fmt.Errorf("packet malformed: length too short")
	ErrPacketLong  = fmt.Errorf("packet malformed: length too long")
)

type Version uint8

type QoS uint8

type Packet interface {
	WriteTo(w io.Writer) (n int64, err error)
	ReadFrom(r io.Reader) (n int64, err error)
	MarshalBinary() (b []byte, err error)
}
