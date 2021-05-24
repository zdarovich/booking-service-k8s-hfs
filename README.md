# booking-service-k8s-hfs
Booking service based on blockchain

### Techstack
- Hyperledger Fabric
- Couchdb
- Postgres
- Kubernetes
- Minikube

### Minikube configuration
Suggested minikube start configuration:
```
minikube start --memory=6144 --cpus=4
```

### Instructions to build the cluster

Run the commands in the following order:

``` bash
kubectl apply -f k8s/kafka/
```

### Configure kafka-manager

Open kafka-manager:

``` bash
open $(minikube service kafka-manager --url)
```
Add new cluster, and use the following data for Cluster Zookeeper Hosts:
```
zookeeper-service:2181
```

## Instructions to build the HFS cluster


### Deploy Fabric tools pod and volume claims
Run the commands in the following order:
``` bash
kubectl apply -f k8s/hfs/fabric-pv.yaml
kubectl apply -f k8s/hfs/fabric-pvc.yaml
kubectl apply -f k8s/hfs/fabric-tools.yaml
```

###  Copy configuration resources to fabric-tools pod
Run the commands in the following order:
``` bash
kubectl exec -it fabric-tools -- mkdir /fabric/config
kubectl cp config/configtx.yaml fabric-tools:/fabric/config/
kubectl cp config/core.yaml fabric-tools:/fabric/config/
kubectl cp config/orderer.yaml fabric-tools:/fabric/config/
kubectl cp config/crypto-config.yaml fabric-tools:/fabric/config/
kubectl cp chaincode/ fabric-tools:/fabric/config/
```

### Generate COKE and PEPSI certificates and blocks required to run ledger
Run the commands in the following order:

``` bash
kubectl exec -it fabric-tools -- /bin/bash
cryptogen generate --config /fabric/config/crypto-config.yaml

cp -r crypto-config /fabric/
for file in $(find /fabric/ -iname *_sk); do echo $file; dir=$(dirname $file); mv ${dir}/*_sk ${dir}/key.pem; done
cp /fabric/config/configtx.yaml /fabric/
cp /fabric/config/core.yaml /fabric/
cp /fabric/config/orderer.yaml /fabric/

cd /fabric

export FABRIC_CFG_PATH="/fabric"

configtxgen -profile TwoOrgsOrdererGenesis -outputBlock genesis.block -channelID channel1

configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./COKEanchors.tx -channelID channel1 -asOrg COKE
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./PEPSIanchors.tx -channelID channel1 -asOrg PEPSI

chmod a+rx /fabric/* -R
exit
```


### Deploy Fabric CA and Orderer pods
Run the commands in the following order:

``` bash
kubectl apply -f k8s/hfs/blockchain-ca_deploy.yaml
kubectl apply -f k8s/hfs/blockchain-ca_svc.yaml
kubectl apply -f k8s/hfs/blockchain-orderer_deploy.yaml
kubectl apply -f k8s/hfs/blockchain-orderer_svc.yaml
```

### Deploy COKE and PEPSI peer pods
Run the commands in the following order:

``` bash
kubectl apply -f k8s/hfs/blockchain-org1peer1_deploy.yaml
kubectl apply -f k8s/hfs/blockchain-org1peer1_svc.yaml
kubectl apply -f k8s/hfs/blockchain-org2peer1_deploy.yaml
kubectl apply -f k8s/hfs/blockchain-org2peer1_svc.yaml
```

### Create and add COKE, PEPSI to the channel
Run the commands in the following order:

