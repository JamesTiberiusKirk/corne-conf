BINDIR ?= $(HOME)/.local/bin

.PHONY: all install uninstall clean

all: cornd cornectl

cornd:
	cd host && go install ./cmd/cornd/

cornectl:
	cd host && go install ./cmd/cornectl/

install: cornd cornectl

uninstall:
	rm -f $(BINDIR)/cornd $(BINDIR)/cornectl

clean:
	rm -f host/cmd/cornd/cornd host/cmd/cornectl/cornectl
