package cmd

import (
	"errors"
	"reflect"

	"dario.cat/mergo"
	"github.com/joeshaw/envdecode"
)

func PopulateFromEnv(obj interface{}) error {
	s := reflect.ValueOf(obj)
	if s.Kind() != reflect.Ptr {
		return errors.New("not a pointer")
	}
	//Create a zero valued copy of the passed struct
	optionsFromEnv := reflect.New(s.Elem().Type()).Interface()
	if err := envdecode.Decode(optionsFromEnv); err != nil {
		return err
	}
	//Populate empty fields with values from the environment
	if err := mergo.Merge(obj, optionsFromEnv); err != nil {
		return err
	}
	return nil
}