``` bash
kubectl exec -it fabric-tools -- /bin/bash
export CHANNEL_NAME="channel1"
cd /fabric
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ${CHANNEL_NAME}.tx -channelID ${CHANNEL_NAME}

export ORDERER_URL="blockchain-orderer:31010"
export CORE_PEER_ADDRESSAUTODETECT="false"
export CORE_PEER_NETWORKID="nid1"
export FABRIC_CFG_PATH="/fabric"

export CORE_PEER_LOCALMSPID="COKE"
export CORE_PEER_MSPID="COKE"
export CORE_PEER_MSPCONFIGPATH="/fabric/crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/msp/"
export CORE_PEER_ADDRESS="blockchain-org1peer1:30110"

peer channel create -o ${ORDERER_URL} -c ${CHANNEL_NAME} -f /fabric/${CHANNEL_NAME}.tx

export CORE_PEER_MSPCONFIGPATH="/fabric/crypto-config/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
export FABRIC_CFG_PATH="/fabric"

peer channel fetch newest -o ${ORDERER_URL} -c ${CHANNEL_NAME}
peer channel join -b ${CHANNEL_NAME}_newest.block
rm -rf /${CHANNEL_NAME}_newest.block

export CORE_PEER_LOCALMSPID="PEPSI"
export CORE_PEER_MSPID="PEPSI"
export CORE_PEER_MSPCONFIGPATH="/fabric/crypto-config/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp"
export CORE_PEER_ADDRESS="blockchain-org2peer1:30110"

peer channel fetch newest -o ${ORDERER_URL} -c ${CHANNEL_NAME}
peer channel join -b ${CHANNEL_NAME}_newest.block
rm -rf /${CHANNEL_NAME}_newest.block

exit
```

### Install booking chaincode
Run the commands in the following order:

``` bash
kubectl exec -it fabric-tools -- /bin/bash
cp -r /fabric/config/chaincode $GOPATH/src/
export CHAINCODE_NAME="cc"
export CHAINCODE_VERSION="1.0"
export FABRIC_CFG_PATH="/fabric"
export CORE_PEER_MSPCONFIGPATH="/fabric/crypto-config/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
export CORE_PEER_LOCALMSPID="COKE"

export CORE_PEER_ADDRESS="blockchain-org1peer1:30110"
peer chaincode install -n ${CHAINCODE_NAME} -v ${CHAINCODE_VERSION} -p chaincode/

export CORE_PEER_ADDRESS="blockchain-org1peer2:30110"
peer chaincode install -n ${CHAINCODE_NAME} -v ${CHAINCODE_VERSION} -p chaincode/

exit
```

### Initialize booking chaincode
Run the commands in the following order:

``` bash
kubectl exec -it fabric-tools -- /bin/bash
export CHANNEL_NAME="channel1"
export CHAINCODE_NAME="cc"
export CHAINCODE_VERSION="1.0"
export FABRIC_CFG_PATH="/fabric"
export CORE_PEER_MSPCONFIGPATH="/fabric/crypto-config/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp"
export CORE_PEER_LOCALMSPID="COKE"
export CORE_PEER_ADDRESS="blockchain-org1peer1:30110"
export ORDERER_URL="blockchain-orderer:31010"

peer chaincode instantiate -o ${ORDERER_URL} -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -v ${CHAINCODE_VERSION} -P "AND('Org1MSP.member','Org2MSP.member','Org3MSP.member','Org4MSP.member')" -c '{"Args":[]}'
exit
```

### Setup anchor peers
Run the commands in the following order:

``` bash
pod=$(kubectl get pods | grep blockchain-org1peer1 | awk '{print $1}')
kubectl exec -it $pod -- peer channel update -f /fabric/COKEanchors.tx -c channel1 -o blockchain-orderer:31010
```

### Deploy blockchain explorer
Run the commands in the following order:

``` bash
kubectl apply -f k8s/hfs/blockchain-explorer-db_deploy.yaml
kubectl apply -f k8s/hfs/blockchain-explorer-db_svc.yaml
kubectl cp config/explorer/app/config.json fabric-tools:/fabric/config/explorer/app/

chmod +x config/explorer/app/run.sh
kubectl cp config/explorer/app/run.sh fabric-tools:/fabric/config/explorer/app/

kubectl apply -f k8s/hfs/blockchain-explorer-app_deploy.yaml
```

##Validation
```
pod=$(kubectl get pods | grep blockchain-org1peer1 | awk '{print $1}')
kubectl exec -it $pod -- /bin/bash
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="COKE"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:7051
export TARGET_TLS_OPTIONS=(-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt")
```

### Query and invoke using 'COKE' identity
```
peer chaincode query -C mychannel -n booking -c '{"function":"QueryRoomByName","Args":["C3"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryAllRooms","Args":[]}'

peer chaincode invoke "${TARGET_TLS_OPTIONS[@]}" -C mychannel -n booking -c '{"function":"CreateBooking","Args":["C1","2006-01-02T15:00:00Z","2006-01-02T16:00:00Z"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryBookingByDate","Args":["C1","2006-01-02T15:00:00Z","2006-01-02T16:00:00Z"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryAllBookings","Args":[]}'
```
