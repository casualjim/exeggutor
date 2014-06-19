#!/bin/sh

echo 'installing cask'
brew tap phinze/cask
brew install brew-cask

echo 'installing ansible'
brew install ansible 

echo 'installing vagrant'
brew cask install vagrant
vagrant plugin install vagrant-vbox-snapshot vagrant-vbguest
vagrant box add casualjim/trusty-vagrant

echo 'installing virtualbox'
brew cask install virtualbox

