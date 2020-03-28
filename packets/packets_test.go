package packets

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/stretchr/testify/assert"
)

type ErrWriter struct {
	Err error
}

func (e *ErrWriter) Write(b []byte) (int, error) {
	if e.Err == nil {
		e.Err = io.ErrShortWrite
	}
	return -1, e.Err
}

type ErrReader struct {
	Buf []byte
	i   int
	Err error
}

func (e *ErrReader) Read(b []byte) (int, error) {
	if e.Buf != nil {
		if e.i < len(e.Buf) {
			i := copy(b, e.Buf[e.i:])
			e.i += i
			return i, nil
		}
	}
	if e.Err == nil {
		e.Err = io.ErrShortBuffer
	}
	return -1, e.Err
}

func TestConnect(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := &Connect{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, conn)
	assert.NoError(t, err)
	p, err := Recv(buf)
	if assert.IsType(t, conn, p) {
		assert.Equal(t, conn, p)
	}
	// TODO Test a fully configured encode decode
	// TODO all the error cases
	conn.ClientID = "foobar"
	conn.CleanSession = true
	conn.KeepAlive = 123
	conn.Username = "foo@bar.com"
	conn.Password = "foobar"
	conn.WillMessage = []byte("foo")
	conn.WillRetain = true
	conn.WillTopic = mqtt.Topic{
		Name: "foo",
		QoS:  mqtt.QoS1,
	}
	buf.Reset()
	_, err = Send(buf, conn)
	assert.NoError(t, err)
	p, err = Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, conn, p) {
		assert.Equal(t, conn, p)
	}

	// Illegal received mesage
	r := &ErrReader{}
	_, err = conn.ReadFrom(r)
	assert.Error(t, err)
	r.Buf = []byte{3, 1, 2, 3}
	_, err = conn.ReadFrom(r)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	r.i = 0
	r.Buf = []byte{13}
	_, err = conn.ReadFrom(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{13, 1, 2, 'M', 'Q', 'T', 'T', 7, 8, 9, 10}
	_, err = conn.ReadFrom(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{13, 1, 2, 'P', 'Q', 'T', 'T', 7, 8, 9, 10}
	_, err = conn.ReadFrom(r)
	assert.Error(t, err)

	// Truncate buffer to cause io errors
	b, err := conn.MarshalBinary()
	buf.Write(b)
	// Error reading password
	buf.Truncate(len(b) - 10)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error reading username
	buf.Write(b)
	buf.Truncate(len(b) - 20)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Error reading will message
	buf.Write(b)
	buf.Truncate(len(b) - 25)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Error reading will message length
	buf.Write(b)
	buf.Truncate(len(b) - 28)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Error reading will topic
	buf.Write(b)
	buf.Truncate(len(b) - 30)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Error reading will client id
	buf.Write(b)
	buf.Truncate(len(b) - 35)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Shorten remaining length
	// Error at client id
	b[1] = 15
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error at will topic
	b[1] = 20
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error at will message length
	b[1] = 23
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error at will message
	b[1] = 25
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error at username
	b[1] = 30
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	// Error at username
	b[1] = 42
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()

	b[9] &= ^ConnectFlagWill
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)

	// Invalid QoS value
	conn.WillTopic.QoS = mqtt.QoS(3)
	_, err = conn.MarshalBinary()
	assert.Error(t, err)
	buf.Reset()
	conn.WillTopic.QoS = mqtt.QoS2
	// Illegal will message
	conn.WillMessage = make([]byte, int(^uint16(0))+1)
	_, err = Send(buf, conn)
	assert.Error(t, err)
}

func TestConnAck(t *testing.T) {
	buf := &bytes.Buffer{}
	connAck := &ConnAck{
		Version:        mqtt.MQTTv311,
		SessionPresent: true,
	}
	_, err := Send(buf, connAck)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, connAck, p) {
		rsp := p.(*ConnAck)
		assert.Equal(t, connAck, rsp)
	}

	connAck.SessionPresent = false
	b, _ := connAck.MarshalBinary()
	b[2] |= 0x0F
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
	buf.Reset()
	b[1] = 4
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())
	buf.Reset()
	b[1] = 0
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
}

func TestDisconnect(t *testing.T) {
	buf := &bytes.Buffer{}
	d := &Disconnect{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, d)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	assert.IsType(t, d, p)
	b, _ := d.MarshalBinary()
	b[1] = 2
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
}

