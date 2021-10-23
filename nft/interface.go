package nft

type Store interface {
	WriteMintToken(collection []byte, id []byte, user string) error
	ReadMintCollection(collection []byte) (*Collection, error)
	ReadMintToken(collection, token []byte) (*Token, error)
}

type Collection struct {
	Key         []byte
	Creator     string
	Circulation int
}

type Token struct {
	Collection []byte
	Key        []byte
}
