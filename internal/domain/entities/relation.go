package entities

type Relation struct {
	BaseEntity
	Type       string
	FromNode   *Node
	ToNode     *Node
	Properties map[string]interface{}
}

func NewRelation(id string, typ string, from *Node, to *Node) *Relation {
	return &Relation{
		BaseEntity: BaseEntity{ID: id},
		Type:       typ,
		FromNode:   from,
		ToNode:     to,
		Properties: make(map[string]interface{}),
	}
}
