package main

// This program evaluates rolling hash algorithms on a few different critera:
//   - It measures the time required to consume a megabyte of random data,
//     "rolling" and computing a new digest on each byte.
//   - It measures the likelihood of each output bit being zero,
//     reporting when the likelihood departs from the expected value of 50%.
//   - It measures the likelihood that each pair of output bits is correlated,
//     reporting when the likelihood departs from the expected value of 50%.
//   - It performs a "strict avalanche" test,
//     in which varying the input by a single bit should change each output bit with likelihood 50%.
//
// Run with -all to test all rolling-hash algorithms.
// Run with algorithm names to test only those algorithms.
//
// To add new algorithms, update the `hashes` map below.
// It maps an algorithm name to a factory function that produces a ready-to-use `roller`.
//
// Note that the abstract `roller` interface adds a few nanoseconds to each method call.
// This makes the timing results merely an adequate starting point for comparing algorithms.

import (
	"flag"
	"fmt"
	"hash/crc32"
	"math/rand"
	"time"

	"github.com/chmduquesne/rollinghash/adler32"
	"github.com/chmduquesne/rollinghash/bozo32"
	"github.com/chmduquesne/rollinghash/buzhash32"
	"github.com/chmduquesne/rollinghash/buzhash64"
	"github.com/chmduquesne/rollinghash/rabinkarp64"
	"go4.org/rollsum"
)

type roller interface {
	Roll(byte)
	Digest() uint32
}

func main() {
	var (
		all  = flag.Bool("all", false, "evaluate all hashes")
		seed = flag.Int64("seed", time.Now().Unix(), "RNG seed")
	)
	flag.Parse()

	fmt.Printf("Using RNG seed %d\n", *seed)

	if *all {
		for name, factory := range hashes {
			fmt.Printf("%s:\n", name)
			eval(factory, *seed)
		}
	} else {
		for _, name := range flag.Args() {
			fmt.Printf("%s:\n", name)
			eval(hashes[name], *seed)
		}
	}
}

// Assigning to this variable from inside the timing loop,
// below,
// prevents the compiler from optimizing away the call to Digest().
var digest uint32

func eval(factory func() roller, seed int64) {
	src := rand.NewSource(seed)
	rng := rand.New(src)
	var megabyte [1024 * 1024]byte
	rng.Read(megabyte[:])

	var (
		zeroes       [32]int
		correlations [32 * 32]int
	)

	r := factory()
	start := time.Now()
	for i := 0; i < len(megabyte); i++ {
		r.Roll(megabyte[i])
		digest = r.Digest()
	}
	fmt.Printf("  Elapsed time to roll/digest a megabyte of random data: %s\n", time.Since(start))

	r = factory()
	for i := 0; i < len(megabyte); i++ {
		r.Roll(megabyte[i])
		digest = r.Digest()

		for i := 0; i < 32; i++ {
			if digest&(1<<i) == 0 {
				zeroes[i]++
			}
			for j := i + 1; j < 32; j++ {
				if ((digest & (1 << i)) == 0) == ((digest & (1 << j)) == 0) {
					correlations[32*i+j]++
				}
			}
		}
	}

	fmt.Println("  Bits departing from 50% likelihood of being zero:")
	for i, z := range zeroes {
		frac := float32(z) / float32(len(megabyte))
		if frac < .49 || frac > .51 {
			fmt.Printf("    Bit %d is zero %.1f%% of the time\n", i, 100.0*frac)
		}
	}
	fmt.Println("  Bit-pair correlations departing from 50% likelihood:")
	for i := 0; i < 31; i++ {
		for j := i + 1; j < 32; j++ {
			frac := float32(correlations[32*i+j]) / float32(len(megabyte))
			if frac < .49 || frac > .51 {
				fmt.Printf("    Bit %d == bit %d %.1f%% of the time\n", i, j, 100.0*frac)
			}
		}
	}

	var (
		inp      = megabyte[:256]
		differed [32]int
	)

	r = factory()
	for i := 0; i < len(inp); i++ {
		r.Roll(inp[i])
	}
	digest = r.Digest() // reference value

	// Try every way to flip a single bit in inp.
	// Compare the resulting digests with the reference value.
	// Under the "strict avalanche" criterion,
	// each output bit should change with likelihood 50%.
	for j := 0; j < len(inp); j++ {
		for k := 0; k < 8; k++ {
			r = factory()
			for i := 0; i < len(inp); i++ {
				byt := inp[i]
				if i == j {
					byt = byt ^ (1 << k)
				}
				r.Roll(byt)
			}
			d2 := r.Digest()
			for i := 0; i < 32; i++ {
				if ((digest & (1 << i)) == 0) != ((d2 & (1 << i)) == 0) {
					differed[i]++
				}
			}
		}
	}

	fmt.Println("  On 1-bit input change, digest bits departing from 50% likelihood of change:")
	for i := 0; i < 32; i++ {
		frac := float32(differed[i]) / float32(8*len(inp))
		if frac < .49 || frac > .51 {
			fmt.Printf("    Bit %d varied %.1f%% of the time\n", i, 100.0*frac)
		}
	}
}

