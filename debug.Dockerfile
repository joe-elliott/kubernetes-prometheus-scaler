FROM golang

ENV GOPATH=/opt/go:$GOPATH \
    PATH=/opt/go/bin:$PATH

# snag delve
RUN go get github.com/derekparker/delve/cmd/dlv

# copy code in
ADD . /opt/go/src/local/myorg/myapp
WORKDIR /opt/go/src/local/myorg/myapp 

# build and setup container.  technically these lines aren't necessary b/c delve will build the app itself, but /shrug
RUN go build -o main main.go
CMD ["./main"]