

export PATH=${PWD}/../bin:${PWD}:$PATH
export FABRIC_CFG_PATH=${PWD}


DEV_MODE=true


function printHelp () {
  echo "Usage: "
  echo "  trade.sh up|down|restart|generate|reset|clean|upgrade|createneworg|startneworg|stopneworg [-c <channel name>] [-f <docker-compose-file>] [-i <imagetag>] [-o <logfile>] [-dev]"
  echo "  trade.sh -h|--help (print this message)"
  echo "    <mode> - one of 'up', 'down', 'restart' or 'generate'"
  echo "      - 'up' - bring up the network with docker-compose up"
  echo "      - 'down' - clear the network with docker-compose down"
  echo "      - 'restart' - restart the network"
  echo "      - 'generate' - generate required certificates and genesis block"
  echo "      - 'reset' - delete chaincode containers while keeping network artifacts" 
  echo "      - 'clean' - delete network artifacts" 
  echo "      - 'upgrade'  - upgrade the network from v1.0.x to v1.1"
  echo "    -c <channel name> - channel name to use (defaults to \"tradechannel\")"
  echo "    -f <docker-compose-file> - specify which docker-compose file use (defaults to docker-compose-e2e.yaml)"
  echo "    -i <imagetag> - the tag to be used to launch the network (defaults to \"latest\")"
  echo "    -d - Apply command to the network in dev mode."
  echo
  echo "Typically, one would first generate the required certificates and "
  echo "genesis block, then bring up the network. e.g.:"
  echo
  echo "	trade.sh generate -c tradechannel"
  echo "	trade.sh up -c tradechannel -o logs/network.log"
  echo "        trade.sh up -c tradechannel -i 1.1.0-alpha"
  echo "	trade.sh down -c tradechannel"
  echo "        trade.sh upgrade -c tradechannel"
  echo
  echo "Taking all defaults:"
  echo "	trade.sh generate"
  echo "	trade.sh up"
  echo "	trade.sh down"
}


pushd () {
    command pushd "$@" > /dev/null
}


popd () {
    command popd "$@" > /dev/null
}


function askProceed () {
  read -p "Continue? [Y/n] " ans
  case "$ans" in
    y|Y|"" )
      echo "proceeding ..."
    ;;
    n|N )
      echo "exiting..."
      exit 1
    ;;
    * )
      echo "invalid response"
      askProceed
    ;;
  esac
}



function clearContainers () {
  CONTAINER_IDS=$(docker ps -aq)
  if [ -z "$CONTAINER_IDS" -o "$CONTAINER_IDS" == " " ]; then
    echo "---- No containers available for deletion ----"
  else
    docker rm -f $CONTAINER_IDS
  fi
}




function removeUnwantedImages() {
  DOCKER_IMAGE_IDS=$(docker images | grep "dev\|none\|test-vp\|peer[0-9]-" | awk '{print $3}')
  if [ -z "$DOCKER_IMAGE_IDS" -o "$DOCKER_IMAGE_IDS" == " " ]; then
    echo "---- No images available for deletion ----"
  else
    docker rmi -f $DOCKER_IMAGE_IDS
  fi
}




function checkPrereqs() {
  
  
  LOCAL_VERSION=$(configtxlator version | sed -ne 's/ Version: //p')
  DOCKER_IMAGE_VERSION=$(docker run --rm hyperledger/fabric-tools:$IMAGETAG peer version | sed -ne 's/ Version: //p'|head -1)

  echo "LOCAL_VERSION=$LOCAL_VERSION"
  echo "DOCKER_IMAGE_VERSION=$DOCKER_IMAGE_VERSION"

  if [ "$LOCAL_VERSION" != "$DOCKER_IMAGE_VERSION" ] ; then
     echo "=================== WARNING ==================="
     echo "  Local fabric binaries and docker images are  "
     echo "  out of  sync. This may cause problems.       "
     echo "==============================================="
  fi
}


function networkUp () {
  checkPrereqs
  
  if [ "$DEV_MODE" = true ] ; then
     pushd ./devmode
     export FABRIC_CFG_PATH=${PWD}
  fi
  
  if [ ! -d "crypto-config" ]; then
    generateCerts
    replacePrivateKey
    generateChannelArtifacts
  fi
  
  LOG_DIR=$(dirname $LOG_FILE)
  if [ ! -d $LOG_DIR ]
  then
    mkdir -p $LOG_DIR
  fi
  IMAGE_TAG=$IMAGETAG docker-compose -f $COMPOSE_FILE up >$LOG_FILE 2>&1 &

  if [ "$DEV_MODE" = true ] ; then
     popd
     export FABRIC_CFG_PATH=${PWD}
  fi

  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start network"
    exit 1
  fi
}


