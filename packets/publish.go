package packets

import (
	"encoding/binary"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/x/util"
)

const (
	cmdPublish uint8 = 0x30
	cmdPubAck  uint8 = 0x40
	cmdPubRec  uint8 = 0x50
	cmdPubRel  uint8 = 0x60
	cmdPubComp uint8 = 0x70

	// Flags
	PublishFlagDuplicate uint8 = 0x08
	PublishFlagRetain    uint8 = 0x01
)

type Publish struct {
	mqtt.Topic
	Version mqtt.Version

	// Flags
	Duplicate bool
	Retain    bool

	// Variable header
	PacketIdentifier uint16

	Payload []byte
}

type PubAck struct {
	Version mqtt.Version

	// Variable header
	PacketIdentifier uint16
}

type PubRec struct {
	Version mqtt.Version

	// Variable header
	PacketIdentifier uint16
}

type PubRel struct {
	Version mqtt.Version

	// Variable header
	PacketIdentifier uint16
}

type PubComp struct {
	Version mqtt.Version

	// Variable header
	PacketIdentifier uint16
}

func (p *Publish) MarshalBinary() (b []byte, err error) {
	var buf [4]byte
	var i int
	fixedHeader := cmdPublish
	if p.Duplicate {
		fixedHeader |= PublishFlagDuplicate
	}
	if p.QoS > 0 {
		fixedHeader |= (uint8(p.QoS) << 1)
	}
	if p.Retain {
		fixedHeader |= PublishFlagRetain
	}
	// Remaining length = len(utf-8(topicName))
	//                  + len(payload)
	//                  + (qos > 0 ) ? len(packet id) : 0
	remLength := uint32(len(p.Topic.Name) + 2 + len(p.Payload))
	if p.Topic.QoS > 0 {
		remLength += 2
	}

	n, err := util.EncodeUvarint(buf[:], remLength)
	if err != nil {
		return nil, err
	}

	// Length = remLength + len(remLength) + len(fixedHeader)
	b = make([]byte, int(remLength)+n+1)

	// FixedHeader
	b[i] = fixedHeader
	i++
	i += copy(b[i:], buf[:n])

	// Variable header
	n, err = util.EncodeUTF8(b[i:], p.Topic.Name)
	i += n
	if err != nil {
		return nil, err
	}
	if p.Topic.QoS > 0 {
		binary.BigEndian.PutUint16(b[i:], p.PacketIdentifier)
		i += 2
	}
	copy(b[i:], p.Payload)
	return b, err
}

func (p *Publish) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.MarshalBinary()
	if err != nil {
		return n, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

// ReadFrom reads a publish packet (minus command byte) from the stream.
// CAUTION: The least significant nibble from the command will not be parsed
// and must be set outside the scope of this function.
func (p *Publish) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	length := int(remLength)
	n = int64(N)
	if err != nil {
		return n, err
	}
	p.Topic.Name, N, err = util.ReadUTF8(r)
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	} else if length <= 0 {
		return n, mqtt.ErrPacketShort
	}
	if p.QoS > 0 {
		N, err = r.Read(buf[:])
		length -= N
		n += int64(N)
		if err != nil {
			return n, err
		} else if length < 0 {
			// NOTE: payload can be zero length
			return n, mqtt.ErrPacketShort
		}
		p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	}
	p.Payload = make([]byte, length)
	N, err = r.Read(p.Payload)
	n += int64(N)
	return n, err
}

func (p *PubAck) MarshalBinary() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubAck
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubAck) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := p.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PubAck) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	N, err := r.Read(buf[:1])
	n = int64(N)
	if err != nil {
		return n, err
	} else if buf[0] < byte(2) {
		return n, mqtt.ErrPacketShort
	} else if buf[0] > byte(2) {
		return n, mqtt.ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubRec) MarshalBinary() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubRec
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubRec) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := p.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PubRec) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	N, err := r.Read(buf[:1])
	n = int64(N)
	if err != nil {
		return n, err
	} else if buf[0] < byte(2) {
		return n, mqtt.ErrPacketShort
	} else if buf[0] > byte(2) {
		return n, mqtt.ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubRel) MarshalBinary() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubRel
	if p.Version == mqtt.MQTTv311 {
		b[0] |= 0x02
	}
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubRel) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := p.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PubRel) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	N, err := r.Read(buf[:1])
	n = int64(N)
	if err != nil {
		return n, err
	} else if buf[0] < byte(2) {
		return n, mqtt.ErrPacketShort
	} else if buf[0] > byte(2) {
		return n, mqtt.ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubComp) MarshalBinary() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubComp
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubComp) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := p.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

func (p *PubComp) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	N, err := r.Read(buf[:1])
	n = int64(N)
	if err != nil {
		return n, err
	} else if buf[0] < byte(2) {
		return n, mqtt.ErrPacketShort
	} else if buf[0] > byte(2) {
		return n, mqtt.ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}
