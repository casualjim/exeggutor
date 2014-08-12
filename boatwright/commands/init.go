package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"code.google.com/p/gopass"
	"github.com/reverb/exeggutor/boatwright"
	"gopkg.in/yaml.v1"
)

// InitCommand is used to initialize the boatwright application
type InitCommand struct {
	Force bool `short:"f" long:"force" description:"When this is set then the config file will be reinitialized"`
}

// Execute runs this command
func (i *InitCommand) Execute(config *boatwright.Config) {
	dpath := os.Getenv("HOME") + "/.boatwright"
	pth := dpath + "/config.yml"
	if _, err := os.Stat(pth); os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Couldn't find a configuration file, do you want to create one (Y/n)")
		createYn, _ := reader.ReadString('\n')
		if strings.HasPrefix(strings.ToUpper(createYn), "N") {
			fmt.Print("Can't proceed without the configuration file ~/.boatwright/config.yml:")
			fmt.Println(`
---
common: &common
  caprica:
    user: example
    pass: guessme
ssh:
  private_key: ~/.ssh/id_rsa
  user: example93
dev:
  <<: *common
  caprica:
    url: https://caprica-dev.helloreverb.com
prod:
  <<: *common
  caprica:
    url: https://caprica.helloreverb.com
  `)
			os.Exit(1)
		}

		fmt.Print("Caprica user: ")
		devuser := ""
		fmt.Scanln(&devuser)

		devpass, _ := gopass.GetPass("Caprica password: ")
		fmt.Print("ssh key file: ($HOME/.ssh/id_rsa)")
		keyfile := ""
		fmt.Scanln(&keyfile)
		if strings.TrimSpace(keyfile) != "" {
			if keyfile[:2] == "~/" {
				keyfile = os.Getenv("HOME") + keyfile[:1]
			}
		} else {
			keyfile = "$HOME/.ssh/id_rsa"
		}

		u, _ := user.Current()
		fmt.Print("ssh user: (" + u.Username + ")")
		sshuser := ""
		fmt.Scanln(&sshuser)
		if strings.TrimSpace(sshuser) == "" {
			sshuser = u.Username
		}

		if _, err := os.Stat(os.Getenv("HOME") + "/.boatwright"); os.IsNotExist(err) {
			os.MkdirAll(dpath, 0700)
		}

		cfgTempl := `---
common: &common
  caprica: 
    user: %s
    pass: %s
ssh:
  private_key: %s
  user: %s
dev:
  <<: *common
  caprica: 
    url: https://caprica-dev.helloreverb.com
prod:
  <<: *common
  caprica:
    url: https://caprica.helloreverb.com
`

		cfgStr := fmt.Sprintf(cfgTempl, devuser, devpass, os.ExpandEnv(keyfile), sshuser)
		err = ioutil.WriteFile(pth, []byte(cfgStr), 0600)
		if err != nil {
			fmt.Errorf("Failed to write the config file at %s, because %v", pth, err)
			os.Exit(1)
		}

		err = yaml.Unmarshal([]byte(cfgStr), config)
		if err != nil {
			fmt.Errorf("Failed to parse config file at %s, because %v", pth, err)
			os.Exit(1)
		}

	} else {
		data, err := ioutil.ReadFile(pth)
		if err != nil {
			fmt.Errorf("Failed to read the config file at %s, because %v", pth, err)
			os.Exit(1)
		}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			fmt.Errorf("Failed to parse config file at %s, because %v", pth, err)
			os.Exit(1)
		}
	}

}
