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

- $U_{32}$, The set of integers in the range $[0, 2^{32})$.
- $U_8$, The set of integers in the range $[0, 2^8)$, aka bytes.
- $V_8$, The set of *sequences* of bytes, i.e. sequences of
  $U_8$.
- $V_v$, The set of *sequences* of *sequences* of bytes, i.e.
  sequences of elements of $V_8$.
- $V_{32}$, The set of sequences of elements of $U_{32}$.

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
- $x \vee y$ denotes the bitwise *inclusive* OR of $x$ and $y$
- $x \oplus y$ denotes the bitwise *exclusive* OR of $x$ and $y$
- $x \ll n$ denotes shifting $x$ to the left $n$ bits, i.e.
  $x \ll n = x2^{n}$
- $x \gg n$ denotes a *logical* right shift -- it shifts $x$ to the
  right by $n$ bits, i.e. $x \gg n = x / 2^n$
- $X \mathbin{\|} Y$ denotes the concatenation of two sequences $X$ and
  $Y$,
  i.e. if $X = \langle X_0, \dots, X_N \rangle$ and $Y = \langle Y_0,
  \dots, Y_M \rangle$ then $X \mathbin{\|} Y = \langle X_0, \dots, X_N, Y_0, \dots, Y_M
  \rangle$
- $\operatorname{min}(x, y)$ denotes the minimum of $x$ and $y$.
- $\operatorname{ROT}_L(x, n)$ denotes the rotation of $x$ to the left
  by $n$ bits, i.e. $\operatorname{ROT}_L(x, n) = (x \ll n) \vee (x \gg
  (32 - n))$

We use standard mathematical notation for summation. For example:

$\sum_{i = 0}^{n} i$

denotes the sum of integers in the range $[0, n]$.

We define a similar notation for exclusive or:

$\bigoplus_{i = 0}^{n} i$

denotes the bitwise exclusive or of theintegers in $[0, n]$.

# Splitting

The primary result of this specification is to define a family of
functions:

$\operatorname{SPLIT}_C \in V_8 \rightarrow V_v$

...which is parameterized by a configuration $C$, consisting of:

- $S_{\text{min}} \in U_{32}$, the minimum split size
- $S_{\text{max}} \in U_{32}$, the maximum split size
- $H \in V_8 \rightarrow U_{32}$, the hash function
- $T \in U_{32}$, the threshold

The configuration must satisfy $S_{\text{max}} \ge S_{\text{min}} > 0$.

## Definitions

We define the constant $W$, which we call the "window size," to be 64.

The "split index" $I(X)$ of a sequence $X$ is either the smallest
non-negative integer $i$ satisfying:

- $i \le |X|$ and
- $S_{\text{max}} \ge i \ge S_{\text{min}}$ and
- $H(\langle X_{i-W}, \dots, X_{i-1} \rangle) \mod 2^T = 0$

...or $\operatorname{min}(|X|, S_{\text{max}})$, if no such $i$ exists. For the
purposes of this definition we set $X_i = 0$ for $i < 0$.

The “prefix” $P(X)$ of a non-empty sequence $X$ is $\langle X_0, \dots, X_{I(X)-1} \rangle$.

The “remainder” $R(X)$ of a non-empty sequence $X$ is $\langle X_{I(X)}, \dots, X_{|X|-1} \rangle$.

We define $\operatorname{SPLIT}_C(X)$ recursively, as follows:

- If $|X| = 0$, $\operatorname{SPLIT}_C(X) = \langle \rangle$
- Otherwise, $\operatorname{SPLIT}_C(X) = \langle P(X) \rangle \mathbin{\|} \operatorname{SPLIT}_C(R(X))$

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

The “hashval” $V(X)$ of a sequence $X$ is:

$H(\langle X_{\operatorname{max}(0, |X|-W)}, \dots, X_{|X|-1} \rangle)$

(i.e., the hash of the last $W$ bytes of $X$).

The “level” $L(X)$ of a sequence $X$ is $Q - T$,
where $Q$ is the largest integer such that

- $Q \le 32$ and
- $V(P(X)) \mod 2^Q = 0$

