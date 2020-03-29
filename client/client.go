package client

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/packets"
	"github.com/satori/go.uuid"
)

const (
	maxKeepAlive = ^uint16(0)
)

var (
	ErrIllegalResponse  = fmt.Errorf("illegal response received from server")
	ErrInternalConflict = fmt.Errorf("received unexpected packet")
)

type Client struct {
	// ClientID is the identity communicated with the server on connect.
	ClientID string
	version  mqtt.Version

	pendingPacketIDs []uint16
	pendingPackets   map[uint16]mqtt.Packet
	packetIDCounter  uint32

	expiresAt time.Time

	transport net.Conn

	// errChan is an internal error channel detecting asyncronous fatal
	// errors.
	errChan chan error
	// subscribeChans that maps topic names to chan []byte
	subscribeChans topicChan

	// respChan is used to pass ConnAck and PingResp responses to the
	// caller goroutine.
	pingResp chan *packets.PingResp
	// ackChan is used to pass SubAck and UnsubAck responses to the caller
	// goroutine. The callee is responsible for setting up a channel
	// prior to sending the Subscribe/Unsubscribe packets.
	ackChan map[uint16]chan mqtt.Packet
	connAck chan *packets.ConnAck
}

// NewClient initialize a new MQTT client with the given configuration and
// connection. After initializing the client, the user MUST call Connect before
// using the rest of the client API. Upon calling Connect, the client takes
// complete ownership of the connection and any reads or writes to the
// connection will lead to the client throwing an error.
func NewClient(connection net.Conn, options ...*ClientOptions) (client *Client) {
	var r [2]byte
	id, _ := uuid.NewV4()
	client = &Client{
		ClientID: id.String(),
		version:  mqtt.MQTTv311,

		transport: connection,

		pendingPackets: make(map[uint16]mqtt.Packet),
		ackChan:        make(map[uint16]chan mqtt.Packet),
		errChan:        make(chan error, 1),
		pingResp:       make(chan *packets.PingResp, 1),
		connAck:        make(chan *packets.ConnAck, 1),
	}
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if opt.Version != nil {
			client.version = *opt.Version
		}
		if opt.ClientID != nil {
			client.ClientID = *opt.ClientID
		}
	}
	rand.Read(r[:])
	defer func() {
		// In case binary package panics (should never occur)
		if recover() != nil {
			client.packetIDCounter = 0
		}
	}()
	initID := binary.LittleEndian.Uint16(r[:])
	client.packetIDCounter = uint32(initID)
	go client.recvRoutine()
	return client
}

// Connect establishes connection to the mqtt broker.
func (c *Client) Connect(options ...*ConnectOptions) error {
	conn := &packets.Connect{
		Version:  c.version,
		ClientID: c.ClientID,
	}
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if opt.KeepAlive != nil {
			conn.KeepAlive = *opt.KeepAlive
		}
		if opt.CleanSession != nil {
			conn.CleanSession = *opt.CleanSession
		}
		if opt.Username != nil {
			conn.Username = *opt.Username
		}
		if opt.Password != nil {
			conn.Password = *opt.Password
		}
	}

	if conn.KeepAlive > 0 {
		c.expiresAt = time.Now().
			Add(time.Second * time.Duration(conn.KeepAlive))
	}
	_, err := packets.Send(c.transport, conn)
	if err != nil {
		return err
	}
	select {
	case connAck := <-c.connAck:
		switch connAck.ReturnCode {
		case packets.ConnAckAccepted:
			return nil
		case packets.ConnAckBadVersion:
			return mqtt.ErrConnectBadVersion
		case packets.ConnAckIDNotAllowed:
			return mqtt.ErrConnectIDNotAllowed
		case packets.ConnAckServerUnavail:
			return mqtt.ErrConnectUnavailable
		case packets.ConnAckBadCredentials:
			return mqtt.ErrConnectCredentials
		case packets.ConnAckUnauthorized:
			return mqtt.ErrConnectUnauthorized
		default:
			return ErrIllegalResponse
		}
	case err := <-c.errChan:
		return err
	}
}

