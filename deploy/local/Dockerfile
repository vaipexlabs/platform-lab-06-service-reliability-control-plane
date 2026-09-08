FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS build

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
