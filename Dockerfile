# GoReleaser supplies the compiled binary for each target platform.
# Keep the runtime base aligned with depbot's static, nonroot image.
FROM cgr.dev/chainguard/static:latest@sha256:c12f0107f8afb0bbc1b0eff96ff8675e85a10c6083d039b777c51fef5c27420f
ARG TARGETPLATFORM
COPY --chmod=0555 ${TARGETPLATFORM}/enduro /usr/local/bin/enduro
COPY LICENSE /usr/share/licenses/enduro/LICENSE
USER 65532:65532
ENV HOME=/home/nonroot
WORKDIR /home/nonroot
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/enduro"]
CMD ["--config", "/etc/enduro/enduro.toml"]
