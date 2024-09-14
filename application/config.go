package application

import "time"

// JoinRequest is a struct that
// defines the request body for
// joining the federated network
type JoinRequest struct {
	Ip        string `json:"ip"`
	PublicKey string `json:"publickey"`
}

// NeighborConfig is a struct that
// defines the configuration of a neighbor
// in the federated network
type NeighborConfig struct {
	Id        int64  `json:"id" yaml:"id"`
	Ip        string `json:"ip" yaml:"ip"`
	PublicKey string `json:"publickey" yaml:"publickey"`
}

// FederatorConfig is a struct that
// defines the configuration of a federator
// in the federated network
type FederatorConfig struct {
	Id              int64            `json:"id" yaml:"id"`
	Host            string           `json:"ip" yaml:"host"`
	Neighbors       []NeighborConfig `json:"neighbors" yaml:"neighbors"`
	Redundancy      int              `json:"redundancy" yaml:"redundancy"`
	CoreAnnInterval time.Duration    `json:"coreAnnInterval" yaml:"core_ann_interval"`
	BeaconInterval  time.Duration    `json:"beaconInterval" yaml:"beacon_interval"`
	PrivateKey      string           `json:"privateKey" yaml:"privateKey"`
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
