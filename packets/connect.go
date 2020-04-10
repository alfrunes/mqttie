package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/x/util"
	"github.com/satori/go.uuid"
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
	ConnectMaskWillQoS        uint8 = 0x18

	connPropSessionExpire       uint8 = 0x11
	connPropReceiveMax          uint8 = 0x21
	connPropMaxPacketSize       uint8 = 0x27
	connPropTopicAliasMax       uint8 = 0x22
	connPropRequestResponseInfo uint8 = 0x19
	connPropDisableProblemInfo  uint8 = 0x17
	connPropUserProperty        uint8 = 0x26
	connPropAuthMethod          uint8 = 0x15
	connPropAuthData            uint8 = 0x16
	connPropWillDelay           uint8 = 0x18
	connPropWillUTF8            uint8 = 0x01
	connPropWillExpire          uint8 = 0x02
	connPropWillContentType     uint8 = 0x03
	connPropWillResponseTopic   uint8 = 0x08
	connPropWillCorrelationData uint8 = 0x09
	connPropWillUserProps       uint8 = 0x26

	// ConnAck status codes
	ConnAckAccepted       = 0x00
	ConnAckBadVersion     = 0x01
	ConnAckIDNotAllowed   = 0x02
	ConnAckServerUnavail  = 0x03
	ConnAckBadCredentials = 0x04
	ConnAckUnauthorized   = 0x05
)

// Connect contains a structural representation of a connect packet. Some of
// the parameters are dependent, for instance: Password requires Username to
// be set. Other parameters are version dependent, these are highlighted in
// the parameter description.
type Connect struct {
	// Version holds the protocol version of this packet (see mqtt package).
	Version mqtt.Version
	// CleanSession stores the clean session flag (MQTT 3.1.1) or clean
	// start flag (MQTT 5.0).
	//
	// For MQTT 3.1.1 the clean session flag forces the server to discard
	// any previous session state and start a new one which last until the
	// client disconnects.
	//
	// For MQTT 5.0 however, the clean start flag only notifies the server
	// to discard session state on connect. What happens after disconnect
	// is determined by session expiry interval.
	CleanSession bool
	// KeepAlive contains the duration in seconds for the client to remain
	// inactive before getting disconnected.
	KeepAlive uint16

	// WillTopic is an optional topic to publish on a successful connect.
	WillTopic mqtt.Topic
	// WillMessage is the payload message for the WillTopic (max 64KiB).
	// NOTE: WillMessage requires WillTopic to be set, otherwise the
	// parameter is ignored.
	WillMessage []byte
	// WillRetain holds the retain flag for the published WillTopic. If
	// set to true, the server retains the message for future subscribers
	// on the WillTopic.
	WillRetain bool

	// ClientID stores the client identifier presented to the server. If
	// left empty a random UUID (v4) is automatically generated.
	ClientID string
	// Username holds the Username credential if the server has access
	// control enabled (cannot be an empty string).
	Username string
	// Password stores the password credential (cannot be an empty string).
	// NOTE: (MQTTv311) Password requires username to be assigned,
	// otherwise the parameter is ignored.
	Password string

	// The following parameters applies only to Version == MQTTv5

	// SessionExpiryInterval holds the duration in seconds the server is
	// required to store the session state. A value of 0xFFFFFFFF
	// (max(uint32)) makes the session does not expire. If the value 0 is
	// used, the session expire when the network connection is closed
	// (defaults to 0).
	SessionExpiryInterval uint32
	// MaxPacketSize tells the server the maximum packet size the client
	// is willing to accept (defaults to 0: no limit).
	MaxPacketSize uint32
	// ReceiveMax notifies the server about the number of QOS1 and QOS2
	// publish packets the client is willing to process simultaneously
	// (defaults to 65535).
	ReceiveMax uint16
	// TopicAliasMax sets the limit on the highest number of topic aliases
	// the client is willing to accept from the server (defaults to 0).
	TopicAliasMax uint16
	// RequestResponseInfo requests the server to return response
	// information in the ConnAck packet. (defaults to false).
	RequestResponseInfo bool
	// DisableProblemInfo requests the server NOT to return a reason string
	// or user properties on Publish, ConnAck and Disconnect packet
	// (defaults to false: enabled).
	DisableProblemInfo bool
	// ConnUserProperties contains user specified (connection related)
	// key-value pairs. The meaning of these properties is not defined by
	// the MQTT 5.0 specification
	// (ref. https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.pdf :871).
	ConnUserProperties map[string]string
	// AuthMethod specifies (and enables) the name of the authentication
	// method used for extensive authentication. If specified, the client
	// must not follow up with any other packets than Auth or Disconnect
	// packets until a ConnAck is received. (defaults to none/disabled).
	AuthMethod string
	// AuthData holds binary data associated with the specified AuthMethod.
	// If an AuthMethod is not specified this parameter is ignored.
	AuthData []byte

	// WillDelayInterval requests the server to delay publishing the
	// WillMessage after the set amount of seconds (defaults to 0).
	WillDelayInterval uint32
	// WillMessageExpiry sets the expiry interval for the published
	// WillMessage (defaults to 0: unset).
	WillMessageExpiry uint32
	// WillFormatUTF8 notifies the server that the WillMessage is encoded
	// using UTF8, otherwise the payload is treated as a stream of bytes
	// (default).
	WillFormatUTF8 bool
	// WillContentType provides a string content-type descriptor of the
	// published WillMessage (unset by default).
	WillContentType string
	// WillResponseTopic provides a response topic the recipients should
	// use to respond to the WillMessage. Setting the WillResponseTopic
	// enables request/response interaction between MQTT clients. (defaults
	// to none).
	WillResponseTopic string
	// WillCorrelationData is used by the sender of the request message to
	// identify which request the response message is for. The value is
	// ignored if WillResponseTopic is not set (defaults to unset).
	WillCorrelationData []byte
	// WillUserProperties provides user-specified key:value pairs of data
	// to the WillMessage. The interpretation of these parameters are
	// completely up to the user's application (think of it as custom HTTP
	// headers).
	WillUserProperties map[string]string
}

