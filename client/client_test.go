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

func TestPublish(t *testing.T) {
	testCases := []struct {
		Name string

		Version mqtt.Version

		ReadErr  error
		WriteErr error
		PubErr   error

		Topic   mqtt.Topic
		Payload []byte
		Retain  bool
	}{
		{
			Name: "Sucessful publish QoS0",

			Version: mqtt.MQTTv311,
			Topic: mqtt.Topic{
				Name: "foo/bar",
				QoS:  mqtt.QoS0,
			},
			Payload: []byte("foobar"),
		},
		{
			Name: "Sucessful publish QoS1",

			Version: mqtt.MQTTv311,
			Topic: mqtt.Topic{
				Name: "foo/bar",
				QoS:  mqtt.QoS1,
			},
			Payload: []byte("foobar"),
		},
		{
			Name: "Sucessful publish QoS2",

			Version: mqtt.MQTTv311,
			Topic: mqtt.Topic{
				Name: "foo/bar",
				QoS:  mqtt.QoS2,
			},
			Retain:  true,
			Payload: []byte("foobar"),
		},
		{
			Name: "Error illegal QoS",

			PubErr:  mqtt.ErrIllegalQoS,
			Version: mqtt.MQTTv311,
			Topic: mqtt.Topic{
				Name: "foo/bar",
				QoS:  mqtt.QoS(250),
			},
			Retain:  true,
			Payload: []byte("foobar"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			conn := NewFakeConn(2)
			defer conn.Close()
			client := NewClient(conn)
			pubOpts := NewPublishOptions()
			if testCase.Retain {
				pubOpts.SetRetain(true)
			} else {
				pubOpts = nil
			}
			conn.On("Close").Return(nil)

			switch testCase.Topic.QoS {
			case mqtt.QoS0:
				conn.On("Write", mock.Anything).
					Return(0, nil)
				if testCase.ReadErr != nil {
					conn.On("Read", mock.Anything).
						Return(0, testCase.ReadErr)
				} else {
					conn.On("Read", mock.Anything).
						Return(0, nil).
						Times(4)
				}

			case mqtt.QoS1:
				pubAck := &packets.PubAck{
					Version: testCase.Version,
					PacketIdentifier: uint16(
						client.packetIDCounter + 1),
				}
				b, _ := pubAck.MarshalBinary()
				conn.ReadChan <- b
				if testCase.ReadErr != nil {
					conn.On("Read", mock.Anything).
						Return(0, testCase.ReadErr)
				} else {
					conn.On("Read", mock.Anything).
						Return(0, nil).
						Times(4)
				}
				if testCase.WriteErr != nil {
					conn.On("Write", mock.Anything).
						Return(0, testCase.WriteErr)
				} else {
					conn.On("Write", mock.Anything).
						Return(0, nil).
						Twice()
				}

			case mqtt.QoS2:
				pubRec := &packets.PubRec{
					Version: testCase.Version,
					PacketIdentifier: uint16(
						client.packetIDCounter + 1),
				}
				b, _ := pubRec.MarshalBinary()
				conn.ReadChan <- b

				if testCase.ReadErr != nil {
					conn.On("Read", mock.Anything).
						Return(0, testCase.ReadErr)
				} else {
					conn.On("Read", mock.Anything).
						Return(0, nil).
						Times(6)
				}
				if testCase.WriteErr != nil {
					conn.On("Write", mock.Anything).
						Return(0, testCase.WriteErr)
				} else {
					conn.On("Write", mock.Anything).
						Return(0, nil).
						Times(3)
				}
			}
			err := client.Publish(
				testCase.Topic,
				testCase.Payload,
				pubOpts,
			)
			if testCase.PubErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err,
					testCase.PubErr.Error())
			}
		})
	}
}