function newOrgNetworkUp () {
  checkPrereqs
  
  if [ ! -d "crypto-config/peerOrganizations/exportingentityorg.trade.com" ]; then
    generateCertsForNewOrg
    replacePrivateKeyForNewOrg
    generateChannelConfigForNewOrg
  fi
  
  LOG_DIR=$(dirname $LOG_FILE_NEW_ORG)
  if [ ! -d $LOG_DIR ]
  then
    mkdir -p $LOG_DIR
  fi
  IMAGE_TAG=$IMAGETAG docker-compose -f $COMPOSE_FILE_NEW_ORG up >$LOG_FILE_NEW_ORG 2>&1 &
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start network"
    exit 1
  fi
}






function upgradeNetwork () {
  docker inspect  -f '{{.Config.Volumes}}' orderer.trade.com |grep -q '/var/hyperledger/production/orderer'
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! This network does not appear to be using volumes for its ledgers, did you start from fabric-samples >= v1.0.6?"
    exit 1
  fi

  LEDGERS_BACKUP=./ledgers-backup

  
  mkdir -p $LEDGERS_BACKUP

  export IMAGE_TAG=$IMAGETAG
  COMPOSE_FILES="-f $COMPOSE_FILE"

  echo "Upgrading orderer"
  docker-compose $COMPOSE_FILES stop orderer.trade.com
  docker cp -a orderer.trade.com:/var/hyperledger/production/orderer $LEDGERS_BACKUP/orderer.trade.com
  docker-compose $COMPOSE_FILES up --no-deps orderer.trade.com

  for PEER in peer0.exporterorg.trade.com peer0.importerorg.trade.com peer0.carrierorg.trade.com peer0.regulatororg.trade.com; do
    echo "Upgrading peer $PEER"

    
    docker-compose $COMPOSE_FILES stop $PEER
    docker cp -a $PEER:/var/hyperledger/production $LEDGERS_BACKUP/$PEER/

    
    CC_CONTAINERS=$(docker ps | grep dev-$PEER | awk '{print $1}')
    if [ -n "$CC_CONTAINERS" ] ; then
        docker rm -f $CC_CONTAINERS
    fi
    CC_IMAGES=$(docker images | grep dev-$PEER | awk '{print $1}')
    if [ -n "$CC_IMAGES" ] ; then
        docker rmi -f $CC_IMAGES
    fi

    
    docker-compose $COMPOSE_FILES up --no-deps $PEER
  done
}


function networkDown () {
  
  if [ "$DEV_MODE" = true ] ; then
     pushd ./devmode
  fi

  docker-compose -f $COMPOSE_FILE down --volumes

  for PEER in peer0.exporterorg.trade.com peer0.importerorg.trade.com peer0.carrierorg.trade.com peer0.regulatororg.trade.com; do
    
    CC_CONTAINERS=$(docker ps -a | grep dev-$PEER | awk '{print $1}')
    if [ -n "$CC_CONTAINERS" ] ; then
      docker rm -f $CC_CONTAINERS
    fi
  done

  if [ "$DEV_MODE" = true ] ; then
     popd
  fi
}


function newOrgNetworkDown () {
  docker-compose -f $COMPOSE_FILE_NEW_ORG down --volumes

  for PEER in peer0.exportingentityorg.trade.com; do
    
    CC_CONTAINERS=$(docker ps -a | grep dev-$PEER | awk '{print $1}')
    if [ -n "$CC_CONTAINERS" ] ; then
      docker rm -f $CC_CONTAINERS
    fi
  done
}


function networkClean () {
  
  clearContainers
  
  removeUnwantedImages
  
  if [ "$DEV_MODE" = true ] ; then
     pushd ./devmode
  fi
  
  rm -rf channel-artifacts crypto-config add_org/crypto-config
  
  rm -f docker-compose-e2e.yaml add_org/docker-compose-exportingEntityOrg.yaml
  
  rm -rf client-certs
  if [ "$DEV_MODE" = true ] ; then
     popd
  fi
}




