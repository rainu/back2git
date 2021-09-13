package main

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"time"
)

type config struct {
	Repository struct {
		Url    string `yaml:"url"`
		Branch string `yaml:"branch"`
		Auth   struct {
			Basic *struct {
				Username        string  `yaml:"username"`
				Password        *string `yaml:"password"`
				PasswordCommand *struct {
					Name string   `yaml:"name"`
					Args []string `yaml:"args"`
				} `yaml:"passwordCommand"`
			} `yaml:"basic"`
			Token *string `yaml:"token"`
			SSH   *struct {
				Username          string  `yaml:"username"`
				PrivateKey        string  `yaml:"privateKey"`
				PKPassword        *string `yaml:"pkPassword"`
				PKPasswordCommand *struct {
					Name string   `yaml:"name"`
					Args []string `yaml:"args"`
				} `yaml:"pkPasswordCommand"`
			} `yaml:"ssh"`
		} `yaml:"auth"`
		Path         string        `yaml:"path"`
		PushInterval time.Duration `yaml:"pushInterval"`
	} `yaml:"repository"`
	Files map[string]interface{} `yaml:"files"`
}

func LoadConfig() *config {
	if len(os.Args) != 2 {
		log.Fatal("Invalid arguments")
	}

	cfg := &config{}

	cfgFile, err := os.Open(os.Args[1])
	if err != nil {
		log.WithError(err).Fatal("Unable to read config file!")
	}

	err = yaml.NewDecoder(cfgFile).Decode(cfg)
	if err != nil {
		log.WithError(err).Fatal("Unable to parse config file!")
	}

	return cfg
}

func (c *config) Authentication() (transport.AuthMethod, error) {
	auth := c.Repository.Auth
	if auth.Basic != nil {
		password := ""

		if auth.Basic.Password != nil {
			password = *auth.Basic.Password
		} else if auth.Basic.PasswordCommand != nil {
			cmd := exec.Command(auth.SSH.PKPasswordCommand.Name, auth.SSH.PKPasswordCommand.Args...)
			rawPw, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("error while execute password command: %w", err)
			}
			password = string(rawPw)
		}

		return &http.BasicAuth{
			Username: auth.Basic.Username,
			Password: password,
		}, nil
	}
	if auth.Token != nil {
		return &http.BasicAuth{
			Username: "git", //the value must only be not empty
			Password: *auth.Token,
		}, nil
	}
	if auth.SSH != nil {
		password := ""
		if auth.SSH.PKPassword != nil {
			password = *auth.SSH.PKPassword
		} else if auth.SSH.PKPasswordCommand != nil {
			cmd := exec.Command(auth.SSH.PKPasswordCommand.Name, auth.SSH.PKPasswordCommand.Args...)
			rawPw, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("error while execute password command: %w", err)
			}
			password = string(rawPw)
		}

		publicKeys, err := ssh.NewPublicKeysFromFile(auth.SSH.Username, auth.SSH.PrivateKey, password)
		if err != nil {
			return nil, fmt.Errorf("generate publickeys failed: %w", err)
		}
		return publicKeys, nil
	}

	return nil, nil
}
