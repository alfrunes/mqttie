package client

import (
	"time"

	"github.com/alfrunes/mqttie/mqtt"
)

type ClientOptions struct {
	// Version signifies the protocol version to be used. Defaults to 3.1.1
	Version *mqtt.Version
	// The client identity communicated with the server. Defaults to random
	// UUID (version 4).
	ClientID *string
	// Timeout sets the duration for how long the client blocks on requests.
	Timeout *time.Duration
}

func NewClientOptions() *ClientOptions {
	return new(ClientOptions)
}

func (opts *ClientOptions) SetVersion(version mqtt.Version) {
	opts.Version = &version
}

func (opts *ClientOptions) SetClientID(id string) {
	opts.ClientID = &id
}

func (opts *ClientOptions) SetTimeout(timeout time.Duration) {
	opts.Timeout = &timeout
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

func NewConnectOptions() *ConnectOptions {
	return &ConnectOptions{}
}

func (opts *ConnectOptions) SetCleanSession(cleanSession bool) {
	opts.CleanSession = &cleanSession
}

func (opts *ConnectOptions) SetKeepAlive(duration time.Duration) {
	secs := int64(duration.Seconds())
	if secs > int64(^uint16(0)) {
		secs = int64(^uint16(0))
	}
	secsUint16 := uint16(secs)
	opts.KeepAlive = &secsUint16
}

func (opts *ConnectOptions) SetUsername(username string) {
	opts.Username = &username
}

func (opts *ConnectOptions) SetPassword(password string) {
	opts.Password = &password
}

func (opts *ConnectOptions) SetWillTopic(topic mqtt.Topic, retain bool) {
	opts.WillTopic = &topic
	opts.WillRetain = &retain
}

func (opts *ConnectOptions) SetWillMessage(message []byte) {
	opts.WillMessage = message
}

type PublishOptions struct {
	Retain *bool
}

func NewPublishOptions() *PublishOptions {
	return &PublishOptions{}
}

func (opts *PublishOptions) SetRetain(retain bool) {
	opts.Retain = &retain
}
