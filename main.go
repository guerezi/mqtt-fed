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

	"github.com/sandipmavani/hardwareid"
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

		privateKey, publicKey, err := keys.GenerateECDHKeyPair()

		if err != nil {
			panic(err)
		}

		id, err := hardwareid.ID()

		if err != nil {
			panic(err)
		}

		fmt.Println("Hardware ID: ", id)

		body, _ := json.Marshal(&application.JoinRequest{
			Ip:         os.Getenv("ADVERTISED_LISTENER"),
			PublicKey:  keys.ConvertECDSAPublicKeyToBytes(publicKey),
			HardwareId: id,
		})
		payload := bytes.NewBuffer(body)

		fmt.Println("Joining the federated network with body: ", payload)
		//TODO: A WHITELIST ESTÁ AQUI no topology
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

		fmt.Println("Data bytes: ", string(dataBytes))

		err = json.Unmarshal(dataBytes, &federatorConfig)
		if err != nil {
			panic(err)
		}

		federatorConfig.CoreAnnInterval = time.Duration(federatorConfig.CoreAnnInterval)
		federatorConfig.BeaconInterval = time.Duration(federatorConfig.BeaconInterval)
		federatorConfig.PrivateKey = privateKey
		federatorConfig.PublicKey = publicKey

		serverKey, _ := keys.ConvertBytesToECDSAPublicKey(privateKey, federatorConfig.ServerPublicKey)

		mySharedKey, _ := keys.GenerateSharedSecret(privateKey, serverKey)
		fmt.Println("Server SharedKey: ", string(federatorConfig.SharedKey))
		// TODO: DO NOT SEND THE SHARED KEY IN THE RESPONSE ANYMORE
		// ITS HERE JUST FOR DEBUGGING

		fmt.Println(string(federatorConfig.SharedKey))
		fmt.Println(string(mySharedKey))
		fmt.Println("Mathing KEYS:", keys.CheckKeys(federatorConfig.SharedKey, mySharedKey))

		fmt.Println("Id:", federatorConfig.Id)
		fmt.Println("Host:", federatorConfig.Host)
		fmt.Println("Neighbors:", federatorConfig.Neighbors)
		fmt.Println("Redundancy:", federatorConfig.Redundancy)
		fmt.Println("CoreAnnInterval:", federatorConfig.CoreAnnInterval)
		fmt.Println("BeaconInterval:", federatorConfig.BeaconInterval)
		fmt.Println("ServerPublicKey:", string(federatorConfig.ServerPublicKey))
		fmt.Println("SharedKey:", string(federatorConfig.SharedKey))
		fmt.Println("PrivateKey:", federatorConfig.PrivateKey)
		fmt.Println("PublicKey:", federatorConfig.PublicKey)

	} else {
		panic("No configuration provided")
	}

	return federatorConfig
}
