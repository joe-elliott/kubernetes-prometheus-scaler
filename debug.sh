#!/bin/bash

eval $(minikube docker-env)

docker build . -t go-app:debug -f debug.Dockerfile

kubectl replace -f ./debug.podspec.yml --force

minikube service go-debug-svc --url