FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /oss-contributions-ledger .

FROM gcr.io/distroless/static-debian12
COPY --from=build /oss-contributions-ledger /oss-contributions-ledger
ENTRYPOINT ["/oss-contributions-ledger"]
