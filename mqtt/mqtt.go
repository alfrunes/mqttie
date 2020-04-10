package mqtt

import (
	"fmt"
)

// QoS type passes QoS information to publish and subscribe requests.
type QoS uint8

const (
	// QoS0 is the lowest quality of service: "At most once delivery".
	// A publish request with this level of service is not acknowledged
	// by the server and contains the least amount of overhead.
	QoS0 QoS = 0
	// QoS1 is the higher quality of service option guaranteeing: "At least
	// once delivery". A message published with this level is acknowledged
	// by the server when all subscribers have acknowledged the message.
	QoS1 QoS = 1
	// QoS2 is the highest level of QoS, guaranteeing: "Exactly once
	// delivery". The request is acknowledged in multiple steps: first
	// the server acknowledge the request is received, then the client
	// (issuer) must send a publish release, and finally when the server
	// has completed the publish request the client gets notified.
	QoS2 QoS = 2
)

// Version defines version level definitions.
type Version uint8

const (
	// MQTTv311 defines the MQTT protocol version 3.1.1
	MQTTv311 Version = 0x04
	// MQTTv5 defines the MQTT protocol version 5.0
	MQTTv5 Version = 0x05
)

var (
	// ErrConnectBadVersion is returned by client.Connect if the protocol
	// version is not supported by the server.
	ErrConnectBadVersion = fmt.Errorf("protocol not supported by server")
	// ErrConnectIDNotAllowed is returned by a connect request if the client
	// id is not allowed by the server.
	ErrConnectIDNotAllowed = fmt.Errorf("client id unacceptable by server")
	// ErrConnectUnavailable is returned by client.Connect if the server is
	// currently unavailable.
	ErrConnectUnavailable = fmt.Errorf("server unavailable")
	// ErrConnectCredentials is returned by client.Connect if the given
	// credentials were incorrect.
	ErrConnectCredentials = fmt.Errorf("incorrect username/password")
	// ErrConnectUnauthorized is returned by client.Connect if the client
	// is not authorized by the server.
	ErrConnectUnauthorized = fmt.Errorf("client unauthorized with server")
)

var (
	// ErrPacketShort is returned if a received packet is shorter than
	// given in the header.
	ErrPacketShort = fmt.Errorf("malformed packet: length too short")
	// ErrPacketLong is returned if a received packet is longer than
	// given in the header.
	ErrPacketLong = fmt.Errorf("malformed packet: length too long")

	// ErrIllegalQoS is returned if an invalid QoS value is passed to a
	// publish/subscribe request.
	ErrIllegalQoS = fmt.Errorf("invalid QoS value")
)

// Topic describes a topic name along with it's QoS value.
type Topic struct {
	// Name of the topic (e.g. foo/bar/+).
	Name string
	// QoS value for this topic.
	QoS QoS
}

// Subscription defines a topic with a channel to pass incoming messages for
// the topic.
type Subscription struct {
	// Topic for the subscription.
	Topic
	// Messages will receive incoming publish messages on the topic.
	Messages chan<- []byte
}
