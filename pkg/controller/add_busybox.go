package controller

import (
	"persistent.com/busybox/busybox-go-dc-operator/pkg/controller/busybox"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, busybox.Add)
}
