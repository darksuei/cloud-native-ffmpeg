FROM jrottenberg/ffmpeg:7.1.2-ubuntu2404 AS ffmpeg

FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cloud-native-ffmpeg ./cmd/app

FROM jrottenberg/ffmpeg:7.1.2-ubuntu2404
COPY --from=builder /cloud-native-ffmpeg /usr/local/bin/cloud-native-ffmpeg
ENTRYPOINT ["/usr/local/bin/cloud-native-ffmpeg"]