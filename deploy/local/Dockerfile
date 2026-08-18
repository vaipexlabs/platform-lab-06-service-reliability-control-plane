FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS build

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
