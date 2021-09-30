package mtg

type Configuration struct {
	ClientId   string
	SessionId  string
	PrivateKey string
	PinToken   string
	PIN        string
	Members    []string
	Threshold  int
}
