FROM golang:1.23.3-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o album-service

EXPOSE 3001
EXPOSE 5051  

CMD ["./album-service"]
