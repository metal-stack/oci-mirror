FROM gcr.io/distroless/static-debian12:nonroot

COPY bin/server /server
ENTRYPOINT [ "/server" ]