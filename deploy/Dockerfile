# Image du serveur Lumenfall : Nakama + notre greffon Go.
# Le greffon doit être compilé avec exactement la même version de Go que
# Nakama : l'image « pluginbuilder » de la même version s'en charge.
ARG NAKAMA_VERSION=3.41.0

FROM heroiclabs/nakama-pluginbuilder:${NAKAMA_VERSION} AS builder
ENV GO111MODULE=on CGO_ENABLED=1
WORKDIR /src
COPY server/ .
RUN go build --trimpath --buildmode=plugin -o /out/lumenfall.so .

FROM heroiclabs/nakama:${NAKAMA_VERSION}
COPY --from=builder /out/lumenfall.so /nakama/data/modules/lumenfall.so
COPY deploy/nakama.yml /nakama/data/lumenfall.yml
