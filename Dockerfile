FROM golang

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /workspace

COPY go.* ./

RUN go mod download

COPY *.go ./

COPY ./pkg/ ./pkg/

RUN CGO_ENABLED=0 go build -ldflags '-s' -o /bin/comments

FROM scratch

COPY --from=0 /etc/passwd /etc/passwd

COPY --from=0 /bin/comments /bin/comments

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER nobody

CMD ["/bin/comments"]
