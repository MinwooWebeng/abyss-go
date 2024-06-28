package player

type Player interface {
	Get(path string)
	Head(path string)
	Post(path string)
	Put(path string)

	Join(path string)
}
