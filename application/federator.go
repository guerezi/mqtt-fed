package application

import (
	"fmt"
	paho "mqtt-fed/infra/queue"
	"os"
	"strconv"
	"time"

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
	CacheSize       int
	Neighbors       map[int64]*paho.Client
	HostClient      *paho.Client
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
		TOPOLOGY_ANN_LEVEL: 2,
		CORE_ANNS:          2,
		MEMB_ANNS:          2,
		ROUTING_TOPICS:     2,
		FEDERATED_TOPICS:   2,
		BEACONS:            2,
	}

	// Message handler for consuming messages
	var messageHandler mqtt.MessageHandler = func(client mqtt.Client, mqttMsg mqtt.Message) {
		// Deserialize the message
		msg, err := Deserialize(mqttMsg)

		// Get the federated topic
		federatedTopic := msg.Topic

		fmt.Print("Received message: ", msg, "\n")

		if err == nil {
			// Check if the message is a topology announcement
			// and add or remove the neighbor from the neighbors
			if msg.Type == "TopologyAnn" {
				// TODO: HERE, CRIAR UM ESTADO INTERMEDIARIO ANTES DE REALMNENTE ENTRAR, CRIAR UM HANDSHAKE
				if msg.TopologyAnn.Action == "NEW" {
					fmt.Println("Topology ann received, adding ", msg.TopologyAnn.Neighbor.Id, " to neighbors")
					mqttClient, err := paho.NewClient(msg.TopologyAnn.Neighbor.Ip, f.Ctx.HostClient.ClientID)

					if err == nil {
						f.Ctx.Neighbors[msg.TopologyAnn.Neighbor.Id] = mqttClient
					} else {
						fmt.Println(err)
					}

				} else if msg.TopologyAnn.Action == "REMOVE" {
					fmt.Println("Topology ann received, removing ", msg.TopologyAnn.Neighbor.Id, " from neighbors")
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
		}
	}

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

	// Create neighbors clients
	neighborsClients := createNeighborsClients(federatorConfig.Neighbors, clientId)
	// Create host client
	hostClient := createHostClient(clientId)

	// Create federator context
	ctx := FederatorContext{
		Id:              federatorConfig.Id,
		CoreAnnInterval: federatorConfig.CoreAnnInterval,
		BeaconInterval:  federatorConfig.BeaconInterval,
		Redundancy:      federatorConfig.Redundancy,
		CacheSize:       1000,
		Neighbors:       neighborsClients,
		HostClient:      hostClient,
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
