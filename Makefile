all:
	CGO_ENABLED=0 go build
	docker build -t thockin/iamyouare .

push: all
	docker push thockin/iamyouare

