package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	connect := &Connect{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(connect)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.Equal(t, connect, p)
	connect.ClientID = "foobar"
	connect.CleanSession = true
	connect.KeepAlive = 123
	connect.Username = "foo@bar.com"
	connect.Password = "foobar"
	connect.WillMessage = []byte("foo")
	connect.WillRetain = true
	connect.WillTopic = mqtt.Topic{
		Name: "foo",
		QoS:  mqtt.QoS1,
	}
	buf.Reset()
	err = bufIO.Send(connect)
	assert.NoError(t, err)
	p, err = bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, connect, p) {
		assert.Equal(t, connect, p)
	}

	// Illegal received mesage
	buf.Write([]byte{cmdConnect, 3, 1, 2, 3})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	buf.Reset()
	buf.Write([]byte{cmdConnect, 13})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdConnect, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdConnect, 13, 1, 2, 'M', 'Q', 'T', 'T', 7, 8, 9, 10})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdConnect, 13, 1, 2, 'P', 'Q', 'T', 'T', 7, 8, 9, 10})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Truncate buffer to cause io errors
	b, _ := connect.MarshalBinary()
	buf.Write(b)
	// Error reading password
	buf.Truncate(len(b) - 10)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error reading username
	buf.Write(b)
	buf.Truncate(len(b) - 20)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Error reading will message
	buf.Write(b)
	buf.Truncate(len(b) - 25)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Error reading will message length
	buf.Write(b)
	buf.Truncate(len(b) - 28)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Error reading will topic
	buf.Write(b)
	buf.Truncate(len(b) - 30)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Error reading will client id
	buf.Write(b)
	buf.Truncate(len(b) - 35)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Shorten remaining length
	// Error at client id
	b[1] = 15
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error at will topic
	b[1] = 20
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error at will message length
	b[1] = 23
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error at will message
	b[1] = 25
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error at username
	b[1] = 30
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	// Error at username
	b[1] = 42
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()

	b[9] &= ^ConnectFlagWill
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Invalid QoS value
	connect.WillTopic.QoS = mqtt.QoS(3)
	_, err = connect.MarshalBinary()
	assert.Error(t, err)
	buf.Reset()
	connect.WillTopic.QoS = mqtt.QoS2
	// Illegal will message
	connect.WillMessage = make([]byte, int(^uint16(0))+1)
	err = bufIO.Send(connect)
	assert.Error(t, err)
}

func TestConnAck(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(1))
	connAck := &ConnAck{
		Version:        mqtt.MQTTv311,
		SessionPresent: true,
	}
	err := bufIO.Send(connAck)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, connAck, p) {
		assert.Equal(t, connAck, p)
	}

	connAck.SessionPresent = false
	b, _ := connAck.MarshalBinary()
	b[2] |= 0x0F
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	b[1] = 4
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())
	buf.Reset()
	b[1] = 0
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
}

func TestDisconnect(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(1))
	d := &Disconnect{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(d)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.IsType(t, d, p)
	b, _ := d.MarshalBinary()
	b[1] = 2
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
	err = bufIO.Close()
	assert.NoError(t, err)
}

func TestPing(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	ping := &PingReq{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(ping)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, ping) {
		pingRet := p.(*PingReq)
		assert.Equal(t, pingRet, ping)
	}

	// Test decode malformed packet
	raw, err := ping.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 5
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Reset()
	buf.Write([]byte{cmdPingReq})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(ping)
	assert.Error(t, err)

	// Test broken reader
	conn.readErr = fmt.Errorf("bar")
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestPingResp(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pingResp := &PingResp{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(pingResp)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, pingResp) {
		assert.Equal(t, pingResp, p)
	}

	b, err := pingResp.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())
	buf.Reset()
	buf.Write([]byte{cmdPingResp})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(pingResp)
	assert.Error(t, err)

	// Test broken reader
	conn.readErr = fmt.Errorf("foo")
	_, err = bufIO.Recv()
	assert.Error(t, err)

}

