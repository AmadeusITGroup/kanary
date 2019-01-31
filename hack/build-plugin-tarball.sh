#!/bin/sh
set -e

GIT_ROOT=$(git rev-parse --show-toplevel)
OUTPUT_FOLDER=$GIT_ROOT/plugin/output/
OUTPUT_BIN_LINUX=$OUTPUT_FOLDER/bin/linux
OUTPUT_BIN_DARWIN=$OUTPUT_FOLDER/bin/darwin
TARBALL_NAME="kubectl-kanary_$VERSION.tar.gz"

cd ..
#Linux
mkdir -p $OUTPUT_BIN_LINUX
CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo -ldflags '-w' -o $OUTPUT_BIN_LINUX/kubectl-kanary $GIT_ROOT/cmd/kubectl-kanary/main.go

# Darwin
mkdir -p $OUTPUT_BIN_DARWIN
CGO_ENABLED=0 GOOS=darwin go build -i -installsuffix cgo -ldflags '-w' -o $OUTPUT_BIN_DARWIN/kubectl-kanary $GIT_ROOT/cmd/kubectl-kanary/main.go


cd $OUTPUT_FOLDER && tar -cvf $GIT_ROOT/plugin/$TARBALL_NAME .
TAR_SHA=$(shasum -a 256 $GIT_ROOT/plugin/$TARBALL_NAME | cut -f1 -d" ")
cp $GIT_ROOT/plugin/kanary_template.yaml $OUTPUT_FOLDER/kanary.yaml

sed -i 's/TAR_SHA256/'$TAR_SHA'/g' $OUTPUT_FOLDER/kanary.yaml
sed -i 's/vVERSION/'$VERSION'/g' $OUTPUT_FOLDER/kanary.yaml