// Disconnect sends a disconnect packet to the server and closes the connection.
func (c *Client) Disconnect() (err error) {
	dc := &packets.Disconnect{
		Version: c.version,
	}
	defer func() {
		errClose := c.transport.Close()
		if err == nil {
			err = errClose
		}
	}()
	_, err = packets.Send(c.transport, dc)
	return
}

// Ping sends a ping packet to the server and blocks for a response.
func (c *Client) Ping() error {
	p := &packets.PingReq{
		Version: c.version,
	}
	_, err := packets.Send(c.transport, p)
	if err != nil {
		return err
	}
	select {
	case <-c.pingResp:
	case err := <-c.errChan:
		select {
		case c.errChan <- err:
		default:
		}
		return err
	}
	return nil
}

// Publish publishes a new packet to the specified topic.
func (c *Client) Publish(topic mqtt.Topic, payload []byte) error {
	// Reserve packet identifier
	packetID := c.aquirePacketID()
	pub := &packets.Publish{
		Version: c.version,

		Topic:   topic,
		Payload: payload,
	}

	switch topic.QoS {
	case mqtt.QoS0:
		// Nothing to do here.
	case mqtt.QoS2:
		c.ackChan[packetID] = make(chan mqtt.Packet, 1)
		defer func() { delete(c.ackChan, packetID) }()
		fallthrough
	case mqtt.QoS1:
		pub.PacketIdentifier = packetID
		c.pendingPackets[packetID] = pub
		c.pendingPacketIDs = append(c.pendingPacketIDs, packetID)
	default:
		return mqtt.ErrIllegalQoS
	}

	_, err := packets.Send(c.transport, pub)
	if err != nil && topic.QoS == mqtt.QoS2 {
		<-c.ackChan[packetID]
	}
	return err
}

// Subscribe sends a subscribe request with the given topics. On success
// the list of status codes corresponding to the provided topics are returned.
func (c *Client) Subscribe(
	topics []mqtt.Topic,
	topicChans []chan<- []byte,
) ([]uint8, error) {
	var statusCodes []uint8
	if len(topics) == 0 {
		return nil, nil
	} else if len(topics) != len(topicChans) {
		return nil, fmt.Errorf(
			"Invalid arguments: len(topics) != len(topicChans)")
	}

	// Reserve packet id
	packetID := c.aquirePacketID()
	// Setup ack channel
	ackChan := make(chan mqtt.Packet)
	c.ackChan[packetID] = ackChan
	defer func() { delete(c.ackChan, packetID) }()
	for i, topic := range topics {
		// Reserve receive channels
		c.subscribeChans.add(topic.Name, topicChans[i])
	}
	// Prepare and send packet.
	sub := &packets.Subscribe{
		Version:          c.version,
		PacketIdentifier: packetID,
		Topics:           topics,
	}
	_, err := packets.Send(c.transport, sub)
	if err != nil {
		return nil, err
	}
	select {
	case ack := <-ackChan:
		if subAck, ok := ack.(*packets.SubAck); ok {
			statusCodes = subAck.ReturnCodes
			// Remove subscribe channels with bad status code.
			for i, status := range statusCodes {
				if status > 2 {
					c.subscribeChans.remove(topics[i].Name)
				}
			}
		} else {
			return nil, ErrInternalConflict
		}

	case err := <-c.errChan:
		// Push error back in channel buffer and abort
		c.errChan <- err
		return nil, err
	}
	return statusCodes, nil
}

// Unsubscribe sends an unsubscribe packet to the topic names. The
// client will no longer receive packets on the given topics.
func (c *Client) Unsubscribe(topicNames ...string) error {
	if len(topicNames) == 0 {
		return nil
	}
	packetID := c.aquirePacketID()
	p := &packets.Unsubscribe{
		Version: c.version,

		Topics:           topicNames,
		PacketIdentifier: packetID,
	}
	c.ackChan[packetID] = make(chan mqtt.Packet, 1)
	_, err := packets.Send(c.transport, p)
	<-c.ackChan[packetID]
	delete(c.ackChan, packetID)
	return err
}