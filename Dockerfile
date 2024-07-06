FROM golang:1.22

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

RUN apt update -y && apt install -y openssh-server
RUN service ssh start

COPY . .
RUN go build -v -o /usr/local/bin/app ./...

EXPOSE 2323 3000

ENTRYPOINT [ "app" ]