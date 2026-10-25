.PHONY: build test race integration cover vet tidy fuzz check

build:
	go build ./...

test:
	go run ./test/run.go -count=1 ./...

race:
	go run ./test/run.go -race -count=1 ./...

integration:
	WS_INTEGRATION=1 go run ./test/run.go -race -count=1 -run Integration -v ./bootstrap ./internal/gen

cover:
	go run ./test/run.go -race -count=1 -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go run ./test/run.go vet ./...

tidy:
	go run ./test/run.go tidy

# 每个入口独立 fuzz；可用 make fuzz FUZZTIME=45s 调整时间。
FUZZTIME ?= 45s
FUZZFLAGS ?= -fuzzminimizetime=1s
fuzz:
	go run ./test/run.go ./internal/gen -run='^$$' -fuzz='^FuzzParseSQL$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)
	go run ./test/run.go ./internal/gen -run='^$$' -fuzz='^FuzzNewName$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)
	go run ./test/run.go ./app/middlewares -run='^$$' -fuzz='^FuzzRedactJSON$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)
	go run ./test/run.go ./app/middlewares -run='^$$' -fuzz='^FuzzValidRequestID$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)
	go run ./test/run.go ./pkg/httpx -run='^$$' -fuzz='^FuzzHandleBind$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)
	go run ./test/run.go ./utils -run='^$$' -fuzz='^FuzzStringHelpers$$' -fuzztime=$(FUZZTIME) $(FUZZFLAGS)

check: build vet race
