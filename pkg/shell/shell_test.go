package shell

import (
	"context"
	"strconv"
	"testing"
)

func Test_RunCommand(t *testing.T) {
	testCases := []struct {
		name             string
		executable       string
		envVars          []string
		arguments        []string
		expectedOutput   []string
		expectedExitCode int
		errorMatcher     func(err error) bool
	}{
		{
			name:             "case 0: non-existing command",
			executable:       "sdfsdgg",
			envVars:          []string{},
			arguments:        []string{"foo"},
			expectedOutput:   []string{""},
			expectedExitCode: -1,
			errorMatcher:     IsCoudlNotStart,
		},
		{
			name:             "case 1: sleep 1",
			executable:       "sleep",
			envVars:          []string{},
			arguments:        []string{"1"},
			expectedOutput:   []string{""},
			expectedExitCode: 0,
			errorMatcher:     nil,
		},
		{
			name:       "case 2: ls nonexist",
			executable: "ls",
			envVars:    []string{},
			arguments:  []string{"nonexist"},
			expectedOutput: []string{
				"ls: nonexist: No such file or directory\n",
				"ls: cannot access 'nonexist': No such file or directory\n",
			},
			expectedExitCode: 1,
			errorMatcher:     IsProblemInExecution,
		},
		{
			name:             "case 3: environment variable",
			executable:       "bash",
			envVars:          []string{"TESTVAR=myecho"},
			arguments:        []string{"-c", "echo -n ${TESTVAR}"},
			expectedOutput:   []string{"myecho"},
			expectedExitCode: 0,
			errorMatcher:     nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			output, exitCode, err := RunCommand(context.Background(), tc.executable, tc.envVars, tc.arguments...)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !stringSliceContains(tc.expectedOutput, output) {
				t.Fatalf("\n\nUnexpected output: %s\n", output)
			}
		})
	}
}

func stringSliceContains(stringSlice []string, needle string) bool {
	for _, elmt := range stringSlice {
		if elmt == needle {
			return true
		}
	}
	return false
}
