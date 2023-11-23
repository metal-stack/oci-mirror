FROM gcr.io/distroless/static-debian12

COPY bin/server /server
ENTRYPOINT [ "/server" ]