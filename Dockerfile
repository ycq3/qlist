# 使用 alpine 作为基础镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 复制二进制文件
COPY qlist-linux-* ./qlist

# 设置可执行权限
RUN chmod +x /app/qlist

# 暴露端口
EXPOSE 8080

# 启动应用
CMD ["/app/qlist"]