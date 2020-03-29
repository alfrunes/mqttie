package client

import (
	log "github.com/sirupsen/logrus"
	"io"
	"reflect"
	"sync/atomic"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/packets"
)

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
		if err == io.EOF {
			return
		} else if err != nil {
			c.errChan <- err
			return
		}
		switch packet.(type) {
		case *packets.PingResp:
			// Bypass to response channel.
			c.pingResp <- packet.(*packets.PingResp)
		case *packets.ConnAck:
			c.connAck <- packet.(*packets.ConnAck)
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
			// Update pending packets and send PubRel
			pubRec := packet.(*packets.PubRec)
			if ackChan, ok := c.ackChan[pubRec.
				PacketIdentifier]; ok {
				select {
				case ackChan <- pubRec:
				default:
					log.Warn("Packet discarded: PUBREC")
				}
			}
			pubRel := &packets.PubRel{
				Version:          c.version,
				PacketIdentifier: pubRec.PacketIdentifier,
			}
			c.pendingPackets[pubRec.PacketIdentifier] = pubRel
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
