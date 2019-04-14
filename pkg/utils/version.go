package utils

import (
	"fmt"
	"time"
)

//BUILDTIME should be populated by at build time: -ldflags "-w -X github.com/amadeusitgroup/kubervisor/pkg/utils.BUILDTIME=${DATE}
//with for example DATE=$(shell date +%Y-%m-%d/%H:%M:%S )   (pay attention not to use space!)
var BUILDTIME string

//TAG should be populated by at build time: -ldflags "-w -X github.com/amadeusitgroup/kubervisor/pkg/utils.TAG=${TAG}
//with for example TAG=$(shell git tag|tail -1)
var TAG string

//COMMIT should be populated by at build time: -ldflags "-w -X github.com/amadeusitgroup/kubervisor/pkg/utils.COMMIT=${COMMIT}
//with for example COMMIT=$(shell git rev-parse HEAD)
var COMMIT string

//BRANCH should be populated by at build time: -ldflags "-w -X github.com/amadeusitgroup/kubervisor/pkg/utils.BRANCH=${BRANCH}
//with for example BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
var BRANCH string

// BuildInfos returns binary build information
func BuildInfos() {
	fmt.Println("Program started at: " + time.Now().String())
	fmt.Println("BUILDTIME=" + BUILDTIME)
	fmt.Println("TAG=" + TAG)
	fmt.Println("COMMIT=" + COMMIT)
	fmt.Println("BRANCH=" + BRANCH)
}
