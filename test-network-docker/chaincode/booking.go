package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"time"
)

const (
	ROOM    = "room"
	BOOKING = "booking"
)

type (
	SmartContract struct {
		contractapi.Contract
	}

	// Asset struct of object stored on the ledger
	Asset struct {
		Id      string      `json:"id"`
		DocType string      `json:"docType"`
		Owner   string      `json:"owner"`
		Data    interface{} `json:"data"`
	}
	// Struct for room data
	Room struct {
		Name string `json:"name"`
	}
	// Struct for booking data
	Booking struct {
		Name  string    `json:"name"`
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}
)

// Entry point of smartcontract build and deployment on HFS
func main() {
	bookingSmartContract, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating booking chaincode: %v", err)
	}

	if err := bookingSmartContract.Start(); err != nil {
		log.Panicf("Error starting booking chaincode: %v", err)
	}
}

// Init creates 20 rooms in the ledger for COKE and PEPSI.
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	idx := 0
	for _, company := range []string{"COKE", "PEPSI"} {
		for i := 0; i < 10; i++ {
			name := fmt.Sprintf("%s%d", string(company[0]), i+1)

			asset := &Asset{
				Id:      name,
				DocType: ROOM,
				Owner:   company,
				Data: &Room{
					Name: name,
				},
			}
			assetBytes, err := json.Marshal(asset)
			if err != nil {
				return err
			}

			err = ctx.GetStub().PutState(asset.Id, assetBytes)
			if err != nil {
				return err
			}
			idx++
		}
	}
	return nil
}

// Create booking functions accepts room name, booking start and end dates
func (s *SmartContract) CreateBooking(ctx contractapi.TransactionContextInterface, name, start, end string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is not a valid string")
	}
	startTime, err := time.Parse("2006-01-02 15:04:05", start)
	if err != nil {
		return fmt.Errorf("start is not a valid datetime string")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", end)
	if err != nil {
		return fmt.Errorf("end is not a valid datetime string")
	}

	rooms, err := s.QueryRoomByName(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get rooms")
	}
	if len(rooms) == 0 {
		return fmt.Errorf("no such room exists")
	}

	bookings, err := s.QueryBookingByNameAndEndDate(ctx, name, start)
	if err != nil {
		return fmt.Errorf("failed to get bookings")
	}

	if len(bookings) != 0 {
		return fmt.Errorf("room is booked")
	}

	user, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return err
	}
	id, err := ctx.GetStub().CreateCompositeKey(BOOKING, []string{name, start, end})
	if err != nil {
		return err
	}
	hash := md5.Sum([]byte(id))
	asset := &Asset{
		Id:      fmt.Sprintf("%x", hash),
		DocType: BOOKING,
		Owner:   user,
		Data: &Booking{
			Name:  name,
			Start: startTime,
			End:   endTime,
		},
	}
	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(asset.Id, assetBytes)
}

// Delete booking accepts asset id
func (s *SmartContract) DeleteBooking(ctx contractapi.TransactionContextInterface, id string) error {
	if len(id) == 0 {
		return fmt.Errorf("id is not a valid string")
	}

	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}
	user, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return err
	}
	if asset.Owner != user {
		return fmt.Errorf("not an owner of the booking")
	}

	return ctx.GetStub().DelState(id)
}

// Queries all existent rooms on the ledger
func (s *SmartContract) QueryAllRooms(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"%s"}}`, ROOM)
	return getQueryResultForQueryString(ctx, queryString)
}

// Queries room by specified name
func (s *SmartContract) QueryRoomByName(ctx contractapi.TransactionContextInterface, roomName string) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"%s", "data":{"name":"%s"}}}`, ROOM, roomName)

	return getQueryResultForQueryString(ctx, queryString)
}

// Queries booking by name and booking date between start and end
func (s *SmartContract) QueryBookingByDate(ctx contractapi.TransactionContextInterface, roomName, start, end string) ([]*Asset, error) {
	startTime, err := time.Parse("2006-01-02 15:04:05", start)
	if err != nil {
		return nil, fmt.Errorf("start is not a valid datetime string")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", end)
	if err != nil {
		return nil, fmt.Errorf("end is not a valid datetime string")
	}

	queryString := fmt.Sprintf(`{"selector":{"docType":"%s", "data":{"start": {"$gte":"%s"}, "end": {"$lte":"%s"}, "name":"%s"}}}`, BOOKING, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), roomName)

	return getQueryResultForQueryString(ctx, queryString)
}

// Queries booking by name and end booking date, which is greater than provided end argument
func (s *SmartContract) QueryBookingByNameAndEndDate(ctx contractapi.TransactionContextInterface, roomName, end string) ([]*Asset, error) {
	endTime, err := time.Parse("2006-01-02 15:04:05", end)
	if err != nil {
		return nil, fmt.Errorf("start is not a valid datetime string")
	}

	queryString := fmt.Sprintf(`{"selector":{"docType":"%s", "data":{"end": {"$gt":"%s"}, "name":"%s"}}}`, BOOKING, endTime.Format(time.RFC3339), roomName)

	return getQueryResultForQueryString(ctx, queryString)
}

// Queries all existent bookings
func (s *SmartContract) QueryAllBookings(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"%s"}}`, BOOKING)
	return getQueryResultForQueryString(ctx, queryString)
}

// Utility method for query iterator
func getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

// Utility method for constructing an array of Assets from the ledger response
func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) ([]*Asset, error) {
	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset Asset
		err = json.Unmarshal(queryResult.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// Read asset by id on the ledger. Needed for DeleteBooking function
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, assetID string) (*Asset, error) {
	assetBytes, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset %s: %v", assetID, err)
	}
	if assetBytes == nil {
		return nil, fmt.Errorf("asset %s does not exist", assetID)
	}

	var asset Asset
	err = json.Unmarshal(assetBytes, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {

	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}
