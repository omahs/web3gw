FROM docker.io/golang:alpine as builder


WORKDIR /src

ADD . /src

RUN apk add build-base

RUN cd server &&\
    GOOS=linux CGO_ENABLED=1 go build -o webproxy &&\
    chmod +x webproxy

FROM alpine:3.14

ARG SFTPGOBRANCH

COPY --from=builder /src/server/webproxy /usr/bin/webproxy

COPY --from=builder /src/sftpgo.json /var/lib/sftpgo/sftpgo.json

WORKDIR /var/lib/sftpgo

RUN apk add git

# specify branch
RUN git clone https://github.com/alichaddad/sftpgo.git -b ${SFTPGOBRANCH} /sftpgo

EXPOSE 8080 8060


ENTRYPOINT  ["/usr/bin/webproxy", "-sftp-config-dir", "."]
