# Introduction

# Notation

This section discusses notation used in this specification.

All operations in this document are implicitly performed modulo
$2^{32}$. We use standard mathematical notation for addition,
subtraction, multiplication, and exponentiation. Division always
denotes integer division, i.e. any remainder is dropped.

We also use the following operators:

- $x \wedge y$ denotes the bitwise AND of $x$ and $y$
- $x \vee y$ denotes the bitwise OR of $x$ and $y$
- $x \ll n$ denotes shifting $x$ to the left $n$ bits, i.e.
  $x \ll n = x2^{n}$
- $x \gg n$ denotes a *logical* right shift -- it shifts $x$ to the
  right by $n$ bits, i.e. $x \gg n = \frac{x}{2^n}$


# The Splitting Algorithm

The splitting algorithm is parametrized over:

- $S_{min}$, the minimum split size
- $S_{max}$, the maximum split size
- $H$, the hash function
- $W$, the window size
- $m$, the split mask
- $v$, the split value

The parameters must satisfy:

- $S_{max} \ge S_{min}$
- $S_{min} \ge W$

## Definitions

For a given sequence of bytes $X_0, X_1, ..., X_N$, we define:

The "preliminary split index" $I_p(X)$ is the smallest non-negative
integer $i$ satisfying both:

- $i \ge S_{min}$
- $H(X_{i-W+1}, ..., X_i) \wedge m = v$

The "split index" $I(X)$ is:

- $I_p(X)$ if $I_p(X) < S_{max}$
- $S_{max}$ otherwise

# Rolling Hash Functions

## The RRS Rolling Checksums

The `rrs` family of checksums is based on a rolling first used
in [rsync][rsync], and later adapted for use in [bup][bup] and
[perkeep][perkeep]. `rrs` was originally inspired by the adler-32
checksum. The name `rrs` was chosen for this specification, and stands
for `rsync rolling sum`.

### Definition

A concrete `rrs` checksum is defined by the parameters:

- $M$, the modulus
- $C$, the character offset

Given a sequence of bytes $X_0, X_1, ..., X_N$ and a choice of $M$ and
$C$, the `rrs` hash of the sub-sequence $X_k, ..., X_l$ is $s(k, l)$,
where:

$a(k, l) = (\sum_{i = k}^{l} (X_i + C)) \mod M$

$b(k, l) = (\sum_{i = k}^{l} (l - i + 1)(X_i + C)) \mod  M$

$s(k, l) = b(k, l) + 2^{16}a(k, l)$

#### RRS1

The concrete hash called `rrs1` uses the values:

- $M = 2^{16}$
- $C = 31$

`rrs1` is used by both Bup and Perkeep, and implemented by the go
package `go4.org/rollsum`.

### Implementation

#### Rolling

`rrs` is a family of _rolling_ hashes. We can compute hashes in a
rolling fashion by taking advantage of the fact that:

$a(k + 1, l + 1) = (a(k, l) - (X_k + C) + (X_{l+1} + C)) \mod M$

$b(k + 1, l + 1) = (b(k, l) - (l - k + 1)(X_k + C) + a(k + 1, l + 1)) \mod M$

So, a typical implementation will work like:

- Keep $X_k, ..., X_l$ in a ring buffer.
- Also store $a(k, l)$ and $b(k, l)$.
- When $X_{l+1}$ is added to the hash:
  - Dequeue $X_k$ from the ring buffer, and enqueue $X_{l+1}$.
  - Use $X_k$, $X_{l+1}$, and the stored $a(k, l)$ and $b(k, l)$ to compute
    $a(k + 1, l + 1)$ and $b(k + 1, l + 1)$. Then use those values to
    compute $s(k + 1, l + 1)$ and also store them for future use.

#### Choice of M

Choosing $M = 2^{16}$ has the advantages of simplicity and efficiency,
as it allows $s(k, l)$ to be computed using only shifts and bitwise
operators:

$s(k, l) = b(k, l) \vee (a(k, l) \ll 16)$

[rsync]: https://rsync.samba.org/tech_report/node3.html
[bup]: https://bup.github.io/
[perkeep]: https://perkeep.org/
