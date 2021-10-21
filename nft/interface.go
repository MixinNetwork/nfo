package nft

type Store interface {
	WriteMintToken(group []byte, id []byte, user string) error
	ReadMintGroup(group []byte) (*Group, error)
	ReadMintToken(group, token []byte) (*Token, error)
}

type Group struct {
	Key         []byte
	Creator     string
	Circulation int
}

type Token struct {
	Group []byte
	Key   []byte
}
