.PHONY: qiniu-ssl
qiniu-ssl:
	go build -ldflags="-s -w" -o qiniu-ssl ./cmd/qiniu-ssl

.PHONY: clean
clean:
	rm -f qiniu-ssl
