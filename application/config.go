package application

import (
	"crypto/ecdsa"
	"time"
)

// JoinRequest is a struct that
// defines the request body for
// joining the federated network
type JoinRequest struct {
	Ip         string `json:"ip"`
	PublicKey  []byte `json:"publicKey"`
	HardwareId string `json:"hardwareId"`
}

// NeighborConfig is a struct that
// defines the configuration of a neighbor
// in the federated network
type NeighborConfig struct {
	Id int64  `json:"id"`
	Ip string `json:"ip"`
	// SharedKey string `json:"sharedKey"`
}

// FederatorConfig is a struct that
// defines the configuration of a federator
// in the federated network
type FederatorConfig struct {
	Id              int64             `json:"id"`
	Host            string            `json:"ip"`
	Neighbors       []NeighborConfig  `json:"neighbors"`
	Redundancy      int               `json:"redundancy"`
	CoreAnnInterval time.Duration     `json:"coreAnnInterval"`
	BeaconInterval  time.Duration     `json:"beaconInterval"`
	ServerPublicKey []byte            `json:"publicKey"` // Public key of the topology manager
	SharedKey       []byte            `json:"sharedKey"` // Shared key with the topology manager
	PrivateKey      *ecdsa.PrivateKey // My private Key
	PublicKey       *ecdsa.PublicKey  // My public Key
}

// HTTPResponse is a struct that
// defines the response body for
// HTTP requests
type HTTPResponse struct {
	Status      string      `json:"status"`
	Code        int         `json:"code"`
	Data        interface{} `json:"data"`
	Description string      `json:"description"`
}
