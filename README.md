# go-minikube-vscode-dev

This repo is a template to get started writing golang applications designed to run in Kubernetes.  It requires that minikube (https://github.com/kubernetes/minikube) is already installed.

### How to use 

1. Type a bunch of go code
2. Run `./debug.sh` in the root directory.  
3. Choose the `Remote Docker` launch configuration in vscode
4. Set a breakpoint or whatever
5. F5

### Issues/Improvements

- Figure out how to get debug output printed into the vscode debug console
- Run `debug.sh` on F5
