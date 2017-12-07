package cmd

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"
)

func init() {
	//Add cidr validation
	govalidator.TagMap["cidr"] = govalidator.IsCIDR
}

type Validator interface {
	Validate(c *cobra.Command, args []string) error
}

func Validate(obj interface{}, c *cobra.Command, args []string) error {
	if err := PopulateFromEnv(obj); err != nil {
		return err
	}
	fmt.Printf("obj = %+v\n", obj)
	if _, err := govalidator.ValidateStruct(obj); err != nil {
		return err
	}
	fmt.Println("no error")
	if v, ok := obj.(Validator); ok {
		return v.Validate(c, args)
	}
	return nil

}
