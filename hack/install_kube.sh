#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.14.1/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
curl -Lo kind https://github.com/kubernetes-sigs/kind/releases/download/0.2.1/kind-linux-amd64 && chmod +x kind && sudo mv kind /usr/local/bin/

kind create cluster