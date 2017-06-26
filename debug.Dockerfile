FROM golang

ENV GOPATH=/opt/go:$GOPATH \
    PATH=/opt/go/bin:$PATH

# snag delve
RUN go get github.com/derekparker/delve/cmd/dlv

# copy binary in
COPY ./main /usr/local/bin
WORKDIR /usr/local/bin

CMD ["./main"]