package models

type Graph struct {
	Nodes     []*Node
	Relations []*Relation
}

type Node struct {
	Label      string
	Properties map[string]interface{}
}

type Relation struct {
	Type       string
	From       string
	To         string
	Properties map[string]interface{}
}