type ConnAck struct {
	SessionPresent bool
	ReturnCode     uint8
	Version        mqtt.Version
}

type Disconnect struct {
	Version mqtt.Version
}

// the following private functions compute the length of the respective packet
// sections note that all length of Binary type and UTF-8 type data are cast to
// uint16 to avoid breaking the packet if the length if above 65535. Instead
// the message is truncated to the overflown value, making it up to the user
// to keep the lengths within the boundaries.

func (c *Connect) computeConnectPropLen() uint64 {
	var length uint64
	if c.SessionExpiryInterval > 0 {
		// uint32
		length += 5
	}
	if c.ReceiveMax > 0 {
		// uint16
		length += 3
	}
	if c.MaxPacketSize > 0 {
		// uint32
		length += 5
	}
	if c.TopicAliasMax > 0 {
		// uint16
		length += 3
	}
	if c.RequestResponseInfo {
		// byte
		length += 2
	}
	if c.DisableProblemInfo {
		// byte
		length += 2
	}
	if len(c.ConnUserProperties) > 0 {
		for key, value := range c.ConnUserProperties {
			// UTF8-encoded key/value (+ property byte)
			length += uint64(uint16(len(key)) + 5)
			length += uint64(uint16(len(value)))
		}
	}
	if c.AuthMethod != "" {
		// UTF-8 string
		length += uint64(uint16(len(c.AuthMethod)) + 3)
	}
	if c.AuthData != nil {
		// Binary data
		length += uint64(uint16(len(c.AuthData)) + 3)
	}
	return length
}

func (c *Connect) computeWillPropLen() uint64 {
	var length uint64
	if c.WillDelayInterval > 0 {
		// uint32
		length += 5
	}
	if c.WillFormatUTF8 {
		// byte
		length += 2
	}
	if c.WillMessageExpiry > 0 {
		// uint32
		length += 5
	}
	if c.WillContentType != "" {
		// UTF-8 string
		length += uint64(uint16(len(c.WillContentType)) + 3)
	}
	if c.WillResponseTopic != "" {
		// UTF-8 string
		length += uint64(uint16(len(c.WillResponseTopic)) + 3)

		if c.WillCorrelationData != nil {
			// Binary data
			length += uint64(uint16(len(c.WillCorrelationData)) + 3)
		}
	}
	if len(c.WillUserProperties) > 0 {
		// UTF-8 key/value pairs
		for key, value := range c.WillUserProperties {
			length += uint64(uint16(len(key)) + 5)
			length += uint64(uint16(len(value)))
		}
	}
	return length
}

