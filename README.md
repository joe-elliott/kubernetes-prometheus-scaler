# go-minikube-vscode-dev

This repo is a template to get started writing golang applications designed to run in Kubernetes.  It requires that minikube (https://github.com/kubernetes/minikube) is already installed.

### How to use 

1. Type a bunch of go code
2. Make sure that the "Remote Minikube" launch configuration is selected in vscode
4. Set a breakpoint or whatever
5. F5

### How it works

- The "Remote Minikube" launch configuration has `deployToMinikube` setup as a prelaunch task
- `deployToMinikube` runs `./debug.sh`
- `./debug.sh` builds a container with `debug.Dockerfile` 
  - `./dockerignore` ignores everything not under the `./src` directory
- `./debug.sh` uses `./debug.podspec.yml` to install your application/service into minikube.  Adjust as necessary to expose other ports.  Delve uses 2345 so leave that one alone.
- The last line of `./debug.sh` prints out the exposed endpoints on your container.  Currently `launch.json` is hardcoded to try to connect to the debugger at 192.168.99.100.  This will need to be adjusted if the service is exposed at a different IP.

### Issues/Improvements

- Print application output into vscode debug console
- Occassionally the pod/container fail to terminate. 
