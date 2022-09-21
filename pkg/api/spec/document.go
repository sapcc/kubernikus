package spec

import (
	"fmt"

	"github.com/go-openapi/loads"
)

var doc *loads.Document = nil

// Spec returns the analyzed swagger document
func Spec() (*loads.Document, error) {
	var err error
	if doc == nil {
		if doc, err = loads.Analyzed(SwaggerJSON, ""); err != nil {
			doc = nil
			return nil, err
		}
	}
	return doc, nil
}

func MustDefaultString(definition, property string) string {
	d, err := DefaultString(definition, property)
	if err != nil {
		panic(err)
	}
	return d
}

func DefaultString(definition, property string) (string, error) {
	defaultVal, err := lookupDefault(definition, property)
	if err != nil {
		return "", err
	}
	defaultString, ok := defaultVal.(string)
	if !ok {
		return "", fmt.Errorf("default value is not of type string")
	}
	return defaultString, nil
}

func MustDefaultInt64(definition, property string) int64 {
	d, err := DefaultInt64(definition, property)
	if err != nil {
		panic(err)
	}
	return d
}

func DefaultInt64(definition, property string) (int64, error) {
	defaultVal, err := lookupDefault(definition, property)
	if err != nil {
		return 0, err
	}
	defaultFloat, ok := defaultVal.(float64)
	if !ok {
		return 0, fmt.Errorf("default value is not of expected type float64: %T", defaultVal)
	}

	//round damn json float to int64
	return int64(defaultFloat + 0.5), nil
}

func lookupDefault(definition, property string) (interface{}, error) {
	document, err := Spec()
	if err != nil {
		return "", err
	}

	def, ok := document.Spec().Definitions[definition]
	if !ok {
		return nil, fmt.Errorf("definition %s not found", definition)
	}
	prop, ok := def.Properties[property]
	if !ok {
		return nil, fmt.Errorf("property %s not found in definition %s", property, definition)
	}
	if prop.Default == nil {
		return nil, fmt.Errorf("No default found for property %s", property)
	}
	return prop.Default, nil
}
