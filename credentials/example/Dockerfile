FROM ubuntu as certsource

RUN apt update
RUN apt install -y ca-certificates

FROM scratch
COPY --from=certsource /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ADD main /
CMD ["/main"]