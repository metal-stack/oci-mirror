FROM gcr.io/distroless/static-debian13:nonroot

COPY bin/server /server
ENTRYPOINT [ "/server" ]