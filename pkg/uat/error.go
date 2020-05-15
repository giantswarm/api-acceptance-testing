package uat

import "github.com/giantswarm/microerror"

// assertionFailedError is used when we find the API not behaving as expected.
var assertionFailedError = &microerror.Error{
	Kind: "assertionFailedError",
}

// IsAssertionFailed asserts assertionFailedError.
func IsAssertionFailed(err error) bool {
	return microerror.Cause(err) == assertionFailedError
}

// requestFailedError is used when an API results in an error.
var requestFailedError = &microerror.Error{
	Kind: "requestFailedError",
}

// IsRequestFailed asserts requestFailedError.
func IsRequestFailed(err error) bool {
	return microerror.Cause(err) == requestFailedError
}

// notYetAvailableError is used when an API results in an error.
var notYetAvailableError = &microerror.Error{
	Kind: "notYetAvailableError",
}

// IsNotYetAvailable asserts notYetAvailableError.
func IsNotYetAvailable(err error) bool {
	return microerror.Cause(err) == notYetAvailableError
}
