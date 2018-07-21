FROM golang:1.10.3

WORKDIR /go/src/info
COPY . .

RUN go get -d -v ./...
RUN go build -v ./...

CMD ["./info"]
