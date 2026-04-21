
FROM golang:latest

WORKDIR /app

COPY api/. .

RUN go mod tidy
RUN go build -o main

EXPOSE 8080

CMD ["./main"]