package nft

type Store interface {
	WriteProperty(key, val []byte) error
	ReadProperty(key []byte) ([]byte, error)
}
