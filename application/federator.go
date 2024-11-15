package application

import (
	"crypto/ecdsa"
	"fmt"
	paho "mqtt-fed/infra/queue"
	"os"
	"strconv"
	"time"

	keys "mqtt-fed/infra/crypto"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// FederatorContext is a struct that
// defines the context of a federator
// in the federated network
type FederatorContext struct {
	Id              int64
	CoreAnnInterval time.Duration
	BeaconInterval  time.Duration
	Redundancy      int
	Neighbors       map[int64]*paho.Client
	HostClient      *paho.Client
	TopologyClient  *paho.Client
	CacheSize       int
	PrivateKey      *ecdsa.PrivateKey // my private key (can be stored as ecdsa.PrivateKey cuz will not be shared)
	PublicKey       []byte            // my public key
	SharedKey       []byte            // shared key with the topology manager
}

// Federator is a struct that
// defines the behavior of a federator
// in the federated network
// It is responsible for consuming messages
// from the federated network and dispatching
// them to the appropriate workers
type Federator struct {
	Ctx     *FederatorContext
	Workers map[string]*TopicWorkerHandle
}

// Run starts the federator
// and consumes messages from the
// federated network
func (f *Federator) Run() {
	topics := map[string]byte{
		TOPOLOGY_ANN_LEVEL:      2,
		CORE_ANNS:               2,
		MEMB_ANNS:               2,
		MEMB_ACK:                2,
		ROUTING_TOPICS:          2,
		SECURE_ROUTING_TOPICS:   2,
		FEDERATED_TOPICS:        2,
		SECURE_FEDERATED_TOPICS: 2,
		BEACONS:                 2,
		SECURE_BEACONS:          2,
		NODE_ANN_LEVEL + strconv.FormatInt(f.Ctx.Id, 10): 2,
	}

	// Message handler for consuming messages
	messageHandler := func(client mqtt.Client, mqttMsg mqtt.Message) {
		// Deserialize the message
		msg, err := f.Deserialize(mqttMsg)

		// Get the federated topic
		federatedTopic := msg.Topic

		if err == nil {
			// Check if the message is a topology announcement
			// and add or remove the neighbor from the neighbors
			if msg.Type == "TopologyAnn" {
				fmt.Println("Topology ann received: ", msg.TopologyAnn.Neighbor.Id, " Action: ", msg.TopologyAnn.Action)

				if msg.TopologyAnn.Action == "NEW" {
					mqttClient, err := paho.NewClient(msg.TopologyAnn.Neighbor.Ip, f.Ctx.HostClient.ClientID)

					if err == nil {
						f.Ctx.Neighbors[msg.TopologyAnn.Neighbor.Id] = mqttClient
					} else {
						fmt.Println("Erro on adding neighbor:", err)
					}

				} else if msg.TopologyAnn.Action == "REMOVE" {
					delete(f.Ctx.Neighbors, msg.TopologyAnn.Neighbor.Id)
				}
			} else {
				// Dispatch the message to the appropriate worker
				if worker, ok := f.Workers[federatedTopic]; ok {
					worker.Dispatch(*msg)
				} else {
					// Create a new worker for the federated topic
					worker := NewTopicWorkerHandle(federatedTopic, f.Ctx)
					worker.Dispatch(*msg)
					f.Workers[federatedTopic] = worker
				}
			}
		} else {
			fmt.Println("error on handle message: ", err)
		}
	}

	fmt.Println("Federator", f.Ctx.Id, "started!")

	// Consume messages from the federated network
	_, err := f.Ctx.HostClient.Consume(topics, messageHandler)

	if err != nil {
		panic(err)
	}
}

// Run starts the federator
// and consumes messages from the
// federated network
func Run(federatorConfig FederatorConfig) {
	// Create a client id
	clientId := "federator_" + strconv.FormatInt(federatorConfig.Id, 10)

	// Create neighbors clients (Usually starts empty and is updated by topology announcements)
	neighborsClients := createNeighborsClients(federatorConfig.Neighbors, clientId)
	// Create host client
	hostClient := createHostClient(clientId)
	// Create topology client
	topologyClient := createTopologyClient(clientId)

	// Create federator context
	ctx := FederatorContext{
		Id:              federatorConfig.Id,
		CoreAnnInterval: federatorConfig.CoreAnnInterval,
		BeaconInterval:  federatorConfig.BeaconInterval,
		Redundancy:      federatorConfig.Redundancy,
		CacheSize:       1000,
		Neighbors:       neighborsClients,
		HostClient:      hostClient,
		TopologyClient:  topologyClient,
		PrivateKey:      federatorConfig.PrivateKey,
		PublicKey:       keys.ConvertECDSAPublicKeyToBytes(federatorConfig.PublicKey),
		SharedKey:       federatorConfig.SharedKey,
	}

	// Create federator instance and then run it
	federator := Federator{
		Ctx:     &ctx,
		Workers: make(map[string]*TopicWorkerHandle),
	}

	federator.Run()
}

// createNeighborsClients creates a map of neighbors clients
// from the neighbors configuration
func createNeighborsClients(neighbors []NeighborConfig, clientId string) map[int64]*paho.Client {
	neighborsClients := make(map[int64]*paho.Client)

	for _, neighbor := range neighbors {
		mqttClient, err := paho.NewClient(neighbor.Ip, clientId)

		if err == nil {
			neighborsClients[neighbor.Id] = mqttClient
		} else {
			fmt.Println(err)
		}
	}

	fmt.Println("Neighbors clients created", neighborsClients)
	return neighborsClients
}

// createHostClient creates a host client for the federator
// it connects to the local mosquitto broker
func createHostClient(clientId string) *paho.Client {
	fmt.Println("Creating host client as ", clientId)
	mosquittoPort := os.Getenv("MOSQUITTO_PORT")

	if mosquittoPort == "" {
		mosquittoPort = "1883"
	}

	mqttClient, err := paho.NewClient("tcp://localhost:"+mosquittoPort, clientId)

	if err != nil {
		panic(err)
	}

	return mqttClient
}

// createTopologyClient creates a client for the federator
// that connects to the topology manager
func createTopologyClient(clientId string) *paho.Client {
	fmt.Println("Creating topology client as ", clientId)

	mqttClient, err := paho.NewClient("tcp://topology-manager:1883", clientId)

	if err != nil {
		fmt.Println("Error on creating topology client: ", err)
		panic(err)
	}

	return mqttClient
}
