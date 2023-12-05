FROM golang:1.21.4-bookworm

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build

CMD ["./bin/a0feed"]
