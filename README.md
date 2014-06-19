# Exeggutor

Exeggutor is a project that contains several different smaller services, to handle deployments over mesos and enforce SLA's
[Exeggutor](http://bulbapedia.bulbagarden.net/wiki/Exeggutor_(Pok%C3%A9mon)) is the name of a pokemon

* [Getting Started](#getting-started)

## TODO

Before this can be put to real work the mdb usage should be revised.  
The revision should try to group all the stores into a single file
It should also make the queues all use a single file.
Furthermore it should look into zookeeper or raft for providing leader election and log replication.

## Developing the app

There is a Makefile included that has a setup task and that will download all the necessary dependencies. Once you've completed the steps below for getting a dev env set up

```bash
git clone https://github.com/reverb/exeggutor $GOPATH/src/github.com/reverb/exeggutor
cd $GOPATH/src/github.com/reverb/exeggutor
make setup
```

running tests, this works in every subdirectory:

```bash
make test
```

Create a distribution:

```bash
cd agora
make distclean
make dist
```

### Setting up linux build environment

For this we'll need vagrant, virtualbox and ansible, if you already followed the instructions
in the ansible playbooks repo, then you can skip this part.

```bash
# you can run this by invoking ./install-tools.sh
brew install --cross-compile-common go
brew tap phinze/cask
brew install brew-cask
brew install ansible 
brew cask install vagrant
brew cask install virtualbox
vagrant plugin install vagrant-vbox-snapshot vagrant-vbguest
vagrant box add casualjim/trusty-vagrant
```

You should now be able to start the virtual machine, which will also provision it.

```bash
vagrant up
```

If you want ssh access which has sudo access then you can run 

```bash
vagrant ssh
```


### Setting up a go dev env

Install golang

```
brew install --cross-compile-common go
```


Create a folder on your file system. This will be root of your go workspace but not the project.

```bash
mkdir -p ~/projects/wordnik/go # or wherever you want to put the go files
export GOPATH=$HOME/projects/wordnik/go
export PATH=$GOPATH/bin:$PATH
```

Now time to edit .bashrc or .zshrc and perhaps also /etc/launchd.conf if you're on a mac

Add the following to .bashrc or .zshrc

```bash
export DOCKER_HOST=tcp://192.168.11.253:4243
export GOPATH=$HOME/projects/wordnik/go # bring godep into scope
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```

Open a new shell or reload your shell with 

```bash
exec $SHELL  # this is preferred over sourcing since it keeps your env clean 
```

Add this to /etc/launchd.conf if you want vim, sublime, ... to find your go stuff

```bash
setenv GOPATH $HOME/projects/wordnik/go
setenv PATH  $HOME/bin:$GOPATH/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
```

This will keep it across reboots, now to enable it for your current session

```bash
launchctl setenv PATH  $HOME/bin:$GOPATH/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
```

If you're using an editor like Sublime, Vim or Emacs you will probably need all or most of the tools below:

* github.com/tools/godep
* code.google.com/p/go.tools/cmd/vet
* code.google.com/p/go.tools/cmd/godoc
* code.google.com/p/go.tools/cmd/goimports
* code.google.com/p/go.tools/cmd/oracle
* code.google.com/p/go.tools/cmd/gotype
* code.google.com/p/go.tools/cmd/cover
* code.google.com/p/rog-go/exp/cmd/godef
* github.com/axw/gocov/gocov
* gopkg.in/matm/v1/gocov-html
* github.com/AlekSi/gocov-xml
* github.com/nsf/gocode
* github.com/golang/lint/golint
* github.com/kisielk/errcheck
* github.com/jstemmer/gotags 

Managing dependencies for a project.  Currently we can use godep until the community figures out a clear winner.

You can install all these tools with:

```bash
make devenv
```

Editors: 

For sublime there is the SublimeGo package

For vim the best one seems to be fatih/vim-go




----
notes

autosourcing a file

in bash:

```bash
cd () {
  builtin cd "$@"
  case $PWD in
    /some/directory) . ./projectSettings.bash;;
  esac
}
```

in zsh:

```bash
# http://stackoverflow.com/questions/17051123/source-a-file-in-zsh-when-entering-a-directory
autoload -U add-zsh-hook
load-local-conf() {
     # check file exists, is regular file and is readable:
     if [[ -f .reverb-env && -r .reverb-env ]]; then
       source .reverb-env
     fi
}
add-zsh-hook chpwd load-local-conf
```
