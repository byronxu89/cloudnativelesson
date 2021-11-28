FROM golang:alpine as builder
WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 GOOS=linux && go build -o httpserver .

FROM alpine:latest as prod
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app/httpserver .
EXPOSE 8090


CMD ["./httpserver"]