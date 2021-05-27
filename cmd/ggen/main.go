package main

import (
	"github.com/olvrng/ggen/generators/sample"
	"github.com/olvrng/ggen/ggencmd"
)

func main() {
	ggencmd.Start(sample.New())
}
