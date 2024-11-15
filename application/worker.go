package application

import (
	"fmt"
	keys "mqtt-fed/infra/crypto"
	paho "mqtt-fed/infra/queue"
	"reflect"
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// TopicWorkerHandle is a struct that
// defines the handle of a topic worker
// in the federated network
type TopicWorkerHandle struct {
	FederatedTopic string
	Channel        chan Message
}

// Dispatch sends a message to the
// topic worker
func (t TopicWorkerHandle) Dispatch(msg Message) {
	t.Channel <- msg
}

// NewTopicWorkerHandle creates a new TopicWorkerHandle instance
func NewTopicWorkerHandle(federatedTopic string, ctx *FederatorContext) *TopicWorkerHandle {
	fmt.Println("Creating a new topic worker handle for:", federatedTopic)

	// Create a new channel
	channel := make(chan Message)

	// Create a new topic worker
	worker := NewTopicWorker(federatedTopic, ctx, channel)
	go worker.Run()

	// Return the topic worker handle
	return &TopicWorkerHandle{
		FederatedTopic: federatedTopic,
		Channel:        channel,
	}
}

type CoreBroker struct {
	Id                   int64
	LatestSeqn           int
	Dist                 int
	LastHeard            time.Time
	Parents              []Parent
	HasUnansweredParents bool
}

type Core struct {
	Myself Announcer
	Other  CoreBroker
}

type Parent struct {
	Id          int64
	WasAnswered bool
}

type TopicWorker struct {
	Topic        string
	Ctx          *FederatorContext
	Channel      chan Message
	Cache        *lru.Cache
	NextId       int
	LatestBeacon time.Time
	CurrentCore  Core
	Children     map[int64]time.Time
	SessionKey   []byte
}

// Run starts the topic worker
// and consumes messages from the
// federated network
func (t TopicWorker) Run() {
	// Consume messages from the federated network
	for msg := range t.Channel {
		if msg.Type == "NodeAnn" {
			t.handleNodeAnn(msg.NodeAnn)
		} else if msg.Type == "SecureRoutedPub" {
			t.handleSecureRoutedPub(msg.SecureRoutedPub)
		} else if msg.Type == "RoutedPub" {
			t.handleRoutedPub(msg.RoutedPub)
		} else if msg.Type == "SecureFederatedPub" {
			t.handleSecureFederatedPub(msg.SecureFederatedPub)
		} else if msg.Type == "FederatedPub" {
			t.handleFederatedPub(msg.FederatedPub)
		} else if msg.Type == "CoreAnn" {
			t.handleCoreAnn(msg.CoreAnn)
		} else if msg.Type == "MeshMembAck" {
			t.handleMembAck(msg.MeshMembAck)
		} else if msg.Type == "MeshMembAnn" {
			t.handleMembAnn(msg.MeshMembAnn)
		} else if msg.Type == "SecureBeacon" {
			t.handleSecureBeacon(msg.SecureBeacon)
		} else if msg.Type == "Beacon" {
			t.handleBeacon()
		}
	}

}

// handleNodeAnn handles a node announcement message
// it updates the topic password if the action is UPDATE_PASSWORD
// Any other handler for node <> federator must be done here
func (t *TopicWorker) handleNodeAnn(nodeAnn NodeAnn) {
	fmt.Println("Node Ann ", t.Topic, " received: ", nodeAnn)

	if nodeAnn.Id == t.Ctx.Id {
		return
	}

	if nodeAnn.Action == "UPDATE_PASSWORD" {
		fmt.Println("Updating topic password", string(nodeAnn.Password), t.Topic)
		t.SessionKey = nodeAnn.Password
	}
}

// handleRoutedPub handles a routed publication
func (t *TopicWorker) handleRoutedPub(routedPub RoutedPub) {
	fmt.Println("Routed Pub ", t.Topic, " received: ", string(routedPub.Payload))

	// Check if the cache contains the publication ID
	if t.Cache.Contains(routedPub.PubId) {
		return
	}

	// Add the publication ID to the cache
	t.Cache.Add(routedPub.PubId, true)

	// Check if the topic worker has local subscribers
	// and send the publication to the local subscribers (sensors and stuff)
	if t.hasLocalSub() {
		fmt.Println("sending pub to local subs ", t.Topic)

		_, err := t.Ctx.HostClient.Publish(t.Topic, string(routedPub.Payload), 2, false)

		if err != nil {
			fmt.Println("Error while send to local subscribers ", err)
		}
	}

	senderId := routedPub.SenderId
	routedPub.SenderId = t.Ctx.Id

	topic, replieRoutedPub := routedPub.Serialize(t.Topic)

	// send to mesh parents
	var parents []int64
	for _, parent := range t.CurrentCore.Other.Parents {
		if senderId != parent.Id {
			parents = append(parents, parent.Id)
		}
	}

	fmt.Println("Routed Pub Sending to parents: ", parents)
	SendTo(topic, replieRoutedPub, parents, t.Ctx.Neighbors)

	// send to mesh children
	var children []int64
	for id, child := range t.Children {
		elapsed := time.Since(child)

		if id != senderId && elapsed < 3*t.Ctx.CoreAnnInterval {
			children = append(children, id)
		}
	}

	fmt.Println("Routed Pub Sending to children: ", children)
	SendTo(topic, replieRoutedPub, children, t.Ctx.Neighbors)
}

// handleSecureRoutedPub handles a secure routed publication
// it decrypts the payload and checks the MAC
// it creates a new publication ID and sends the publication
// to the mesh parents and children
func (t *TopicWorker) handleSecureRoutedPub(secureRoutedPub SecureRoutedPub) {
	fmt.Println("Secure Routed Pub ", t.Topic, " received: ", string(secureRoutedPub.Payload))

	// Check if the cache contains the publication ID
	if t.Cache.Contains(secureRoutedPub.PubId) {
		return
	}

	// Add the publication ID to the cache
	t.Cache.Add(secureRoutedPub.PubId, true)

	// Check if the topic worker has local subscribers
	// and send the publication to the local subscribers (sensors and stuff)
	if t.hasLocalSub() {
		// It only decrypts the payload if it has local subscribers
		payload, er := keys.DecryptSimple(secureRoutedPub.Payload, t.SessionKey)

		if er != nil {
			fmt.Println("Error while decrypting the payload", er)
		} else {
			fmt.Println("Payload decrypted successfully", string(payload))
		}

		var sessionKey [16]byte
		copy(sessionKey[:], t.SessionKey[:16])
		if !keys.ValidateMAC(sessionKey, payload, secureRoutedPub.Mac) {
			fmt.Println("Message was tampered")
			return
		}

		fmt.Println("sending pub to local subs ", t.Topic)

		_, err := t.Ctx.HostClient.Publish(t.Topic, string(payload), 2, false)

		if err != nil {
			fmt.Println("Error while send to local subscribers ", err)
		}
	}

	senderId := secureRoutedPub.SenderId
	secureRoutedPub.SenderId = t.Ctx.Id

	topic, replieRoutedPub := secureRoutedPub.Serialize(t.Topic)

	var parents, children []int64

	// send to mesh parents
	for _, parent := range t.CurrentCore.Other.Parents {
		if senderId != parent.Id {
			parents = append(parents, parent.Id)
		}
	}

	fmt.Println("Secure Routed Pub Sending to parents: ", parents)
	SendTo(topic, replieRoutedPub, parents, t.Ctx.Neighbors)

	// send to mesh children
	for id, child := range t.Children {
		elapsed := time.Since(child)

		if id != senderId && elapsed < 3*t.Ctx.CoreAnnInterval {
			children = append(children, id)
		}
	}

	fmt.Println("Secure Routed Pub Sending to children: ", children)
	SendTo(topic, replieRoutedPub, children, t.Ctx.Neighbors)

}

// handleFederatedPub handles a federated publication
// it creates a new publication ID and sends the publication
// to the mesh parents and children
func (t *TopicWorker) handleFederatedPub(msg FederatedPub) {
	fmt.Println("Federted Pub ", t.Topic, " received: ", string(msg.Payload))

	newId := PubId{
		OriginId: t.Ctx.Id,
		Seqn:     t.NextId,
	}

	t.NextId += 1

	// Check if the cache contains the publication ID
	if t.Cache.Contains(newId) {
		return
	}

	// Add the publication ID to the cache
	t.Cache.Add(newId, true)

	pub := RoutedPub{
		PubId:    newId,
		Payload:  msg.Payload,
		SenderId: t.Ctx.Id,
	}

	topic, routedPub := pub.Serialize(t.Topic)

	// send to mesh parents
	var parents []int64
	for _, parent := range t.CurrentCore.Other.Parents {
		parents = append(parents, parent.Id)
	}

	fmt.Println("Federted Pub Sending to parents: ", parents)
	SendTo(topic, routedPub, parents, t.Ctx.Neighbors)

	// send to mesh children
	var children []int64
	for id, child := range t.Children {
		elapsed := time.Since(child)

		if elapsed < 3*t.Ctx.CoreAnnInterval {
			children = append(children, id)
		}
	}

	fmt.Println("Federted Pub Sending to children: ", children)
	SendTo(topic, routedPub, children, t.Ctx.Neighbors)
}

// handleSecureFederatedPub handles a secure federated publication
// it encrypts the payload and generates a MAC for the payload
func (t *TopicWorker) handleSecureFederatedPub(msg SecureFederatedPub) {
	fmt.Println("Secure Federted Pub ", t.Topic, " received: ", msg.Payload)

	if t.SessionKey == nil {
		fmt.Println("No session key available")
		return
	}

	fmt.Println("Session key: ", string(t.SessionKey))

	payload, err := keys.EncryptSimple(msg.Payload, t.SessionKey)

	if err != nil {
		fmt.Println("Error while encrypting the payload", err)
	} else {
		fmt.Println("Payload encrypted successfully", payload)
	}

	var sessionKey [16]byte
	copy(sessionKey[:], t.SessionKey[:16])
	mac := keys.GenerateMAC(sessionKey, msg.Payload)

	newId := PubId{
		OriginId: t.Ctx.Id,
		Seqn:     t.NextId,
	}

	t.NextId += 1

	pub := SecureRoutedPub{
		PubId:    newId,
		Payload:  payload,
		SenderId: t.Ctx.Id,
		Mac:      mac,
	}

	topic, secureRoutedPub := pub.Serialize(t.Topic)
	fmt.Println("Secure Federted Pub after: ", string(secureRoutedPub))

	t.Cache.Add(newId, true)
	var parents, children []int64

	// send to mesh parents
	for _, parent := range t.CurrentCore.Other.Parents {
		parents = append(parents, parent.Id)
	}

	fmt.Println("Secure Federted Pub Sending to parents: ", parents)
	SendTo(topic, secureRoutedPub, parents, t.Ctx.Neighbors)

	// send to mesh children
	for id, child := range t.Children {
		elapsed := time.Since(child)

		if elapsed < 3*t.Ctx.CoreAnnInterval {
			children = append(children, id)
		}
	}

	fmt.Println("Federted Pub Sending to children: ", children)
	SendTo(topic, secureRoutedPub, children, t.Ctx.Neighbors)
}

// handleCoreAnn handles a core announcement
// it updates the core information and forwards
// the core announcement to the mesh neighbors
func (t *TopicWorker) handleCoreAnn(coreAnn CoreAnn) {
	// if the core ann is from the current core or from the sender, ignore it
	// because we are not interested in our own core anns or in core anns from
	// the core that we are receiving the core anns
	if coreAnn.CoreId == t.Ctx.Id || coreAnn.SenderId == t.Ctx.Id {
		return
	}

	fmt.Println("Core Ann ", t.Topic, " received: ", coreAnn)

	coreAnn.Dist += 1

	// filter the core information and get the valid core
	core := FilterValid(t.CurrentCore, t.Ctx.CoreAnnInterval)
	fmt.Println("Core: ", core)

	if core != nil {
		currentCoreId := t.Ctx.Id

		if c, ok := core.(CoreBroker); ok {
			currentCoreId = c.Id
		}

		if coreAnn.CoreId == currentCoreId {
			core := core.(CoreBroker)
			// received a core ann with a diferent distance to the core: because we
			// are keeping only parents with same distance, the current parents are no
			// longer valid, so we clean the parents list and add the neighbor from the
			// receiving core ann as unique parent for now
			if coreAnn.Seqn > core.LatestSeqn || coreAnn.Dist <= core.Dist {
				t.CurrentCore.Other.LatestSeqn = coreAnn.Seqn
				t.CurrentCore.Other.Dist = coreAnn.Dist
				t.CurrentCore.Other.LastHeard = time.Now()

				wasAnswered := false

				if hasLocalSub(t.LatestBeacon, t.Ctx) {
					answer(coreAnn, t.Topic, t.Ctx)
					wasAnswered = true
				}

				t.CurrentCore.Other.Parents = t.CurrentCore.Other.Parents[:0]
				fmt.Println("Adding parent ", coreAnn.SenderId, " to ", t.Ctx.Id)
				t.CurrentCore.Other.Parents = append(t.CurrentCore.Other.Parents, Parent{
					Id:          coreAnn.SenderId,
					WasAnswered: wasAnswered,
				})
				t.CurrentCore.Other.HasUnansweredParents = !wasAnswered

				t.forward(coreAnn)

				// neighbor is not already a parent: make it parent if the redundancy
				// permits or if it has a lower id
			} else if coreAnn.Seqn == core.LatestSeqn || coreAnn.Dist == core.Dist {

				var isParent bool

				// check if the neighbor is already a parent
				for _, p := range core.Parents {
					if p.Id == coreAnn.SenderId {
						isParent = true
					}
				}

				// if the neighbor is not a parent, add it as a parent if the redundancy
				if !isParent {
					// check if the redundancy permits
					if len(core.Parents) <= t.Ctx.Redundancy {
						// check if the neighbor has local subscribers
						if len(core.Parents) == t.Ctx.Redundancy {
							t.CurrentCore.Other.Parents = t.CurrentCore.Other.Parents[:len(t.CurrentCore.Other.Parents)-1]
						}

						wasAnswered := false

						// check if the neighbor has local subscribers
						if hasLocalSub(t.LatestBeacon, t.Ctx) {
							answer(coreAnn, t.Topic, t.Ctx)
							wasAnswered = true
						}

						// add the neighbor as a parent
						parent := Parent{
							Id:          coreAnn.SenderId,
							WasAnswered: wasAnswered,
						}

						fmt.Println("Adding parent ", parent.Id, " to ", t.Ctx.Id)

						t.CurrentCore.Other.Parents = append(t.CurrentCore.Other.Parents, parent)
						t.CurrentCore.Other.HasUnansweredParents = !wasAnswered
					}
				}
			}
			// received a core ann from a core with a higher id: depose the current core
		} else if coreAnn.CoreId < currentCoreId {
			fmt.Println(currentCoreId, " Core deposed", coreAnn.CoreId, " New core elected")
			fmt.Println("Children on : ", t.Children, "will be empty")

			if coreAnn.CoreId == t.Ctx.Id {
				newNodeAnn := NodeAnn{
					Id:     t.Ctx.Id,
					Topic:  t.Topic,
					Action: "UPDATE_CORE",
				}
				_, payload := newNodeAnn.Serialize(strconv.FormatInt(t.Ctx.Id, 10))

				t.sendToTopology(payload)
			}

			t.Children = make(map[int64]time.Time)

			wasAnswered := false

			// check if the new core has local subscribers
			if t.hasLocalSub() {
				answer(coreAnn, t.Topic, t.Ctx)
				wasAnswered = true
			}

			// create a new core with the new core as the elected core
			var parents []Parent
			parents = append(parents, Parent{
				Id:          coreAnn.SenderId,
				WasAnswered: wasAnswered,
			})

			// update the current core
			newCore := Core{
				Other: CoreBroker{
					Id:                   coreAnn.CoreId,
					Parents:              parents,
					LatestSeqn:           coreAnn.Seqn,
					LastHeard:            time.Now(),
					Dist:                 coreAnn.Dist,
					HasUnansweredParents: !wasAnswered,
				},
			}

			t.CurrentCore = newCore

			// forward the core ann
			t.forward(coreAnn)
		}
		// received a core ann from a core with a higher id: depose the current core
	} else {
		fmt.Println("Children: ", t.Children, "will be empty")
		t.Children = make(map[int64]time.Time)

		wasAnswered := false

		if t.hasLocalSub() {
			answer(coreAnn, t.Topic, t.Ctx)
			wasAnswered = true
		}

		var parents []Parent
		parents = append(parents, Parent{
			Id:          coreAnn.SenderId,
			WasAnswered: wasAnswered,
		})

		newCore := Core{
			Other: CoreBroker{
				Id:                   coreAnn.CoreId,
				Parents:              parents,
				LatestSeqn:           coreAnn.Seqn,
				LastHeard:            time.Now(),
				Dist:                 coreAnn.Dist,
				HasUnansweredParents: !wasAnswered,
			},
		}

		t.CurrentCore = newCore

		fmt.Println(coreAnn.CoreId, " new core elected", newCore)

		t.forward(coreAnn)

		// if the core ann is from the current core, update the core
		if coreAnn.CoreId == t.Ctx.Id {
			newNodeAnn := NodeAnn{
				Id:     t.Ctx.Id,
				Topic:  t.Topic,
				Action: "UPDATE_CORE",
			}
			_, payload := newNodeAnn.Serialize(strconv.FormatInt(t.Ctx.Id, 10))

			t.sendToTopology(payload)
		}
	}
}

// handleMembAnn handles a mesh membership announcement
func (t *TopicWorker) handleMembAnn(membAnn MeshMembAnn) {
	fmt.Println("Memb Ann ", t.Topic, " received: ", membAnn)

	// if the memb ann is from the current core or from the sender, ignore it
	if membAnn.CoreId == t.Ctx.Id || membAnn.SenderId == t.Ctx.Id {
		return
	}

	// if the memb ann seqn is the same as the latest seqn, answer the parents
	if membAnn.Seqn == t.CurrentCore.Other.LatestSeqn {
		fmt.Println("Adding child ", membAnn.SenderId, " to ", t.Ctx.Id)
		t.Children[membAnn.SenderId] = time.Now()
		answerParents(&t.CurrentCore.Other, t.Ctx, t.Topic)

		if t.Ctx.Neighbors[membAnn.SenderId] != nil {
			fmt.Println("Sending my memb ack using key ", t.SessionKey)

			pub := MeshMembAck{
				CoreId:     t.CurrentCore.Other.Id,
				Seqn:       t.CurrentCore.Other.LatestSeqn,
				SenderId:   t.Ctx.Id,
				SessionKey: t.SessionKey,
			}

			// serialize the mesh ack announcement
			topic, myMembAck := pub.Serialize(t.Topic)

			if t.Ctx.Neighbors[membAnn.SenderId] != nil {
				fmt.Println("Sending my memb ack CHILD to ", membAnn.SenderId, " On topic ", topic)
				_, err := t.Ctx.Neighbors[membAnn.SenderId].Publish(topic, string(myMembAck), 2, false)

				if err != nil {
					fmt.Println("error while send my memb ack")
				}
			}
		}
	}
}

// handleMembAck handles a mesh membership acknowledgment
// it checks if the shared key matches and updates the shared key
func (t *TopicWorker) handleMembAck(membAck MeshMembAck) {
	if membAck.SenderId == t.Ctx.Id || membAck.CoreId == t.Ctx.Id {
		return
	}

	if reflect.DeepEqual(t.SessionKey, membAck.SessionKey) {
		fmt.Println("Shared key matched")
	} else {
		// t.SessionKey = membAck.SessionKey
		fmt.Println("Shared key did not match")
	}
}

// handleBeacon handles a beacon message and
// updates the latest beacon time,
//
// it is used to intent flag a worker as a member of the federated topic network
// You must recieve a beacon to be a member of the federated topic network
func (t *TopicWorker) handleBeacon() {
	t.LatestBeacon = time.Now()

	// check if the current core has local subscribers
	core := FilterValid(t.CurrentCore, t.Ctx.CoreAnnInterval)

	if core != nil {
		fmt.Println("Has Beancon for ", t.Topic)

		// if the current core is a core broker, answer the parents
		if c, ok := core.(CoreBroker); ok {
			answerParents(&c, t.Ctx, t.Topic)
		}
	} else {
		fmt.Println("Broker has no local subscribers, Creating an announcer")
		// if the current core is an announcer, create a new core
		announcer := NewAnnouncer(t.Topic, t.Ctx)

		t.CurrentCore = Core{
			Myself: *announcer,
		}

		fmt.Println("Children on beacon: ", t.Children, "will be empty")
		t.Children = make(map[int64]time.Time)
	}
}

// handleSecureBeacon handles a secure beacon message
// it is used to intent flag a worker as a member of the secure federated topic network
// You must recieve a beacon to be a member of the secure federated topic network
// It will send a message to the topology manager to update the core or join the network
func (t *TopicWorker) handleSecureBeacon(_ SecureBeacon) {
	fmt.Println("Secure Beacon ", t.Topic, " received")

	core := FilterValid(t.CurrentCore, t.Ctx.CoreAnnInterval)

	// Check if the cache contains the publication ID
	if t.Cache.Contains(t.Topic) {
		return
	}

	// Add the publication ID to the cache
	t.Cache.Add(t.Topic, true)

	time.AfterFunc(2*time.Second, func() {
		t.Cache.Remove(t.Topic)
	})

	// I will be the core, must create a session key
	if core == nil {
		newNodeAnn := NodeAnn{
			Id:     t.Ctx.Id,
			Topic:  t.Topic,
			Action: "UPDATE_CORE",
		}
		_, payload := newNodeAnn.Serialize(strconv.FormatInt(t.Ctx.Id, 10))

		t.sendToTopology(payload)
	} else if t.SessionKey == nil {
		// Im sending join every time I receive a secure beacon, this is not correct
		newNodeAnn := NodeAnn{
			Id:     t.Ctx.Id,
			Topic:  t.Topic,
			Action: "JOIN",
		}
		_, payload := newNodeAnn.Serialize(strconv.FormatInt(t.Ctx.Id, 10))

		t.sendToTopology(payload)
	}

	t.handleBeacon()
}

// hasLocalSub checks if the topic worker has local subscribers
// by checking if the latest beacon time is not zero and if the
// elapsed time is less than 3 times the beacon interval
// LocalSubscribers are subscribers that are in the same broker as the topic worker
func (t TopicWorker) hasLocalSub() bool {
	// check if the latest beacon time is not zero
	// and if the elapsed time is less than 3 times
	if !t.LatestBeacon.IsZero() {
		elapsed := time.Since(t.LatestBeacon)

		return elapsed < 3*t.Ctx.BeaconInterval
	} else {
		return false
	}
}

// forwards (publish) a core announcement to the mesh neighbors
func (t TopicWorker) forward(coreAnn CoreAnn) {
	pub := CoreAnn{
		Dist:     coreAnn.Dist + 1,
		SenderId: t.Ctx.Id,
		Seqn:     coreAnn.Seqn,
		CoreId:   coreAnn.CoreId,
	}

	topic, myCoreAnn := pub.Serialize(t.Topic)

	for id, ngbrClient := range t.Ctx.Neighbors {
		if id != coreAnn.SenderId {
			fmt.Println("Forwarding core ann to", id, "On topic", topic)
			_, err := ngbrClient.Publish(topic, string(myCoreAnn), 2, false)
			if err != nil {
				fmt.Println("Error while forward message to", ngbrClient.ClientID)
			}
		}
	}
}

// Sends a message to topology manager, the message is encrypted
// with the shared key of the federated topic.
// The message can be UPDATE_CORE, JOIN or LEAVE
func (t TopicWorker) sendToTopology(message []byte) {

	topic := NODE_ANN_LEVEL + strconv.FormatInt(t.Ctx.Id, 10)
	payload, err := keys.Encrypt(message, t.Ctx.SharedKey)

	if err != nil {
		fmt.Println("Error while encrypting the payload", err)
		return
	}

	fmt.Println("Sending to topology", topic)

	t.Ctx.TopologyClient.Publish(topic, string(payload), 2, false)
}

// hasLocalSub checks if the topic worker has local subscribers
func hasLocalSub(latestBeacon time.Time, ctx *FederatorContext) bool {
	if !latestBeacon.IsZero() {
		elapsed := time.Since(latestBeacon)

		return elapsed < 3*ctx.BeaconInterval
	} else {
		return false
	}
}

// NewTopicWorker creates a new topic worker instance with the given federated topic
// and federator context and channel to receive messages from the federated network
func NewTopicWorker(federatedTopic string, ctx *FederatorContext, channel chan Message) *TopicWorker {

	cache, _ := lru.New(ctx.CacheSize)
	return &TopicWorker{
		Topic:    federatedTopic,
		Ctx:      ctx,
		Channel:  channel,
		Cache:    cache,
		NextId:   0,
		Children: make(map[int64]time.Time),
	}
}

// FilterValid filters the core information and returns the valid core
func FilterValid(core Core, coreAnnInterval time.Duration) interface{} {
	// deepequal is used to compare the core information
	// if the core information is not empty, check if the
	// other core is not empty and if the elapsed time is
	// less than 3 times the core announcement interval
	if !reflect.DeepEqual(core.Other, CoreBroker{}) {
		elapsed := time.Since(core.Other.LastHeard)

		if elapsed < 3*coreAnnInterval {
			return core.Other
		}
	} else if !reflect.DeepEqual(core.Myself, Announcer{}) {
		return core.Myself
	}

	return nil
}

// SendTo sends a message to the mesh neighbors
func SendTo(topic string, message []byte, ids []int64, neighbors map[int64]*paho.Client) {
	if len(ids) <= 0 {
		return
	}

	// The first is the core broker
	firstId := ids[0]

	// remove the first id from the list because it will be sent separately latter
	ids = append(ids[:0], ids[0+1:]...)

	for _, id := range ids {

		if neighbors[id] != nil {
			fmt.Println("Sending:", topic, "With message:", string(message), "to ", id)

			_, err := neighbors[id].Publish(topic, string(message), 2, false)
			if err != nil {
				fmt.Println("problem creating or queuing the message for broker id ", id)
			}
		} else {
			fmt.Println("broker", id, "is not a neighbor")
		}
	}

	// send to the first id if it is a neighbor
	if neighbors[firstId] != nil {
		fmt.Println("Sending:", topic, "With message:", string(message), "to ", firstId)

		_, err := neighbors[firstId].Publish(topic, string(message), 2, false)

		if err != nil {
			fmt.Println("problem creating or queuing the message for broker id ", firstId)
		}
	} else {
		fmt.Println("broker", firstId, "is not a neighbor")
	}
}

// Answers the parents of a core broker with the latest sequence number
func answerParents(core *CoreBroker, context *FederatorContext, topic string) {
	fmt.Println("Answering parents of ", core.Id, " On topic ", topic)

	if core.HasUnansweredParents {
		pub := MeshMembAnn{
			CoreId:   core.Id,
			Seqn:     core.LatestSeqn,
			SenderId: context.Id,
		}

		// serialize the mesh membership announcement
		topic, myMembAnn := pub.Serialize(topic)

		for _, parent := range core.Parents {
			if !parent.WasAnswered {
				if context.Neighbors[parent.Id] != nil {
					fmt.Println("Sending my memb ann PARENTS to ", parent.Id, " On topic ", topic)
					_, err := context.Neighbors[parent.Id].Publish(topic, string(myMembAnn), 2, false)
					if err != nil {
						fmt.Println("error while send my memb ann")
					}
				}

				parent.WasAnswered = true
			}
		}

		// set the core broker has no unanswered parents after answering the parents
		core.HasUnansweredParents = false
	}
}

// Answers a core announcement with a mesh membership announcement
func answer(coreAnn CoreAnn, topic string, context *FederatorContext) {
	fmt.Println("Answering core ann from", coreAnn.SenderId, "as", context.Id, "On topic", topic)

	pub := MeshMembAnn{
		CoreId:   coreAnn.CoreId,
		Seqn:     coreAnn.Seqn,
		SenderId: context.Id,
	}

	// serialize the mesh membership announcement
	topic, myMembAnn := pub.Serialize(topic)

	// send the mesh membership announcement to the sender
	if context.Neighbors[coreAnn.SenderId] != nil {
		fmt.Println("Sending my memb ann to ", coreAnn.SenderId, " On topic ", topic)
		_, err := context.Neighbors[coreAnn.SenderId].Publish(topic, string(myMembAnn), 2, false)
		if err != nil {
			fmt.Println("error while send my memb ann to ", coreAnn.SenderId)
		}
	} else {
		fmt.Println(coreAnn.SenderId, " is not a neighbor")
	}
}
