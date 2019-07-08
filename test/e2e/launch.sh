#!/bin/bash
set -x

if [ -z "$TAG" ]
then
TAG=latest
fi

if [ -z "$SKIPBUILD" ]
then
SKIPBUILD=0
fi

set -o errexit
set -o nounset
set -o pipefail

DIR="$(cd "$(dirname "${0}")" && pwd)"
CURRENT=$PWD
cd "$DIR/../.."

# Load the image
if [ ! "$SKIPBUILD" = "1" ]
then
    make build
    make TAG=$TAG container
fi
kind load docker-image kanaryoperator/operator:$TAG

#Laucnh proxy
kubectl proxy --port=8001&
proxyPID=$!
function killProxy {
  echo "Stopping kubeproxy"
  kill -9 ${proxyPID}
  cd "$CURRENT"
}
trap killProxy EXIT

#run the test
GO111MODULE=on operator-sdk test local ./test/e2e --image kanaryoperator/operator:$TAG