func TestPing(t *testing.T) {
	var buf *bytes.Buffer = &bytes.Buffer{}
	ping := &PingReq{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, ping)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, ping) {
		pingRet := p.(*PingReq)
		assert.Equal(t, pingRet, ping)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, ping)
	assert.Error(t, err)

	// Test broken reader
	r := &ErrReader{}
	r.Buf = []byte{cmdPingReq}
	_, err = Recv(r)
	assert.Error(t, err)

	// Test decode malformed packet
	raw, err := ping.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 5
	buf.Write(raw)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())
}

func TestPingResp(t *testing.T) {
	buf := &bytes.Buffer{}
	pingResp := &PingResp{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, pingResp)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, pingResp) {
		rsp := p.(*PingResp)
		assert.Equal(t, pingResp, rsp)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, pingResp)
	assert.Error(t, err)

	// Test broken reader
	r := &ErrReader{}
	r.Buf = []byte{cmdPingResp}
	_, err = Recv(r)
	assert.Error(t, err)

	b, err := pingResp.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())
}

func TestPublish(t *testing.T) {
	buf := &bytes.Buffer{}
	pub := &Publish{
		Version: mqtt.MQTTv311,
		Topic: mqtt.Topic{
			Name: "foo/bar",
			QoS:  mqtt.QoS0,
		},
		Payload: []byte("baz"),
	}
	_, err := Send(buf, pub)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, pub, p) {
		assert.Equal(t, pub, p)
	}

	pub.Duplicate = true
	pub.Topic.QoS = mqtt.QoS2
	pub.Retain = true
	_, err = Send(buf, pub)
	assert.NoError(t, err)
	p, err = Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, pub, p) {
		assert.Equal(t, pub, p)
	}

	b, _ := pub.MarshalBinary()
	buf.Write(b)
	buf.Truncate(1)
	_, err = Recv(buf)
	assert.Error(t, err)

	buf.Reset()
	buf.Write(b)
	buf.Truncate(2)
	_, err = Recv(buf)
	assert.Error(t, err)

	buf.Reset()
	buf.Write(b)
	buf.Truncate(len(b) - len(pub.Payload) - 2)
	_, err = Recv(buf)
	assert.Error(t, err)

	buf.Reset()
	b[1] = byte(len(pub.Topic.Name))
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)

	buf.Reset()
	b[1] = byte(len(pub.Topic.Name) + 3)
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
}

func TestPubAck(t *testing.T) {
	buf := &bytes.Buffer{}
	pubAck := &PubAck{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, pubAck)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	assert.Equal(t, pubAck, p)

	r := &ErrReader{}
	_, err = pubAck.ReadFrom(r)
	assert.Error(t, err)
	r.Buf = []byte{cmdPubAck, 1}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	r.i = 0
	r.Buf = []byte{cmdPubAck, 3}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	r.i = 0
	r.Buf = []byte{2}
	_, err = pubAck.ReadFrom(r)
	assert.Error(t, err)
}

func TestPubRec(t *testing.T) {
	buf := &bytes.Buffer{}
	pubRec := &PubRec{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, pubRec)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	assert.Equal(t, pubRec, p)

	r := &ErrReader{}
	_, err = pubRec.ReadFrom(r)
	assert.Error(t, err)
	r.Buf = []byte{cmdPubRec, 1}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	r.i = 0
	r.Buf = []byte{cmdPubRec, 3}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	r.i = 0
	r.Buf = []byte{2}
	_, err = pubRec.ReadFrom(r)
	assert.Error(t, err)
}

func TestPubRel(t *testing.T) {
	buf := &bytes.Buffer{}
	pubRel := &PubRel{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, pubRel)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	assert.Equal(t, pubRel, p)

	r := &ErrReader{}
	_, err = pubRel.ReadFrom(r)
	assert.Error(t, err)
	r.Buf = []byte{cmdPubRel, 1}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	r.i = 0
	r.Buf = []byte{cmdPubRel, 3}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	r.i = 0
	r.Buf = []byte{2}
	_, err = pubRel.ReadFrom(r)
	assert.Error(t, err)
}

func TestPubComp(t *testing.T) {
	buf := &bytes.Buffer{}
	pubComp := &PubComp{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, pubComp)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	assert.Equal(t, pubComp, p)

	r := &ErrReader{}
	_, err = pubComp.ReadFrom(r)
	assert.Error(t, err)
	r.Buf = []byte{cmdPubComp, 1}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	r.i = 0
	r.Buf = []byte{cmdPubComp, 3}
	_, err = Recv(r)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	r.i = 0
	r.Buf = []byte{2}
	_, err = pubComp.ReadFrom(r)
	assert.Error(t, err)
}

