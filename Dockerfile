FROM golang:latest

ARG USER="airvisual"
ARG PASS="nopass"
ARG HOST="127.0.0.1"

ENV USER ${USER}
ENV PASS ${PASS}
ENV HOST ${HOST}

WORKDIR airqmetrics

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /airqmetrics

EXPOSE 1280

CMD exec /airqmetrics --user ${USER} --pass ${PASS} --host ${HOST} --listen :1280
