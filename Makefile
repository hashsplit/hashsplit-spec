MARKDOWN = pandoc -f markdown+tex_math_dollars

all: build/spec.html build/spec.pdf
clean:
	rm -rf build/

build/spec.html: build/.dir spec.md
	$(MARKDOWN) -t html --mathml < $^ > $@
build/spec.pdf: build/.dir spec.md
	$(MARKDOWN) -t pdf < $^ > $@
build/.dir:
	mkdir -p build
	touch build/.dir

.PHONY: all clean
