FROM golang:1.23 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/room ./cmd/room

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/room /room
ENTRYPOINT ["/room"]
