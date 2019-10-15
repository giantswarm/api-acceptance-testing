package cliutil

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// ExitIfError handles an error of it gets one, by printing it
// and exiting with status code 1.
func ExitIfError(err error) {
	if err == nil {
		return
	}
	Complain(err)
	os.Exit(1)
}

// Complain handles an error of it gets one, by printing it
// but does not exit.
func Complain(err error) {
	if err == nil {
		return
	}

	fmt.Printf("%s: %s\n", color.RedString("ERROR"), color.WhiteString(err.Error()))
}

// PrintSuccess just prints a success message.
func PrintSuccess(message string, v ...interface{}) {
	fmt.Printf("%s: %s\n", color.GreenString("OK"), color.WhiteString(message, v...))
}
