# Exeggutor

A container project for generating spark projects.
[Exeggutor](http://bulbapedia.bulbagarden.net/wiki/Exeggutor_(Pok%C3%A9mon)) is the name of a pokemon

* [Getting Started](#getting-started)

## Getting Started

Install Vagrant and VirtualBox, see the [ansible-playbooks repo](https://github.com/reverb/ansible-playbooks/blob/master/README.md#install-vagrant-and-virtual-box)

To configure ansible for your system layout it needs an environment variable to be set.  
You can do this for example by adding the following line to ~/.profile or ~/.zprofile

```bash
export REVERB_ANSIBLE_ROLES=$HOME/projects/wordnik/ansible-playbooks/playbooks/roles
```

This allows you to use the roles defined in the common repository which standardize the way we install applications everywhere.

To start the VM:

```bash
vagrant up
```

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