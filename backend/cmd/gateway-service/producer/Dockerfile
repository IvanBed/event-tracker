FROM golang:alpine AS builder

WORKDIR /build

COPY . .

ADD go.mod .

ADD go.sum .

RUN go mod download

RUN go build -o producer producer.go

FROM alpine

WORKDIR /build

COPY --from=builder /build/producer /build/producer

CMD [". /producer"]
