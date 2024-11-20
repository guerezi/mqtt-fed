#!/bin/bash

send_beacon_messages() {
    mqtt pub -t federator/beacon/s/generic -m "test message" -h localhost -p 1880
    mqtt pub -t federator/beacon/s/generic -m "test message" -h localhost -p 1881
    mqtt pub -t federator/beacon/s/generic -m "test message" -h localhost -p 1882
    mqtt pub -t federator/beacon/s/generic -m "test message" -h localhost -p 1883
    mqtt pub -t federator/beacon/s/generic -m "test message" -h localhost -p 1884

    mqtt pub -t federator/beacon/generictest -m "test message" -h localhost -p 1880
    mqtt pub -t federator/beacon/generictest -m "test message" -h localhost -p 1881
    mqtt pub -t federator/beacon/generictest -m "test message" -h localhost -p 1882
    mqtt pub -t federator/beacon/generictest -m "test message" -h localhost -p 1883
    mqtt pub -t federator/beacon/generictest -m "test message" -h localhost -p 1884

    mqtt pub -t federator/beacon/s/data -m "test message" -h localhost -p 1884
    mqtt pub -t federator/beacon/s/data -m "test message" -h localhost -p 1885
    mqtt pub -t federator/beacon/s/data -m "test message" -h localhost -p 1886
    mqtt pub -t federator/beacon/s/data -m "test message" -h localhost -p 1887
    mqtt pub -t federator/beacon/s/data -m "test message" -h localhost -p 1888
    
    mqtt pub -t federator/beacon/datatest -m "test message" -h localhost -p 1884
    mqtt pub -t federator/beacon/datatest -m "test message" -h localhost -p 1885
    mqtt pub -t federator/beacon/datatest -m "test message" -h localhost -p 1886
    mqtt pub -t federator/beacon/datatest -m "test message" -h localhost -p 1887
    mqtt pub -t federator/beacon/datatest -m "test message" -h localhost -p 1888

    echo "Sent beacon messages"
}

# Função para enviar mensagens repetidamente por 1.5 segundos
send_repeated_messages() {
    for ((i=1; i<=20; i++)); do
        mqtt pub -t federated/s/generic -m "test message" -h localhost -p 1884
        mqtt pub -t federated/s/data -m "test message" -h localhost -p 1884
        
        mqtt pub -t federated/generictest -m "test message" -h localhost -p 1884
        mqtt pub -t federated/datatest -m "test message" -h localhost -p 1884

        echo "Sent $i repeated messages"
    done
}

echo "Starting round"

send_beacon_messages
sleep 15
send_beacon_messages
sleep 15
send_beacon_messages
sleep 5
send_repeated_messages
echo "Ended round 1"

send_beacon_messages
sleep 10
send_repeated_messages
echo "Ended round 2"

send_beacon_messages
sleep 10
send_repeated_messages
echo "Ended round 3"

send_beacon_messages
sleep 10
send_repeated_messages
echo "Ended round 4"

send_beacon_messages
sleep 10
send_repeated_messages
echo "Ended round 5"
