package queue

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	ClientID string
	client   mqtt.Client
}

// NewClient creates a new MQTT client
// broker: the MQTT broker URL
// clientID: the client ID
// returns a new MQTT client
func NewClient(broker string, clientID string) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &Client{
		ClientID: clientID,
		client:   client,
	}, nil
}

// Consume subscribes to a list of topics
// topics: a map of topics to subscribe to
// messageHandler: the message handler
// returns a boolean indicating if the subscription was successful and an error
func (c Client) Consume(topics map[string]byte, messageHandler mqtt.MessageHandler) (bool, error) {
	token := c.client.SubscribeMultiple(topics, messageHandler)
	token.Wait()

	if token.Error() != nil {
		return false, token.Error()
	}

	return true, nil
}

// Publish publishes a message to a topic
// topic: the topic to publish to
// message: the message to publish
// qos: the quality of service
// retained: whether the message should be retained
// returns a boolean indicating if the publication was successful and an error
func (c Client) Publish(topic string, message string, qos byte, retained bool) (bool, error) {
	token := c.client.Publish(topic, qos, retained, message)
	token.Wait()

	if token.Error() != nil {
		return false, token.Error()
	}

	return true, nil
}

// Disconnect disconnects the client
// the 10 ms timeout is hardcoded
func (c Client) Disconnect() {
	c.client.Disconnect(10)
}
