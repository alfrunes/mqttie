# MQTTie - a MQTT protocol library written in go
[![Actions Status](https://github.com/alfrunes/mqttie/workflows/Go/badge.svg)](https://github.com/alfrunes/mqttie/actions)
[![codecov](https://codecov.io/gh/alfrunes/mqttie/branch/master/graph/badge.svg)](https://codecov.io/gh/alfrunes/mqttie)

## Package structure:
├── client - The MQTT client package.  
├── mqtt - Common MQTT definitions.  
├── packets - Low-level implementation of packets and packet I/O.  
└── util - Utility library, e.g. functions for encoding mqtt-specific types.  

## High-level project goals
 * Create a low-level packet interface for the mqtt protocol (Partially complete)
 * Create a high-level client interface for the mqtt protocol (Started)
 * Support all versions of the protocol (currently 3.1.1)
 * Create a an mqtt broker (not started)