func TestPublish(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pub := &Publish{
		Version: mqtt.MQTTv311,
		Topic: mqtt.Topic{
			Name: "foo/bar",
			QoS:  mqtt.QoS0,
		},
		Payload: []byte("baz"),
	}
	err := bufIO.Send(pub)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, pub, p) {
		assert.Equal(t, pub, p)
	}

	pub.Duplicate = true
	pub.Topic.QoS = mqtt.QoS2
	pub.Retain = true
	err = bufIO.Send(pub)
	assert.NoError(t, err)
	p, err = bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, pub, p) {
		assert.Equal(t, pub, p)
	}

	b, _ := pub.MarshalBinary()
	buf.Write(b)
	buf.Truncate(1)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	buf.Reset()
	buf.Write(b)
	buf.Truncate(2)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	buf.Reset()
	buf.Write(b)
	buf.Truncate(len(b) - len(pub.Payload) - 2)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	buf.Reset()
	b[1] = byte(len(pub.Topic.Name))
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	buf.Reset()
	b[1] = byte(len(pub.Topic.Name) + 3)
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestPubAck(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pubAck := &PubAck{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(pubAck)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.Equal(t, pubAck, p)

	buf.Write([]byte{cmdPubAck})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Write([]byte{cmdPubAck, 1})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	buf.Write([]byte{cmdPubAck, 3})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Write([]byte{cmdPubAck, 2})
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestPubRec(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pubRec := &PubRec{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(pubRec)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.Equal(t, pubRec, p)

	buf.Write([]byte{cmdPubRec})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Write([]byte{cmdPubRec, 1})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	buf.Write([]byte{cmdPubRec, 3})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Write([]byte{cmdPubRec, 2})
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestPubRel(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pubRel := &PubRel{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(pubRel)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.Equal(t, pubRel, p)

	buf.Write([]byte{cmdPubRel})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Write([]byte{cmdPubRel, 1})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	buf.Write([]byte{cmdPubRel, 3})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Write([]byte{cmdPubRel, 2})
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestPubComp(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	pubComp := &PubComp{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(pubComp)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	assert.Equal(t, pubComp, p)

	buf.Write([]byte{cmdPubComp})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Write([]byte{cmdPubComp, 1})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	buf.Write([]byte{cmdPubComp, 3})
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Write([]byte{cmdPubComp, 2})
	_, err = bufIO.Recv()
	assert.Error(t, err)
}

func TestSubscribe(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	sub := &Subscribe{
		Version: mqtt.MQTTv311,
		Topics: []mqtt.Topic{
			{
				Name: "foo",
				QoS:  mqtt.QoS0,
			},
			{
				Name: "foo/bar",
				QoS:  mqtt.QoS1,
			},
			{
				Name: "foo/bar/baz",
				QoS:  mqtt.QoS2,
			},
		},
	}

	err := bufIO.Send(sub)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, sub) {
		assert.Equal(t, sub, p)
	}

	buf.Write([]byte{cmdSubscribe, 2, 0, 1})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdSubscribe, 0})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdSubscribe, 6, 0, 0, 0, 1, 'f'})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdSubscribe, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test malformed packet length
	buf.Reset()
	b, err := sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	buf.Reset()
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 127
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, io.EOF.Error())

	// Remaining length field too high
	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	binary.PutUvarint(b[1:], ^uint64(0))
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(sub)
	assert.Error(t, err)
}

func TestSubAck(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	subAck := &SubAck{
		Version: mqtt.MQTTv311,
		ReturnCodes: []uint8{
			0,
			1,
			2,
			128,
		},
	}
	err := bufIO.Send(subAck)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, subAck) {
		assert.Equal(t, subAck, p)
	}

	// Test broken reader
	buf.Write([]byte{cmdSubAck})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Write([]byte{cmdSubAck, 2, 0, 1})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdSubAck, 0})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test decode malformed packet
	raw, err := subAck.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 0
	buf.Reset()
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	raw[1] = 1
	buf.Reset()
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(subAck)
	assert.Error(t, err)

}

func TestUnsubAck(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	uAck := &UnsubAck{
		Version: mqtt.MQTTv311,
	}
	err := bufIO.Send(uAck)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, uAck) {
		assert.Equal(t, uAck, p)
	}

	// Test broken reader
	buf.Write([]byte{cmdUnsubAck, 2})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test decode malformed packet length
	buf.Reset()
	raw, err := uAck.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 5
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf.Reset()
	raw[1] = 0
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	buf.Reset()
	raw = append(raw, make([]byte, 10)...)
	binary.PutUvarint(raw[1:], uint64(^uint32(0)))
	buf.Write(raw)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(uAck)
	assert.Error(t, err)
}

func TestUnsubscribe(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	sub := &Unsubscribe{
		Version: mqtt.MQTTv311,
		Topics:  []string{"foo", "foo/bar", "foo/bar/baz"},
	}

	err := bufIO.Send(sub)
	assert.NoError(t, err)
	p, err := bufIO.Recv()
	assert.NoError(t, err)
	if assert.IsType(t, p, sub) {
		assert.Equal(t, sub, p)
	}

	// Test broken reader
	buf.Write([]byte{cmdUnsubscribe, 2, 0, 1})
	_, err = bufIO.Recv()
	assert.Error(t, err)
	buf.Reset()
	buf.Write([]byte{cmdUnsubscribe, 0})
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test malformed packet length
	buf.Reset()
	b, err := sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	buf.Reset()
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 127
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.EqualError(t, err, io.EOF.Error())

	// Remaining length field too high
	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	binary.PutUvarint(b[1:], ^uint64(0))
	buf.Write(b)
	_, err = bufIO.Recv()
	assert.Error(t, err)

	// Test broken writer
	conn.writeErr = fmt.Errorf("foo")
	err = bufIO.Send(sub)
	assert.Error(t, err)
}

func TestIllegalCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := NewBufferConn(buf)
	bufIO := NewPacketIO(conn, mqtt.MQTTv311, time.Duration(0))
	buf.Write([]byte{0})
	_, err := bufIO.Recv()
	assert.Error(t, err)
}
