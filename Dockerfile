FROM golang:1.8.3 as build
WORKDIR /go/src/kubernetes-prometheus-scaler

# install glide
RUN curl https://glide.sh/get | sh

# copy in code, resolve dependencies and build
COPY . .
RUN    glide up -v \
    && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest  
WORKDIR /root/
COPY --from=build /go/src/kubernetes-prometheus-scaler/app .
CMD ["./app"] 