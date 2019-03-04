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
	assetid := 1
	intByte := []byte(strconv.Itoa(assetid))
	stub.PutState("assetid",intByte)
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
	 } else if function == "AddAsset" {
		 return t.AddAsset(stub,invokerOrg, invokerCertIssuer,args)
	 } else if function == "Response_RequestOfVictimToReliefCamp" {
		 return t.Response_RequestOfVictimToReliefCamp(stub,invokerOrg,invokerCertIssuer,args)
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
	err = stub.PutState(rid,requestBytes)
	if err != nil {
		fmt.Println("error occured while writing request info onto the ledger")
		return shim.Error(err.Error())
	}
	//############## create a composite key ################33

	indexName := "request-reliefcamp-status-email-rid"
	request_camp_emailIndexKey, err := stub.CreateCompositeKey(indexName,[]string{"r",victim.Reliefcamp,"requested",email,rid})
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

//func (t *DisasterChaincode) FetchRequestInfoByStatus(stub shim.ChaincodeStubInterface,invokerOrg string, invokerCertIssuer string, args []string) pb.Response{
	
	

//}	

func (t *DisasterChaincode) Response_RequestOfVictimToReliefCamp(stub shim.ChaincodeStubInterface,invokerOrg string, invokerCertIssuer string, args []string) pb.Response{

	var err error
	if !t.testMode && !authenticateReliefCamp(invokerOrg,invokerCertIssuer){
		return shim.Error("Caller not a member of the relief camp. access denied")
	}

	if len(args) != 4 {
		err := errors.New(fmt.Sprintf("Incorrect number of arguments. Expecting 2.Found %d",len(args)))
		return shim.Error(err.Error())
	}

	camp := args[0]		//fetch request sent to a particular camp which are pending to be addressed
	status := args[1]	// that is , which are still in requested state

	keysIterator,err := stub.GetStateByPartialCompositeKey("request-reliefcamp-status-email-rid",[]string{"r",camp,status})
	if err != nil {
		return shim.Error("Error while accessing state :"+ err.Error())
	}

	defer keysIterator.Close()
	var keys []string
	for keysIterator.HasNext() {
		keyv,err := keysIterator.Next()
		key := keyv.Key
		if err != nil{
			return shim.Error("Error while accessing state through an iterator")
		}

		keys = append(keys,key)

		var attr []string
		_,attr,err = stub.SplitCompositeKey(key)
		if err != nil{
			return shim.Error("error while splitting the composite key : " + err.Error())
		}

		//kemail := attr[3]   commented out later , please check if needed. #check
		krid := attr[4]
		// fetch the particular requeset
		requestBytes,err := stub.GetState(krid)
		if err != nil {
			fmt.Println("error while retieving the request info from the ledger " + err.Error())
			return shim.Error(err.Error())
		} else if requestBytes == nil {
			fmt.Println("request does not exist")
		}

		request := VictimRequest{}
		err = json.Unmarshal(requestBytes,&request)

		if err != nil {
			fmt.Println("error while unmarshalling request" + err.Error())
			return shim.Error(err.Error())
		}

	//process the request
		resourcetype := request.Need
		assetIterator,err := stub.GetStateByPartialCompositeKey("loc-type-id",[]string{request.RequestTo,resourcetype})
		if err != nil {
			return shim.Error("Error while accessing state(by partial composite key) : " +  err.Error())
		}
		defer assetIterator.Close()
		for assetIterator.HasNext(){
			var arr []string
			assetCompKeyvalue,err := assetIterator.Next() //composite key loc-type-id specific to the asset which we 
			assetCompKey := assetCompKeyvalue.Key
			if err != nil {								// are going to dispatch
				return shim.Error("some error occured while fetching the next composite key in the list")
			}
			_,arr,err = stub.SplitCompositeKey(assetCompKey)
			if err != nil {
				return shim.Error("some error occured while splitting the composite key")
			}
			
			dispatchAssetBytes,err := stub.GetState(arr[2]) // arr[2] in the composite key store the respective id
			if err != nil{
				return shim.Error("error occured while fetching the particular asset  whose id was obtained from the composite key")
			}

			//food - change the required parameters
			if request.Need == "food" {
				dispatchAsset := Food{}
				err = json.Unmarshal(dispatchAssetBytes,&dispatchAsset)
				if err != nil {
					fmt.Println("error while unmarshalling request" + err.Error())
					return shim.Error(err.Error())
				}
				dispatchAsset.Res.Beneficiary = request.Email
				dispatchAsset.Res.Status = "dispatched"
				dispatchAsset.Res.CurrentLocation = dispatchAsset.Res.CurrentLocation+"-dispatched"
				dispatchAssetBytes,err = json.Marshal(dispatchAsset)
				if err != nil {
					return shim.Error("error while marshalling")
				}
				//re-write the asset on the ledger
				err = stub.PutState(arr[2],dispatchAssetBytes)
				if err != nil{
					return shim.Error("error while writing the asset onto the ledger")
				}
				//delete the composite key because the asset does not exist in the camp anymore
				// and create a new relevant composite key
				stub.PutState(assetCompKey,nil)
				indexName := "loc-type-id"
				assetCompKey, err = stub.CreateCompositeKey(indexName,[]string{dispatchAsset.Res.CurrentLocation,arr[1],arr[2]})
				value := []byte{0x00}
				err = stub.PutState(assetCompKey,value)
				break

			} else if request.Need == "clothes" { 			//clothes
				dispatchAsset := Clothes{}
				err = json.Unmarshal(dispatchAssetBytes,&dispatchAsset)
				if err != nil {
					fmt.Println("error while unmarshalling request" + err.Error())
					return shim.Error(err.Error())
				}
				dispatchAsset.Res.Beneficiary = request.Email
				dispatchAsset.Res.Status = "dispatched"
				dispatchAsset.Res.CurrentLocation = dispatchAsset.Res.CurrentLocation+"-dispatched"
				dispatchAssetBytes,err = json.Marshal(dispatchAsset)
				if err != nil {
					return shim.Error("error while marshalling")
				}
				//re-write the asset on the ledger
				err = stub.PutState(arr[2],dispatchAssetBytes)
				if err != nil{
					return shim.Error("error while writing the asset onto the ledger")
				}
				//delete the composite key because the asset does not exist in the camp anymore
				// and create a new relevant composite key
				stub.PutState(assetCompKey,nil)
				indexName := "loc-type-id"
				assetCompKey, err = stub.CreateCompositeKey(indexName,[]string{dispatchAsset.Res.CurrentLocation,arr[1],arr[2]})
				value := []byte{0x00}
				err = stub.PutState(assetCompKey,value)
				break
			} else if request.Need == "moveinshelter" {  			//moveinshelter
				dispatchAsset := MoveInShelter{}
				err = json.Unmarshal(dispatchAssetBytes,&dispatchAsset)
				if err != nil {
					fmt.Println("error while unmarshalling request" + err.Error())
					return shim.Error(err.Error())
				}
				dispatchAsset.Capacity = dispatchAsset.Capacity - 1
				var x string
				if dispatchAsset.Capacity == 0{
					x = dispatchAsset.Res.CurrentLocation + "-dispatched"
				}
				dispatchAsset.Res.Beneficiary = dispatchAsset.Res.Beneficiary+","+request.Email
				//here, the value is not dispatched but it is accepted
				dispatchAsset.Res.Status = "accepted"
				dispatchAssetBytes,err = json.Marshal(dispatchAsset)
				if err != nil {
					return shim.Error("error while marshalling")
				}
				//re-write the asset on the ledger
				err = stub.PutState(arr[2],dispatchAssetBytes)
				if err != nil{
					return shim.Error("error while writing the asset onto the ledger")
				}
				//delete the composite key because the asset does not exist in the camp anymore
				// and create a new relevant composite key
				stub.PutState(assetCompKey,nil)
				indexName := "loc-type-id"
				assetCompKey, err = stub.CreateCompositeKey(indexName,[]string{x,arr[1],arr[2]})
				value := []byte{0x00}
				err = stub.PutState(assetCompKey,value)
				break
			} else if request.Need == "shelter" {   			//shelter
				dispatchAsset := Shelter{}
				err = json.Unmarshal(dispatchAssetBytes,&dispatchAsset)
				if err != nil {
					fmt.Println("error while unmarshalling request" + err.Error())
					return shim.Error(err.Error())
				}
				dispatchAsset.Res.Beneficiary = request.Email
				dispatchAsset.Res.Status = "dispatched"
				dispatchAsset.Res.CurrentLocation = dispatchAsset.Res.CurrentLocation+"-dispatched"
				dispatchAssetBytes,err = json.Marshal(dispatchAsset)
				if err != nil {
					return shim.Error("error while marshalling")
				}
				//re-write the asset on the ledger
				err = stub.PutState(arr[2],dispatchAssetBytes)
				if err != nil{
					return shim.Error("error while writing the asset onto the ledger")
				}
				//delete the composite key because the asset does not exist in the camp anymore
				// and create a new relevant composite key
				stub.PutState(assetCompKey,nil)
				indexName := "loc-type-id"
				assetCompKey, err = stub.CreateCompositeKey(indexName,[]string{dispatchAsset.Res.CurrentLocation,arr[1],arr[2]})
				value := []byte{0x00}
				err = stub.PutState(assetCompKey,value)
				break
			} else if request.Need == "medicalkit" {//medicalkit
				dispatchAsset := MedicalKit{}
				err = json.Unmarshal(dispatchAssetBytes,&dispatchAsset)
				if err != nil {
					fmt.Println("error while unmarshalling request" + err.Error())
					return shim.Error(err.Error())
				}
				dispatchAsset.Res.Beneficiary = request.Email
				dispatchAsset.Res.Status = "dispatched"
				dispatchAsset.Res.CurrentLocation = dispatchAsset.Res.CurrentLocation+"-dispatched"
				dispatchAssetBytes,err = json.Marshal(dispatchAsset)
				if err != nil {
					return shim.Error("error while marshalling")
				}
				//re-write the asset on the ledger
				err = stub.PutState(arr[2],dispatchAssetBytes)
				if err != nil{
					return shim.Error("error while writing the asset onto the ledger")
				}
				//delete the composite key because the asset does not exist in the camp anymore
				// and create a new relevant composite key
				stub.PutState(assetCompKey,nil)
				indexName := "loc-type-id"
				assetCompKey, err = stub.CreateCompositeKey(indexName,[]string{dispatchAsset.Res.CurrentLocation,arr[1],arr[2]})
				value := []byte{0x00}
				err = stub.PutState(assetCompKey,value)
				break
			}
		}

		if err != nil{
			return shim.Error("Some error occured while writing stuff to ledger in the last step")
		}
	}
	return shim.Success(nil)	
}

func (t *DisasterChaincode) AddAsset(stub shim.ChaincodeStubInterface,invokerOrg string, invokerCertIssuer string, args []string) pb.Response{
	if !t.testMode && !authenticateReliefCamp(invokerOrg,invokerCertIssuer){
		return shim.Error("Caller not a member of the relief camp. access denied")
	}
	var err error
	Type := args[0]
	var nargs int
	
	if Type == "moveinshelter"{
		nargs = 11
	} else {
		nargs = 8
	}

	if nargs != len(args) {
		err := errors.New(fmt.Sprintf("Incorrect number of arguments. Expecting 8 or 11.Found %d",len(args)))
		return shim.Error(err.Error())
	}
//###### create a resource object ###############
	quantity := args[1]
	beneficiary := args[2]
	source := args[3]
	status := args[4]
	carrier := args[5]
	description := args[6]
	Currentlocation := args[7]
	var address string 
	var food bool 
	var capacity int
	if(nargs == 11){
		capacity,err = strconv.Atoi(args[8])
		address = args[9]
		if args[10] == "yes"{
			food = true
			} else { food = false }
		
	}

//################ generate asset id #####################################

idbytes,err := stub.GetState("assetid")
if err != nil{
	return shim.Error("error while getting the asset id value :  " + err.Error())
}
ids := string(idbytes[:])
id,err := strconv.Atoi(ids)
id = id + 1;
ids = strconv.Itoa(id)
err = stub.PutState("assetid",[]byte(ids))	
if err != nil{
	return shim.Error("error while getting the asset id value : " + err.Error())
}

//############## create a composite key to store assets of a particlar type and status################

indexName := "loc-type-id"
loc_asset_id , err := stub.CreateCompositeKey(indexName,[]string{Currentlocation,Type,ids})
if err != nil {
	fmt.Println("error while creating a composite key for asset")
	return shim.Error(err.Error())
}
value := []byte{0x00}
stub.PutState(loc_asset_id,value)        

//################ create a particular type of resource ##################
	resource := Resource{Type,quantity,beneficiary,source,status,carrier,description,Currentlocation}
	if Type == "food"{
		asset := Food{resource}
		assetbytes, err := json.Marshal(asset)	//Marshal the asset structure into a sequence of bytes
		if err != nil {
			return shim.Error("Error marshalling asset " + err.Error())
		}
		//########### Store the asset details in the ledger ###########

		err = stub.PutState(ids,assetbytes)
		if err != nil {
			return shim.Error(err.Error())
		}

	} else if Type == "clothes"{
	    asset := Clothes{resource}
		assetbytes, err := json.Marshal(asset)	//Marshal the asset structure into a sequence of bytes
		if err != nil {
			return shim.Error("Error marshalling asset " + err.Error())
		}

		//########### Store the asset details in the ledger ###########

		err = stub.PutState(ids,assetbytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else if Type == "shelter"{
		asset := Shelter{resource}
		assetbytes, err := json.Marshal(asset)	//Marshal the asset structure into a sequence of bytes
		if err != nil {
			return shim.Error("Error marshalling asset " + err.Error())
		}

		//########### Store the asset details in the ledger ###########

		err = stub.PutState(ids,assetbytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else if Type == "medicalkit"{
		asset := MedicalKit{resource}
		assetbytes, err := json.Marshal(asset)	//Marshal the asset structure into a sequence of bytes
		if err != nil {
			return shim.Error("Error marshalling asset " + err.Error())
		}

		//########### Store the asset details in the ledger ###########

		err = stub.PutState(ids,assetbytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else if Type == "moveinshelter"{
		asset := MoveInShelter{capacity,address,food,resource}
		assetbytes, err := json.Marshal(asset)	//Marshal the asset structure into a sequence of bytes
		if err != nil {
			return shim.Error("Error marshalling asset(move in sheleter) " + err.Error())
		}

		//########### Store the asset details in the ledger ###########

		err = stub.PutState(ids,assetbytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	return shim.Success(nil)
}

func main() {
	twc := new(DisasterChaincode)
	twc.testMode = true
	err := shim.Start(twc)
	if err != nil {
		fmt.Printf("Error starting Disaster chaincode: %s", err.Error())
	}
}