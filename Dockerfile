FROM golang:1.13-alpine as build

RUN \
  echo -e "\e[32madd build dependency packages\e[0m" \
  && apk --no-cache add \
    ca-certificates \
    git

WORKDIR /go/src/

COPY atlassian_status_exporter.go go.mod go.sum ./

RUN \
  echo -e "\e[32mBuild the binary\e[0m" \
  && env GOOS=linux GOARCH=386 go build -v

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/atlassian_status_exporter /bin/

EXPOSE 9997

ENTRYPOINT ["/bin/atlassian_status_exporter"]
