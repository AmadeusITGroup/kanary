#!/bin/bash
set -x
set -o errexit
set -o nounset
set -o pipefail

kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller --upgrade --wait