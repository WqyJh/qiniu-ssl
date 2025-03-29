.PHONY: qiniu-ssl
qiniu-ssl:
	go build -o qiniu-ssl cmd/qiniu-ssl/main.go

.PHONY: clean
clean:
	rm -f qiniu-ssl
