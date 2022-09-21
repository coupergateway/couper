package cache

type Storage interface {
	Del(string)
	Get(string) (interface{}, error)
	Set(string, interface{}, int64)
}
