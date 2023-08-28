FROM golang:1.20-alpine

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
#COPY src/go.mod src/go.sum ./
#RUN go mod download && go mod verify

COPY src .
RUN mkdir -p /usr/local/bin/
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/app ./cmd/server
RUN chmod +x /usr/local/bin/app

CMD ["app"]
