#!/bin/bash

# if minikube isn't running, start it
minikube ip
if [ $? -ne 0 ]; then
  minikube start
fi

# sets the docker env to point at minikube so the images we build will be available in minikube
eval $(minikube docker-env)

# build the debug image
docker build . -t go-app:debug -f debug.Dockerfile

# delete old stuff
kubectl delete po go-debug
kubectl delete svc go-debug-svc

#wait for stuff to be deleted
while kubectl get po go-debug > /dev/null; do :; done
while kubectl get svc go-debug-svc > /dev/null; do :; done

# make the new stuff
kubectl create -f ./debug.podspec.yml

# display to the user the endpoints to put in launch.json
minikube service go-debug-svc --url