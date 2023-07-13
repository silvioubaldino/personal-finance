FROM golang:1.19-alpine

RUN mkdir /app
ADD . /app
WORKDIR /app/cmd/api
RUN go build -o main .
CMD ["/app/cmd/api/main"]