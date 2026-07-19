# ---- 构建阶段 ----
FROM golang:1.25-alpine AS build
WORKDIR /src
# 先拷贝依赖清单，利用层缓存（仅当 go.mod/go.sum 变化时才重新下载）
COPY backend/go.mod backend/go.sum ./
RUN go mod download
# 再拷贝全部源码
COPY backend/ .
# 纯 Go（modernc.org/sqlite，无 cgo），-a 强制重编以纳入 embed 的前端资源
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /out/lan-nav .

# ---- 运行阶段 ----
FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/lan-nav /app/lan-nav

# 统一数据目录：数据库与上传的 logo 都落在 /data，便于挂载持久化卷
ENV DATA_DIR=/data
EXPOSE 5725
VOLUME ["/data"]
ENTRYPOINT ["/app/lan-nav"]
