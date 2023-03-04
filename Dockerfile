FROM alpine:3
WORKDIR /
RUN apk add gcompat && mkdir -p /etc/zonepop
COPY zonepop /bin/zonepop
CMD ["/bin/zonepop", "-config-file", "/etc/zonepop/config.lua"]
