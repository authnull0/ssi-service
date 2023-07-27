FROM golang:1.20-alpine
ENV APPHOME=/app
WORKDIR $APPHOME
COPY . ${APPHOME}
RUN go mod download
Run go mod tidy
RUN go build -o main cmd/main.go
EXPOSE 5000
CMD [ "./main" ] 
