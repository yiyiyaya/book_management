FROM golang:1.6.2

EXPOSE 8000

ENV TIME_ZONE=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone

COPY . /go/src/github.com/yiyiyaya/book_management

WORKDIR /go/src/github.com/yiyiyaya/book_management

RUN go build

CMD ["sh", "-c", "./book_management -port=8000"]
