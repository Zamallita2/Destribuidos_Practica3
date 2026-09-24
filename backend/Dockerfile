FROM golang:alpine

WORKDIR /app

RUN apk add --no-cache gcc musl-dev tzdata openssl

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN go build -o airres-api .

EXPOSE 8080

CMD ["./airres-api"]
