FROM golang:alpine AS gateway

WORKDIR /gateway

COPY . .

ADD go.mod .

ADD go.sum .

RUN go mod download

RUN go build -o gateway ./cmd/main

FROM alpine

WORKDIR /build

COPY --from=gateway /build/gateway /build/gateway

CMD [". /gateway"]
