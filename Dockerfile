FROM registry.gitlab.com/etke.cc/base/build AS builder

WORKDIR /app
COPY . .
RUN just build

FROM registry.gitlab.com/etke.cc/base/app

COPY --from=builder /app/suae /bin/suae

USER app

ENTRYPOINT ["/bin/suae"]

