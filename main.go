package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"mqtt-fed/application"
	keys "mqtt-fed/infra/crypto"
	"net/http"
	"os"
)

func main() {
	federatorConfig := getConfig()

	application.Run(federatorConfig)
	fmt.Println("Federator", federatorConfig.Id, "started!")

	select {}
}

func getConfig() application.FederatorConfig {
	var federatorConfig application.FederatorConfig

	if os.Getenv("TOPOLOGY_MANAGER_URL") != "" {

		privateKey, publicKey := keys.GetKeys()

		fmt.Println("Public Key: ", publicKey, " Private Key: ", privateKey)

		body, _ := json.Marshal(&application.JoinRequest{
			Ip:        os.Getenv("ADVERTISED_LISTENER"),
			PublicKey: string(publicKey),
		})
		payload := bytes.NewBuffer(body)

		fmt.Println("Joining the federated network with body: ", payload)
		resp, err := http.Post(os.Getenv("TOPOLOGY_MANAGER_URL")+"/api/v1/join", "application/json", payload)

		if err != nil {
			panic(err)
		}

		var response application.HTTPResponse

		err = json.NewDecoder(resp.Body).Decode(&response)

		if err != nil {
			panic(err)
		}

		if response.Code != 200 {
			panic(response.Description)
		}

		dataBytes, _ := json.Marshal(response.Data)

		err = json.Unmarshal(dataBytes, &federatorConfig)
		if err != nil {
			panic(err)
		}

		federatorConfig.CoreAnnInterval = time.Duration(federatorConfig.CoreAnnInterval)
		federatorConfig.BeaconInterval = time.Duration(federatorConfig.BeaconInterval)
		// TODO: Dont liek this, private Key is not part of the config
		federatorConfig.PrivateKey = string(privateKey)

		fmt.Println("Federator config: ", federatorConfig)
	} else {
		panic("No configuration provided")
	}

	return federatorConfig
}
