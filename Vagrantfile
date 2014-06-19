# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "casualjim/trusty-vagrant"

  config.vm.provider "virtualbox" do |v|
    v.memory = 2048
  end
  
  config.vm.define "exeggutor-box" do |b|
    b.vm.network :private_network, ip: "192.168.11.253"
    b.vm.synced_folder ".", "/vagrant", disabled: true
    b.vm.synced_folder ".", "/usr/share/exeggutor"
  end

  config.vm.provision "ansible" do |ansible|
    ansible.playbook = "./playbook.yml"
    ansible.sudo = true
  end

end

# export DOCKER_HOST=tcp://192.168.11.253:4243
