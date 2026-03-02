#!/bin/bash


go mod tidy
go build -o srv ./server 
sudo mv srv /usr/local/bin/go-deployer
sudo systemctl restart go-deployer.service
