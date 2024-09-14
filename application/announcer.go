package application

import (
	"fmt"
	"time"
)

// Announcer is an interface that
// defines the behavior of an announcer
type Announcer struct {
	FederatedTopic string
	stop           chan bool
}

// Drop stops the Announcer
// from sending core announcements
// to the federated network
// TODO: UNUSED
func (a Announcer) Drop() {
	a.stop <- true
	fmt.Println("Stop announcing as core")
}

// NewAnnouncer creates a new Announcer instance
// that sends core announcements to all neighbors
// in the federated network
func NewAnnouncer(federatedTopic string, ctx *FederatorContext) *Announcer {
	ann := CoreAnn{
		CoreId:   ctx.Id,
		Seqn:     0,
		Dist:     0,
		SenderId: ctx.Id,
	}

	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-stop:
				fmt.Println("Stop announcing as core goroutine")
				return
			default:
				time.Sleep(ctx.CoreAnnInterval)

				// Send core announcement to all neighbors
				for _, neighbor := range ctx.Neighbors {

					// Serialize the core announcement
					topic, coreAnn := ann.Serialize(federatedTopic)

					// Publish the core announcement
					// TODO: ENCRYPT USING CORE PUBLIC KEY ? (NOT IMPLEMENTED)
					fmt.Println("Sending core announcement to neighbor: ", neighbor.ClientIP, " On Topic: ", topic, " With CoreAnn: ", string(coreAnn))
					_, err := neighbor.Publish(topic, string(coreAnn), 2, true)
					if err != nil {
						fmt.Println("error while send coreAnn")
					}
				}

				ann.Seqn += 1
			}
		}
	}()

	fmt.Println(ctx.Id, "Start announcing as core")

	return &Announcer{
		FederatedTopic: federatedTopic,
		stop:           stop,
	}
}
