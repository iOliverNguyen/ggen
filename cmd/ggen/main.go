package ggen

import (
	"github.com/ng-vu/ggen/gens/sample"
	"github.com/ng-vu/ggen/ggencmd"
)

func main() {
	ggencmd.Start(sample.New())
}
