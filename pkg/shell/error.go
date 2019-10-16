package shell

import "github.com/giantswarm/microerror"

var couldNotStartError = &microerror.Error{
	Kind: "couldNotStartError",
	Desc: "The command could not be started, as it perhaps could not be found.",
}

// IsCoudlNotStart asserts couldNotStartError.
func IsCoudlNotStart(err error) bool {
	return microerror.Cause(err) == couldNotStartError
}

var problemInExecutionError = &microerror.Error{
	Kind: "problemInExecutionError",
}

// IsProblemInExecution asserts problemInExecutionError.
func IsProblemInExecution(err error) bool {
	return microerror.Cause(err) == problemInExecutionError
}
