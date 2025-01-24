package entities

type Graph struct {
	BaseEntity
	Nodes     []*Node
	Relations []*Relation
}

func NewGraph(id string) *Graph {
	return &Graph{
		BaseEntity: BaseEntity{ID: id},
		Nodes:      make([]*Node, 0),
		Relations:  make([]*Relation, 0),
	}
}
