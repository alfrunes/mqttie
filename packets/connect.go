package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/util"
	"github.com/google/uuid"
)

const (
	cmdConnect    uint8 = 0x10
	cmdConnAck    uint8 = 0x20
	cmdDisconnect uint8 = 0xE0

	// Flags
	ConnectFlagUsername       uint8 = 0x80
	ConnectFlagPassword       uint8 = 0x40
	ConnectFlagWillRetain     uint8 = 0x20
	ConnectFlagWill           uint8 = 0x04
	ConnectFlagCleanSession   uint8 = 0x02
	ConnAckFlagMaskv311       uint8 = 0x01
	ConnAckFlagSessionPresent uint8 = 0x01
)

type Connect struct {
	Version      mqtt.Version
	CleanSession bool
	KeepAlive    uint16

	ClientID string

	WillTopic   string
	WillMessage []byte
	WillQoS     uint8
	WillRetain  bool

	Username string
	Password string
}

type ConnAck struct {
	SessionPresent bool
	ReturnCode     uint8
	Version        mqtt.Version
}

type Disconnect struct {
	Version mqtt.Version
}

func NewConnectPacket(version mqtt.Version, keepAlive uint16) *Connect {
	return &Connect{
		Version:   version,
		KeepAlive: keepAlive,
	}
}

func NewConnAckPacket(version mqtt.Version) *ConnAck {
	return &ConnAck{
		Version: version,
	}
}

func NewDisconnectPacket(version mqtt.Version) *Disconnect {
	return &Disconnect{
		Version: version,
	}
}

func (c *Connect) MarshalBinary() (b []byte, err error) {
	var i int
	var flags uint8
	var buf [4]byte
	// Initialize length to fixed variable header length:
	//     "MQTT" + version + Flags + KeepAlive
	var length int = 10
	if c.WillQoS > 2 {
		return nil, fmt.Errorf("illegal QoS value (highest: 2)")
	}
	// Compute Flags
	if c.CleanSession {
		flags |= ConnectFlagCleanSession
	}
	if c.Username != "" {
		length += len(c.Username) + 2
		flags |= ConnectFlagUsername
		if c.Password != "" {
			length += len(c.Password) + 2
			flags |= ConnectFlagPassword
		}
	}
	if c.WillTopic != "" {
		length += len(c.WillTopic) + len(c.WillMessage) + 4
		flags |= (c.WillQoS << 3) | ConnectFlagWill
		if c.WillRetain {
			flags |= ConnectFlagWillRetain
		}
	}

	if len(c.ClientID) == 0 {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		c.ClientID = uuid.String()
	}
	length += len(c.ClientID) + 2
	l, err := util.EncodeUvarint(buf[:], uint32(length))

	// Encode message to stream
	// Fixed header
	b = make([]byte, length+1+l)
	b[0] = cmdConnect
	i++

	// Variable header
	i += copy(b[i:], buf[:l])
	// Variable header
	i += copy(b[i:], []byte{0, 4, 'M', 'Q', 'T', 'T',
		uint8(c.Version), flags})
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(b[i:], c.KeepAlive)
	i += 2

	// Payload
	n, err := util.EncodeUTF8(b[i:], c.ClientID)
	if err != nil {
		return nil, err
	}
	i += n

	if c.WillTopic != "" {
		n, err = util.EncodeUTF8(b[i:], c.WillTopic)
		if err != nil {
			return nil, err
		}
		i += n
		l := len(c.WillMessage) + 2
		if l > 0xFFFFFFFF {
			return nil, fmt.Errorf("connect: WillMessage too long")
		}
		binary.BigEndian.PutUint16(b[i:], uint16(l))
		i += 2
		i += copy(b[i:], c.WillMessage)
	}

	if c.Username != "" {
		n, err = util.EncodeUTF8(b[i:], c.Username)
		if err != nil {
			return nil, err
		}
		i += n
		if c.Password != "" {
			n, err = util.EncodeUTF8(b[i:], c.Password)
			if err != nil {
				return nil, err
			}
		}
	}
	return b, nil
}

