FROM golang:1.20-alpine as builder
RUN apk add --no-cache git make curl build-base
ENV GOOS=linux

WORKDIR /src

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./
RUN make test
RUN make check
RUN make alpine

FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
RUN export PATH=$PATH:/app
WORKDIR /app
COPY --from=builder /src/bin/bifrost /app/bifrost
COPY --from=builder /src/templates /app/templates
CMD ["/app/bifrost"]
