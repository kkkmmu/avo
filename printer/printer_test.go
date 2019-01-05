package printer_test

import (
	"fmt"

	"github.com/mmcloughlin/avo/printer"
)

func ExampleConfig_GeneratedWarning() {
	// Default configuration named "avo".
	cfg := printer.NewDefaultConfig()
	fmt.Println(cfg.GeneratedWarning())

	// Name can be customized.
	cfg = printer.Config{
		Name: "mildred",
	}
	fmt.Println(cfg.GeneratedWarning())

	// Argv takes precedence.
	cfg = printer.Config{
		Argv: []string{"echo", "hello", "world"},
		Name: "mildred",
	}
	fmt.Println(cfg.GeneratedWarning())

	// Output:
	// Code generated by avo. DO NOT EDIT.
	// Code generated by mildred. DO NOT EDIT.
	// Code generated by command: echo hello world. DO NOT EDIT.
}