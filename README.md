WIP formal spec for a version of the content-based splitting algorithms
used by systems such as perkeep, bup, kopia, asuran, and others.

Right now all that's here is a first pass at specifying one rolling hash
function, which we've named `rrs1`.

# Why

Many systems use some version of this hash-splitting algorithm, but
because they all compute slightly different splits, it's difficult to
write interoperable tools around them.

The goal of this project is to provide a formal spec for a parametrized
version of the algorithm, along with test suites, to facilitate building
interoperable implementations. The idea is, a tool using hash splitting
should be able to just say in its docs which hashsplit function it uses,
just as tools which use cryptographic hashes can just say "sha256" or
the like, and developers working in other languages with other libraries
can easily make use of that information.

# Building

Install [pandoc](https://pandoc.org) and type `make`.

There is also a built PDF version of the WIP spec [here][1]

[1]: https://raw.githubusercontent.com/hashsplit/hashsplit-spec/gh-pages/spec.pdf
