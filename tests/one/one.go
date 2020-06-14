package one

// +gen
// +gen:sample=10

// this is comment of A
//
// +gen: this directive should be ignored from comment text
type A struct {
	Zero struct{}

	// comment of One
	One int

	Two string

	//
	// comment of Three
	//
	Three bool
}

// +gen:b: this directive should be ignored
type B int

// +gen:last: 20: number:int * x
