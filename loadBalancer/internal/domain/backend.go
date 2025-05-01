package domain

type Backend interface {
	IsAlive() bool
	SetAlive(value bool)
}
