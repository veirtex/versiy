FROM golang:1.25-alpine AS build
WORKDIR /app

RUN apk add --no-cache make

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o versiy ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/versiy .
EXPOSE 3000
CMD ["./versiy"]
