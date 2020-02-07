FROM python:alpine as update_db
WORKDIR /usr/src/app
COPY chinaip .
RUN pip3 install -r requirements.txt
RUN python3 update_db.py

FROM golang:alpine as builder
WORKDIR /go/src/github.com/tuna/freedns-go
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=update_db /usr/src/app/db.go chinaip/
RUN go build -o ./build/freedns-go


FROM alpine
COPY --from=builder /go/src/github.com/tuna/freedns-go/build/freedns-go ./
ENTRYPOINT ["./freedns-go"]
CMD ["-f", "114.114.114.114:53", "-c", "8.8.8.8:53", "-l", "0.0.0.0:53"]
