FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build

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
