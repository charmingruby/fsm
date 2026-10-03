// Package greeting provides simple greeting functions.
package greeting

import "fmt"

// Greet returns a greeting message for the given name.
func Greet(name string) string {
	return fmt.Sprintf("hi, %s!", name)
}
