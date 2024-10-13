package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	keys "mqtt-fed/infra/crypto"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const TOPOLOGY_ANN_LEVEL = "federator/topology_ann"

const BEACONS = "federator/beacon/#"
const SECURE_BEACONS = "federator/beacon/s/#"
const BEACON_TOPIC_LEVEL = "federator/beacon/"
const SECURE_BEACON_TOPIC_LEVEL = "federator/beacon/s/"

const CORE_ANNS = "federator/core_ann/#"
const CORE_ANN_TOPIC_LEVEL = "federator/core_ann/"

const MEMB_ANNS = "federator/memb_ann/#"
const MEMB_ANN_TOPIC_LEVEL = "federator/memb_ann/"

const MEMB_ACK = "federator/memb_ack/#"
const MEMB_ACK_TOPIC_LEVEL = "federator/memb_ack/"

const FEDERATED_TOPICS = "federated/#"
const SECURE_FEDERATED_TOPICS = "federated/s/#"
const FEDERATED_TOPICS_LEVEL = "federated/"
const SECURE_FEDERATED_TOPICS_LEVEL = "federated/s/"

const ROUTING_TOPICS = "federator/routing/#"
const SECURE_ROUTING_TOPICS = "federator/routing/s/#"
const ROUTING_TOPICS_LEVEL = "federator/routing/"
const SECURE_ROUTING_TOPICS_LEVEL = "federator/routing/s/"

type Message struct {
	Topic string
	Type  string
	TopologyAnn
	FederatedPub
	SecureFederatedPub
	RoutedPub
	SecureRoutedPub
	CoreAnn
	MeshMembAnn
	MeshMembAck
	Beacon
	SecureBeacon
}

type TopologyAnn struct {
	Neighbor NeighborConfig `json:"neighbor"`
	Action   string         `json:"action"`
}

type RoutedPub struct {
	PubId    PubId
	SenderId int64
	Payload  []byte
}

type SecureRoutedPub struct {
	PubId    PubId
	SenderId int64
	Payload  []byte
	Mac      []byte
}

type FederatedPub struct {
	Payload []byte
}

type SecureFederatedPub struct {
	Payload []byte
	Mac     []byte
}

// TODO, move public key to this struct?
type CoreAnn struct {
	CoreId   int64
	SenderId int64
	Seqn     int
	Dist     int
}

// TODO, move public key to this struct?
type MeshMembAnn struct {
	CoreId    int64
	SenderId  int64
	Seqn      int
	PublicKey []byte // My public key to be used by the sender to generate the shared key
}

type MeshMembAck struct {
	CoreId     int64
	SenderId   int64
	Seqn       int
	PublicKey  []byte // The public key of the sender to be used by the receiver to generate the shared key
	SessionKey []byte // The shared key just for debugging
}

type PubId struct {
	OriginId int64
	Seqn     int
}

type Beacon struct {
	Payload []byte
}

type SecureBeacon struct {
	Payload []byte
}

