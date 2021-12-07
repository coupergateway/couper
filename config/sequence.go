package config

type SequenceItem interface {
	Body
	Add(item SequenceItem)
	Deps() []SequenceItem
	GetName() string
}
