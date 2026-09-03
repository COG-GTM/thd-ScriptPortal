from golang:alpine
RUN apk update
RUN apk add git gcc musl-dev bash python3
RUN go get github.com/homedepot/ScriptPortal
WORKDIR /go/src/github.com/homedepot/ScriptPortal
RUN ./build.sh
RUN ./install.sh
RUN addgroup -S scriptportal && adduser -S -G scriptportal scriptportal \
 && chown -R scriptportal:scriptportal /var/lib/ScriptPortal /go/src/github.com/homedepot/ScriptPortal
USER scriptportal
EXPOSE 8080
CMD ["ScriptPortal", "-p", "8080"]
