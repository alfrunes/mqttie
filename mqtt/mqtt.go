package mqtt

import (
	"fmt"
	"io"
)

var (
	ErrPacketShort = fmt.Errorf("packet malformed: length too short")
	ErrPacketLong  = fmt.Errorf("packet malformed: length too long")
)

type Packet interface {
	WriteTo(w io.Writer) (n int64, err error)
	ReadFrom(r io.Reader) (n int64, err error)
	Marshal() (b []byte, err error)
}

type version uint8

const (
	// MQTT commands
	FORBIDDEN      uint8 = 0x00
	cmdConnect     uint8 = 0x10
	cmdConnAck     uint8 = 0x20
	cmdPublish     uint8 = 0x30
	cmdPubAck      uint8 = 0x40
	cmdPubRec      uint8 = 0x50
	cmdPubRel      uint8 = 0x60
	cmdPubComp     uint8 = 0x70
	cmdSubscribe   uint8 = 0x80
	cmdSubAck      uint8 = 0x90
	cmdUnsubscribe uint8 = 0xA0
	cmdUnsubAck    uint8 = 0xB0
	cmdPingReq     uint8 = 0xC0
	cmdPingResp    uint8 = 0xD0
	cmdDisconnect  uint8 = 0xE0
	cmdAuth        uint8 = 0xF0

	// Version definitions
	MQTTv311 version = 0x04

	// CONNECT
	// Flags
	ConnectFlagUsername     uint8 = 0x80
	ConnectFlagPassword     uint8 = 0x40
	ConnectFlagWillRetain   uint8 = 0x20
	ConnectFlagWill         uint8 = 0x04
	ConnectFlagCleanSession uint8 = 0x02

	// CONNACK
	// ReturnCodes (MQTTv311)
	ConnAckAccepted     uint8 = 0x00
	ConnAckUnacceptable uint8 = 0x01
	ConnAckRejected     uint8 = 0x02
	ConnAckUnavailable  uint8 = 0x03
	ConnAckInvalidLogin uint8 = 0x04
	ConnAckUnauthorized uint8 = 0x05

	// Flags
	ConnAckFlagMaskv311       uint8 = 0x01
	ConnAckFlagSessionPresent uint8 = 0x01

	// PUBLISH
	PublishFlagDuplicate uint8 = 0x08
	PublishFlagRetain    uint8 = 0x01
)

func Send(w io.Writer, p Packet) (n int, err error) {
	N, err := p.WriteTo(w)
	n = int(N)
	return n, err
}

func Recv(r io.Reader) (p Packet, err error) {
	var buf [1]byte
	_, err = r.Read(buf[:])
	if err != nil {
		return nil, err
	}
	cmdByte := buf[0]
	cmd := uint8(buf[0] & 0xF0)

	switch cmd {
	// TODO: Support for different MQTT versions
	case cmdConnect:
		pkg := &Connect{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdConnAck:
		pkg := &ConnAck{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPublish:
		pkg := &Publish{
			Version: MQTTv311,
		}
		if cmdByte&PublishFlagDuplicate > 0 {
			pkg.Duplicate = true
		}
		if cmdByte&PublishFlagRetain > 0 {
			pkg.Retain = true
		}

		_, err = pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubAck:
		pkg := &PubAck{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubRec:
		pkg := &PubRec{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubRel:
		pkg := &PubRel{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubComp:
		pkg := &PubComp{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdSubscribe:
		pkg := &Subscribe{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdSubAck:
		pkg := &SubAck{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdUnsubscribe:
		pkg := &Unsubscribe{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdUnsubAck:
		pkg := &UnsubAck{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPingReq:
		pkg := &PingReq{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPingResp:
		pkg := &PingResp{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdDisconnect:
		pkg := &Disconnect{
			Version: MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdAuth:
		// TODO: MQTT v5
		fallthrough
	default:
		return nil, fmt.Errorf("invalid command byte: 0x%02X", cmd)
	}

	return p, err
}