// WriteTo marshals and writes the connect request to the stream w.
func (c *Connect) WriteTo(w io.Writer) (n int64, err error) {
	b, err := c.MarshalBinary()
	if err != nil {
		return 0, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

// ReadFrom reads and unmarshals a connect request from the stream.
// NOTE: it is assumed that the command byte has already been consumed.
func (c *Connect) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [10]byte

	l, N, err := util.ReadVarint(r)
	if err != nil {
		return n, err
	}
	length := int(l)
	n = int64(N)

	// Read variable header
	N, err = r.Read(buf[:])
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	} else if length <= 0 {
		return n, mqtt.ErrPacketShort
	}
	// Parse variable header
	if bytes.Compare(buf[2:6], []byte{'M', 'Q', 'T', 'T'}) != 0 {
		return n, fmt.Errorf(
			"connect: unknown protocol: %s", string(buf[2:6]))
	}
	switch mqtt.Version(buf[6]) {
	case mqtt.MQTTv311:
		c.Version = mqtt.MQTTv311
	default:
		return n, fmt.Errorf(
			"connect: unknown protocol version: %d", buf[6])
	}
	flags := uint8(buf[7])
	if flags&ConnectFlagWillRetain > 0 {
		if flags&ConnectFlagWill > 0 {
			return n, fmt.Errorf(
				"connect: illegal flag composition: 0x%02X",
				flags,
			)
		}
		c.WillRetain = true
	}
	if flags&ConnectFlagCleanSession > 0 {
		c.CleanSession = true
	}
	binary.BigEndian.PutUint16(buf[8:], c.KeepAlive)

	// Payload
	c.ClientID, N, err = util.ReadUTF8(r)
	n += int64(len(c.ClientID) + 2)
	if err != nil {
		return n, err
	} else if length -= N; length < 0 {
		return n, mqtt.ErrPacketShort
	}
	if flags&ConnectFlagWill > 0 {
		c.WillTopic, N, err = util.ReadUTF8(r)
		n += int64(len(c.WillTopic) + 2)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, mqtt.ErrPacketShort
		}
		N, err = r.Read(buf[:2])
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, mqtt.ErrPacketShort
		}
		l16 := binary.BigEndian.Uint16(buf[:2])
		c.WillMessage = make([]byte, l16-2)
		N, err := r.Read(c.WillMessage)
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, mqtt.ErrPacketShort
		}
	}

	if flags&ConnectFlagUsername > 0 {
		c.Username, N, err = util.ReadUTF8(r)
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, mqtt.ErrPacketShort
		}
		if flags&ConnectFlagPassword > 0 {
			c.Password, N, err = util.ReadUTF8(r)
			n += int64(N)
			if err != nil {
				return n, err
			} else if length -= N; length < 0 {
				return n, mqtt.ErrPacketShort
			}
		}
	}

	return n, nil
}

func (c *ConnAck) MarshalBinary() (b []byte, err error) {
	b = []byte{cmdConnAck, 2, 0, c.ReturnCode}
	if c.SessionPresent {
		b[2] |= ConnAckFlagSessionPresent
	}
	return b, nil
}

// WriteTo writes the marshaled ConnAck packet to the stream w.
func (c *ConnAck) WriteTo(w io.Writer) (n int64, err error) {
	b, err := c.MarshalBinary()
	if err != nil {
		return 0, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

// ReadFrom reads and unmarshals the ConnAck request from stream.
// NOTE: it is assumed that the command byte is already consumed from the reader.
func (c *ConnAck) ReadFrom(r io.Reader) (n int64, err error) {
	var raw [4]byte
	N, err := r.Read(raw[:])
	n = int64(N)
	flags := raw[2]
	if flags > ConnAckFlagSessionPresent {
		return n, fmt.Errorf("connack: illegal flags: %02X", flags)
	} else if flags&ConnAckFlagSessionPresent > 0 {
		c.SessionPresent = true
	}
	c.ReturnCode = raw[3]
	return n, nil
}

func (d *Disconnect) MarshalBinary() (b []byte, err error) {
	return []byte{cmdDisconnect, 0}, nil
}

// WriteTo writes the marshaled Disconnect request to stream.
func (d *Disconnect) WriteTo(w io.Writer) (n int64, err error) {
	b, err := d.MarshalBinary()
	if err != nil {
		return 0, err
	}
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

// ReadFrom reads the final length byte from stream, verifying that the packet
// is indeed a disconnect request.
func (d *Disconnect) ReadFrom(r io.Reader) (n int64, err error) {
	var b [1]byte
	N, err := r.Read(b[:])
	n = int64(N)
	if b[0] != byte(0) {
		return n, fmt.Errorf("disconnect: unexpected payload")
	}
	return n, err
}
