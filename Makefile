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

e2e: build container
	operator-sdk test local ./test/e2e

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f ${ARTIFACT}

validate:
	gometalinter --vendor ./... -e test -e zz_generated --deadline 9m -D gocyclo

generate:
	operator-sdk generate k8s

install-tools:
	BINDIR=${GOPATH}/bin ./hack/install-gometalinter.sh
	./hack/install-operator-sdk.sh

.PHONY: build push clean test e2e validate install-tools