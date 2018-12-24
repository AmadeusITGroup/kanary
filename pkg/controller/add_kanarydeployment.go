package controller

import (
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kanarydeployment.Add)
}
