all: build/spec.html
clean:
	rm -rf build/

build/spec.html: build/.dir spec.md
	pandoc -f markdown+tex_math_dollars -t html --mathml < $^ > $@
build/.dir:
	mkdir -p build
	touch build/.dir

.PHONY: all clean
