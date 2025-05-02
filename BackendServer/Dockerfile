FROM golang:1.24-alpine

WORKDIR /app
COPY go.mod ./

RUN go mod tidy
RUN go mod download
COPY . .

ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -o main cmd/main.go

EXPOSE 9000

CMD ["./main"]