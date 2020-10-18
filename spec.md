# Introduction

This specification describes a mechanism
for splitting a byte stream into blocks of varying size
with split boundaries based solely on the content of the input.
It also describes a mechanism for organizing those blocks into a (probabilistically) balanced tree
whose shape is likewise determined solely by the content of the input.

The general technique has been used by various systems such as:

- [Perkeep](https://perkeep.org)
- [Bup](https://bup.github.io/)
- [RSync](https://rsync.samba.org/)
- [Low-Bandwidth Network Filesystem (LBFS)](https://pdos.csail.mit.edu/papers/lbfs:sosp01/lbfs.pdf)
- [Syncthing](https://syncthing.net/)
- [Kopia](https://github.com/kopia/kopia)

...and many others.
The technique permits the efficient representation of slightly different versions of the same data
(e.g. successive revisions of a file in a version control system),
since changes in one part of the input generally do not affect the boundaries of any but the adjacent blocks.

However, the exact functions used by these
systems differ in details, and thus do not produce identical splits,
making interoperability for some use cases more difficult than it
should be.

The role of this specification is therefore to fully and formally
describe a concrete function on which future systems may standardize,
improving interoperability.

# Notation

This section discusses notation used in this specification.

We define the following sets:

- $U_{32}$, The set of integers in the range $[0, 2^{32})$
- $U_8$, The set of integers in the range $[0, 2^8)$, aka bytes.
- $V_8$, The set of *sequences* of bytes, i.e. sequences of
  $U_8$.
- $V_v$, The set of *sequences* of *sequences* of bytes, i.e.
  sequences of elements of $V_8$.

All arithmetic operations in this document are implicitly performed
modulo $2^{32}$. We use standard mathematical notation for addition,
subtraction, multiplication, and exponentiation. Division always
denotes integer division, i.e. any remainder is dropped.

We use the notation $\langle X_0, X_1, \dots, X_k \rangle$ to denote
an ordered sequence of values.

$|X|$ denotes the length of the sequence $X$, i.e. the number of
elements it contains.

We also use the following operators and functions:

- $x \wedge y$ denotes the bitwise AND of $x$ and $y$
- $x \vee y$ denotes the bitwise OR of $x$ and $y$
- $x \ll n$ denotes shifting $x$ to the left $n$ bits, i.e.
  $x \ll n = x2^{n}$
- $x \gg n$ denotes a *logical* right shift -- it shifts $x$ to the
  right by $n$ bits, i.e. $x \gg n = x / 2^n$
- $X \mathbin{\|} Y$ denotes the concatenation of two sequences $X$ and $Y$,
  i.e. if $X = \langle X_0, \dots, X_N \rangle$ and $Y = \langle Y_0,
  \dots, Y_M \rangle$ then $X \mathbin{\|} Y = \langle X_0, \dots, X_N, Y_0, \dots, Y_M
  \rangle$
- $\min(x, y)$ denotes the minimum of $x$ and $y$ and $\max(x, y)$ denotes the maximum
- $\operatorname{Type}(x)$ denotes the type of $x$.

Finally, we define the “prefix” $\mathbb{P}_q(X)$
of a non-empty sequence $X$
with respect to a given predicate $q$
to be the initial subsequence $X^\prime$ of $X$
up to and including the first member that makes $q(X^\prime)$ true.
And we define the “remainder” $\mathbb{R}_q(X)$
to be everything left after removing the prefix.

Formally,
given a sequence $X = \langle X_0, \dots, X_{|X|-1} \rangle$
and a predicate $q \in \operatorname{Type}(X) \rightarrow \{\text{true},\text{false}\}$,

$\mathbb{P}_q(X) = \langle X_0, \dots, X_e \rangle$

for the smallest integer $e$ such that:

- $0 \le e < |X|$ and
- $q(\langle X_0, \dots, X_e \rangle) = \text{true}$

or $|X|-1$ if no such integer exists.
(I.e., if nothing satisfies $q$, the prefix is the whole sequence.)
And:

$\mathbb{R}_q(X) = \langle X_b, \dots, X_{|X|-1} \rangle$

where $b = |\mathbb{P}_q(\langle X_0, \dots, X_{|X|-1} \rangle)|$.

Note that when $\mathbb{P}_q(X) = X$, $\mathbb{R}_q(X) = \langle \rangle$.

# Splitting

The primary result of this specification is to define a family of
functions:

$\operatorname{SPLIT}_C \in V_8 \rightarrow V_v$

...which is parameterized by a _configuration_ $C$, consisting of:

- $S_{\text{min}} \in U_{32}$, the minimum split size
- $S_{\text{max}} \in U_{32}$, the maximum split size
- $H \in V_8 \rightarrow U_{32}$, the hash function
- $T \in U_{32}$, the threshold

The configuration must satisfy $S_{\text{max}} \ge S_{\text{min}} > 0$.

<!---
NOTE

It might help the clarity of what follows
if we make parameterization by C implicit
rather than repeating C in subscripts everywhere.
-->

## Definitions

We define the constant $W$, which we call the "window size," to be 64.

<!---
NOTE

Fixing W at 64
arbitrarily rules out using this section
to describe and reason about hashsplit algorithms
that it otherwise could.

I think fixing it at 64 is more properly a part of the recommendation section.
-->

We define the predicate $q_C(X)$
on a non-empty byte sequence $X$
with respect to a configuration $C$
to be:

- $\text{true}$ if $|X| = S_{\text{max}}$; otherwise
- $\text{true}$ if $|X| \ge S_{\text{min}}$ and $H(\langle X_{\max(0,|X|-W)}, \dots, X_{|X|-1} \rangle) \mod 2^T = 0$
  (i.e., the last $W$ bytes of $X$ hash to a value with at least $T$ trailing zeroes);
  otherwise
- $\text{false}$.

<!---
NOTE

This previously used H(<X_(|X|-W) ... X_(|X|-1)>) and defined X_i = 0 for i<0.
However, this unnecessarily constrains the choice of hash function.
If the hash function wants to treat input shorter than W as being prefixed by zeroes,
it can specify that;
but if it wants to handle input shorter than W differently,
it should be allowed to do that too.
-->

We define $\operatorname{SPLIT}_C(X)$ recursively, as follows:

- If $|X| = 0$, $\operatorname{SPLIT}_C(X) = \langle \rangle$
- Otherwise, $\operatorname{SPLIT}_C(X) = \langle \mathbb{P}_{q_C}(X) \rangle \mathbin{\|} \operatorname{SPLIT}_C(\mathbb{R}_{q_C}(X))$

# Tree Construction

If sequence $X$ and sequence $Y$ are largely the same,
$\operatorname{SPLIT}_C$ will produce mostly the same chunks,
choosing the same locations for chunk boundaries
except in the vicinity of whatever differences there are
between $X$ and $Y$.

This has obvious benefits for storage and bandwidth,
as the same chunks can represent both $X$ and $Y$ with few exceptions.
But while only a small number of chunks may change,
the _sequence_ of chunks may get totally rewritten,
as when a difference exists near the beginning of $X$ and $Y$
and all subsequent chunks have to “shift position” to the left or right.
Representing the two different sequences may therefore require space
that is linear in the size of $X$ and $Y$.

We can do better,
requiring space that is only _logarithmic_ in the size of $X$ and $Y$,
by organizing the chunks in a tree whose shape,
like the chunk boundaries themselves,
is determined by the content of the input.
The trees representing two slightly different versions of the same input
will differ only in the subtrees in the vicinity of the differences.

## Definitions

A “chunk” is a member of the sequence produced by $\operatorname{SPLIT}_C$.

The “hashval” $V_C(X)$ of a byte sequence $X$ is:

$H(\langle X_{\max(0, |X|-W)}, \dots, X_{|X|-1} \rangle)$

(i.e., the hash of the last $W$ bytes of $X$).

A “node” $N_{h,i}$ in a hashsplit tree
at non-negative “height” $h$
is a sequence of children.
The children of a node at height 0 are chunks.
The children of a node at height $h+1$ are nodes at height $h$.

A “tier” of a hashsplit tree is a sequence of nodes
$N_h = \langle N_{h,0}, \dots, N_{h,k} \rangle$
at a given height $h$.

The function $\operatorname{Rightmost}(N_{h,i})$
on a node $N_{h,i} = \langle S_0, \dots, S_e \rangle$
produces the “rightmost leaf chunk”
defined recursively as follows:

- If $h = 0$, $\operatorname{Rightmost}(N_{h,i}) = S_e$
- If $h > 0$, $\operatorname{Rightmost}(N_{h,i}) = \operatorname{Rightmost}(S_e)$

The “level” $L_C(X)$ of a given chunk $X$
is $\max(0, Q - T)$,
where $Q$ is the largest integer such that

- $Q \le 32$ and
- $V_C(\mathbb{P}_{q_C}(X)) \mod 2^Q = 0$

(i.e., the level is the number of trailing zeroes in the hashval
in excess of the threshold needed
to produce the prefix chunk $\mathbb{P}_{q_C}(X)$).

The level $L_C(N)$ of a given _node_ $N$
is the level of its rightmost leaf chunk:
$L_C(N) = L_C(\operatorname{Rightmost}(N))$

The predicate $z_{C,h}(K)$
on a sequence $K = \langle K_0, \dots, K_e \rangle$
of chunks or of nodes
with respect to a height $h$
is defined as:

- $\text{true}$ if $L(K_e) > h$; otherwise
- $\text{false}$.

<!---
NOTE

Still needed:
a way to specify
the minimum and maximum branching factor
(akin to S_min and S_max for SPLIT_C).
-->

For conciseness, define

- $P_C(X) = \mathbb{P}_{z_{C,0}}(\operatorname{SPLIT}_C(X))$ and
- $R_C(X) = \mathbb{R}_{z_{C,0}}(\operatorname{SPLIT}_C(X))$

## Algorithm

This section contains two descriptions of hashsplit trees:
an algebraic description for formal reasoning,
and a procedural description for practical construction.

### Algebraic description

The tier $N_0$
of hashsplit tree nodes
for a given byte sequence $X$
is equal to

$\langle P_C(X) \rangle \mathbb{\|} R_C(X)$

The tier $N_{h+1}$
of hashsplit tree nodes
for a given byte sequence $X$
is equal to

$\langle \mathbb{P}_{z_{C,h+1}}(N_h) \rangle \mathbb{\|} \mathbb{R}_{z_{C,h+1}}(N_h)$

(I.e., each node in the tree has as its children
a sequence of chunks or lower-tier nodes,
as appropriate,
up to and including the first one
whose “level” is greater than the node’s height.)

The root of the hashsplit tree is $N_{h^\prime,0}$
for the smallest value of $h^\prime$
such that $|N_{h^\prime}| = 1$

### Procedural description

For this description we use $N_h$ to denote a single node at height $h$.
The algorithm must keep track of the “rightmost” such node for each tier in the tree.

To compute a hashsplit tree from a byte sequence $X$,
compute its “root node” as follows.

1. Let $N_0$ be $\langle\rangle$ (i.e., a node at height 0 with no children).
2. If $|X| = 0$, then:
    a. Let $h$ be the largest height such that $N_h$ exists.
    b. If $|N_0| > 0$, then:
        i. For each integer $i$ in $[0 .. h]$, “close” $N_i$ (see below).
        ii. Set $h \leftarrow h+1$.
    c. [pruning] While $h > 0$ and $|N_h| = 1$, set $h \leftarrow h-1$ (i.e., traverse from the prospective tree root downward until there is a node with more than one child).
    d. **Terminate** with $N_h$ as the root node.
3. Otherwise, set $N_0 \leftarrow N_0 \mathbin{\|} \langle P_C(X) \rangle$ (i.e., add $P_C(X)$ to the list of children in $N_0$).
4. For each integer $i$ in $[0 .. L_C(X))$, “close” the node $N_i$ (see below).
5. Set $X \leftarrow R_C(X)$.
6. Go to step 2.

To “close” a node $N_i$:

1. If no $N_{i+1}$ exists yet, let $N_{i+1}$ be $\langle\rangle$ (i.e., a node at height ${i + 1}$ with no children).
2. Set $N_{i+1} \leftarrow N_{i+1} \mathbin{\|} \langle N_i \rangle$ (i.e., add $N_i$ as a child to $N_{i+1}$).
3. Let $N_i$ be $\langle\rangle$ (i.e., new node at height $i$ with no children).

# Rolling Hash Functions

## The RRS Rolling Checksums

The `rrs` family of checksums is based on an algorithm first used
in [rsync][rsync], and later adapted for use in [bup][bup] and
[perkeep][perkeep]. `rrs` was originally inspired by the adler-32
checksum. The name `rrs` was chosen for this specification, and stands
for `rsync rolling sum`.

### Definition

A concrete `rrs` checksum is defined by the parameters:

- $M$, the modulus
- $c$, the character offset

Given a sequence of bytes $\langle X_0, X_1, \dots, X_N \rangle$ and a
choice of $M$ and $c$, the `rrs` hash of the sub-sequence $\langle X_k,
\dots, X_l \rangle$ is $s(k, l)$, where:

$a(k, l) = (\sum_{i = k}^{l} (X_i + c)) \mod M$

$b(k, l) = (\sum_{i = k}^{l} (l - i + 1)(X_i + c)) \mod M$

$s(k, l) = b(k, l) + 2^{16}a(k, l)$

#### RRS1

The concrete hash called `rrs1` uses the values:

- $M = 2^{16}$
- $c = 31$

`rrs1` is used by both Bup and Perkeep, and implemented by the Go
package `go4.org/rollsum`.

### Implementation

#### Rolling

`rrs` is a family of _rolling_ hashes. We can compute hashes in a
rolling fashion by taking advantage of the fact that, for $l \geq k \geq 0$:

$a(k + 1, l + 1) = (a(k, l) - (X_k + c) + (X_{l+1} + c)) \mod M$

$b(k + 1, l + 1) = (b(k, l) - (l - k + 1)(X_k + c) + a(k + 1, l + 1)) \mod M$

So, a typical implementation will work like this:

- Keep $\langle X_k, \dots, X_l \rangle$ in a ring buffer.
- Also store $a(k, l)$ and $b(k, l)$.
- When $X_{l+1}$ is added to the hash:
  - Dequeue $X_k$ from the ring buffer, and enqueue $X_{l+1}$.
  - Use $X_k$, $X_{l+1}$, and the stored $a(k, l)$ and $b(k, l)$ to compute
    $a(k + 1, l + 1)$ and $b(k + 1, l + 1)$. Then use those values to
    compute $s(k + 1, l + 1)$ and also store them for future use.

In all cases the ring buffer should initially contain all zero bytes, reflecting
the use of $X_i = 0$ for $i < 0$ in ["Splitting"](#splitting), above.

#### Choice of M

Choosing $M = 2^{16}$ has the advantages of simplicity and efficiency,
as it allows $s(k, l)$ to be computed using only shifts and bitwise
operators:

$s(k, l) = b(k, l) \vee (a(k, l) \ll 16)$

[rsync]: https://rsync.samba.org/tech_report/node3.html
[bup]: https://bup.github.io/
[perkeep]: https://perkeep.org/
