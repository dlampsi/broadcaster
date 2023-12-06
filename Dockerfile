FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN apk add --no-cache make git && make build 

# --------------------------------------------------
FROM alpine:3.14

COPY --from=builder /app/bin/broadcaster /app/
RUN chmod +x /app/broadcaster

WORKDIR /app

ENTRYPOINT [ "./broadcaster" ]
CMD ["server"]
