package client

import (
	"strings"
)

type topicChan struct {
	Chan  chan<- []byte
	Child map[string]*topicChan
}

func (t *topicChan) add(topic string, c chan []byte) {
	topicPtr := t
	scopes := strings.Split(topic, "/")
	for _, name := range scopes {
		if s, ok := topicPtr.Child[name]; ok {
			topicPtr = s
		} else {
			topicPtr.Child[name] = &topicChan{
				Child: make(map[string]*topicChan),
			}
		}
		topicPtr = topicPtr.Child[name]
	}
	topicPtr.Chan = c
}

func (t *topicChan) get(topic string) chan<- []byte {
	// TODO: wildcard match ALL matching topics
	topicPtr := t
	scopes := strings.Split(topic, "/")
	for i, scope := range scopes {
		if wildcard, ok := topicPtr.Child["*"]; ok {
			if i == len(scopes)-1 && wildcard.Chan != nil {
				return wildcard.Chan
			} else {
				return nil
			}
		}
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
