MARKDOWN = pandoc -f markdown+tex_math_dollars

all: build/spec.html
clean:
	rm -rf build/ spec.pdf

build/spec.html: build/.dir spec.md
	$(MARKDOWN) -t html --mathml < $^ > $@
build/.dir:
	mkdir -p build
	touch build/.dir

spec.pdf: spec.md
	$(MARKDOWN) -t pdf < $^ > $@

.PHONY: all clean
