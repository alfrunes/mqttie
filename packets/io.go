package packets

import (
	"fmt"
	"net"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
)

type IO interface {
	Send(p mqtt.Packet) (err error)
	Recv() (p mqtt.Packet, err error)
	Close() error
}

type PacketIO struct {
	timeout   time.Duration
	conn      net.Conn
	version   mqtt.Version
	sendMutex chan struct{}
	recvMutex chan struct{}
}

func NewPacketIO(
	conn net.Conn,
	version mqtt.Version,
	timeout time.Duration,
) IO {
	return &PacketIO{
		timeout:   timeout,
		conn:      conn,
		version:   version,
		sendMutex: make(chan struct{}, 1),
		recvMutex: make(chan struct{}, 1),
	}
}

// Send writes the packet p to stream w, ensuring mutual exclusive access.
func (p *PacketIO) Send(pkt mqtt.Packet) (err error) {
	p.sendMutex <- struct{}{}
	defer func() { <-p.sendMutex }()
	if p.timeout > time.Duration(0) {
		p.conn.SetWriteDeadline(time.Now().Add(p.timeout))
	}
	_, err = pkt.WriteTo(p.conn)
	return err
}

// Recv reads and encodes a packet from stream. The Recv operation is protected
// by a mutex, but should only be handled by a single goroutine.
func (p *PacketIO) Recv() (pkg mqtt.Packet, err error) {
	var buf [1]byte
	p.recvMutex <- struct{}{}
	defer func() { <-p.recvMutex }()
	if p.timeout > time.Duration(0) {
		p.conn.SetWriteDeadline(time.Now().Add(p.timeout))
	}
	_, err = p.conn.Read(buf[:])
	if err != nil {
		return nil, err
	}
	cmdByte := buf[0]
	cmd := uint8(buf[0] & 0xF0)

	switch cmd {
	// TODO: Support for different MQTT versions
	case cmdConnect:
		connect := &Connect{
			Version: p.version,
		}
		_, err := connect.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = connect

	case cmdConnAck:
		connAck := &ConnAck{
			Version: p.version,
		}
		_, err := connAck.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = connAck

	case cmdPublish:
		pub := &Publish{
			Version: p.version,
		}
		if cmdByte&PublishFlagDuplicate > 0 {
			pub.Duplicate = true
		}
		if cmdByte&PublishFlagRetain > 0 {
			pub.Retain = true
		}
		pub.Topic.QoS = mqtt.QoS((cmdByte & 0x06) >> 1)

		_, err = pub.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pub

	case cmdPubAck:
		pubAck := &PubAck{
			Version: p.version,
		}
		_, err := pubAck.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pubAck

	case cmdPubRec:
		pubRec := &PubRec{
			Version: p.version,
		}
		_, err := pubRec.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pubRec

	case cmdPubRel:
		pubRel := &PubRel{
			Version: p.version,
		}
		_, err := pubRel.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pubRel

	case cmdPubComp:
		pubComp := &PubComp{
			Version: p.version,
		}
		_, err := pubComp.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pubComp

	case cmdSubscribe:
		sub := &Subscribe{
			Version: p.version,
		}
		_, err := sub.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = sub

	case cmdSubAck:
		subAck := &SubAck{
			Version: p.version,
		}
		_, err := subAck.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = subAck

	case cmdUnsubscribe:
		unSub := &Unsubscribe{
			Version: p.version,
		}
		_, err := unSub.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = unSub

	case cmdUnsubAck:
		unsubAck := &UnsubAck{
			Version: p.version,
		}
		_, err := unsubAck.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = unsubAck

	case cmdPingReq:
		ping := &PingReq{
			Version: p.version,
		}
		_, err := ping.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = ping

	case cmdPingResp:
		pingRsp := &PingResp{
			Version: p.version,
		}
		_, err := pingRsp.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = pingRsp

	case cmdDisconnect:
		disconnect := &Disconnect{
			Version: p.version,
		}
		_, err := disconnect.ReadFrom(p.conn)
		if err != nil {
			return nil, err
		}
		pkg = disconnect

	default:
		return nil, fmt.Errorf("invalid command byte: 0x%02X", cmd)
	}

	return pkg, err
}

func (p *PacketIO) Close() error {
	return p.conn.Close()
}
