package client

import (
	"strings"

	"github.com/alfrunes/mqttie/packets"
)

type subMap map[string]interface{}

func (s subMap) Add(topic string, c chan<- []byte) bool {
	i := strings.Index(topic, "+")
	if i < 0 {
		s[topic] = c
		return true
	}
	if m, ok := s[topic[:i+1]].(subMap); ok {
		return m.Add(topic[i+1:], c)
	}
	m := make(subMap)
	if m.Add(topic[i+1:], c) {
		s[topic[:i+1]] = m
		return true
	}
	return false
}

func (s subMap) Get(topic string) chan<- []byte {
	if c, ok := s[topic].(chan<- []byte); ok {
		return c
	}
	var i, j int
	for {
		// Check multi-level wildcard (highest precedence)
		if c, ok := s[topic[:i]+"#"].(chan<- []byte); ok {
			return c
		}
		if tmp, ok := s[topic[:i]+"+"].(subMap); ok {
			// Carve out and replace scope with wildcard
			// and recurse onward.
			j = strings.Index(topic[i:], "/")
			if c := tmp.Get(topic[i+j:]); c != nil {
				return c
			}
		}
		// Advance index
		j = strings.Index(topic[i:], "/")
		if j < 0 {
			break
		}
		i += j + 1
	}
	return nil
}

func (s subMap) Del(topic string) {
	i := strings.Index(topic, "+")
	if i == -1 {
		delete(s, topic)
		return
	}
	if m, ok := s[topic[:i+1]].(subMap); ok {
		if len(m) <= 1 {
			delete(s, topic[:i+1])
		} else {
			m.Del(topic[i+1:])
		}
	}
}

type packetMap struct {
	packets map[uint16]packets.Packet
	mutex   chan struct{}
}

func newPacketMap() *packetMap {
	return &packetMap{
		packets: make(map[uint16]packets.Packet),
		mutex:   make(chan struct{}, 1),
	}
}

func (p *packetMap) Add(packetID uint16, packet packets.Packet) bool {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	if _, ok := p.packets[packetID]; ok {
		return false
	}
	p.packets[packetID] = packet
	return true
}

func (p *packetMap) Set(packetID uint16, packet packets.Packet) {
	p.mutex <- struct{}{}
	p.packets[packetID] = packet
	<-p.mutex
}

func (p *packetMap) Get(packetID uint16) (packets.Packet, bool) {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	packet, ok := p.packets[packetID]
	return packet, ok
}

func (p *packetMap) Del(packetID uint16) {
	p.mutex <- struct{}{}
	delete(p.packets, packetID)
	<-p.mutex
}

type packetChanMap struct {
	chans map[uint16]chan packets.Packet
	mutex chan struct{}
}

func newPacketChanMap() *packetChanMap {
	return &packetChanMap{
		chans: make(map[uint16]chan packets.Packet),
		mutex: make(chan struct{}, 1),
	}
}

func (p *packetChanMap) New(packetID uint16) bool {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	if _, ok := p.chans[packetID]; ok {
		return false
	}
	p.chans[packetID] = make(chan packets.Packet, 1)
	return true
}

func (p *packetChanMap) Get(packetID uint16) (chan packets.Packet, bool) {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	c, ok := p.chans[packetID]
	return c, ok
}

func (p *packetChanMap) Del(packetID uint16) {
	p.mutex <- struct{}{}
	delete(p.chans, packetID)
	<-p.mutex
}
