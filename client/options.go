package client

import (
	"github.com/alfrunes/mqttie/mqtt"
)

type ClientOptions struct {
	// Version signifies the protocol version to be used. Defaults to 3.1.1
	Version *mqtt.Version
	// The client identity communicated with the server. Defaults to random
	// UUID (version 4).
	ClientID *string
}

type ConnectOptions struct {
	// CleanSession indicates whether the server should discard any
	// previously stored session-state for the client.
	CleanSession *bool
	// KeepAlive is the number of seconds the session is active, defaults
	// to 0 ("infinite").
	KeepAlive *uint16

	// Username MQTT credentials. (Defaults to none)
	Username *string
	// Password MQTT credentials. (Defaults to none)
	// NOTE: If Password is set Username MUST also be set.
	Password *string

	// WillTopic is the topic to publish when connection is established.
	// Defaults to none.
	WillTopic *mqtt.Topic
	// WillMessage is the payload to the will topic that will be published
	// on connect. Defaults to empty.
	WillMessage []byte
	// WillRetain determines whether the server should retain the packet
	// for inbound subscribers for the lifetime of the session.
	// NOTE: if the WillTopic QoS is QoS0 the server may discard the packet
	//       at any time.
	WillRetain *bool
}
