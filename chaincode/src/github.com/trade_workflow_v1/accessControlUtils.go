package main

import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/lib/cid"
	"crypto/x509"
)

func getTxCreatorInfo(stub shim.ChaincodeStubInterface) (string, string, error) {
	var mspid string
	var err error
	var cert *x509.Certificate

	mspid, err = cid.GetMSPID(stub)
	if err != nil {
		fmt.Printf("Error getting MSP identity: %s\n", err.Error())
		return "", "", err
	}

	cert, err = cid.GetX509Certificate(stub)
	if err != nil {
		fmt.Printf("Error getting client certificate: %s\n", err.Error())
		return "", "", err
	}

	return mspid, cert.Issuer.CommonName, nil
}

// For now, just hardcode an ACL
// We will support attribute checks in an upgrade

func authenticateReliefCamp(mspID string, certCN string) bool {
	return (mspID == "ReliefCampMSP") && (certCN == "ca.reliefcamp.disaster.com")
}

func authenticateLocalHub(mspID string, certCN string) bool {
	return (mspID == "LocalHubMSP") && (certCN == "ca.localhub.disaster.com")
}

func authenticateLogistics(mspID string, certCN string) bool {
	return (mspID == "LogisticsMSP") && (certCN == "ca.logistics.disaster.com")
}

func authenticateNGO(mspID string, certCN string) bool {
	return (mspID == "NGOMSP") && (certCN == "ca.ngo.disaster.com")
}

func authenticateGovernment(mspID string, certCN string) bool {
	return (mspID == "GovernmentMSP") && (certCN == "ca.government.disaster.com")
}