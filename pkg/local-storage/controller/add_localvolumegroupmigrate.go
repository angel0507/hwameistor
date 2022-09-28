package controller

import "github.com/hwameistor/hwameistor/pkg/local-storage/controller/localvolumegroupmigrate"

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, localvolumegroupmigrate.Add)
}
