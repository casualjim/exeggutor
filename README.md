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

There is a Makefile included that has a setup task and that will download all the necessary dependencies. Once you've completed the above steps

```bash
git clone https://github.com/reverb/exeggutor $GOPATH/src/github.com/reverb/exeggutor
cd $GOPATH/src/github.com/reverb/exeggutor
make setup
```

### Setting up a go dev env

Create a folder on your file system. This will be root of your go workspace but not the project.

```bash
mkdir -p ~/projects/wordnik/go
export GOPATH=$HOME/projects/wordnik/go
export PATH=$GOPATH/bin:$PATH
```

Now time to edit .bashrc or .zshrc and perhaps also /etc/launchd.conf if you're on a mac

Add the following to .bashrc or .zshrc

```bash
export GOPATH=$HOME/projects/wordnik/go # bring godep into scope
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOPATH=$(godep env):$GOPATH # override gopath to prefer the vendor directory
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

Install the dependency management tool

```bash
go get -u github.com/tools/godep
```

If you're using an editor like Sublime, Vim or Emacs you will probably need all or most of the tools below (make setup took care of this):

```bash
go get code.google.com/p/go.tools/cmd/vet
go get code.google.com/p/go.tools/cmd/godoc
go get code.google.com/p/go.tools/cmd/goimports
go get code.google.com/p/go.tools/cmd/oracle
go get code.google.com/p/go.tools/cmd/gotype
go get code.google.com/p/go.tools/cmd/cover
go get code.google.com/p/rog-go/exp/cmd/godef
go get github.com/axw/gocov/gocov
go get gopkg.in/matm/v1/gocov-html
go get github.com/AlekSi/gocov-xml
go get github.com/nsf/gocode
go get github.com/golang/lint/golint
go get github.com/kisielk/errcheck
go get github.com/jstemmer/gotags 
```

Managing dependencies for a project.  Currently we can use godep until the community figures out a clear winner.

```bash
go get github.com/tools/godep
```


Editors: 

For sublime there is the SublimeGo package

For vim the best one seems to be fatih/vim-go


### Install build dependencies

Install protoc-gen-go, you will also need protobuf, protobuf-c, gcc and g++

```bash
go get code.google.com/p/goprotobuf/{proto,protoc-gen-go}
```





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
