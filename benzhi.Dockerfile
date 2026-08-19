FROM golang:1.21

ENV GOTOOLCHAIN=local
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download || true

COPY . .

RUN go build ./...

CMD ["bash"]
