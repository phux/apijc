restart-docker:
	docker-compose stop
	docker-compose up -d

run-example:
	go run main.go \
		--urlFile .testdata/urlfile_example.json \
		--baseDomain "http://localhost:8080" \
		--newDomain "http://localhost:8081" \
		--rateLimit 1000 \

full-example: restart-docker
full-example:
	sleep 2 # wiremock servers take about 2 seconds to boot
	make run-example

cov:
	go test -coverprofile coverage.out ./...

release-local:
	goreleaser release --snapshot --clean

test-linux-release:
	./dist/apijc_linux_amd64_v1/apijc \
		--urlFile .testdata/urlfile_example.json \
		--baseDomain "http://localhost:8080" \
		--newDomain "http://localhost:8081" \
		--rateLimit 1000 \