(i.e., the level is the number of trailing zeroes in the rolling checksum in excess of the threshold needed to produce the prefix chunk $P(X)$).

(Note:
When $|R(X)| > 0$,
$L(X)$ is non-negative,
because $P(X)$ is defined in terms of a hash with $T$ trailing zeroes.
But when $|R(X)| = 0$,
that hash may have fewer than $T$ trailing zeroes,
and so $L(X)$ may be negative.
This makes no difference to the algorithm below, however.)

A “node” in a hashsplit tree
is a pair $(D, C)$
where $D$ is the node’s “depth”
and $C$ is a sequence of children.
The children of a node at depth 0 are chunks
(i.e., subsequences of the input).
The children of a node at depth $D > 0$ are nodes at depth $D - 1$.

The function $\operatorname{Children}(N)$ on a node $N = (D, C)$ produces $C$
(the sequence of children).

## Algorithm

To compute a hashsplit tree from sequence $X$,
compute its “root node” as follows.

1. Let $N_0$ be $(0, \langle\rangle)$ (i.e., a node at depth 0 with no children).
2. If $|X| = 0$, then:
    a. Let $d$ be the largest depth such that $N_d$ exists.
    b. If $|\operatorname{Children}(N_0)| > 0$, then:
        i. For each integer $i$ in $[0 .. d]$, “close” $N_i$.
        ii. Set $d \leftarrow d+1$.
    c. [pruning] While $d > 0$ and $|\operatorname{Children}(N_d)| = 1$, set $d \leftarrow d-1$ (i.e., traverse from the prospective tree root downward until there is a node with more than one child).
    d. **Terminate** with $N_d$ as the root node.
3. Otherwise, set $N_0 \leftarrow (0, \operatorname{Children}(N_0) \mathbin{\|} \langle P(X) \rangle)$ (i.e., add $P(X)$ to the list of children in $N_0$).
4. For each integer $i$ in $[0 .. L(X))$, “close” the node $N_i$ (see below).
5. Set $X \leftarrow R(X)$.
6. Go to step 2.

To “close” a node $N_i$:

1. If no $N_{i+1}$ exists yet, let $N_{i+1}$ be $(i+1, \langle\rangle)$ (i.e., a node at depth ${i + 1}$ with no children).
2. Set $N_{i+1} \leftarrow (i+1, \operatorname{Children}(N_{i+1}) \mathbin{\|} \langle N_i \rangle)$ (i.e., add $N_i$ as a child to $N_{i+1}$).
3. Let $N_i$ be $(i, \langle\rangle)$ (i.e., new node at depth $i$ with no children).

# Rolling Hash Functions

## CP32

The `cp32` hash function is based on cyclic polynomials. The family of
related functions is sometimes also called "buzhash." `cp32` is the
recommended hash function for use with hashsplit; use it unless you have
clear reasons for doing otherwise.

### Definition

We define the function $\operatorname{CP32} \in V_8 \rightarrow U_{32}$
as:

$\operatorname{CP32}(X) = \bigoplus_{i = 0}^{|X| - 1}
\operatorname{ROT}_L(g(X_i), |X| - i + 1)$

Where $g(n) = G_n$ and the sequence $G \in V_{32}$ is defined in the
appendix.

The sequence $G$ was chosen at random. Note that $|G| = 256$, so
$g(n)$ is always defined.

### Implementation

## Rolling

$\operatorname{CP32}$ can be computed in a rolling fashion; for
sequences

$X = \langle X_0, \dots, X_N \rangle$

and

$Y = \langle X_1, \dots, X_N, y \rangle$

Given $\operatorname{CP32}(X)$, $X_0$ and $y$, we can compute
$\operatorname{CP32}(Y)$ as:

$\operatorname{CP32}(Y) = \operatorname{ROT}_L(\operatorname{CP32}(X),
1) \oplus \operatorname{ROT}_L(g(X_0), |X| \mod 32) \oplus g(y)$.

Note that the splitting algorithm only computes hashes on sequences of
size $W = 64$, and since 64 is a multiple of 32 this means that for the
purposes of splitting, the above can be simplified to:

