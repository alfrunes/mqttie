package client

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/packets"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
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
	respChan chan mqtt.Packet
	// ackChan is used to pass SubAck and UnsubAck responses to the caller
	// goroutine. The callee is responsible for setting up a channel
	// prior to sending the Subscribe/Unsubscribe packets.
	ackChan map[uint16]chan mqtt.Packet
}

// NewClient initialize a new MQTT client with the given configuration and
// connection. After initializing the client, the user MUST call Connect before
// using the rest of the client API. Upon calling Connect, the client takes
// complete ownership of the connection and any reads or writes to the
// connection will lead to the client throwing an error.
func NewClient(connection net.Conn, options ...ClientOptions) (client *Client) {
	var r [2]byte
	id, _ := uuid.NewV4()
	client = &Client{
		ClientID: id.String(),
		version:  mqtt.MQTTv311,

		transport: connection,

		pendingPackets: make(map[uint16]mqtt.Packet),
		ackChan:        make(map[uint16]chan mqtt.Packet),
		errChan:        make(chan error, 1),
		respChan:       make(chan mqtt.Packet, 2),
	}
	for _, opt := range options {
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
	return client
}

func (c *Client) aquirePacketID() uint16 {
	// Thread safe method to aquire unique packet ID.
	for i := 0; i < int(^uint16(0)); i++ {
		newVal := atomic.AddUint32(&c.packetIDCounter, 1)
		ret := uint16(newVal)
		if _, ok := c.pendingPackets[ret]; ok {
			continue
		} else if _, ok := c.ackChan[ret]; ok {
			continue
		} else {
			return ret
		}
	}
	panic("ran out of packet ids")
}

func (c *Client) recvRoutine() {
	for {
		packet, err := packets.Recv(c.transport)
		if err != nil && err != io.EOF {
			c.errChan <- err
			return
		}
		switch packet.(type) {
		case *packets.ConnAck, *packets.PingResp:
			// Bypass to response channel.
			var cached bool
			for !cached {
				select {
				case c.respChan <- packet:
					cached = true
				default:
					// Pop one response and count it as loss
					lost, ok := <-c.respChan
					if ok {
						pType := reflect.
							ValueOf(lost).Type()
						log.Errorf(
							"Packet lost: %s",
							pType.Name(),
						)
					}
				}
			}
		case *packets.SubAck, *packets.UnsubAck:
			// Use generic reflection of the (dereferenced) value
			pVal := reflect.ValueOf(packet).Elem()
			// Extract packet ID.
			id := pVal.FieldByName("PacketIdentifier")
			packetID := id.Interface().(uint16)
			// Verify that the channel is present
			if c, ok := c.ackChan[packetID]; ok {
				// Non-blocking send on channel
				//  - May receive multiple copies.
				select {
				case c <- packet:
				default:
				}
			} else {
				log.Errorf("Package lost: %s; packet id: %d",
					pVal.Type().Name(), packetID,
				)
			}

		case *packets.Publish:
			pub := packet.(*packets.Publish)

			subChan := c.subscribeChans.get(pub.Topic.Name)
			if subChan != nil {
				select {
				case subChan <- pub.Payload:

				default:
					log.Errorf("Subscriber channel %s is "+
						"full, discarding payload",
						pub.Topic.Name)
				}
			} else {
				log.Warnf("Internal error: no subscriber "+
					"chan for topic %s", pub.Topic.Name)
				continue
			}

			switch pub.QoS {
			case mqtt.QoS0:
				// We're done here

			case mqtt.QoS1:
				// Send puback and delete packet from pending.
				pubAck := &packets.PubAck{
					Version:          c.version,
					PacketIdentifier: pub.PacketIdentifier,
				}
				_, err := packets.Send(c.transport, pubAck)
				if err != nil {
					log.Error(err)
					c.errChan <- err
				}
				delete(c.pendingPackets, pub.PacketIdentifier)

			case mqtt.QoS2:
				// Send PubRec and update pending packet.
				pubRec := &packets.PubRec{
					Version:          c.version,
					PacketIdentifier: pub.PacketIdentifier,
				}
				_, err := packets.Send(c.transport, pubRec)
				if err != nil {
					log.Error(err)
					c.errChan <- err
					return
				}
				c.pendingPackets[pub.
					PacketIdentifier] = &packets.PubRel{
					Version:          c.version,
					PacketIdentifier: pub.PacketIdentifier,
				}
			}

		case *packets.PubAck:
			// Delete pending packet; publish completed
			pubAck := packet.(*packets.PubAck)
			delete(c.pendingPackets, pubAck.PacketIdentifier)

		case *packets.PubComp:
			// Delete pending packet; publish completed
			pubComp := packet.(*packets.PubComp)
			delete(c.pendingPackets, pubComp.PacketIdentifier)

		case *packets.PubRel:
			// Discard cached packet and send publish complete
			pub := packet.(*packets.PubRel)
			delete(c.pendingPackets, pub.PacketIdentifier)
			pubComp := &packets.PubComp{
				Version:          c.version,
				PacketIdentifier: pub.PacketIdentifier,
			}
			_, err := packets.Send(c.transport, pubComp)
			if err != nil {
				log.Error(err)
				c.errChan <- err
				return
			}

		case *packets.PubRec:
			pubRec := packet.(*packets.PubRec)
			delete(c.pendingPackets, pubRec.PacketIdentifier)
			pubRel := &packets.PubRel{
				Version:          c.version,
				PacketIdentifier: pubRec.PacketIdentifier,
			}
			_, err := packets.Send(c.transport, pubRel)
			if err != nil {
				log.Error(err)
				c.errChan <- err
				return
			}

		default:
			log.Error(ErrIllegalResponse)
			c.errChan <- ErrIllegalResponse
			return
		}
	}
}

// Connect establishes connection to the mqtt broker.
func (c *Client) Connect(options ...ConnectOptions) error {
	conn := &packets.Connect{
		Version:  c.version,
		ClientID: c.ClientID,
	}
	for _, opt := range options {
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
	// TODO handle unclean session

	packet, err := packets.Recv(c.transport)
	if err != nil {
		return err
	}
	if _, ok := packet.(*packets.ConnAck); !ok {
		return ErrIllegalResponse
	}
	go c.recvRoutine()
	return err
}

// Disconnect sends a disconnect packet to the server and closes the connection.
func (c *Client) Disconnect() error {
	dc := &packets.Disconnect{
		Version: c.version,
	}
	_, err := packets.Send(c.transport, dc)
	if err != nil {
		return err
	}
	return c.transport.Close()
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
	for i := 0; i < cap(c.respChan); i++ {
		r := <-c.respChan
		if _, ok := r.(*packets.PingResp); ok {
			return nil
		} else {
			c.respChan <- r
		}
	}
	return ErrIllegalResponse
}

// Publish publishes a new packet to the specified topic.
func (c *Client) Publish(topic mqtt.Topic, payload []byte) error {
	// Reserve packet identifier
	packetID := c.aquirePacketID()
	pub := &packets.Publish{
		Version: c.version,

		Topic:            topic,
		Payload:          payload,
		PacketIdentifier: packetID,
	}

	switch topic.QoS {
	case mqtt.QoS0:
		// Nothing to do here.
	case mqtt.QoS1:
		fallthrough
	case mqtt.QoS2:
		c.pendingPackets[packetID] = pub
		c.pendingPacketIDs = append(c.pendingPacketIDs, packetID)
	default:
		return mqtt.ErrIllegalQoS
	}

	_, err := packets.Send(c.transport, pub)
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
