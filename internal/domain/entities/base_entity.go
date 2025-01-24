package entities

type BaseEntity struct {
	ID string
}

func (e *BaseEntity) GetID() string {
	return e.ID
}
