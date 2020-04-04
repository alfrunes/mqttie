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
		if _, ok := c.pendingPackets.Get(ret); ok {
			continue
		} else if _, ok := c.ackChan.Get(ret); ok {
			continue
		} else {
			return ret
		}
	}
	panic("ran out of packet ids")
}

func (c *Client) recvRoutine() {
	for {
		packet, err := c.io.Recv()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Error(err)
			c.errChan <- err
			return
		}
		switch packet := packet.(type) {
		case *packets.PingResp:
			// Bypass to response channel.
			c.pingResp <- packet
		case *packets.ConnAck:
			c.connAck <- packet
		case *packets.SubAck, *packets.UnsubAck:
			// Use generic reflection of the (dereferenced) value
			pVal := reflect.ValueOf(packet).Elem()
			// Extract packet ID.
			id := pVal.FieldByName("PacketIdentifier")
			packetID := id.Interface().(uint16)
			// Verify that the channel is present
			if c, ok := c.ackChan.Get(packetID); ok {
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
			subChan := c.subs.Get(packet.Topic.Name)
			if subChan != nil {
				select {
				case subChan <- packet.Payload:

				default:
					log.Errorf("Subscriber channel %s is "+
						"full, discarding payload",
						packet.Topic.Name)
				}
			} else {
				log.Warnf("Internal error: no subscriber "+
					"chan for topic %s", packet.Topic.Name)
			}
			switch packet.QoS {
			case mqtt.QoS0:
				// We're done here

			case mqtt.QoS1:
				// Send puback and delete packet from pending.
				pubAck := &packets.PubAck{
					Version: c.version,
					PacketIdentifier: packet.
						PacketIdentifier,
				}
				err := c.io.Send(pubAck)
				if err != nil {
					log.Error(err)
					c.errChan <- err
				}
				c.pendingPackets.Del(packet.PacketIdentifier)

			case mqtt.QoS2:
				// Send PubRec and update pending packet.
				packetID := packet.PacketIdentifier
				pubRec := &packets.PubRec{
					Version:          c.version,
					PacketIdentifier: packetID,
				}
				err := c.io.Send(pubRec)
				if err != nil {
					log.Error(err)
					c.errChan <- err
					return
				}
				c.pendingPackets.Set(
					packet.PacketIdentifier,
					pubRec)
			}

		case *packets.PubAck:
			// Delete pending packet; publish completed
			c.pendingPackets.Del(packet.PacketIdentifier)

		case *packets.PubComp:
			// Delete pending packet; publish completed
			c.pendingPackets.Del(packet.PacketIdentifier)

		case *packets.PubRel:
			// Discard cached packet and send publish complete
			c.pendingPackets.Del(packet.PacketIdentifier)
			pubComp := &packets.PubComp{
				Version:          c.version,
				PacketIdentifier: packet.PacketIdentifier,
			}
			err := c.io.Send(pubComp)
			if err != nil {
				log.Error(err)
				c.errChan <- err
				return
			}

		case *packets.PubRec:
			// Update pending packets and send PubRel
			if ackChan, ok := c.ackChan.
				Get(packet.PacketIdentifier); ok {
				select {
				case ackChan <- packet:
				default:
					log.Warn("Packet discarded: PUBREC")
				}
			} else {
				log.Error("[internal] ACK chan not present")
			}
			pubRel := &packets.PubRel{
				Version:          c.version,
				PacketIdentifier: packet.PacketIdentifier,
			}
			c.pendingPackets.Set(packet.PacketIdentifier, pubRel)
			err := c.io.Send(pubRel)
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
