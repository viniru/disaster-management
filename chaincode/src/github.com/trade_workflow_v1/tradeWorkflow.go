package main

import (
	"fmt"
	"errors"
	"strconv"
	"strings"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	//"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// DisasterChaincode implementation
type DisasterChaincode struct {			//implements the shim.chaincode interface
	testMode bool
}

//Init method is called once the chaincode has been installed onto the blockchain.It is called only once
//by each endorsing peer that deploys its own instance of the chaincode.

//This mehtod can be used for initialising, bootstrapping and setting up the chaincode.

func (t *DisasterChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Initializing Disaster Management")
	return shim.Success(nil)
}

func (t *DisasterChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	fmt.Println("Disaster Management Invoke")

	var err error
	var invokerOrg, invokerCertIssuer string

	if !t.testMode {
		invokerOrg, invokerCertIssuer, err = getTxCreatorInfo(stub)
		if err != nil {
			fmt.Println("Error extracting invoker identity info: %s\n", err.Error())
			fmt.Errorf("Error extracting invoker identity info: %s\n", err.Error())
			return shim.Error(err.Error())
		}
		fmt.Printf("TradeWorkflow Invoke by '%s', '%s'\n", invokerOrg, invokerCertIssuer)
	}

	function, args := stub.GetFunctionAndParameters()

	if function == "RegisterVictim" {
		 return t.RegisterVictim(stub,invokerOrg, invokerCertIssuer,args)

	 } else if function == "Request_VictimToReliefCamp" {
		 return t.Request_VictimToReliefCamp(stub,invokerOrg, invokerCertIssuer,args)
	 }

	 return shim.Error("invalid mehtod invokation")
}

func (t *DisasterChaincode) RegisterVictim(stub shim.ChaincodeStubInterface,invokerOrg string, invokerCertIssuer string, args []string) pb.Response{
var err error

	if !t.testMode && !authenticateReliefCamp(invokerOrg,invokerCertIssuer){
		return shim.Error("Caller not a member of the relief camp. access denied")
	}

	if(len(args) != 5) {
		err = errors.New(fmt.Sprintf("Incorrect number of arguments. Expecting 5.Found %d",len(args)))
		return shim.Error(err.Error())
	}
	reliefcamp := strings.ToLower(args[0])
	email := strings.ToLower(args[2])
	
	//=== check if the user already exitsts===
	victimBytes, err := stub.GetState(email)
	if err != nil {
		fmt.Println("Failed to check whether the email exists or not")
		return shim.Error("Failed to check whether the email exists or not " + err.Error())
	} else if victimBytes != nil {
		fmt.Println("This marble already exists " + email)
		return shim.Error("The requested email already exists " + email)
	}

	//########### create victim object and marshal to json ##############
	
	var victim Victim
	victim.Reliefcamp = reliefcamp
	victim.HealthCondition = args[1]
	victim.NumRequests = 0
	victim.Details = Participant{
	Email : email,
	Location : args[3],
	Description	 : args[4],
	}

	victimBytes, err = json.Marshal(victim)		//Marshal the victim structure into a sequence of bytes
	if err != nil {
		return shim.Error("Error marshalling trade Agreement structure")
	}


	//########### Store the victim details in the ledger ###########

	err = stub.PutState(email,victimBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//######### create a composite key ##########

	indexName := "Reliefcamp-email"
	camp_emailIndexKey, err := stub.CreateCompositeKey(indexName,[]string{reliefcamp, email})
	if err != nil {
		fmt.Println("error while creating a composite key")
		return shim.Error(err.Error())
	}

	//store the index name onto the ledger, just the index and not the info about the corresponding victim
	
	value := []byte{0x00}
	stub.PutState(camp_emailIndexKey,value)

	//victim info saved successfully
	fmt.Println("victim info saved successfully")
	
	return shim.Success(nil)
}

func (t *DisasterChaincode) Request_VictimToReliefCamp(stub shim.ChaincodeStubInterface,invokerOrg string, invokerCertIssuer string, args []string) pb.Response{

	if !t.testMode && !authenticateReliefCamp(invokerOrg,invokerCertIssuer){
		return shim.Error("Caller not a member of the relief camp. access denied")
	}

	if(len(args) != 4) {
		err := errors.New(fmt.Sprintf("Incorrect number of arguments. Expecting 4.Found %d",len(args)))
		return shim.Error(err.Error())
	}


	email := args[0]

	//check if the victim info is there on the ledger
	victimBytes, err := stub.GetState(email)

	if err != nil {
		fmt.Println("error while retieving the victim info from the ledger " + err.Error())
		return shim.Error(err.Error())
	} else if victimBytes == nil {
		fmt.Println("victim does not exist")
	}

	victim := Victim{}
	err = json.Unmarshal(victimBytes,&victim)

	if err != nil {
		fmt.Println("error while unmarshalling " + err.Error())
		return shim.Error(err.Error())
	}

	//go on with considering the request
	rid := email + strconv.Itoa((victim.NumRequests+1))	//generate request id for this particular victim and request
	victim.NumRequests = victim.NumRequests+1

	//create the request asset

	request := VictimRequest{email,rid,victim.Reliefcamp,"requested",args[1],args[2],args[3]}

	//update the data of the victim on the ledger

	victimPutBytes,err := json.Marshal(victim)
	if err != nil {
		fmt.Println("error occured while marhsalling victim info")
		return shim.Error(err.Error())
	}
	err = stub.PutState(email,victimPutBytes)
	if err != nil {
		fmt.Println("error occured while writing victim info onto the ledger")
		return shim.Error(err.Error())
	}

	//write the request data onto the ledger
	requestBytes, err := json.Marshal(request)
	if err != nil {
		fmt.Println("error occured while marhsalling request info")
		return shim.Error(err.Error())
	}
	err = stub.PutState(email,requestBytes)
	if err != nil {
		fmt.Println("error occured while writing request info onto the ledger")
		return shim.Error(err.Error())
	}
	//############## create a composite key ################33

	indexName := "request-reliefcamp-email-requestid"
	request_camp_emailIndexKey, err := stub.CreateCompositeKey(indexName,[]string{"r",victim.Reliefcamp,email,rid})
	if err != nil {
		fmt.Println("error while creating a composite key")
		return shim.Error(err.Error())
	}

	//store the index name onto the ledger, just the index and not the info about the corresponding victim
	
	value := []byte{0x00}
	stub.PutState(request_camp_emailIndexKey,value)
	fmt.Println("done")
	return shim.Success(nil)
}

func main() {
	twc := new(DisasterChaincode)
	twc.testMode = true
	err := shim.Start(twc)
	if err != nil {
		fmt.Printf("Error starting Disaster chaincode: %s", err)
	}
}