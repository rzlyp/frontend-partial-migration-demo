FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/pos-backend .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/pos-backend /usr/local/bin/pos-backend
EXPOSE 8080
CMD ["pos-backend"]
