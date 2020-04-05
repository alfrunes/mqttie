package client

import (
	"time"

	"github.com/alfrunes/mqttie/mqtt"
)

// ClientOptions holds configuration options to initialize a new Client.
type ClientOptions struct {
	// Version signifies the protocol version to be used. Defaults to 3.1.1
	Version *mqtt.Version
	// The client identity communicated with the server. Defaults to random
	// UUID (version 4).
	ClientID *string
	// Timeout sets the duration for how long the client blocks on requests.
	Timeout *time.Duration
}

// NewClientOptions initializes a new empty client options struct.
func NewClientOptions() *ClientOptions {
	return new(ClientOptions)
}

// SetVersion sets the protocol version used by this client.
func (opts *ClientOptions) SetVersion(version mqtt.Version) {
	opts.Version = &version
}

// SetClientID sets the client id communicated with the server.
func (opts *ClientOptions) SetClientID(id string) {
	opts.ClientID = &id
}

// SetTimeout sets the timeout duration for blocking on send and receive to
// the connection. If unset, the client blocks indefinitely.
func (opts *ClientOptions) SetTimeout(timeout time.Duration) {
	opts.Timeout = &timeout
}

// ConnectOptions holds configuration options for making a connect request.
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

// NewConnectOptions initializes a new connect options struct.
func NewConnectOptions() *ConnectOptions {
	return &ConnectOptions{}
}

// SetCleanSession sets the clean-session flag.
func (opts *ConnectOptions) SetCleanSession(cleanSession bool) {
	opts.CleanSession = &cleanSession
}

// SetKeepAlive sets the keep alive to the given duration.
// NOTE: If the duration is longer than the maximum 18:12:15 (hr:min:sec),
// the value will be truncated to this maximum.
func (opts *ConnectOptions) SetKeepAlive(duration time.Duration) {
	secs := int64(duration.Seconds())
	if secs > int64(^uint16(0)) {
		secs = int64(^uint16(0))
	}
	secsUint16 := uint16(secs)
	opts.KeepAlive = &secsUint16
}

// SetUsername sets the username credential.
func (opts *ConnectOptions) SetUsername(username string) {
	opts.Username = &username
}

// SetPassword sets the password credential.
func (opts *ConnectOptions) SetPassword(password string) {
	opts.Password = &password
}

// SetWillTopic sets the will topic to publish on connect.
func (opts *ConnectOptions) SetWillTopic(topic mqtt.Topic, retain bool) {
	opts.WillTopic = &topic
	opts.WillRetain = &retain
}

// SetWillMessage sets the will message payload to the given buffer.
func (opts *ConnectOptions) SetWillMessage(message []byte) {
	opts.WillMessage = message
}

// PublishOptions contains configuration options for making a publish request.
type PublishOptions struct {
	// Retain determines whether the server should retain the application
	// message and it's QoS to be delivered to future subscribers.
	Retain *bool
}

// NewPublishOptions initializes a new blank publish options struct.
func NewPublishOptions() *PublishOptions {
	return &PublishOptions{}
}

// SetRetain sets the retain flag to the given value.
func (opts *PublishOptions) SetRetain(retain bool) {
	opts.Retain = &retain
}
