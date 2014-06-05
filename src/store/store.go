package store

type Log interface {
	Write(s string) (error)
}