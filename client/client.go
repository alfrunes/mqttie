package client

import (
	"fmt"
	"net"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/packets"
	log "github.com/sirupsen/logrus"
)

const (
	maxKeepAlive = ^uint16(0)
)

var (
	ErrIllegalResponse = fmt.Errorf("illegal response received from server")
)

type ClientConfig struct {
	Version   mqtt.Version
	KeepAlive uint16

	Username string
	Password string
}

type Client struct {
	ClientConfig

	pendingPacketIDs []uint16
	pendingPackets   map[uint16]mqtt.Packet
	packetIDCounter  uint16

	expiresAt time.Time

	transport net.Conn

	// errChan is an internal error channel detecting asyncronous fatal
	// errors.
	errChan chan error
	// subscribeChans that maps topic names to chan []byte
	subscribeChans topicChan

	// Internally used channels to wait for server to client acknowledge
	// on connect, ping, subscribe and unsubscribe requests.
	// NOTE: These channels MUST be cleared by the main goroutine,
	//       if requests are made asyncronously; spawn a trivial goroutine
	//       that clears the buffer on response.
	connAck  chan *packets.ConnAck
	pingResp chan *packets.PingResp
	subAck   chan *packets.SubAck
	unsubAck chan *packets.UnsubAck
}

// NewClient initialize a new MQTT client with the given configuration and
// connection. After initializing the client, the user MUST call Connect before
// using the rest of the client API. Upon calling Connect, the client takes
// complete ownership of the connection and any reads or writes to the
// connection will lead to the client throwing an error.
func NewClient(config ClientConfig, connection net.Conn) *Client {
	client := &Client{
		ClientConfig: config,
		transport:    connection,
	}
	return client
}

func (c *Client) clientRoutine() {
	for {
		packet, err := packets.Recv(c.transport)
		if err != nil {
			c.errChan <- err
			return
		}
		switch packet.(type) {
		case *packets.ConnAck:
			c.connAck <- packet.(*packets.ConnAck)
		case *packets.PingResp:
			c.pingResp <- packet.(*packets.PingResp)
		case *packets.Publish:
			pub := packet.(*packets.Publish)

			subChan := c.subscribeChans.get(pub.TopicName)
			if subChan != nil {
				select {
				case subChan <- pub.Payload:

				default:
					log.Errorf("Subscriber channel %s is "+
						"full, discarding payload",
						pub.TopicName)
				}
			} else {
				log.Warnf("Internal error: no subscriber "+
					"chan for topic %s", pub.TopicName)
			}

			switch pub.QoSLevel {
			case mqtt.QoS0:
				// We're done here

			case mqtt.QoS1:
				// TODO send puback
				pubAck := &packets.PubAck{
					Version:          c.Version,
					PacketIdentifier: pub.PacketIdentifier,
				}
				_, err := packets.Send(c.transport, pubAck)
				if err != nil {
					log.Error(err)
					c.errChan <- err
				}
				delete(c.pendingPackets, pub.PacketIdentifier)

			case mqtt.QoS2:
				// TODO send pubrec
				pubRec := &packets.PubRec{
					Version:          c.Version,
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
					Version:          c.Version,
					PacketIdentifier: pub.PacketIdentifier,
				}
			}

		case *packets.PubAck:
			pubAck := packet.(*packets.PubAck)
			delete(c.pendingPackets, pubAck.PacketIdentifier)

		case *packets.PubComp:
			pubComp := packet.(*packets.PubComp)
			delete(c.pendingPackets, pubComp.PacketIdentifier)

		case *packets.PubRel:
			// Discard cached packet and send publish complete
			pub := packet.(*packets.PubRel)
			delete(c.pendingPackets, pub.PacketIdentifier)
			pubComp := &packets.PubComp{
				Version:          c.Version,
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
				Version:          c.Version,
				PacketIdentifier: pubRec.PacketIdentifier,
			}
			_, err := packets.Send(c.transport, pubRel)
			if err != nil {
				log.Error(err)
				c.errChan <- err
				return
			}

		case *packets.SubAck:
			c.subAck <- packet.(*packets.SubAck)

		case *packets.UnsubAck:
			c.unsubAck <- packet.(*packets.UnsubAck)

		default:
			log.Error(ErrIllegalResponse)
			c.errChan <- ErrIllegalResponse
			return
		}
	}
}

func (c *Client) Connect(cleanSession bool) error {
	conn := packets.NewConnectPacket(c.Version, c.KeepAlive)
	conn.KeepAlive = maxKeepAlive
	if c.KeepAlive > 0 {
		conn.KeepAlive = c.KeepAlive
	}
	conn.CleanSession = cleanSession
	conn.Username = c.Username
	conn.Password = c.Password
	c.expiresAt = time.Now().Add(time.Second * time.Duration(c.KeepAlive))
	_, err := conn.WriteTo(c.transport)
	if err != nil {
		return err
	}
	connAck := packets.ConnAck{
		Version: c.Version,
	}
	_, err = connAck.ReadFrom(c.transport)
	if err != nil {
		return err
	}
	//if c.KeepAlive == 0 {
	//	go c.LoopForever()
	//}
	return err
}
