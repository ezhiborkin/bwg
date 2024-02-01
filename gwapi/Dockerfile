FROM golang:latest

RUN go version
ENV GOPATH=/

COPY ./ ./

# make wait-for-it.sh executable
RUN chmod +x wait-for-postgres.sh

# build go app
RUN go mod download
RUN go build -o gwapi ./cmd/gwapi/main.go

CMD ["./gwapi"]