func (c *Connect) computeFlagsAndLen() (uint8, uint64) {
	// Initialize length to fixed variable header length:
	//     "MQTT" + version + Flags + KeepAlive
	var length uint64 = 10
	var flags uint8
	if c.CleanSession {
		flags |= ConnectFlagCleanSession
	}

	if c.Username != "" {
		length += uint64(uint16(len(c.Username))) + 2
		flags |= ConnectFlagUsername
		if c.Password != "" {
			length += uint64(uint16(len(c.Password))) + 2
			flags |= ConnectFlagPassword
		}
	} else if c.Version >= mqtt.MQTTv5 {
		if c.Password != "" {
			length += uint64(uint16(len(c.Password))) + 2
			flags |= ConnectFlagPassword
		}
	}
	if c.WillTopic.Name != "" {
		length += uint64(uint16(len(c.WillTopic.Name)))
		length += uint64(uint16(len(c.WillMessage))) + 4
		flags |= (uint8(c.WillTopic.QoS) << 3) | ConnectFlagWill
		if c.WillRetain {
			flags |= ConnectFlagWillRetain
		}
	}

	if len(c.ClientID) == 0 {
		id := uuid.NewV4()
		c.ClientID = id.String()
	}
	length += uint64(uint16(len(c.ClientID))) + 2
	return flags, length
}

