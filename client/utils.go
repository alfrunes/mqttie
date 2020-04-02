package client

import (
	"strings"

	"github.com/alfrunes/mqttie/mqtt"
)

type topicChan struct {
	Chan   chan<- []byte
	Child  map[string]*topicChan
	Parent interface{}
}

func (t *topicChan) add(topic string, c chan<- []byte) {
	topicPtr := t
	if topicPtr.Child == nil {
		t.Child = make(map[string]*topicChan)
		topicPtr = t
	}
	scopes := strings.Split(topic, "/")
	for _, name := range scopes {
		if s, ok := topicPtr.Child[name]; ok {
			topicPtr = s
		} else {
			topicPtr.Child[name] = &topicChan{
				Child:  make(map[string]*topicChan),
				Parent: topicPtr,
			}
		}
		if topicPtr.Child == nil {
			topicPtr.Child = make(map[string]*topicChan)
		}
		topicPtr = topicPtr.Child[name]
	}
	topicPtr.Chan = c
}

func (t *topicChan) get(topic string) chan<- []byte {
	// TODO: wildcard match ALL matching topics
	topicPtr := t
	if topicPtr == nil {
		return nil
	}
	scopes := strings.Split(topic, "/")
	for i, scope := range scopes {
		if wildcard, ok := topicPtr.Child["*"]; ok {
			if i == len(scopes)-1 && wildcard.Chan != nil {
				return wildcard.Chan
			} else {
				return nil
			}
		}
		// FIXME should try named branch if wildcard leads to dead end
		if wildcard, ok := topicPtr.Child["+"]; ok {
			topicPtr = wildcard
			continue
		}
		p, ok := topicPtr.Child[scope]
		if !ok {
			return nil
		}
		topicPtr = p
	}

	return topicPtr.Chan
}

func (t *topicChan) remove(topic string) {
	scopes := strings.Split(topic, "/")
	topicPtr := t

	for _, scope := range scopes {
		if child, ok := topicPtr.Child[scope]; ok {
			topicPtr = child
		}
	}
	topicPtr.Chan = nil
	// Clean up branch
	root := topicPtr
	for root.Parent != nil {
		parent := root.Parent.(*topicChan)
		if len(parent.Child) > 1 ||
			parent.Chan != nil {
			break
		}
		root = parent
	}
	root.Child = nil
}

type packetMap struct {
	packets map[uint16]mqtt.Packet
	mutex   chan struct{}
}

func newPacketMap() *packetMap {
	return &packetMap{
		packets: make(map[uint16]mqtt.Packet),
		mutex:   make(chan struct{}, 1),
	}
}

func (p *packetMap) Add(packetID uint16, packet mqtt.Packet) bool {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	if _, ok := p.packets[packetID]; ok {
		return false
	}
	p.packets[packetID] = packet
	return true
}

func (p *packetMap) Set(packetID uint16, packet mqtt.Packet) {
	p.mutex <- struct{}{}
	p.packets[packetID] = packet
	<-p.mutex
}

func (p *packetMap) Get(packetID uint16) (mqtt.Packet, bool) {
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
	chans map[uint16]chan mqtt.Packet
	mutex chan struct{}
}

func newPacketChanMap() *packetChanMap {
	return &packetChanMap{
		chans: make(map[uint16]chan mqtt.Packet),
		mutex: make(chan struct{}, 1),
	}
}

func (p *packetChanMap) New(packetID uint16) bool {
	p.mutex <- struct{}{}
	defer func() { <-p.mutex }()
	if _, ok := p.chans[packetID]; ok {
		return false
	}
	p.chans[packetID] = make(chan mqtt.Packet, 1)
	return true
}

func (p *packetChanMap) Get(packetID uint16) (chan mqtt.Packet, bool) {
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
