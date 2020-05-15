package project

var (
	description = "Test suite for node pools functionality"
	gitSHA      = "n/a"
	name        = "npat"
	source      = "https://github.com/giantswarm/api-acceptance-test"
	version     = "n/a"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
