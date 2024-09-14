package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const TOPOLOGY_ANN_LEVEL = "federator/topology_ann"

const BEACONS = "federator/beacon/#"
const BEACON_TOPIC_LEVEL = "federator/beacon/"

const CORE_ANNS = "federator/core_ann/#"
const CORE_ANN_TOPIC_LEVEL = "federator/core_ann/"

const MEMB_ANNS = "federator/memb_ann/#"
const MEMB_ANN_TOPIC_LEVEL = "federator/memb_ann/"

const FEDERATED_TOPICS = "federated/#"
const FEDERATED_TOPICS_LEVEL = "federated/"

const ROUTING_TOPICS = "federator/routing/#"
const ROUTING_TOPICS_LEVEL = "federator/routing/"

type Message struct {
	Topic string
	Type  string
	TopologyAnn
	FederatedPub
	RoutedPub
	CoreAnn
	MeshMembAnn
	Beacon
}

type TopologyAnn struct {
	Neighbor NeighborConfig `json:"neighbor"`
	Action   string         `json:"action"`
}

type RoutedPub struct {
	PubId    PubId
	SenderId int64
	Payload  []byte
	// MAC
}

type FederatedPub struct {
	Payload []byte
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
	CoreId   int64
	SenderId int64
	Seqn     int
}

type PubId struct {
	OriginId int64
	Seqn     int
}

type Beacon struct {
	Payload []byte
}

// Deserialize deserializes a message from an MQTT message
// mqttMessage: the MQTT message
// returns the deserialized message and an error
func Deserialize(mqttMessage mqtt.Message) (*Message, error) {
	topic := mqttMessage.Topic()

	message := Message{}

	var err error
	fmt.Print("Received message: ", topic)

	// Check the topic and unmarshal the payload accordingly
	// the error is returned if the topic is not recognized
	if strings.HasPrefix(topic, TOPOLOGY_ANN_LEVEL) {
		message.Type = "TopologyAnn"
		err = json.Unmarshal(mqttMessage.Payload(), &message.TopologyAnn)

		fmt.Println("->", message.Type, "Payload:", message.TopologyAnn)
	} else if strings.HasPrefix(topic, ROUTING_TOPICS_LEVEL) {
		message.Type = "RoutedPub"
		message.Topic = strings.TrimPrefix(topic, ROUTING_TOPICS_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.RoutedPub)

		fmt.Println("->", message.Type, "Payload:", message.RoutedPub)
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
	} else if strings.HasPrefix(topic, MEMB_ANN_TOPIC_LEVEL) {
		message.Type = "MeshMembAnn"
		message.Topic = strings.TrimPrefix(topic, MEMB_ANN_TOPIC_LEVEL)
		err = json.Unmarshal(mqttMessage.Payload(), &message.MeshMembAnn)

		fmt.Println("->", message.Type, "Payload:", message.MeshMembAnn)
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

// Serialize serializes a message to an MQTT message for RoutedPub
// returns the topic and payload
func (r *RoutedPub) Serialize(fedTopic string) (string, []byte) {
	topic := ROUTING_TOPICS_LEVEL + fedTopic
	payload, _ := json.Marshal(&r)

	fmt.Println("Serialized RoutedPub: ", string(payload))
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
