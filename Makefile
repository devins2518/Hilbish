PREFIX ?= /usr
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin
LIBDIR ?= $(PREFIX)/share/hilbish

build:
	@go build

install:
	@install -v -d "$(DESTDIR)$(BINDIR)/" && install -m 0755 -v hilbish "$(DESTDIR)$(BINDIR)/hilbish"
	@mkdir -p "$(DESTDIR)$(LIBDIR)"
	@cp libs preload.lua .hilbishrc.lua "$(DESTDIR)$(LIBDIR)" -r
	@echo "$(DESTDIR)$(BINDIR)/hilbish" >> /etc/shells
	@echo "Hilbish Installed"

uninstall:
	@rm -vrf \
			"$(DESTDIR)$(BINDIR)/hilbish" \
			"$(DESTDIR)$(LIBDIR)"
	@sed -i '/hilbish/d' /etc/shells
	@echo "Hilbish Uninstalled"

clean:
	@go clean

all: build install

.PHONY: install uninstall build clean
