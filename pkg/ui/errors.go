package ui

import (
	"fmt"
	"os"
)

// Fatal logs an error message and exits with code 1
func Fatal(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// FatalError logs an error object and exits with code 1
func FatalError(err error) {
	if err != nil {
		Error(err.Error())
		os.Exit(1)
	}
}

// WarningF prints a formatted warning message but continues execution
func WarningF(format string, args ...interface{}) {
	Warning(fmt.Sprintf(format, args...))
}

// ErrorF prints a formatted error message but does not exit
func ErrorF(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

// InfoF prints a formatted info message
func InfoF(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

// SuccessF prints a formatted success message
func SuccessF(format string, args ...interface{}) {
	Success(fmt.Sprintf(format, args...))
}