func TestSubscribe(t *testing.T) {
	subChan := make(chan []byte, 5)
	testCases := []struct {
		Name string

		Topics      []mqtt.Subscription
		ReturnCodes []uint8
		Publishes   []packets.Publish
	}{
		{
			Name: "Sucessful subscription",

			Topics: []mqtt.Subscription{{
				Topic: mqtt.Topic{
					Name: "foo",
					QoS:  mqtt.QoS0,
				},
				Recv: subChan,
			}, {
				Topic: mqtt.Topic{
					Name: "foo/bar",
					QoS:  mqtt.QoS1,
				},
				Recv: subChan,
			}, {
				Topic: mqtt.Topic{
					Name: "foo/+/baz",
					QoS:  mqtt.QoS2,
				},
				Recv: subChan,
			}},
			ReturnCodes: []uint8{0, 1, 2},
		}, {
			Name: "Sucessful subscription w/incoming publish",

			Topics: []mqtt.Subscription{{
				Topic: mqtt.Topic{
					Name: "foo",
					QoS:  mqtt.QoS0,
				},
				Recv: subChan,
			}, {
				Topic: mqtt.Topic{
					Name: "foo/+",
					QoS:  mqtt.QoS1,
				},
				Recv: subChan,
			}, {
				Topic: mqtt.Topic{
					Name: "foo/+/baz",
					QoS:  mqtt.QoS2,
				},
				Recv: subChan,
			}, {
				Topic: mqtt.Topic{
					Name: "n/+",
					QoS:  mqtt.QoS0,
				},
				Recv: subChan,
			}},
			ReturnCodes: []uint8{0, 1, 2, 0x80},
			// NOTE: packet IDs will be assigned in test
			Publishes: []packets.Publish{{
				Topic: mqtt.Topic{
					Name: "foo",
					QoS:  mqtt.QoS0,
				},
				Payload: []byte("foo"),
			}, {
				Topic: mqtt.Topic{
					Name: "foo/bar",
					QoS:  mqtt.QoS1,
				},
				Payload: []byte("foobar"),
			}, {
				Topic: mqtt.Topic{
					Name: "foo/bar/baz",
					QoS:  mqtt.QoS2,
				},
				Payload: []byte("foobarbaz"),
			}, {
				Topic: mqtt.Topic{
					Name: "n/a",
					QoS:  mqtt.QoS0,
				},
				Payload: []byte("foobarbaz"),
			}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			conn := NewFakeConn(2)
			defer conn.Close()
			client := NewClient(conn)
			conn.On("Read", mock.Anything).
				Return(0, nil).
				Times(4)
			conn.On("Close").Return(nil)

			conn.On("Write", mock.Anything).
				Run(func(args mock.Arguments) {
					subAck := packets.SubAck{
						Version: mqtt.MQTTv311,
						PacketIdentifier: uint16(client.
							packetIDCounter),
						ReturnCodes: testCase.ReturnCodes,
					}
					b, _ := subAck.MarshalBinary()
					conn.ReadChan <- b
				}).Return(0, nil).Once()
			ret, err := client.Subscribe(testCase.Topics...)
			assert.NoError(t, err)
			assert.Equal(t, testCase.ReturnCodes, ret)

			finished := make(chan struct{}, 1)
			for _, pub := range testCase.Publishes {
				conn.On("Read", mock.Anything).
					Return(0, nil).Times(8)
				if pub.QoS > mqtt.QoS0 {
					conn.On("Write", mock.Anything).Run(func(
						args mock.Arguments,
					) {
						if pub.QoS != mqtt.QoS2 {
							finished <- struct{}{}
							return
						}
						pubRel := &packets.PubRel{
							Version: mqtt.MQTTv311,
							PacketIdentifier: pub.
								PacketIdentifier,
						}
						b, _ := pubRel.MarshalBinary()
						conn.ReadChan <- b
					}).Return(-1, nil).Once()
					if pub.QoS > mqtt.QoS1 {
						conn.On("Write", mock.Anything).Run(func(
							args mock.Arguments,
						) {
							finished <- struct{}{}
						}).
							Return(-1, nil).Once()
					}
				}
				pub.PacketIdentifier = uint16(client.
					packetIDCounter + 1)
				b, err := pub.MarshalBinary()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				conn.ReadChan <- b
				if c := client.subs.Get(
					pub.Topic.Name,
				); c != nil && cap(c) > 0 {
					<-subChan
				}
				if pub.QoS > mqtt.QoS0 {
					<-finished
				}
			}

			// Unsubscribe from "active" subscription
			for i, topic := range testCase.Topics {
				if testCase.ReturnCodes[i] > 2 {
					continue
				}
				conn.On("Write", mock.Anything).Run(func(
					args mock.Arguments,
				) {
					unsubAck := packets.UnsubAck{
						Version: mqtt.MQTTv311,
						PacketIdentifier: uint16(client.
							packetIDCounter),
					}
					b, _ := unsubAck.MarshalBinary()
					conn.ReadChan <- b
				}).Return(-1, nil)
				conn.On("Read", mock.Anything).
					Return(-1, nil).
					Times(8)
				err := client.Unsubscribe(topic.Name)
				assert.NoError(t, err)
			}

		})
	}
}