function replacePrivateKey () {
  
  cp docker-compose-e2e-template.yaml docker-compose-e2e.yaml
  
  if [ "$DEV_MODE" = true ] ; then
    CURRENT_DIR=$PWD
    cd crypto-config/peerOrganizations/devorg.trade.com/ca/
    PRIV_KEY=$(ls *_sk)
    cd "$CURRENT_DIR"
    sed -i "s/DEVORG_CA_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
  else
    
    
    CURRENT_DIR=$PWD
    cd crypto-config/peerOrganizations/exporterorg.trade.com/ca/
    PRIV_KEY=$(ls *_sk)
    cd "$CURRENT_DIR"
    sed -i "s/EXPORTER_CA_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
    cd crypto-config/peerOrganizations/importerorg.trade.com/ca/
    PRIV_KEY=$(ls *_sk)
    cd "$CURRENT_DIR"
    sed -i "s/IMPORTER_CA_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
    cd crypto-config/peerOrganizations/carrierorg.trade.com/ca/
    PRIV_KEY=$(ls *_sk)
    cd "$CURRENT_DIR"
    sed -i "s/CARRIER_CA_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
    cd crypto-config/peerOrganizations/regulatororg.trade.com/ca/
    PRIV_KEY=$(ls *_sk)
    cd "$CURRENT_DIR"
    sed -i "s/REGULATOR_CA_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
  fi
}

function replacePrivateKeyForNewOrg () {
  
  cp add_org/docker-compose-exportingEntityOrg-template.yaml add_org/docker-compose-exportingEntityOrg.yaml

  
  
  CURRENT_DIR=$PWD
  cd crypto-config/peerOrganizations/exportingentityorg.trade.com/ca/
  PRIV_KEY=$(ls *_sk)
  cd "$CURRENT_DIR"
  sed -i "s/EXPORTINGENTITY_CA_PRIVATE_KEY/${PRIV_KEY}/g" add_org/docker-compose-exportingEntityOrg.yaml
}


function generateCerts (){
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. exiting"
    exit 1
  fi
  echo
  echo "##########################################################"
  echo "##### Generate certificates using cryptogen tool #########"
  echo "##########################################################"
  
  if [ "$DEV_MODE" = true ] ; then
      if [ $(basename $PWD) != "devmode" ] ; then
        pushd ./devmode
        export FABRIC_CFG_PATH=${PWD}
      fi
  fi

  if [ -d "crypto-config" ]; then
    rm -Rf crypto-config
  fi
  set -x
  cryptogen generate --config=./crypto-config.yaml
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  echo
}

function generateCertsForNewOrg (){
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. exiting"
    exit 1
  fi
  echo
  echo "######################################################################"
  echo "##### Generate certificates for new org using cryptogen tool #########"
  echo "######################################################################"

  if [ -d "crypto-config/peerOrganizations/exportingentityorg.trade.com" ]; then
    rm -Rf crypto-config/peerOrganizations/exportingentityorg.trade.com
  fi
  set -x
  cryptogen generate --config=./add_org/crypto-config.yaml
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  echo
}


function generateChannelArtifacts() {
  which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. exiting"
    exit 1
  fi

  mkdir -p channel-artifacts

  echo "###########################################################"
  echo "#########  Generating Orderer Genesis block  ##############"
  echo "###########################################################"
  if [ "$DEV_MODE" = true ] ; then
    PROFILE=OneOrgTradeOrdererGenesis
    CHANNEL_PROFILE=OneOrgTradeChannel
  else 
    PROFILE=FourOrgsTradeOrdererGenesis
    CHANNEL_PROFILE=FourOrgsTradeChannel
  fi

  
  
  set -x
  configtxgen -profile $PROFILE -outputBlock ./channel-artifacts/genesis.block
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate orderer genesis block..."
    exit 1
  fi
  echo
  echo "###################################################################"
  echo "###  Generating channel configuration transaction  'channel.tx' ###"
  echo "###################################################################"
  set -x
  configtxgen -profile $CHANNEL_PROFILE -outputCreateChannelTx ./channel-artifacts/channel.tx -channelID $CHANNEL_NAME
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate channel configuration transaction..."
    exit 1
  fi

  if [ "$DEV_MODE" = false ] ; then
    echo
    echo "#####################################################################"
    echo "#######  Generating anchor peer update for ExporterOrgMSP  ##########"
    echo "#####################################################################"
    set -x
    configtxgen -profile $CHANNEL_PROFILE -outputAnchorPeersUpdate ./channel-artifacts/ExporterOrgMSPanchors.tx -channelID $CHANNEL_NAME -asOrg ExporterOrgMSP
    res=$?
    set +x
    if [ $res -ne 0 ]; then
      echo "Failed to generate anchor peer update for ExporterOrgMSP..."
      exit 1
    fi

    echo
    echo "#####################################################################"
    echo "#######  Generating anchor peer update for ImporterOrgMSP  ##########"
    echo "#####################################################################"
    set -x
    configtxgen -profile $CHANNEL_PROFILE -outputAnchorPeersUpdate \
    ./channel-artifacts/ImporterOrgMSPanchors.tx -channelID $CHANNEL_NAME -asOrg ImporterOrgMSP
    res=$?
    set +x
    if [ $res -ne 0 ]; then
      echo "Failed to generate anchor peer update for ImporterOrgMSP..."
      exit 1
    fi

    echo
    echo "####################################################################"
    echo "#######  Generating anchor peer update for CarrierOrgMSP  ##########"
    echo "####################################################################"
    set -x
    configtxgen -profile $CHANNEL_PROFILE -outputAnchorPeersUpdate \
    ./channel-artifacts/CarrierOrgMSPanchors.tx -channelID $CHANNEL_NAME -asOrg CarrierOrgMSP
    res=$?
    set +x
    if [ $res -ne 0 ]; then
      echo "Failed to generate anchor peer update for CarrierOrgMSP..."
      exit 1
    fi

    echo
    echo "######################################################################"
    echo "#######  Generating anchor peer update for RegulatorOrgMSP  ##########"
    echo "######################################################################"
    set -x
    configtxgen -profile $CHANNEL_PROFILE -outputAnchorPeersUpdate \
    ./channel-artifacts/RegulatorOrgMSPanchors.tx -channelID $CHANNEL_NAME -asOrg RegulatorOrgMSP
    res=$?
    set +x
    if [ $res -ne 0 ]; then
      echo "Failed to generate anchor peer update for RegulatorOrgMSP..."
      exit 1
    fi
    echo
  fi
}

