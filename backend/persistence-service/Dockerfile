FROM golang:alpine AS builder

WORKDIR /build

COPY . .

ADD go.mod .

ADD go.sum .

RUN go mod download

RUN go build -o consumer consumer.go

FROM alpine

WORKDIR /build

COPY --from=builder /build/consumer /build/consumer

CMD [". /consumer"]
