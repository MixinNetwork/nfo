package mtg

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type Configuration struct {
	App struct {
		ClientId   string `toml:"client-id"`
		SessionId  string `toml:"session-id"`
		PrivateKey string `toml:"private-key"`
		PinToken   string `toml:"pin-token"`
		PIN        string `toml:"pin"`
	} `toml:"app"`
	Group struct {
		Members   []string `toml:"members"`
		Threshold int      `toml:"threshold"`
	}
}

func Setup(path string) (*Configuration, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Configuration
	err = toml.Unmarshal(f, &conf)
	return &conf, err
}
