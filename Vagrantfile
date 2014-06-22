# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "casualjim/trusty-vagrant"

  config.vm.provider "virtualbox" do |v|
    v.name = "exeggutor-box"
    v.memory = 4096
    v.cpus = 4
    
    v.customize ["modifyvm", :id, "--cpuexecutioncap", "50"]    
    v.customize ["modifyvm", :id, "--usb", "off"]
    v.customize ["modifyvm", :id, "--usbehci", "off"]
    v.customize ["modifyvm", :id, "--usbcardreader", "off"] 

    # the settings below allow for faster network over the natted interface 
    v.customize ["modifyvm", :id, "--chipset", "ich9"]
    v.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    v.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
  end
  
  config.vm.define "exeggutor-box" do |b|
    b.vm.network :private_network, ip: "192.168.11.253"
  end

  config.vm.provision "ansible" do |ansible|
    ansible.playbook = "./playbook.yml"
    ansible.sudo = true
  end

end

# export DOCKER_HOST=tcp://192.168.11.253:4243
# echo '192.168.11.253 exeggutor-box' | sudo tee -a /etc/hosts
