package packets

import (
	"fmt"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
)

var (
	sendMutex = make(chan struct{}, 1)
	recvMutex = make(chan struct{}, 1)
)

// Send writes the packet p to stream w, ensuring mutual exclusive access.
func Send(w io.Writer, p mqtt.Packet) (n int, err error) {
	sendMutex <- struct{}{}
	defer func() { <-sendMutex }()
	N, err := p.WriteTo(w)
	n = int(N)
	return n, err
}

// Recv reads and encodes a packet from stream. The Recv operation is protected
// by a mutex, but should only be handled by a single goroutine.
func Recv(r io.Reader) (p mqtt.Packet, err error) {
	var buf [1]byte
	recvMutex <- struct{}{}
	defer func() { <-recvMutex }()
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
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdConnAck:
		pkg := &ConnAck{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPublish:
		pkg := &Publish{
			Version: mqtt.MQTTv311,
		}
		if cmdByte&PublishFlagDuplicate > 0 {
			pkg.Duplicate = true
		}
		if cmdByte&PublishFlagRetain > 0 {
			pkg.Retain = true
		}
		pkg.Topic.QoS = mqtt.QoS((cmdByte & 0x06) >> 1)

		_, err = pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubAck:
		pkg := &PubAck{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubRec:
		pkg := &PubRec{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubRel:
		pkg := &PubRel{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPubComp:
		pkg := &PubComp{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdSubscribe:
		pkg := &Subscribe{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdSubAck:
		pkg := &SubAck{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdUnsubscribe:
		pkg := &Unsubscribe{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdUnsubAck:
		pkg := &UnsubAck{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPingReq:
		pkg := &PingReq{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdPingResp:
		pkg := &PingResp{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	case cmdDisconnect:
		pkg := &Disconnect{
			Version: mqtt.MQTTv311,
		}
		_, err := pkg.ReadFrom(r)
		if err != nil {
			return nil, err
		}
		p = pkg

	default:
		return nil, fmt.Errorf("invalid command byte: 0x%02X", cmd)
	}

	return p, err
}
