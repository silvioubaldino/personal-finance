FROM golang:1.19-alpine

RUN mkdir /app
WORKDIR /app
COPY . /app
WORKDIR /app/cmd/api
RUN go build -o main .
COPY db/migrations /app/db/migrations
CMD ["/app/cmd/api/main"]