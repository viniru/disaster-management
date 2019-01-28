package main

import (
	"fmt"
	//"errors"
	//"strconv"
	//"strings"
	//"encoding/json"
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
	function, args := stub.GetFunctionAndParameters()

	 if function == "testOne" {
		 fmt.Printf("testOne invoked and the argument is '%s' \n",args[0])
		 return t.testOne(stub,args)
	 } else if function == "testTwo" {
		 fmt.Printf("testTwo invoked and the arguments are '%s' and '%s' \n", args[0],args[1])
		 return t.testTwo(stub,args)
	 }

	 return shim.Error("invalid mehtod invokation")
}

func (t *DisasterChaincode) testOne(stub shim.ChaincodeStubInterface,args []string) pb.Response{
	return shim.Success(nil)
}

func (t *DisasterChaincode) testTwo(stub shim.ChaincodeStubInterface,args []string) pb.Response{
	return shim.Success(nil)
}

func main() {
	twc := new(DisasterChaincode)
	twc.testMode = false
	err := shim.Start(twc)
	if err != nil {
		fmt.Printf("Error starting Disaster chaincode: %s", err)
	}
}