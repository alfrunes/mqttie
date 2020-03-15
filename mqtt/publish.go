package mqtt

import (
	"encoding/binary"
	"io"

	"github.com/alfrunes/mqttie/util"
)

func NewPublishPacket(
	version version,
	topicName string,
	packetID uint16,
) *Publish {
	return &Publish{
		Version: version,

		TopicName:        topicName,
		PacketIdentifier: packetID,
	}
}

func NewPubAckPacket(version version, packetID uint16) *PubAck {
	return &PubAck{
		Version: version,

		PacketIdentifier: packetID,
	}
}

func NewPubRecPacket(version version, packetID uint16) *PubRec {
	return &PubRec{
		Version: version,

		PacketIdentifier: packetID,
	}
}

func NewPubRelPacket(version version, packetID uint16) *PubRel {
	return &PubRel{
		Version: version,

		PacketIdentifier: packetID,
	}
}

func NewPubCompPacket(version version, packetID uint16) *PubComp {
	return &PubComp{
		Version: version,

		PacketIdentifier: packetID,
	}
}

type Publish struct {
	Version version

	// Flags
	Duplicate bool
	QoSLevel  uint8
	Retain    bool

	// Variable header
	TopicName        string
	PacketIdentifier uint16

	Payload []byte
}

type PubAck struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubRec struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubRel struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

type PubComp struct {
	Version version

	// Variable header
	PacketIdentifier uint16
}

func (p *Publish) Marshal() (b []byte, err error) {
	var buf [4]byte
	var i int
	fixedHeader := cmdPublish
	if p.Duplicate {
		fixedHeader |= PublishFlagDuplicate
	}
	if p.QoSLevel > 0 {
		fixedHeader |= (p.QoSLevel << 1)
	}
	if p.Retain {
		fixedHeader |= PublishFlagRetain
	}
	// Remaining length = len(utf-8(topicName))
	//                  + len(packageIdentifier)
	//                  + len(payload)
	remLength := uint32(len(p.TopicName) + 4 + len(p.Payload))

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
	n, err = util.EncodeUTF8(b[i:], p.TopicName)
	i += n
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(b[i:], p.PacketIdentifier)
	i += 2
	copy(b[i:], p.Payload)
	return b, err
}

func (p *Publish) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	n = int64(len(b))
	return n, err
}

func (p *Publish) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	remLength, N, err := util.ReadVarint(r)
	length := int(remLength)
	n = int64(N)
	if err != nil {
		return n, err
	}
	p.TopicName, N, err = util.ReadUTF8(r)
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	} else if length <= 0 {
		return n, ErrPacketShort
	}
	N, err = r.Read(buf[:])
	length -= N
	n += int64(N)
	if err != nil {
		return n, err
	} else if length < 0 {
		// NOTE: payload can be zero length
		return n, ErrPacketShort
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	p.Payload = make([]byte, length)
	N, err = r.Read(p.Payload)
	n += int64(N)
	return n, err
}

func (p *PubAck) Marshal() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubAck
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubAck) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
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
	} else if n < 2 {
		return n, io.ErrUnexpectedEOF
	} else if n > 2 {
		return n, ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubRec) Marshal() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubRec
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubRec) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
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
	} else if n < 2 {
		return n, io.ErrUnexpectedEOF
	} else if n > 2 {
		return n, ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubRel) Marshal() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubRel
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubRel) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
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
	} else if n < 2 {
		return n, io.ErrUnexpectedEOF
	} else if n > 2 {
		return n, ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}

func (p *PubComp) Marshal() (b []byte, err error) {
	b = make([]byte, 4)
	b[0] = cmdPubComp
	b[1] = 2
	binary.BigEndian.PutUint16(b[2:], p.PacketIdentifier)
	return b, err
}

func (p *PubComp) WriteTo(w io.Writer) (n int64, err error) {
	b, err := p.Marshal()
	if err != nil {
		return n, err
	}
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
	} else if n < 2 {
		return n, io.ErrUnexpectedEOF
	} else if n > 2 {
		return n, ErrPacketLong
	}
	N, err = r.Read(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	}
	p.PacketIdentifier = binary.BigEndian.Uint16(buf[:])
	return n, err
}
