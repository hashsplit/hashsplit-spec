all: build/rrs.html
clean:
	rm -rf build/

build/rrs.html: build/.dir spec/hashes/rrs.md
	pandoc -f markdown+tex_math_dollars -t html --mathml < $^ > $@
build/.dir:
	mkdir -p build
	touch build/.dir

.PHONY: all clean
