# GoReleaser supplies the compiled binary for each target platform.
# The embedded BagIt validator runs a Python interpreter linked against glibc.
FROM cgr.dev/chainguard/glibc-dynamic:latest@sha256:d553920723f9c0dedef07166a6295d35b9f7173a3c37f5a7b5692b4b5f62a7b3
ARG TARGETPLATFORM
COPY --chmod=0555 ${TARGETPLATFORM}/enduro /usr/local/bin/enduro
COPY LICENSE /usr/share/licenses/enduro/LICENSE
USER 65532:65532
ENV HOME=/home/nonroot
WORKDIR /home/nonroot
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/enduro"]
CMD ["--config", "/etc/enduro/enduro.toml"]
