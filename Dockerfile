FROM golang:1.13 AS build

WORKDIR /go/src/humanitec.io/deploymentset-svc

# Cache the gets here - might cause issues if different dependancies update at different times.
RUN go get github.com/julienschmidt/httprouter \
           github.com/lib/pq

# Ideally we would only include ./cmd and ./pkg but the Dockerfile does not allow for directories to be coppied in one gos
COPY . .

RUN go build -o /bin/depset humanitec.io/deploymentset-svc/cmd/depset

RUN ls

ENTRYPOINT ["/bin/depset"]

# Small Light image
#FROM alpine
#COPY --from=build /go/src/humanitec.io/deploymentset-svc/depset /bin/depset

#ENTRYPOINT ["/bin/depset"]
