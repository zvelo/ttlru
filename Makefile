GO_PKGS=$(shell go list ./...)
ALLDIRS=$(shell find . \( -path ./Godeps -o -path ./.git \) -prune -o -type d -print)
GO_FILES=$(foreach dir, $(ALLDIRS), $(wildcard $(dir)/*.go))

ifeq ("$(CIRCLECI)", "true")
	export GIT_BRANCH = $(CIRCLE_BRANCH)
endif

lint:
	@golint ./...
	@go vet ./...

test: $(GO_FILES)
	go test -v -race ./...

coverage: .acc.out

.acc.out: $(GO_FILES)
	@echo "mode: set" > .acc.out
	@for pkg in $(GO_PKGS); do \
		cmd="go test -v -coverprofile=profile.out $$pkg"; \
		eval $$cmd; \
		if test $$? -ne 0; then \
			exit 1; \
		fi; \
		if test -f profile.out; then \
			cat profile.out | grep -v "mode: set" >> .acc.out; \
		fi; \
	done
	@rm -f ./profile.out

coveralls: .coveralls-stamp

.coveralls-stamp: .acc.out
	@if test -n "$(COVERALLS_REPO_TOKEN)"; then \
		goveralls -v -coverprofile=.acc.out -service circle-ci -repotoken $(COVERALLS_REPO_TOKEN); \
	fi
	@touch .coveralls-stamp

clean:
	@rm -f \
		./.acc.out \
		./.coveralls-stamp