// Deserialize deserializes a message from an MQTT message
// mqttMessage: the MQTT message
// returns the deserialized message and an error
func (f *Federator) Deserialize(mqttMessage mqtt.Message) (*Message, error) {
	topic := mqttMessage.Topic()

	message := Message{}

	var err error
	fmt.Print("Received message: ", topic)

	// Check the topic and unmarshal the payload accordingly
	// the error is returned if the topic is not recognized
	if strings.HasPrefix(topic, TOPOLOGY_ANN_LEVEL) {
		message.Type = "TopologyAnn"

		fmt.Println("Decoding TopologyAnn with key: ", string(f.Ctx.SharedKey))
		payload, _ := keys.Decrypt(mqttMessage.Payload(), f.Ctx.SharedKey)
		err = json.Unmarshal(payload, &message.TopologyAnn)

		fmt.Println("->", message.Type, "Payload:", message.TopologyAnn)
	} else if strings.HasPrefix(topic, SECURE_ROUTING_TOPICS_LEVEL) {
		message.Type = "SecureRoutedPub"
		message.Topic = strings.TrimPrefix(topic, SECURE_ROUTING_TOPICS_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.SecureRoutedPub)

		fmt.Println("->", message.Type, "Payload:", message.SecureRoutedPub)
	} else if strings.HasPrefix(topic, ROUTING_TOPICS_LEVEL) {
		message.Type = "RoutedPub"
		message.Topic = strings.TrimPrefix(topic, ROUTING_TOPICS_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.RoutedPub)

		fmt.Println("->", message.Type, "Payload:", message.RoutedPub)
	} else if strings.HasPrefix(topic, SECURE_FEDERATED_TOPICS_LEVEL) {
		message.Type = "SecureFederatedPub"
		message.Topic = strings.TrimPrefix(topic, SECURE_FEDERATED_TOPICS_LEVEL)
		message.SecureFederatedPub.Payload = mqttMessage.Payload()

		fmt.Println("->", message.Type, "Payload:", string(message.SecureFederatedPub.Payload))
	} else if strings.HasPrefix(topic, FEDERATED_TOPICS_LEVEL) {
		message.Type = "FederatedPub"
		message.Topic = strings.TrimPrefix(topic, FEDERATED_TOPICS_LEVEL)
		message.FederatedPub.Payload = mqttMessage.Payload()

		fmt.Println("->", message.Type, "Payload:", string(message.FederatedPub.Payload))
	} else if strings.HasPrefix(topic, CORE_ANN_TOPIC_LEVEL) {
		message.Type = "CoreAnn"
		message.Topic = strings.TrimPrefix(topic, CORE_ANN_TOPIC_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.CoreAnn)

		fmt.Println("->", message.Type, "Payload:", message.CoreAnn)
	} else if strings.HasPrefix(topic, MEMB_ACK_TOPIC_LEVEL) {
		message.Type = "MeshMembAck"
		message.Topic = strings.TrimPrefix(topic, MEMB_ACK_TOPIC_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.MeshMembAck)

		fmt.Println("->", message.Type, "Payload:", message.MeshMembAck)
	} else if strings.HasPrefix(topic, MEMB_ANN_TOPIC_LEVEL) {
		message.Type = "MeshMembAnn"
		message.Topic = strings.TrimPrefix(topic, MEMB_ANN_TOPIC_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.MeshMembAnn)

		fmt.Println("->", message.Type, "Payload:", message.MeshMembAnn)
	} else if strings.HasPrefix(topic, SECURE_BEACON_TOPIC_LEVEL) {
		message.Type = "SecureBeacon"
		message.Topic = strings.TrimPrefix(topic, SECURE_BEACON_TOPIC_LEVEL)
		message.Beacon.Payload = mqttMessage.Payload()

		fmt.Println("->", message.Type, "Payload:", string(message.SecureBeacon.Payload))
	} else if strings.HasPrefix(topic, BEACON_TOPIC_LEVEL) {
		message.Type = "Beacon"
		message.Topic = strings.TrimPrefix(topic, BEACON_TOPIC_LEVEL)
		message.Beacon.Payload = mqttMessage.Payload()

		fmt.Println("->", message.Type, "Payload:", string(message.Beacon.Payload))
	} else {
		err = errors.New("received a packet from a topic it was not supposed to be subscribed to")
	}

	if err != nil {
		return nil, err
	}

	return &message, nil
}

// Serialize serializes a message to an MQTT message for FederatedPub
// returns the topic and payload
func (f *FederatedPub) Serialize(fedTopic string) (string, []byte) {
	topic := CORE_ANN_TOPIC_LEVEL + fedTopic
	payload, _ := json.Marshal(&f)

	fmt.Println("Serialized FederatedPub: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for SecureFederatedPub
// returns the topic and payload
func (f *SecureFederatedPub) Serialize(fedTopic string) (string, []byte) {
	topic := CORE_ANN_TOPIC_LEVEL + fedTopic // TODO: DO I REALLY SEND AS A CORE ANNOUNCEMENT? WHY
	payload, _ := json.Marshal(&f)

	fmt.Println("Serialized SecureFederatedPub: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for RoutedPub
// returns the topic and payload
func (r *RoutedPub) Serialize(fedTopic string) (string, []byte) {
	topic := ROUTING_TOPICS_LEVEL + fedTopic
	payload, _ := json.Marshal(&r)

	fmt.Println("Serialized RoutedPub: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for SecureRoutedPub
// returns the topic and payload
func (r *SecureRoutedPub) Serialize(fedTopic string) (string, []byte) {
	topic := SECURE_ROUTING_TOPICS_LEVEL + fedTopic
	payload, _ := json.Marshal(&r)

	fmt.Println("Serialized SecureRoutedPub: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for CoreAnn
// returns the topic and payload
func (c *CoreAnn) Serialize(fedTopic string) (string, []byte) {
	topic := CORE_ANN_TOPIC_LEVEL + fedTopic
	payload, _ := json.Marshal(&c)

	fmt.Println("Serialized CoreAnn: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for MeshMembAnn
// returns the topic and payload
func (m *MeshMembAnn) Serialize(fedTopic string) (string, []byte) {
	topic := MEMB_ANN_TOPIC_LEVEL + fedTopic
	payload, _ := json.Marshal(&m)

	fmt.Println("Serialized MeshMembAnn: ", string(payload))
	return topic, payload
}

// Serialize serializes a message to an MQTT message for MeshMembAnn
// returns the topic and payload
func (m *MeshMembAck) Serialize(fedTopic string) (string, []byte) {
	topic := MEMB_ACK_TOPIC_LEVEL + fedTopic
	payload, _ := json.Marshal(&m)

	fmt.Println("Serialized MeshMembAck: ", string(payload))
	return topic, payload
}
