package cache

type Storage interface {
	Del(string)
	Get(string) interface{}
	GetAllWithPrefix(string) []interface{}
	Set(string, interface{}, int64)
}
