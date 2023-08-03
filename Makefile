current_dir = $(shell pwd)

release:
	docker run -it --rm -v $(current_dir):/go/src/github.com/demetrio108/rmqdump -w /go/src/github.com/demetrio108/rmqdump -e CGO_ENABLED=0 golang:1-buster go build -buildvcs=false -ldflags "-s -w"

rpm:
	rm -rf .build
	mkdir -p .build/usr/bin
	cp rmqdump .build/usr/bin/
	docker run -it --rm -v $(current_dir):/rmqdump -w /rmqdump lrdevops/fpm fpm -s dir -t rpm -C .build --name rmqdump --version 0.0.3 --iteration 1

deb:
	rm -rf .build
	mkdir -p .build/usr/bin
	cp rmqdump .build/usr/bin/
	docker run -it --rm -v $(current_dir):/rmqdump -w /rmqdump lrdevops/fpm fpm -s dir -t deb -C .build --name rmqdump --version 0.0.3 --iteration 1
