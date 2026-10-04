FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /casdoor-forward-auth .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /casdoor-forward-auth /casdoor-forward-auth
EXPOSE 9999
ENTRYPOINT ["/casdoor-forward-auth"]
