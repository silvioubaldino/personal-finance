FROM golang:1.19-alpine

RUN mkdir /app
ADD . /app
WORKDIR /app/cmd/api
COPY ../env/personal-finance-dd2e2-firebase-adminsdk-is12m-59e71d7427.json .
RUN go build -o main .
CMD ["/app/cmd/api/main"]