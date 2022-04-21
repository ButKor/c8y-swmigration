# Build
FROM golang:1.18-alpine As build
WORKDIR /app
ADD ./ ./
RUN apk add git
RUN go build -o swmigration ./pkg

# Deploy
FROM alpine:3.15.0
WORKDIR /app
COPY --from=build /app/swmigration /app/swmigration
ADD ./pkg/templates ./templates/
EXPOSE 8085

RUN apk add bash --no-cache
RUN apk add wget
RUN wget -O /etc/apk/keys/rmiller-rsa-signing.rsa.pub https://reubenmiller.jfrog.io/artifactory/api/security/keypair/public/repositories/c8y-alpine
RUN sh -c "echo 'https://reubenmiller.jfrog.io/artifactory/c8y-alpine/stable/main'" >> /etc/apk/repositories
RUN apk add go-c8y-cli
RUN apk add jsonnet
RUN apk add jq

ENTRYPOINT [ "./swmigration" ]
