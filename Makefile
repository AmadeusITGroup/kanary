ARTIFACT=operator
ARTIFACT_PLUGIN=kubectl-kanary

# 0.0 shouldn't clobber any released builds
TAG= latest
PREFIX =  kanary/${ARTIFACT}
SOURCEDIR = "."

SOURCES := $(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

all: build

build: ${ARTIFACT}

${ARTIFACT}: ${SOURCES}
	CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o ${ARTIFACT} ./cmd/manager/main.go

build-plugin: ${ARTIFACT_PLUGIN}

${ARTIFACT_PLUGIN}: ${SOURCES}
	CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o ${ARTIFACT_PLUGIN} ./cmd/kubectl-kanary/main.go

container:
	operator-sdk build $(PREFIX):$(TAG)

test:
	./go.test.sh

e2e:
	./test/e2e/launch.sh

simple-server:
	CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o ./test/simple-server/docker/build/simple-server ./test/simple-server
	docker build -t simpleserver:latest ./test/simple-server/docker
    ifeq ($(KINDPUSH), true)
	 kind load docker-image simpleserver:latest
    endif

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f ${ARTIFACT}

validate:
	${GOPATH}/bin/golangci-lint run ./...

generate:
	operator-sdk generate k8s

install-tools:
	./hack/golangci-lint.sh -b ${GOPATH}/bin v1.16.0
	./hack/install-operator-sdk.sh

.PHONY: build push clean test e2e validate install-tools