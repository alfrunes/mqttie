package packets

import (
	"encoding/binary"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/util"
)

const (
	cmdSubscribe   uint8 = 0x80
	cmdSubAck      uint8 = 0x90
	cmdUnsubscribe uint8 = 0xA0
	cmdUnsubAck    uint8 = 0xB0
)

type Subscribe struct {
	Version mqtt.Version

	PacketIdentifier uint16

	// Payload
	Topics []mqtt.Topic
}

type SubAck struct {
	Version mqtt.Version

	PacketIdentifier uint16

	ReturnCodes []uint8
}

type Unsubscribe struct {
	Version mqtt.Version

	PacketIdentifier uint16

	Topics []string
}

type UnsubAck struct {
	Version mqtt.Version

	PacketIdentifier uint16
}

func (s *Subscribe) MarshalBinary() (b []byte, err error) {
	var buf [4]byte
	var i int
	var payloadLength int64
	for _, topic := range s.Topics {
		// Add length of utf-8 encoded topics + QoS byte
		payloadLength += int64(len(topic.Name) + 3)
	}

	// Remaining length = payloadLength + len(packetIdentifier)
	remainingLength := payloadLength + 2
	if remainingLength > int64(^uint32(0)) {
		// Casting to uint32 overflows
		return nil, mqtt.ErrPacketLong
	}
	N, err := util.EncodeUvarint(buf[:], uint32(remainingLength))
	if err != nil {
		return nil, err
	}
	b = make([]byte, int(remainingLength)+N+1)
	// FIXME: the flag section may change across versions
	b[0] = cmdSubscribe | 0x02
	i++
	i += copy(b[i:], buf[:N])
	binary.BigEndian.PutUint16(b[i:], s.PacketIdentifier)
	i += 2

	// Payload
	for _, topic := range s.Topics {
		n, err := util.EncodeUTF8(b[i:], topic.Name)
		if err != nil {
			return nil, err
		}
		i += n
		b[i] = byte(topic.QoS)
		i++
	}
	return b, nil
}

func (s *Subscribe) WriteTo(w io.Writer) (n int64, err error) {
	buf, err := s.MarshalBinary()
	if err != nil {
		return n, err
	}
	N, err := w.Write(buf)
	n = int64(N)
	return n, err
}

func (s *Subscribe) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	n = int64(N)
	if err != nil {
		return n, err
	}
	length := int(remLength)

	N, err = r.Read(buf[:])
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	} else if length <= 0 {
		return n, mqtt.ErrPacketShort
	}
	s.PacketIdentifier = binary.BigEndian.Uint16(buf[:])

	// Payload
	s.Topics = []mqtt.Topic{}
	for length > 0 {
		topicFilter := mqtt.Topic{}
		topicFilter.Name, N, err = util.ReadUTF8(r)
		n += int64(N)
		length -= N
		if err != nil {
			return n, err
		}
		N, err = r.Read(buf[:1])
		n += int64(N)
		length -= N
		if err != nil {
			return n, err
		} else if length < 0 {
			return n, mqtt.ErrPacketShort
		}
		topicFilter.QoS = mqtt.QoS(buf[0])
		s.Topics = append(s.Topics, topicFilter)
	}
	return n, err
}

func (s *SubAck) MarshalBinary() (b []byte, err error) {
	var i int
	var buf [4]byte
	remLength := len(s.ReturnCodes) + 2
	n, err := util.EncodeUvarint(buf[:], uint32(remLength))
	if err != nil {
		return nil, err
	}

	b = make([]byte, n+remLength+1)

	b[0] = cmdSubAck
	i++
	i += copy(b[i:], buf[:n])

	// Variable header
	binary.BigEndian.PutUint16(b[i:], s.PacketIdentifier)
	i += 2

	// Payload
	for _, code := range s.ReturnCodes {
		b[i] = code
		i++
	}
	return b, err
}

func (s *SubAck) WriteTo(w io.Writer) (n int64, err error) {
	b, err := s.MarshalBinary()
	if err != nil {
		return n, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (s *SubAck) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	n = int64(N)
	length := int(remLength)
	if err != nil {
		return n, err
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	} else if length -= N; length <= 0 {
		return n, mqtt.ErrPacketShort
	}
	s.PacketIdentifier = binary.BigEndian.Uint16(buf[:])

	s.ReturnCodes = make([]uint8, length)
	N, err = r.Read(s.ReturnCodes)
	n += int64(N)
	return n, err
}

func (u *Unsubscribe) MarshalBinary() (b []byte, err error) {
	var i int
	var buf [4]byte
	var remLength int = 2
	for _, topic := range u.Topics {
		remLength += len([]byte(topic)) + 2
	}
	n, err := util.EncodeUvarint(buf[:], uint32(remLength))

	b = make([]byte, n+remLength+1)
	// Fixed header
	b[0] = cmdUnsubscribe
	if u.Version == mqtt.MQTTv311 {
		b[0] |= 0x02
	}
	i++

	// Variable header
	i += copy(b[i:], buf[:n])
	binary.BigEndian.PutUint16(b[i:], u.PacketIdentifier)
	i += 2

	// Payload
	for _, topic := range u.Topics {
		n, err := util.EncodeUTF8(b[i:], topic)
		if err != nil {
			return nil, err
		}
		i += n
	}
	return b, nil
}

func (u *Unsubscribe) WriteTo(w io.Writer) (n int64, err error) {
	b, err := u.MarshalBinary()
	if err != nil {
		return 0, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (u *Unsubscribe) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	n = int64(N)
	if err != nil {
		return n, err
	}
	length := int(remLength)
	N, err = r.Read(buf[:])
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	} else if length <= 0 {
		return n, mqtt.ErrPacketShort
	}

	u.Topics = []string{}
	for length > 0 {
		topic, N, err := util.ReadUTF8(r)
		n += int64(N)
		length -= N
		if err != nil {
			return n, err
		} else if length < 0 {
			return n, mqtt.ErrPacketShort
		}
		u.Topics = append(u.Topics, topic)
	}
	return n, err
}

func (u *UnsubAck) MarshalBinary() (b []byte, err error) {
	b = []byte{cmdUnsubAck, 2, 0, 0}
	binary.BigEndian.PutUint16(b[2:], u.PacketIdentifier)
	return b, nil
}

func (u *UnsubAck) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := u.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (u *UnsubAck) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	n = int64(N)
	if err != nil {
		return n, err
	} else if remLength < 2 {
		return n, mqtt.ErrPacketShort
	} else if remLength > 2 {
		return n, mqtt.ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	u.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}