func (c *Connect) MarshalBinary() (b []byte, err error) {
	var i int
	var flags uint8
	var remLen uint64
	var connPropLen uint64
	var willPropLen uint64
	if c.WillTopic.QoS > 2 {
		return nil, fmt.Errorf("illegal QoS value (highest: 2)")
	}
	// Compute packet length.
	flags, remLen = c.computeFlagsAndLen()
	if c.Version >= mqtt.MQTTv5 {
		connPropLen = c.computeConnectPropLen()
		willPropLen = c.computeWillPropLen()
		remLen += connPropLen + uint64(util.GetUvarintLen(connPropLen))
		remLen += willPropLen + uint64(util.GetUvarintLen(willPropLen))
	}
	remLenSize := util.GetUvarintLen(remLen)
	if remLenSize > 4 {
		return nil, fmt.Errorf("packet too large")
	}
	// Encode message to stream
	// Fixed header
	b = make([]byte, int(remLen)+int(remLenSize)+1)
	b[0] = cmdConnect
	i++
	i += binary.PutUvarint(b[i:], remLen)

	// Variable header
	i += copy(b[i:], []byte{0, 4, 'M', 'Q', 'T', 'T',
		uint8(c.Version), flags})
	binary.BigEndian.PutUint16(b[i:], c.KeepAlive)
	i += 2
	if c.Version >= mqtt.MQTTv5 {
		i += binary.PutUvarint(b[i:], connPropLen)
		if c.SessionExpiryInterval > 0 {
			b[i] = connPropSessionExpire
			i++
			i += util.EncodeValue(b[i:], c.SessionExpiryInterval)
		}
		if c.ReceiveMax > 0 {
			b[i] = connPropReceiveMax
			i++
			i += util.EncodeValue(b[i:], c.ReceiveMax)
		}
		if c.MaxPacketSize > 0 {
			b[i] = connPropMaxPacketSize
			i++
			i += util.EncodeValue(b[i:], c.MaxPacketSize)
		}
		if c.TopicAliasMax > 0 {
			b[i] = connPropTopicAliasMax
			i++
			i += util.EncodeValue(b[i:], c.TopicAliasMax)
		}
		if c.RequestResponseInfo {
			i += copy(b[i:], []byte{
				connPropRequestResponseInfo, 0x01,
			})
		}
		if c.DisableProblemInfo {
			i += copy(b[i:], []byte{
				connPropDisableProblemInfo, 0x00,
			})
		}
		if len(c.ConnUserProperties) > 0 {
			for key, value := range c.ConnUserProperties {
				// UTF8-encoded key/value (+ property byte)
				b[i] = connPropUserProperty
				i++
				i += util.EncodeValue(b[i:], key)
				i += util.EncodeValue(b[i:], value)
			}
		}
		if c.AuthMethod != "" {
			// UTF-8 string
			b[i] = connPropAuthMethod
			i += util.EncodeValue(b[i:], c.AuthMethod)
		}
		if c.AuthData != nil {
			// Binary data
			b[i] = connPropAuthData
			i++
			i += util.EncodeValue(b[i:], c.AuthData)
		}
	}

	// Payload
	i += util.EncodeValue(b[i:], c.ClientID)

	if c.Version >= mqtt.MQTTv5 {
		// Will properties
		if c.WillDelayInterval > 0 {
			b[i] = connPropWillDelay
			i++
			i += util.EncodeValue(b, c.WillDelayInterval)
		}
		if c.WillFormatUTF8 {
			i += copy(b[i:], []byte{connPropWillUTF8, 0x01})
		}
		if c.WillMessageExpiry > 0 {
			b[i] = connPropWillExpire
			i++
			i += util.EncodeValue(b[i:], c.WillMessageExpiry)
		}
		if c.WillContentType != "" {
			b[i] = connPropWillContentType
			i++
			i += util.EncodeValue(b[i:], c.WillContentType)
		}
		if c.WillResponseTopic != "" {
			b[i] = connPropWillResponseTopic
			i++
			i += util.EncodeValue(b[i:], c.WillResponseTopic)
			if c.WillCorrelationData != nil {
				b[i] = connPropWillCorrelationData
				i++
				i += util.EncodeValue(
					b[i:], c.WillCorrelationData,
				)
			}
		}
		if len(c.WillUserProperties) > 0 {
			for key, value := range c.WillUserProperties {
				b[i] = connPropWillUserProps
				i++
				i += util.EncodeValue(b[i:], key)
				i += util.EncodeValue(b[i:], value)
			}
		}

	}

	if c.WillTopic.Name != "" {
		i += util.EncodeValue(b[i:], c.WillTopic.Name)
		i += util.EncodeValue(b[i:], c.WillMessage)
	}

	if c.Username != "" {
		i += util.EncodeValue(b[i:], c.Username)
		if c.Password != "" {
			util.EncodeValue(b[i:], c.Password)
		}
	} else if c.Version == mqtt.MQTTv5 {
		if c.Password != "" {
			util.EncodeValue(b[i:], c.Password)
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
	if length <= 12 {
		return n, mqtt.ErrPacketShort
	}

	// Read variable header
	N, err = r.Read(buf[:])
	n += int64(N)
	length -= N
	if err != nil {
		return n, err
	}
	// Parse variable header
	if !bytes.Equal(buf[2:6], []byte{'M', 'Q', 'T', 'T'}) {
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
		if flags&ConnectFlagWill == 0 {
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
	c.KeepAlive = binary.BigEndian.Uint16(buf[8:])

	// Payload
	c.ClientID, N, err = util.ReadUTF8(r)
	n += int64(len(c.ClientID) + 2)
	if err != nil {
		return n, err
	} else if length -= N; length < 0 {
		return n, mqtt.ErrPacketShort
	}
	if flags&ConnectFlagWill > 0 {
		c.WillTopic.QoS = mqtt.QoS(flags&ConnectMaskWillQoS) >> 3
		c.WillTopic.Name, N, err = util.ReadUTF8(r)
		n += int64(len(c.WillTopic.Name) + 2)
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
		c.WillMessage = make([]byte, l16)
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
	}
	if flags&ConnectFlagPassword > 0 {
		c.Password, N, err = util.ReadUTF8(r)
		n += int64(N)
		if err != nil {
			return n, err
		} else if length -= N; length < 0 {
			return n, mqtt.ErrPacketShort
		}
		if (flags&ConnectFlagUsername == 0) &&
			(c.Version <= mqtt.MQTTv311) {
			return n, fmt.Errorf(
				"protocol violation: password without username",
			)
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
	b, _ := c.MarshalBinary()
	N, err := w.Write(b)
	n = int64(N)
	return n, err
}

// ReadFrom reads and unmarshals the ConnAck request from stream.
// NOTE: it is assumed that the command byte is already consumed from the reader.
func (c *ConnAck) ReadFrom(r io.Reader) (n int64, err error) {
	var raw [3]byte
	N, err := r.Read(raw[:])
	n = int64(N)
	if err != nil {
		return n, err
	} else if raw[0] > byte(2) {
		return n, mqtt.ErrPacketLong
	} else if raw[0] < byte(2) {
		return n, mqtt.ErrPacketShort
	}
	flags := raw[1]
	if flags > ConnAckFlagSessionPresent {
		return n, fmt.Errorf("connack: illegal flags: %02X", flags)
	} else if flags&ConnAckFlagSessionPresent > 0 {
		c.SessionPresent = true
	}
	c.ReturnCode = raw[2]
	return n, nil
}

func (d *Disconnect) MarshalBinary() (b []byte, err error) {
	return []byte{cmdDisconnect, 0}, nil
}

// WriteTo writes the marshaled Disconnect request to stream.
func (d *Disconnect) WriteTo(w io.Writer) (n int64, err error) {
	b, _ := d.MarshalBinary()
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
