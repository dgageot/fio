FROM golang as build
WORKDIR /app
COPY main.go go.mod go.sum vendor ./
ARG GOOS=darwin
ARG GOARCH=arm64
RUN GOOS=${GOOS} GOARCH=${GOARCH} go build

FROM scratch AS output
COPY --from=build /app/fio /fio