package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/alfrunes/mqttie/util"
	"github.com/google/uuid"
)

type Connect struct {
	Version      version
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
	Version        version
}

type Disconnect struct {
	Version version
}

func NewConnectPacket(version version, keepAlive uint16) *Connect {
	return &Connect{
		Version:   version,
		KeepAlive: keepAlive,
	}
}

func NewConnAckPacket(version version) *ConnAck {
	return &ConnAck{
		Version: version,
	}
}

func NewDisconnectPacket(version version) *Disconnect {
	return &Disconnect{
		Version: version,
	}
}

// WriteTo marshals and writes the connect request to the stream w.
func (c *Connect) WriteTo(w io.Writer) (n int64, err error) {
	var flags uint8
	var buf [4]byte
	// Initialize length to fixed variable header length:
	//     "MQTT" + version + Flags + KeepAlive
	var length int = 10
	if c.WillQoS > 2 {
		return 0, fmt.Errorf("illegal QoS value (highest: 2)")
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
			return 0, err
		}
		c.ClientID = uuid.String()
	}
	length += len(c.ClientID) + 2
	l, err := util.EncodeUvarint(buf[:], uint32(length))

	// Encode message to stream
	// Fixed header
	N, err := w.Write(append([]byte{cmdConnect}, buf[:l]...))
	n += int64(N)
	if err != nil {
		return n, err
	} else if n < int64(l+1) {
		return n, io.ErrShortWrite
	}
	// Variable header
	N, err = w.Write([]byte{0, 4, 'M', 'Q', 'T', 'T',
		uint8(c.Version), flags})
	n += int64(N)
	if err != nil {
		return n, err
	} else if N < 8 {
		return n, io.ErrShortWrite
	}
	binary.BigEndian.PutUint16(buf[:2], c.KeepAlive)

	// Payload
	binary.BigEndian.PutUint16(buf[2:], uint16(len(c.ClientID)))
	N, err = w.Write(buf[:])
	n += int64(N)
	if err != nil {
		return n, err
	} else if N < 4 {
		return n, io.ErrShortWrite
	}

	N, err = w.Write([]byte(c.ClientID))
	n += int64(N)
	if err != nil {
		return n, err
	} else if n < int64(len(c.ClientID)) {
		return n, io.ErrShortWrite
	}

	if c.WillTopic != "" {
		N, err = util.WriteUTF8(w, c.WillTopic)
		n += int64(N)
		if err != nil {
			return n, err
		}
		l := len(c.WillMessage) + 2
		if l > 0xFFFFFFFF {
			return n, fmt.Errorf("connect: WillMessage too long")
		}
		binary.BigEndian.PutUint16(buf[:2], uint16(l))
		N, err = w.Write(buf[:2])
		n += int64(N)
		if err != nil {
			return n, err
		}
		N, err = w.Write(c.WillMessage)
		n += int64(N)
		if err != nil {
			return n, err
		} else if N < len(c.WillMessage) {
			return n, io.ErrShortWrite
		}
	}

	if c.Username != "" {
		N, err = util.WriteUTF8(w, c.Username)
		n += int64(N)
		if err != nil {
			return n, err
		}
		if c.Password != "" {
			N, err = util.WriteUTF8(w, c.Password)
			n += int64(N)
			if err != nil {
				return n, err
			}
		}
	}
	return n, nil
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
		return n, ErrPacketShort
	}
	// Parse variable header
	if bytes.Compare(buf[2:6], []byte{'M', 'Q', 'T', 'T'}) != 0 {
		return n, fmt.Errorf(
			"connect: unknown protocol: %s", string(buf[2:6]))
	}
	switch version(buf[6]) {
	case MQTTv311:
		c.Version = MQTTv311
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
		return n, ErrPacketShort
	}
	if flags&ConnectFlagWill > 0 {
		c.WillTopic, N, err = util.ReadUTF8(r)
		n += int64(len(c.WillTopic) + 2)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, ErrPacketShort
		}
		N, err = r.Read(buf[:2])
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, ErrPacketShort
		}
		l16 := binary.BigEndian.Uint16(buf[:2])
		c.WillMessage = make([]byte, l16-2)
		N, err := r.Read(c.WillMessage)
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, ErrPacketShort
		}
	}

	if flags&ConnectFlagUsername > 0 {
		c.Username, N, err = util.ReadUTF8(r)
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, ErrPacketShort
		}
		if flags&ConnectFlagPassword > 0 {
			c.Password, N, err = util.ReadUTF8(r)
			n += int64(N)
			if err != nil {
				return n, err
			} else if length -= N; length < 0 {
				return n, ErrPacketShort
			}
		}
	}

	return n, nil
}

// WriteTo writes the marshaled ConnAck packet to the stream w.
func (c *ConnAck) WriteTo(w io.Writer) (n int64, err error) {
	var flags uint8
	if c.SessionPresent {
		flags |= ConnAckFlagSessionPresent
	}

	N, err := w.Write([]byte{cmdConnAck, 2, flags, c.ReturnCode})
	return int64(N), err
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

// WriteTo writes the marshaled Disconnect request to stream.
func (d *Disconnect) WriteTo(w io.Writer) (n int64, err error) {
	N, err := w.Write([]byte{cmdDisconnect, 0})
	return int64(N), err
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
