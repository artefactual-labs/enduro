# GoReleaser supplies the compiled binary for each target platform.
# Keep the runtime base aligned with depbot's static, nonroot image.
FROM cgr.dev/chainguard/static:latest@sha256:8658e9eaeb2294fa0e49b0667f7d66b4e21fa7690b8d2a4c30cd1d013ceb9eb7
ARG TARGETPLATFORM
COPY --chmod=0555 ${TARGETPLATFORM}/enduro /usr/local/bin/enduro
COPY LICENSE /usr/share/licenses/enduro/LICENSE
USER 65532:65532
ENV HOME=/home/nonroot
WORKDIR /home/nonroot
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/enduro"]
CMD ["--config", "/etc/enduro/enduro.toml"]
