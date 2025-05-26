package models

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/apis/apiserver"
	apiserverv1alpha1 "k8s.io/apiserver/pkg/apis/apiserver/v1alpha1"
	apiserverv1beta1 "k8s.io/apiserver/pkg/apis/apiserver/v1beta1"
	apiservervalidation "k8s.io/apiserver/pkg/apis/apiserver/validation"
	authenticationcel "k8s.io/apiserver/pkg/authentication/cel"
)

var decoder runtime.Decoder

func init() {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(apiserverv1beta1.AddToScheme, apiserverv1alpha1.AddToScheme, apiserver.AddToScheme)
	utilruntime.Must(schemeBuilder.AddToScheme(scheme))
	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// AuthenticationConfiguration is a custom string type
type AuthenticationConfiguration string

// Validate ensures that the AuthenticationConfiguration fulfills the go-swagger validation interface
func (a AuthenticationConfiguration) Validate(formats strfmt.Registry) error {
	if a == "" {
		return nil // empty is valid
	}
	obj, schemaVersion, err := decoder.Decode([]byte(a), nil, nil)
	if err != nil {
		return errors.New(http.StatusBadRequest, "failed to decode authenticationConfiguration: %s", err)
	}

	authenticationConfig, ok := obj.(*apiserver.AuthenticationConfiguration)
	if !ok {
		return errors.New(http.StatusBadRequest, "failed to cast authenticationConfiguration type: %v", schemaVersion)
	}

	if errList := apiservervalidation.ValidateAuthenticationConfiguration(authenticationcel.NewDefaultCompiler(), authenticationConfig, nil); len(errList) != 0 {
		return errors.New(http.StatusBadRequest, "invalid authenticationConfiguration: %v", errList)
	}
	// Add additional validation logic here if needed
	return nil
}

// ContextValidate validates the AuthenticationConfiguration based on the context it is used in
func (a AuthenticationConfiguration) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	// Add context-specific validation logic here if needed
	return nil
}
