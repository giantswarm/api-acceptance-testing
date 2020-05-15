package client

import "github.com/giantswarm/microerror"

// invalidConfigError is used when parameters given by the user are invalid.
var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
