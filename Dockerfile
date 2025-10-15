# 构建阶段
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o fxratebot .

# 运行阶段
FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /app/fxratebot .
CMD ["./fxratebot"]