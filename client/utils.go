package client

import (
	"strings"
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
