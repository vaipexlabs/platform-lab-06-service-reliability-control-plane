FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/checkout-demo ./cmd/checkout-demo

FROM scratch
USER 65532:65532
COPY --from=build /out/checkout-demo /checkout-demo
EXPOSE 8080
ENTRYPOINT ["/checkout-demo"]
