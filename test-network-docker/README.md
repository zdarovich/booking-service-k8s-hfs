## Introduction
Booking service deployed on Docker

## Run

```
cd test-network-docker
export PATH=${PWD}/bin:${PWD}:$PATH
export FABRIC_CFG_PATH=$PWD/config/
./network.sh down
```

Run the following command to deploy the test network using Certificate Authorities with default channel name 'mychannel':
```
./network.sh up createChannel -ca
```

You can then use the test network script to deploy the `asset-abe-encrypted` smart contract to a channel on the network:
```
./network.sh deployCC -ccn booking -ccp ./chaincode/ -ccl go -ccv 1 -ccs 1 -cci Init
```

## Register identities with attributes

We will create the identities using the Org1 CA. Set the Fabric CA client home to the MSP of the Org1 CA admin:
```
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
```

Attributes will be added upon enrollment. The following command will register an identity named creator2 with required attributes.
```
fabric-ca-client register --id.name creator2 --id.secret creator2pw --id.type client --id.affiliation org1 --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
```

The following enroll command will add the attribute to the certificate:

```
fabric-ca-client enroll -u https://creator2:creator2pw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/creator2@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
```

Run the command below to copy the Node OU configuration file into the creator2 MSP folder.
```
cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/creator2@org1.example.com/msp/config.yaml"
```

Do the same but for second identity 'creator1'.
```
fabric-ca-client register --id.name creator1 --id.secret creator1pw --id.type client --id.affiliation org1 --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
fabric-ca-client enroll -u https://creator1:creator1pw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
mv -i ${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp/keystore/*_sk ${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp/keystore/priv_sk
cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp/config.yaml"
```

## Create an asset using 'creator1' identity

You can use either identity attribute to create an asset using the `asset-abe-encrypted` smart contract. We will set the following environment variables to use the 'creator1' identity that was generated:

```
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/creator1@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:7051
export TARGET_TLS_OPTIONS=(-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt")
```

### Query and invoke using 'COKE' identity
```
peer chaincode query -C mychannel -n booking -c '{"function":"QueryRoomByName","Args":["C3"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryAllRooms","Args":[]}'

peer chaincode invoke "${TARGET_TLS_OPTIONS[@]}" -C mychannel -n booking -c '{"function":"CreateBooking","Args":["C1","2006-01-02 15:00:00","2006-01-02 16:00:00"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryBookingByDate","Args":["C1","2006-01-02 15:00:00","2006-01-02 16:00:00"]}'
peer chaincode query -C mychannel -n booking -c '{"function":"QueryAllBookings","Args":[]}'
```

## Clean up

When you are finished, you can run the following command to bring down the test network:
```
./network.sh down
```
