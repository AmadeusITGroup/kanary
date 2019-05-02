#!/bin/bash
if [ -z "$TAG" ]
then
TAG=latest
fi

set -o errexit
set -o nounset
set -o pipefail

DIR="$(cd "$(dirname "${0}")" && pwd)"
CURRENT=$PWD
cd $DIR/../..

# Load the image
make build
make TAG=$TAG container
kind load docker-image kanary/operator:$TAG

#Laucnh proxy
kubectl proxy --port=8001&
proxyPID=$!
function killProxy {
  echo "Stopping kubeproxy"
  kill -9 ${proxyPID}
  cd $CURRENT
}
trap killProxy EXIT

#run the test
operator-sdk test local ./test/e2e --image kanary/operator:$TAG --kubeconfig $(kind get kubeconfig-path)