var hashes = map[string]func() roller{
	"rollsum":   func() roller { return rollsum.New() },
	"adler32":   newAdler32(64),
	"bozo32":    newBozo32(64),
	"buzhash32": newBuzhash32(64),
	"buzhash64": newBuzhash64(64),
	"crc32":     newCRC32(64),
	// "rabinkarp64": newRabinKarp64(64), // TODO: debug why this seems to cause eval to hang
}

// adler32

type adler32wrapper struct {
	a *adler32.Adler32
}

func newAdler32(windowSize int) func() roller {
	return func() roller {
		w := &adler32wrapper{a: adler32.New()}

		// Objects in the rollinghash module require an initial call to Write to set up their rolling windows.
		z := make([]byte, windowSize)
		w.a.Write(z)
		return w
	}
}

func (w *adler32wrapper) Roll(b byte) {
	w.a.Roll(b)
}

func (w *adler32wrapper) Digest() uint32 {
	return w.a.Sum32()
}

// bozo32

type bozo32wrapper struct {
	b *bozo32.Bozo32
}

func newBozo32(windowSize int) func() roller {
	return func() roller {
		w := &bozo32wrapper{b: bozo32.New()}

		// Objects in the rollinghash module require an initial call to Write to set up their rolling windows.
		z := make([]byte, windowSize)
		w.b.Write(z)
		return w
	}
}

func (w *bozo32wrapper) Roll(b byte) {
	w.b.Roll(b)
}

func (w *bozo32wrapper) Digest() uint32 {
	return w.b.Sum32()
}

// buzhash32

type buzhash32wrapper struct {
	b *buzhash32.Buzhash32
}

func newBuzhash32(windowSize int) func() roller {
	return func() roller {
		w := &buzhash32wrapper{b: buzhash32.New()}

		// Objects in the rollinghash module require an initial call to Write to set up their rolling windows.
		z := make([]byte, windowSize)
		w.b.Write(z)
		return w
	}
}

func (w *buzhash32wrapper) Roll(b byte) {
	w.b.Roll(b)
}

func (w *buzhash32wrapper) Digest() uint32 {
	return w.b.Sum32()
}

// buzhash64

type buzhash64wrapper struct {
	b *buzhash64.Buzhash64
}

func newBuzhash64(windowSize int) func() roller {
	return func() roller {
		w := &buzhash64wrapper{b: buzhash64.New()}

		// Objects in the rollinghash module require an initial call to Write to set up their rolling windows.
		z := make([]byte, windowSize)
		w.b.Write(z)
		return w
	}
}

func (w *buzhash64wrapper) Roll(b byte) {
	w.b.Roll(b)
}

func (w *buzhash64wrapper) Digest() uint32 {
	return uint32(w.b.Sum64())
}

// crc32 (not strictly a "rolling" hash)

type crc32wrapper struct {
	buf  []byte // circular buffer
	next int    // position in buf of next write
}

func newCRC32(windowSize int) func() roller {
	return func() roller {
		return &crc32wrapper{buf: make([]byte, windowSize)}
	}
}

func (w *crc32wrapper) Roll(b byte) {
	w.buf[w.next] = b
	w.next++
	w.next %= len(w.buf)
}

func (w *crc32wrapper) Digest() uint32 {
	buf := make([]byte, len(w.buf))
	copy(buf[:], w.buf[w.next:])
	copy(buf[len(w.buf)-w.next:], w.buf[:])
	return crc32.ChecksumIEEE(buf)
}

// rabinkarp64

type rabinkarp64wrapper struct {
	r *rabinkarp64.RabinKarp64
}

func newRabinKarp64(windowSize int) func() roller {
	return func() roller {
		w := &rabinkarp64wrapper{r: rabinkarp64.New()}

		// Objects in the rollinghash module require an initial call to Write to set up their rolling windows.
		z := make([]byte, windowSize)
		w.r.Write(z)
		return w
	}
}

func (w *rabinkarp64wrapper) Roll(b byte) {
	w.r.Roll(b)
}

func (w *rabinkarp64wrapper) Digest() uint32 {
	return uint32(w.r.Sum64())
}
