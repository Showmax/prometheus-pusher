.PHONY: help
help:
	@echo -e '$(NAME) Makefile targets:'
	@echo -e 'make [build]\t- build $(NAME)'
	@echo -e 'make bench\t- run benchmarks'
	@echo -e 'make clean\t- remove the built binary'
	@echo -e 'make mrproper\t- clean the built binary and vendor sources'
	@echo -e 'make test\t- run unit tests'
	@echo -e 'make vendor\t- prepare govendor environment\n'
	@echo -e '$(NAME) Makefile variables:'
	@echo -e 'NAME\t\t- filename of the built binary (default: $(NAME))'
	@echo -e 'BUILD_DIR\t- directory where to place the binary (default: $(BUILD_DIR))'
	@echo -e 'BENCH_CORES\t- array of cpu cores to run benchmarks on (default: $(BENCH_CORES))'
	@echo -e 'BENCH_TIME\t- duration of each benchmark (default: $(BENCH_TIME))'

define banner
	@echo -e '\n|\n| $1\n|\n'
endef

define finished
	@echo -e '\n||\n|| BUILD SUCCESSFUL\n||\n|| path:\t$(1)'
	@echo -en '|| version:\t$(2)\n|| size:\t'
	@stat -c %s $(1) | awk '{ split( "B kB MB GB", v ); s=1; while( $$1>10240 ){ $$1/=1024; s++ } printf("%0.3f %s", $$1, v[s]) }'
	@echo -en '\n|| shared libs:'
	@ldd $(1) | awk '{ printf("%s\n||\t", $$0)}'
	@echo -e '\n|| ;)\n||'
endef

MAIN_GOALS = .gitignore bin/.gitkeep vendor $(VENDOR_LIBS)
TEST = test/done
VENDOR_LIBS = $(shell govendor list +v,^u +e +m | awk '{ print "vendor/"$$2 }' | sort | uniq)
PROJECT_ALL_FILES = $(wildcard *.go)
PROJECT_BUILD_FILES = $(filter-out $(wildcard *test.go),$(PROJECT_ALL_FILES))
VERSION = $(file <VERSION)

-include config.mk .config.mk .cfg.mk includes.mk .includes.mk

# overrideable variables
GO ?= $(shell which go)
GOVENDOR ?= govendor
NAME ?= $(shell basename $${PWD})
BUILD_DIR ?= bin
BENCH_CORES ?= 1,2,4
BENCH_TIME ?= 5s

.DEFAULT_GOAL = $(BUILD_DIR)/$(NAME)
.PHONY: build
build: $(.DEFAULT_GOAL)
$(BUILD_DIR)/$(NAME): $(PROJECT_BUILD_FILES) | $(MAIN_GOALS) test VERSION
	$(call banner,BUILDING)
	@make -B VERSION
	$(GOVENDOR) build -a -x -ldflags "-X main.version=$(VERSION) $(LDFLAGS)" -o $@ $^
	$(call finished,$@,$(VERSION))

.PHONY: test
test: $(TEST)
$(TEST): $(PROJECT_ALL_FILES) | $(MAIN_GOALS)
	$(call banner,TESTING)
	$(GOVENDOR) vet -v && $(GOVENDOR) fmt && $(GOVENDOR) test -v -cover -race
	@mkdir -p test
	@touch $@

.PHONY: run
run: $(PROJECT_BUILD_FILES) | $(MAIN_GOALS)
	@-$(GO) run -ldflags "-X main.version=$(VERSION) $(LDFLAGS)" $^ $(GO_RUN_ARGS)

.PHONY: bench
bench: $(PROJECT_ALL_FILES) | $(MAIN_GOALS) $(TEST)
	$(call banner,RUNNING BENCHMARKS)
	$(GOVENDOR) test -v -run NoSuchThing -bench . -benchtime $(BENCH_TIME) -cpu $(BENCH_CORES) -benchmem

vendor:
	$(call banner,BOOTSTRAPPING govendor ENVIRONMENT)
	which govendor >/dev/null 2>&1 || $(GO) get -u github.com/kardianos/govendor
	test -e $@/vendor.json || $(GOVENDOR) init

$(VENDOR_LIBS): vendor/%:
	$(GOVENDOR) fetch -v $*

.PHONY: clean
clean:
	$(call banner,CLEANING)
	rm -f $(BUILD_DIR)/$(NAME) $(TEST)

.PHONY: mrproper
mrproper:
	$(call banner,REMOVING govendor ENVIRONEMNT)
	rm -rf $(BUILD_DIR)/$(NAME) vendor $(TEST)

.gitignore:
	$(call banner,UPDATING .gitignore)
	grep -q $(BUILD_DIR)/$(NAME) .gitignore 2>/dev/null || echo $(BUILD_DIR)/$(NAME) >> .gitignore
	grep -q $(TEST) .gitignore 2>/dev/null || echo $(TEST) >> .gitignore

bin/.gitkeep:
	mkdir -p bin
	touch $@

VERSION:
	test -z "$$(git status --porcelain $(PROJECT_ALL_FILES) $(VENDOR_LIBS))" || echo -e '\n|\n| YOU HAVE TO UPDATE THE VERSION AND COMMIT FIRST\n|\n'
