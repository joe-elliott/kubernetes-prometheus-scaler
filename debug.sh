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

echo "The following endpoints are exposed on your pod."
echo " - 30080 is mapped to 8080 in your container and is useful only if your application provides a service on a port.  Adjust the podspec as necessary to expose other ports or hide this one."
echo " - 32345 is necessary for vscode to connect to the dlv debugger.  You may need to adjust the IP in launch.json if the below doesn't match."

minikube service go-debug-svc --url