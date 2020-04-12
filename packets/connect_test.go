package packets

import (
	"bytes"
	"testing"
	"time"

	"github.com/alfrunes/mqttie/mqtt"
	"github.com/stretchr/testify/assert"
)

func TestConnectSendRecv(t *testing.T) {
	type testCase struct {
		Name string
		*Connect
	}
	testCases := []testCase{
		{
			Name: "Simple v3.1.1",
			Connect: &Connect{
				Version: mqtt.MQTTv311,
			},
		}, {
			Name: "Simple v5.0",
			Connect: &Connect{
				Version: mqtt.MQTTv5,
			},
		}, {
			Name: "Advanced v3.1.1",
			Connect: &Connect{
				Version:      mqtt.MQTTv311,
				ClientID:     "foobar",
				CleanSession: true,
				KeepAlive:    123,
				Username:     "foo@bar.org",
				Password:     "foobarbaz",
				WillMessage:  []byte("Hello there!"),
				WillRetain:   true,
				WillTopic: mqtt.Topic{
					Name: "foo/bar",
					QoS:  mqtt.QoS1,
				},
			},
		}, {
			Name: "Advanced v5.0",
			Connect: &Connect{
				Version:      mqtt.MQTTv5,
				ClientID:     "bobTheBldr",
				CleanSession: true,
				KeepAlive:    12345,
				Username:     "bob@bldr.org",
				Password:     "bldmeapass",
				AuthMethod:   "Trusty auth",
				AuthData:     []byte("authorize me pls"),
				WillMessage:  []byte("Hi, I'm a bldr"),
				WillTopic: mqtt.Topic{
					Name: "bob/bld",
					QoS:  mqtt.QoS2,
				},
				RequestResponseInfo:   true,
				DisableProblemInfo:    true,
				SessionExpiryInterval: 123456,
				ReceiveMax:            10,
				MaxPacketSize:         4096,
				WillContentType:       "application/grbg",
				WillCorrelationData:   []byte("correlate this!"),
				WillDelayInterval:     1234567,
				WillFormatUTF8:        true,
				WillMessageExpiry:     0xFFFFFFFF,
				WillUserProperties: map[string]string{
					"key": "value",
				},
				WillResponseTopic: "rsp/here/pls",
				ConnUserProperties: map[string]string{
					"this":   "is",
					"mostly": "useless",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			conn := NewBufferConn(buf)
			bufIO := NewPacketIO(
				conn,
				testCase.Connect.Version,
				time.Minute,
			)
			err := bufIO.Send(testCase.Connect)
			if assert.NoError(t, err) {
				p, err := bufIO.Recv()
				assert.NoError(t, err)
				assert.Equal(t, testCase.Connect, p)
			}
		})
	}

}