# Generate configuration (policies, certificates) for new org in JSON format
function generateChannelConfigForNewOrg() {
  which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. exiting"
    exit 1
  fi

  mkdir -p channel-artifacts

  echo "####################################################################################"
  echo "#########  Generating Channel Configuration for Exporting Entity Org  ##############"
  echo "####################################################################################"
  set -x
  FABRIC_CFG_PATH=${PWD}/add_org/ && configtxgen -printOrg ExportingEntityOrgMSP > ./channel-artifacts/exportingEntityOrg.json
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate channel configuration for exportingentity org..."
    exit 1
  fi
  echo
}


CHANNEL_NAME="tradechannel"

COMPOSE_FILE=docker-compose-e2e.yaml
COMPOSE_FILE_NEW_ORG=add_org/docker-compose-exportingEntityOrg.yaml

IMAGETAG="latest"

LOG_FILE="logs/network.log"
LOG_FILE_NEW_ORG="logs/network-neworg.log"


MODE=$1;shift

if [ "$MODE" == "up" ]; then
  EXPMODE="Starting"
elif [ "$MODE" == "down" ]; then
  EXPMODE="Stopping"
elif [ "$MODE" == "restart" ]; then
  EXPMODE="Restarting"
elif [ "$MODE" == "clean" ]; then
  EXPMODE="Cleaning"
elif [ "$MODE" == "generate" ]; then
  EXPMODE="Generating certs and genesis block"
elif [ "$MODE" == "upgrade" ]; then
  EXPMODE="Upgrading the network"
elif [ "$MODE" == "createneworg" ]; then
  EXPMODE="Generating certs and configuration for new org"
elif [ "$MODE" == "startneworg" ]; then
  EXPMODE="Starting peer and CA for new org"
elif [ "$MODE" == "stopneworg" ]; then
  EXPMODE="Stopping peer and CA for new org"
else
  printHelp
  exit 1
fi

while getopts "h?m:c:f:i:o:d:" opt; do
  case "$opt" in
    h|\?)
      printHelp
      exit 0
    ;;
    c)  CHANNEL_NAME=$OPTARG
    ;;
    f)  COMPOSE_FILE=$OPTARG
    ;;
    i)  IMAGETAG=`uname -m`"-"$OPTARG
    ;;
    o)  LOG_FILE=$OPTARG
    ;;
    d)  DEV_MODE=$OPTARG 
    ;;
  esac
done


echo "${EXPMODE} with channel '${CHANNEL_NAME}'"

askProceed


if [ "${MODE}" == "up" ]; then
  networkUp
elif [ "${MODE}" == "down" ]; then 
  networkDown
elif [ "${MODE}" == "generate" ]; then 
  generateCerts
  replacePrivateKey
  generateChannelArtifacts
elif [ "${MODE}" == "restart" ]; then 
  networkDown
  networkUp
elif [ "${MODE}" == "reset" ]; then 
  removeUnwantedImages
elif [ "${MODE}" == "clean" ]; then 
  networkClean
elif [ "${MODE}" == "upgrade" ]; then 
  upgradeNetwork
elif [ "${MODE}" == "createneworg" ]; then 
  generateCertsForNewOrg
  replacePrivateKeyForNewOrg
  generateChannelConfigForNewOrg
elif [ "${MODE}" == "startneworg" ]; then 
  newOrgNetworkUp
elif [ "${MODE}" == "stopneworg" ]; then 
  newOrgNetworkDown
else
  printHelp
  exit 1
fi
