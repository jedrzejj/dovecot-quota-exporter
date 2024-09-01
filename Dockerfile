FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/dovecot-quota-exporter


FROM scratch
COPY --from=0 /bin/dovecot-quota-exporter /bin/dovecot-quota-exporter
EXPOSE 9901

ENTRYPOINT ["/bin/dovecot-quota-exporter"]

