# GoReleaser supplies the compiled binary for each target platform.
# Keep the runtime base aligned with depbot's static, nonroot image.
FROM cgr.dev/chainguard/static:latest@sha256:fc42af33e643f20d08baf03cddcc1c02ba57052dfb02edb0d79f3fc47bcea8d9
ARG TARGETPLATFORM
COPY --chmod=0555 ${TARGETPLATFORM}/enduro /usr/local/bin/enduro
COPY LICENSE /usr/share/licenses/enduro/LICENSE
USER 65532:65532
ENV HOME=/home/nonroot
WORKDIR /home/nonroot
EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/enduro"]
CMD ["--config", "/etc/enduro/enduro.toml"]
