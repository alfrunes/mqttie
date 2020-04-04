package mqtt

import (
	"fmt"
	"io"
)

type QoS uint8

const (
	QoS0 QoS = 0
	QoS1 QoS = 1
	QoS2 QoS = 2
)

type Version uint8

const (
	// Version definitions
	MQTTv311 Version = 0x04
	// TODO: MOAR VERSIONS!
)

var (
	ErrConnectBadVersion   = fmt.Errorf("protocol not supported by server")
	ErrConnectIDNotAllowed = fmt.Errorf("client id unacceptable by server")
	ErrConnectUnavailable  = fmt.Errorf("server unavailable")
	ErrConnectCredentials  = fmt.Errorf("incorrect username/password")
	ErrConnectUnauthorized = fmt.Errorf("client unauthorized with server")
)

var (
	ErrPacketShort = fmt.Errorf("malformed packet: length too short")
	ErrPacketLong  = fmt.Errorf("malformed packet: length too long")

	ErrIllegalQoS = fmt.Errorf("invalid QoS value")
)

type Packet interface {
	WriteTo(w io.Writer) (n int64, err error)
	ReadFrom(r io.Reader) (n int64, err error)
	MarshalBinary() (b []byte, err error)
}

type Topic struct {
	Name string
	QoS  QoS
}

type Subscription struct {
	Topic
	Recv chan<- []byte
}
