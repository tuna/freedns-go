FROM golang

WORKDIR $GOPATH/src/github.com/tuna/freedns-go
COPY . ./

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["freedns-go", "-f", "114.114.114.114:53", "-c", "8.8.8.8:53", "-l", "0.0.0.0:53"]
