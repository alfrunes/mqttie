package client

import (
	"testing"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/alfrunes/mqttie/packets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	True       = true
	False      = false
	TestTopics = []mqtt.Topic{
		mqtt.Topic{
			Name: "foo/qos0",
			QoS:  mqtt.QoS0,
		},
		mqtt.Topic{
			Name: "bar/qos1",
			QoS:  mqtt.QoS1,
		},
		mqtt.Topic{
			Name: "baz/qos2",
			QoS:  mqtt.QoS2,
		},
	}
	TestUint16 uint16 = 123
	TestString        = "test"
)

func TestConnect(t *testing.T) {
	clientOpts := NewClientOptions()
	clientOpts.SetClientID("tester")
	clientOpts.SetVersion(mqtt.MQTTv311)

	testCases := []struct {
		Name string

		// Connect options
		cleanSession bool
		keepAlive    time.Duration
		username     string
		password     string
		willTopic    *mqtt.Topic
		willMessage  []byte
		willRetain   bool

		readErr    error
		writeErr   error
		connectErr error
		response   *packets.ConnAck
	}{
		{
			Name: "Successful connect",

			keepAlive: time.Hour * 24,
			username:  "foo",
			password:  "bar",
			willTopic: &mqtt.Topic{
				Name: "foo",
				QoS:  mqtt.QoS0,
			},
			willMessage: []byte("hello there"),
			willRetain:  true,

			response: &packets.ConnAck{
				SessionPresent: true,
				ReturnCode:     packets.ConnAckAccepted,
				Version:        mqtt.MQTTv311,
			},
		},
		{
			Name: "Incorrect version",

			cleanSession: true,

			connectErr: mqtt.ErrConnectBadVersion,
			response: &packets.ConnAck{
				ReturnCode: packets.ConnAckBadVersion,
				Version:    mqtt.Version(250),
			},
		},
		{
			Name: "Bad client id",

			connectErr: mqtt.ErrConnectIDNotAllowed,
			response: &packets.ConnAck{
				ReturnCode: packets.ConnAckIDNotAllowed,
				Version:    mqtt.MQTTv311,
			},
		},
		{
			Name: "Server unavailable",

			connectErr: mqtt.ErrConnectUnavailable,
			response: &packets.ConnAck{
				ReturnCode: packets.ConnAckServerUnavail,
				Version:    mqtt.MQTTv311,
			},
		},
		{
			Name: "Bad credentials",

			username: "foo",
			password: "baz",

			connectErr: mqtt.ErrConnectCredentials,
			response: &packets.ConnAck{
				ReturnCode: packets.ConnAckBadCredentials,
				Version:    mqtt.MQTTv311,
			},
		},
		{
			Name: "Client unauthorized",

			connectErr: mqtt.ErrConnectUnauthorized,
			response: &packets.ConnAck{
				ReturnCode: packets.ConnAckUnauthorized,
				Version:    mqtt.MQTTv311,
			},
		},
		{
			Name: "Unknown response code",

			connectErr: ErrIllegalResponse,
			response: &packets.ConnAck{
				ReturnCode: 69,
				Version:    mqtt.MQTTv311,
			},
		},
		{
			Name: "I/O read error",

			readErr:    ErrInternalConflict,
			connectErr: ErrInternalConflict,
			response:   &packets.ConnAck{},
		},
		{
			Name: "I/O write error",

			writeErr:   ErrInternalConflict,
			connectErr: ErrInternalConflict,
			response:   &packets.ConnAck{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			conn := NewFakeConn(1)
			b, err := testCase.response.MarshalBinary()
			assert.NoError(t, err)
			conn.ReadChan <- b
			connectOpts := NewConnectOptions()
			if testCase.cleanSession {
				connectOpts.SetCleanSession(true)
			}
			if testCase.keepAlive > time.Duration(0) {
				connectOpts.SetKeepAlive(testCase.keepAlive)
			}
			if testCase.username != "" {
				connectOpts.SetUsername(testCase.username)
			}
			if testCase.password != "" {
				connectOpts.SetPassword(testCase.password)
			}
			if testCase.willTopic != nil {
				connectOpts.SetWillTopic(
					*testCase.willTopic,
					testCase.willRetain,
				)
			}
			if testCase.willMessage != nil {
				connectOpts.SetWillMessage(testCase.willMessage)
			}

			conn.On("Write", mock.Anything).
				Return(0, testCase.writeErr)
			conn.On("Read", mock.Anything).Times(10).
				Return(0, testCase.readErr)
			conn.On("Close").
				Return(nil)
			client := NewClient(conn, clientOpts)
			if !assert.NotNil(t, client) {
				t.FailNow()
			}
			err = client.Connect(connectOpts, nil)
			if testCase.connectErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCase.connectErr.Error())
			}
			conn.Close()
		})
	}
}

func TestDisconnect(t *testing.T) {
	testCases := []struct {
		Name string

		writeErr      error
		disconnectErr error
	}{
		{
			Name: "Successful disconnect",
		},
		{
			Name:          "I/O send error",
			writeErr:      ErrInternalConflict,
			disconnectErr: ErrInternalConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			conn := NewFakeConn(1)
			conn.On("Write", mock.Anything).
				Return(0, testCase.writeErr)
			conn.On("Close").
				Return(nil)
			client := NewClient(conn)
			if !assert.NotNil(t, client) {
				t.FailNow()
			}
			err := client.Disconnect()
			if testCase.disconnectErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCase.
					disconnectErr.Error())
			}
		})
	}
}

func TestPing(t *testing.T) {
	testCases := []struct {
		Name string

		writeErr error
		readErr  error
		pingErr  error
	}{
		{
			Name: "Successful disconnect",
		},
		{
			Name:     "I/O send error",
			writeErr: ErrInternalConflict,
			pingErr:  ErrInternalConflict,
		},
		{
			Name:    "I/O read error",
			readErr: ErrInternalConflict,
			pingErr: ErrInternalConflict,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			conn := NewFakeConn(1)
			pingResp := &packets.PingResp{
				Version: mqtt.MQTTv311,
			}
			b, err := pingResp.MarshalBinary()
			assert.NoError(t, err)
			conn.ReadChan <- b
			conn.On("Write", mock.Anything).
				Return(0, testCase.writeErr)
			conn.On("Read", mock.Anything).
				Return(0, testCase.readErr)
			conn.On("Close").
				Return(nil)
			client := NewClient(conn, nil)
			if !assert.NotNil(t, client) {
				t.FailNow()
			}
			err = client.Ping()
			if testCase.pingErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, testCase.
					pingErr.Error())
			}
			conn.Close()
		})
	}
}