$\operatorname{CP32}(Y) = \operatorname{ROT}_L(\operatorname{CP32}(X),
1) \oplus g(X_0) \oplus g(y)$.

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

# Appendix

The definition of $G$ as used by $\operatorname{CP32}$ is:

$\langle$
```
1798466244, 335077821, 492897903, 2272477127, 924078993,
470900288, 3538556835, 2253448369, 3901930249, 1082788922, 1010874156,
714641328, 4136751352, 2988471001, 1873327877, 1925014406, 3717225155,
2144051386, 1334093084, 500713342, 1872178988, 2899116430, 3801501459,
2055021448, 3958647975, 2833424546, 2394367699, 3436276150, 1413805598,
3782782330, 3579224461, 3133673204, 885754503, 2000724393, 1650724833,
2984658034, 3739408072, 662999316, 3056811851, 3138083869, 1474639442,
4259649945, 4017566483, 465238337, 781885572, 928545464, 2787015764,
3078209121, 3631832061, 386384374, 1358863444, 855586437, 2499107874,
707972634, 194016939, 339095673, 2929281836, 1250797697, 1198569924,
4107355101, 1890126859, 2694076458, 1260735760, 609497694, 388343177,
2587066586, 2492394206, 2046329380, 2072888184, 3255373238, 1106749356,
3571012236, 2131471591, 3541399572, 3614800836, 3022576390, 774577410,
77184245, 823105086, 3857914499, 3771555855, 2336796436, 1452192314,
30479627, 871710755, 2699403231, 2367144669, 4219196231, 1301074994,
1716369630, 3566152300, 381894957, 2268738300, 276392481, 3980456184,
1554573746, 259052121, 234173122, 23950250, 1165367973, 412829095,
1254938419, 1679790307, 1496242670, 1260221101, 1124019412, 106214921,
2039485120, 1499412132, 137092054, 58056147, 3245088693, 1464413688,
2731895448, 3753028136, 925430623, 3695831665, 2599322487, 3593331371,
742039893, 1081974509, 3160094770, 1117133092, 345511722, 3339872022,
2504608598, 3557049083, 2989113041, 181657774, 2007372650, 4212900848,
443636792, 3434861085, 102756407, 2245460171, 2324673430, 2506866248,
4065685208, 857327755, 3772175337, 1199813398, 61289795, 624682477,
1093806826, 3753905274, 2536215571, 3435807477, 169664309, 2732339640,
4102264811, 3191810878, 3532983199, 3436341711, 1369853097, 3930511726,
3404499246, 1068382818, 4046345179, 928204364, 558874001, 1620894455,
1633203276, 2806743079, 2045995282, 382386530, 1584848719, 3158410491,
2435624061, 4185806596, 1716376159, 2105767941, 1401343968, 1314413943,
2700759678, 3948708865, 2709965905, 99565343, 3253716852, 947063852,
2912703801, 667765048, 1930933490, 2234259567, 2764681460, 4120047632,
4143875827, 2974525548, 118528551, 218125965, 3643729055, 3697557171,
2374091571, 3441261501, 2281675994, 2163884677, 623218011, 3655599706,
2023054360, 2223459370, 4256043883, 1223603155, 3261217112, 1517615469,
197872117, 809619196, 2779816360, 757709542, 2439696019, 242243149,
2722665646, 2560033869, 3416882218, 1386419121, 831440001, 3295846081,
3641366410, 1441505168, 1326817242, 2836380996, 3502924296, 1549365865,
3764012830, 648405540, 2534689092, 3974284422, 4133264030, 2191784891,
2455575138, 3603219099, 1639524653, 370485705, 930165312, 1760334651,
3984631013, 45275969, 1204803190, 3219286849, 1585735867, 1846351268,
1077427956, 2179343099, 424359820, 206584565, 2382377895, 3844702,
2642110243, 117631095, 884429158, 3518666968, 2610894740, 3902151812,
2641836823, 4185407209, 3757119320, 1237230287, 2699069567
```
$\rangle$

[rsync]: https://rsync.samba.org/tech_report/node3.html
[bup]: https://bup.github.io/
[perkeep]: https://perkeep.org/
