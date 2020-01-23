FROM golang:1.13

WORKDIR /go/src/humanitec.io/deploymentset-svc

# Ideally we would only include ./cmd and ./pkg but the Dockerfile does not allow for directories to be coppied in one gos
COPY . .

RUN go build -o /bin/depsets humanitec.io/deploymentset-svc/cmd/depsets

ENTRYPOINT ["/bin/depsets"]
