FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build

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
