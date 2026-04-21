# worker/Dockerfile

FROM golang:latest

WORKDIR /app

COPY worker/. .

RUN go mod tidy
RUN go build -o main .

CMD ["./main"]