FROM golang:1.26.4 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /asterism ./cmd/asterism

FROM gcr.io/distroless/static-debian12

COPY --from=build /asterism /asterism

ENTRYPOINT ["/asterism"]
