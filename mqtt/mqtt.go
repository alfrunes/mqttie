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
	cmdSuback      uint8 = 0x90
	cmdUnsubscribe uint8 = 0xA0
	cmdUnsuback    uint8 = 0xB0
	cmdPingreq     uint8 = 0xC0
	cmdPingresp    uint8 = 0xD0
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
