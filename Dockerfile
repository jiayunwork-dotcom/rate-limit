FROM golang:1.21-alpine
ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/bin/server .
EXPOSE 8080
CMD ["/app/bin/server", "httpd"]