func TestSubscribe(t *testing.T) {
	buf := &bytes.Buffer{}
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

	_, err := Send(buf, sub)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, sub) {
		rsp := p.(*Subscribe)
		assert.Equal(t, sub, rsp)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, sub)
	assert.Error(t, err)
	r := &ErrReader{}
	_, err = Recv(r)
	r.Buf = []byte{cmdSubscribe, 2, 0, 1}
	_, err = Recv(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{cmdSubscribe, 0}
	_, err = Recv(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{cmdSubscribe, 6, 0, 0, 0, 1, 'f'}
	_, err = Recv(r)
	assert.Error(t, err)

	// Test malformed packet length
	b, err := sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 127
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, io.EOF.Error())

	// Remaining length field too high
	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	binary.PutUvarint(b[1:], ^uint64(0))
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
}

func TestSubAck(t *testing.T) {
	var buf *bytes.Buffer = &bytes.Buffer{}
	subAck := &SubAck{
		Version: mqtt.MQTTv311,
		ReturnCodes: []uint8{
			0,
			1,
			2,
			128,
		},
	}
	_, err := Send(buf, subAck)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, subAck) {
		subAckRet := p.(*SubAck)
		assert.Equal(t, subAck, subAckRet)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, subAck)
	assert.Error(t, err)

	// Test broken reader
	r := &ErrReader{}
	r.Buf = []byte{cmdSubAck}
	_, err = Recv(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{cmdSubAck, 2, 0, 1}
	_, err = Recv(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{cmdSubAck, 0}
	_, err = Recv(r)
	assert.Error(t, err)

	// Test decode malformed packet
	raw, err := subAck.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 0
	buf.Write(raw)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
	raw[1] = 1
	buf = &bytes.Buffer{}
	buf.Write(raw)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())
}

func TestUnsubAck(t *testing.T) {
	var buf *bytes.Buffer = &bytes.Buffer{}
	uAck := &UnsubAck{
		Version: mqtt.MQTTv311,
	}
	_, err := Send(buf, uAck)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, uAck) {
		uAckRet := p.(*UnsubAck)
		assert.Equal(t, uAck, uAckRet)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, uAck)
	assert.Error(t, err)

	// Test broken reader
	r := &ErrReader{}
	_, err = Recv(r)
	r.Buf = []byte{cmdUnsubAck, 2}
	_, err = Recv(r)
	assert.Error(t, err)
	buf = &bytes.Buffer{}

	// Test decode malformed packet length
	raw, err := uAck.MarshalBinary()
	assert.NoError(t, err)
	raw[1] = 5
	buf.Write(raw)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketLong.Error())

	buf = &bytes.Buffer{}
	raw[1] = 0
	buf.Write(raw)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	raw = append(raw, make([]byte, 10)...)
	binary.PutUvarint(raw[1:], uint64(^uint32(0)))
	buf = &bytes.Buffer{}
	buf.Write(raw)
	_, err = Recv(buf)
	assert.Error(t, err)
}

func TestUnsubscribe(t *testing.T) {
	buf := &bytes.Buffer{}
	sub := &Unsubscribe{
		Version: mqtt.MQTTv311,
		Topics:  []string{"foo", "foo/bar", "foo/bar/baz"},
	}

	_, err := Send(buf, sub)
	assert.NoError(t, err)
	p, err := Recv(buf)
	assert.NoError(t, err)
	if assert.IsType(t, p, sub) {
		rsp := p.(*Unsubscribe)
		assert.Equal(t, sub, rsp)
	}

	// Test broken writer
	w := &ErrWriter{}
	_, err = Send(w, sub)
	assert.Error(t, err)

	// Test broken reader
	r := &ErrReader{}
	_, err = Recv(r)
	r.Buf = []byte{cmdUnsubscribe, 2, 0, 1}
	_, err = Recv(r)
	assert.Error(t, err)
	r.i = 0
	r.Buf = []byte{cmdUnsubscribe, 0}
	_, err = Recv(r)
	assert.Error(t, err)

	// Test malformed packet length
	b, err := sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 5
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, mqtt.ErrPacketShort.Error())

	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	b[1] = 127
	buf.Write(b)
	_, err = Recv(buf)
	assert.EqualError(t, err, io.EOF.Error())

	// Remaining length field too high
	buf = &bytes.Buffer{}
	b, err = sub.MarshalBinary()
	assert.NoError(t, err)
	binary.PutUvarint(b[1:], ^uint64(0))
	buf.Write(b)
	_, err = Recv(buf)
	assert.Error(t, err)
}

func TestIllegalCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{0})
	_, err := Recv(buf)
	assert.Error(t, err)
}
