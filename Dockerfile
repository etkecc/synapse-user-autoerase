FROM ghcr.io/etkecc/base/build AS builder

WORKDIR /app
COPY . .
RUN just build

FROM ghcr.io/etkecc/base/app

COPY --from=builder /app/suae /bin/suae

USER app

ENTRYPOINT ["/bin/suae"]

