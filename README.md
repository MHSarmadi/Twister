# Twister Permutation — Specification and Analysis

## 1. Overview

The **Twister** permutation is a 1024-bit cryptographic permutation built from
four mixing layers over a 4×4 word matrix. Its fundamental building block is a
**256-bit quarter round** acting on four 64-bit words. The quarter round is a
**MARX**-class function — **M**ultiply–**A**dd–**R**otate–**X**OR — in which the
only source of non-linearity is **integer carry propagation**, arising from both
`bits.Mul64` and modular addition.

**Repository:** [`github.com/MHSarmadi/Twister`](https://github.com/MHSarmadi/Twister)

The quarter round is implemented as `quarter_ab2ab` and is invoked by four
mixing layers (`permute_col2rdiag`, `permute_rdiag2row`, `permute_row2ldiag`,
`permute_ldiag2col`), which compose to a full 1024-bit permutation in four
super-rounds (64 quarter rounds total).

**Key design claims:**

| Property | Value |
| :--- | :--- |
| State size (per quarter round) | 4 × 64 = 256 bits |
| State size (full permutation) | 16 × 64 = 1024 bits |
| Ideal avalanche at | round 1 (of the QR) |
| Measured avalanche (QR R1) | 128.01 / 256 (ideal 128), σ = 8.00 |
| Per-word symmetry | uniform (per-word mean within 0.1 bits) |
| Max differential probability above `2⁻¹²` | none observed |
| Bijective | yes (proof in §5) |
| Nonlinearity source | integer carries only |
| Total QRs per permutation | 64 (4 super-rounds × 4 phases × 4 QRs) |

### §1.1 — Endpoint symmetry: `ab2ab`

The quarter round is named `ab2ab` because it takes two words `(a, b)` as
inputs, modifies them through a chain of four multiplications and four linear
operations, and delivers the final state back through `a` and `b`:

```
a, b ──▶ chain of 4 muls + pre-mix + post-mix ──▶ a, b   (modified)
         c, d ──▶ sinks (modified in-chain, not at endpoints)
```

This is the deliberate successor to the ARX `a2a` quarter round of the previous
design (V2), which returned the chain to `a` alone. The change is not cosmetic:

- **Dual-symbolic multiplication requires two symbolic operands.** Since each
  `Mul64(a,b)` multiplies two state words, the natural endpoint pairing is
  `(a, b)`, not `a` alone. `ab2ab` aligns the endpoint symmetry with the
  multiplication topology.
- **The permutation can chain through both endpoints.** Every mixing layer's
  starter set contains *both* `a`-positions and `b`-positions of its four QRs
  (see §3.7), so the "hot" indices flow across phase boundaries through two
  channels instead of one.
- **Round 1 avalanche is a direct consequence.** The pre-mix spreads the four
  words to weight-3 of each other; the four muls then amplify through carries;
  the post-mix returns four independent-looking words to the endpoints. The
  measured 128.01 / 256 at R1 is the empirical result of this topology.

### §1.2 — Full permutation architecture

The full permutation applies four mixing layers in a fixed order, five times
(this was reduced to **four** super-rounds in the final version — see §6.10
for the verification required):

| Layer | Geometry | Starter (a,b) pairs |
| :--- | :--- | :--- |
| `permute_col2rdiag` | columns | `{04, 9d, e2, f3}` |
| `permute_rdiag2row` | right diagonals | `{0d, b4, f2, 63}` |
| `permute_row2ldiag` | rows | `{03, 46, b9, df}` |
| `permute_ldiag2col` | left diagonals | `{0f, b6, 2d, 94}` |

Each layer covers all 16 words in 4 QRs — a "perfect shuffle" through the
4×4 matrix. Every word appears in exactly one QR per layer. The four layers
compose to a full permutation in which every input bit reaches every output
bit within 2 layers (256-bit full diffusion per QR × 4 QR per layer, joined
by the pre- and post-mix of each QR).

## 2. Specification

### §2.1 Constants

```go
const (
    c0 uint64 = 0x5477697374657221  // "Twister!"  (LE)
    c1 uint64 = 0x205361726d616469  // " Sarmadi"  (LE)
)
```

The constants are XOR-injected into `s[0]` and `s[1]` at the start of **every**
quarter round. They serve three purposes: (i) they break the all-zero fixed
point, (ii) they provide domain separation for the string `"Twister! Sarmadi"`,
and (iii) the specific bit patterns are chosen (per §3.6) to have high RX
penalty (low Hamming weight of `c ⊕ rotl(c, r)` for all `r ≠ 0`).

Because the injection happens in *every* QR (64 QRs × 2 words = 128
word-injections per permutation), the constant breaks period-2 and period-3
self-similarity that would otherwise enable slide attacks.

### §2.2 Quarter round pseudocode

```
Input:  state (s0, s1, s2, s3) ∈ (Z/2⁶⁴)⁴
Output: state (s0, s1, s2, s3) ∈ (Z/2⁶⁴)⁴

1. Constant injection:
   s0 ← s0 ⊕ c0
   s1 ← s1 ⊕ c1

2. Pre-mix (linear over GF(2)):
   m  ← s0 ⊕ s1 ⊕ s2 ⊕ s3
   s0 ← m ⊕ s3
   s1 ← m ⊕ s2
   s2 ← m ⊕ s1
   s3 ← m ⊕ s0

3. QR1:
   (a, b) ← Mul64(s0, s1)          // 128-bit product, hi / lo
   s0 ← Rotl64(s0, 32)
   s1 ← Rotl64(s1, 32)
   s2 ← s2 ⊕ a
   s3 ← s3 ⊕ b

4. QR2:
   (a, b) ← Mul64(s1, s2)
   s0 ← s0 ⊕ a
   s2 ← Rotl64(s2, 32)
   s3 ← s3 + b

5. QR3:
   (a, b) ← Mul64(s0, s3)
   s1 ← s1 ⊕ a
   s2 ← s2 + b
   s3 ← Rotl64(s3, 32)

6. QR4:
   (a, b) ← Mul64(s2, s3)
   s0 ← s0 + a
   s1 ← s1 + b

7. Post-mix (linear over Z/2⁶⁴):
   m  ← s0 + s1 + s2 + s3
   s0 ← m − s3
   s1 ← m − s2
   s2 ← m − s1
   s3 ← m − s0
```

### §2.3 Reference implementation

See `quarter_ab2ab` in [`github.com/MHSarmadi/Twister`](https://github.com/MHSarmadi/Twister).

### §2.4 — Full permutation pseudocode

```
Input:  state s[0..15] ∈ (Z/2⁶⁴)¹⁶
Output: state s[0..15]

for round in 0..3:
    // col2rdiag (columns → right diagonals)
    quarter_ab2ab(s, {0x0, 0x4, 0x8, 0xc})   // col 0
    quarter_ab2ab(s, {0x9, 0xd, 0x1, 0x5})   // col 1
    quarter_ab2ab(s, {0xe, 0x2, 0x6, 0xa})   // col 2
    quarter_ab2ab(s, {0xf, 0x3, 0x7, 0xb})   // col 3

    // rdiag2row (right diagonals → rows)
    quarter_ab2ab(s, {0x0, 0xd, 0x7, 0xa})   // rdiag 0
    quarter_ab2ab(s, {0xb, 0x4, 0xe, 0x1})   // rdiag 1
    quarter_ab2ab(s, {0xf, 0x2, 0x5, 0x8})   // rdiag 2
    quarter_ab2ab(s, {0x6, 0x3, 0x9, 0xc})   // rdiag 3

    // row2ldiag (rows → left diagonals)
    quarter_ab2ab(s, {0x0, 0x3, 0x1, 0x2})   // row 0
    quarter_ab2ab(s, {0x4, 0x6, 0x5, 0x7})   // row 1
    quarter_ab2ab(s, {0xb, 0x9, 0x8, 0xa})   // row 2
    quarter_ab2ab(s, {0xd, 0xf, 0xe, 0xc})   // row 3

    // ldiag2col (left diagonals → columns)
    quarter_ab2ab(s, {0x0, 0xf, 0x5, 0xa})   // ldiag 0
    quarter_ab2ab(s, {0xb, 0x6, 0xc, 0x1})   // ldiag 1
    quarter_ab2ab(s, {0x2, 0xd, 0x7, 0x8})   // ldiag 2
    quarter_ab2ab(s, {0x9, 0x4, 0xe, 0x3})   // ldiag 3
```

Total: 4 super-rounds × 4 layers × 4 QRs = **64 quarter rounds** = **256 `Mul64`**
per `Permute()`.

## 3. Design Rationale

### §3.1 Integer multiplication, not carryless (CLMUL)

Carryless multiplication is bilinear over GF(2)[X] and adds **zero
cryptographic non-linearity**. Its differential is a fixed linear map, and a
round built solely on CLMUL would be affine. Integer multiplication differs
precisely by its **carry propagation**, which is the sole source of
non-linearity. Empirical comparison:

| Multiplier | Round-1 avalanche | σ |
| :--- | :--- | :--- |
| CLMUL + pre-mix | 125.34 | 17.7 |
| Integer mul + pre-mix | **126.71** | **8.14** |
| Integer mul + pre-mix + rotl-32 | **128.01** | **8.00** |

The carries are not a side effect; they are the primitive.

### §3.2 Rotl-32, not bit reversal

The **triangle structure** of integer multiplication (number of contributing
partial products at bit position `k` is `min(k+1, 127−k)`) means the leak of
diffusion is **concentrated at the low bits** (positions 0–15). Reversal
(`i → 63 − i`) rescues the low bits but **maps the top bit to position 0** — the
single most leaky position — so a top-bit input delta (the one with `DP = 1.0`
for a single multiplication) lands in the worst possible slot for the *next*
multiplication. `Rotl64(x, 32)` rescues the same leaky bits (0–15 → 32–47,
fully mixed) while avoiding the top-bit-to-position-0 penalty, and compiles to a
single instruction (`RORX` on x86-64, `ROR` on ARM64).

Empirically: reversal scored 127.91, rotl-32 scores **128.01**.

### §3.3 XOR then addition ordering

The round uses **XOR first, addition second** for every word:

| Word | XORed at | Added at |
| :--- | :--- | :--- |
| `s0` | QR2 | QR4 |
| `s1` | QR3 | QR4 |
| `s2` | QR1 | QR3 |
| `s3` | QR1 | QR2 |

XOR is GF(2)-linear: it spreads bits without amplifying correlation. Addition
is nonlinear: it amplifies through carries. The ordering **spreads correlation
first, then amplifies the decorrelated result**. This is the "spread-then-amplify"
principle — the opposite of the carry-first ordering used in some ARX designs,
chosen here because the multiplication (the primary non-linearity) already
operates on correlated inputs, and we do not want the addition to amplify
structural artifacts before they have been decorrelated by XOR.

### §3.4 Dual-symbolic multiplication

Both operands of every `Mul64` are **state words**, not constants. This is the
primary departure from classical MARX (Alzette, Sparkle, EchoPulse), where the
multiplier is a fixed constant. The trade-off:

- **Gain**: each multiplication is bilinear in the state, so it contributes a
  much denser differential and algebraic structure than a fixed-multiplier MARX
  step.
- **Cost**: the differential of `Mul64(x, y)` depends on `(x, y)`, not only on
  `(Δx, Δy)`. This makes SMT/MILP trail search intractable at 64-bit word size
  (§6.1).

Empirically, dual-symbolic multiplication is what allows round 1 to reach ideal
avalanche. A fixed-multiplier version of the same topology does not.

### §3.5 Pre-mix / post-mix topology

The pre-mix is the matrix

$$
M_{XOR} = \begin{pmatrix} 1&1&1&0 \\ 1&1&0&1 \\ 1&0&1&1 \\ 0&1&1&1 \end{pmatrix}
$$

over GF(2). Its determinant is `1` (bijective), every row and column has Hamming
weight 3, and a delta in any one word produces a delta in exactly 3 of the 4
output words. The post-mix is the same topology over `Z/2⁶⁴`:

$$
M_{ADD} = \begin{pmatrix} 1&1&1&0 \\ 1&1&0&1 \\ 1&0&1&1 \\ 0&1&1&1 \end{pmatrix} \quad \text{mod } 2^{64}
$$

which is also bijective (`det(M_ADD) = −3`, odd, hence invertible mod `2⁶⁴`).
The use of the *same* topology in two different rings gives the round a symmetry
between bit-level (GF(2)) and word-level (Z/2⁶⁴) mixing.

### §3.6 Constant selection

The constants encode `"Twister! Sarmadi"` in little-endian. They are chosen
such that for every rotation `r ∈ [1, 63]`, the RX penalty
`HW(c0 ⊕ rotl(c0, r)) + HW(c1 ⊕ rotl(c1, r))` is large. The measured minimum
combined RX penalty is **48** over all `r ≠ 0`, ruling out RX trails that
survive the constant injection.

### §3.7 — Permutation index chaining

Each of the four permute layers has four QRs, and each QR has an `(a, b)`
endpoint pair (the two words combined first and modified last). The permute
layer's **starter set** is the union of these 8 words. The four layers are
chained by high overlap between consecutive starter sets:

| From → To | Common starters | Similarity |
| :--- | :--- | :--- |
| `col2rdiag → rdiag2row` | `{0,4,d,f,2,3}` | **75 %** |
| `rdiag2row → row2ldiag` | `{0,3,4,6,b,d,f}` | **87.5 %** |
| `row2ldiag → ldiag2col` | `{0,4,6,b,9,d,f}` | **87.5 %** |
| `ldiag2col → col2rdiag` | `{0,2,4,9,d,f}` | **75 %** |

The **`75 %` transitions bracket the round**, and the **`87.5 %` transitions
carry the round's interior**. Index `0` is common to all four layers — a
deliberate choice — making it the "anchor" of the super-round.

**Why this matters.** In a trail, "hot" indices are the ones most recently
modified — they carry the newest non-linearity. When two consecutive layers
share 6–7 of 8 starters, the hot indices continue hot across the layer
boundary, keeping trail propagation active. Empirically (V2 measurements, ARX
QR, but expected to hold or improve with MARX QR), this composition is
**anti-correlated** — 2-phase trail probability is 100× *lower* than
independent-phase chaining would predict. The chaining is therefore a
defensive property: it accelerates the death of trails at phase boundaries.

Note that `col2rdiag → rdiag2row` and `ldiag2col → col2rdiag` are **both** at
75 %; the chaining argument is symmetric around the round, which is good for
security but worth stating explicitly. (An earlier draft of this document
mislabeled `ldiag2col → col2rdiag` as 87.5 %; the corrected figure is 75 %.)

### §3.8 — Why four geometries

The four geometries (columns, right diagonals, rows, left diagonals) form a
complete decomposition of the 4×4 word grid into 4-word groups. Between them,
they give the permutation **four distinct traversals of the same grid**:

- **Columns** group words by `col = w mod 4`.
- **Rows** group words by `row = ⌊w/4⌋`.
- **Right diagonals** group words by `(w + ⌊w/4⌋) mod 4` (or equivalent).
- **Left diagonals** group words by `(w − ⌊w/4⌋) mod 4`.

Any two consecutive layers jointly cover all four "directions." This is more
than ChaCha's two layers (columns + diagonals) and is why the permutation
reaches full 1024-bit diffusion faster.

## 4. Primitive Set

Per quarter round:

| Operation | Count | Purpose |
| :--- | :--- | :--- |
| `bits.Mul64` (×8 hi/lo output) | 4 | Non-linearity (carries) |
| `bits.RotateLeft64(x, 32)` | 4 | Low-bit rescue, position mixing |
| XOR (`^`) | 13 | Linear diffusion, pre-mix |
| Add/Sub (`+`, `-`) | 11 | Non-linearity (carries), post-mix |
| Constant injection | 2 | Fixed-point break, domain separation |

Every word is **multiplied exactly twice**, **rotated exactly once**,
**XORed once**, and **added once** per round.

Per permutation (64 QRs):

| Operation | Count |
| :--- | :--- |
| `Mul64` | 256 |
| `Rotl64(x,32)` | 256 |
| XOR | 832 |
| Add/Sub | 704 |
| Constant XOR | 128 |

## 5. Algebraic Properties

### §5.1 Bijectivity

The quarter round is a permutation of `(Z/2⁶⁴)⁴`, with a constructive inverse:

- **Constant injection**: `s ⊕ c` is an involution → bijective.
- **Pre-mix**: `M_XOR` has determinant 1 over GF(2) → bijective. It is an
  involution (the mixture word `m` is preserved by the transform).
- **Each QR `k`**: the two source words are modified only by `Rotl64(x, 32)`
  applied *after* the multiplication. The inverse procedure is therefore:
  (i) un-rotate the source words, (ii) recompute `Mul64` on the recovered
  originals, (iii) un-XOR / un-ADD the sink words. Each QR is bijective for
  any function `f: (u64, u64) → (u64, u64)` used in place of `Mul64`.
- **Post-mix**: `M_ADD` has `det = −3`, which is odd, hence invertible mod `2⁶⁴`.
  Constructive inverse: `m_new = 3m`, so `m = m_new · 3⁻¹`, then
  `s0 = m − s0_new`, etc.

Composition of bijections is a bijection. ∎

**Caveat for implementers.** The proof above depends on the multipliers being
recoverable from the outputs, which requires the rotation to be applied *after*
the multiplication. Reordering `s0 ← Rotl64(s0, 32)` to appear before
`Mul64(s0, s1)` would break bijectivity. Any modification must preserve the
current ordering.

### §5.2 Fixed points

The all-zero state is **not** a fixed point, because the constant injection
`s0 ⊕ c0`, `s1 ⊕ c1` breaks the symmetry (`c0, c1 ≠ 0`). Empirically, over
2¹⁸ random samples, `min HW(P²(x) ⊕ x) = 89` and `min HW(P³(x) ⊕ x) = 87`,
both on the theoretical floor for `Binomial(256, ½)`.

### §5.3 Round symmetry

Every word experiences exactly {2 muls, 1 rotation, 1 XOR, 1 add} per round.
The asymmetry in *which* QR does which operation to *which* word is cosmetic;
the per-word operation multiset is identical for all four words.

## 6. Empirical Security Evaluation

All tests are in the repository under `research/mul/`. Sample sizes are chosen
to place the noise floor at `2⁻¹²` (MDP) or `1/(2√N)` (linear, BIC).

### §6.1 Avalanche (256-bit QR)

| Rounds | Mean / 256 | Std-dev | Per-word spread |
| :--- | :--- | :--- | :--- |
| 1 | **128.01** | 8.00 | uniform (≤ 0.1 bit) |
| 2 | 127.99 | 8.00 | uniform |
| 4 | 127.99 | 8.00 | uniform |
| 8 | 127.98 | 8.03 | uniform |

Ideal for a random 256-bit permutation is mean 128, std 8. **Round 1 is at the
ideal plateau.**

### §6.2 Differential

| Test | Result |
| :--- | :--- |
| MDP (160 deltas × 4096 samples) | max DP = 1/4096 = noise floor, all rounds |
| Structured deltas (top-bit, bot-bit, alt, mid) | ideal at R1 (mean 127.9–128.1, std 7.9–8.1) |
| Differential spread | 100 % distinct at R1 |
| Higher-order (2-bit flip) | identical to single-bit |

### §6.3 Bit-independence (BIC)

| Test | Comparisons | Max \|ρ\| | p-value | Verdict |
| :--- | :--- | :--- | :--- | :--- |
| BIC (all pairs) | 193,280 | 4.9σ | 0.19 | excellent |
| BIC (adjacent pairs) | 65,280 | 4.7σ | 0.14 | excellent |
| BIC balance | 65,536 | 4.3σ | 0.80 | excellent |

All maxima are below Bonferroni-corrected significance at α = 0.01
(threshold ≈ 5.2–5.4σ).

### §6.4 Rotational-XOR

| Constant | min HW(c ⊕ rotl(c, r)) | Worst r | Combined min |
| :--- | :--- | :--- | :--- |
| `c0` | 24 | 16 | — |
| `c1` | 22 | 24, 40 | — |
| Combined | — | — | **48** |

No rotation gives low RX penalty in the tested pair.

### §6.5 Rotational symmetry

Over 4 rounds, `HW(P(rotl(x, r)) ⊕ rotl(P(x), r))` has mean 127.3–128.7 for
every `r ∈ [1, 63]`, minimum ≥ 96. **No rotational symmetry.**

### §6.6 Integral / cube

Over cube sizes `k ∈ {8, 12, 16}` and rounds 1–4: `0/8` zero-sum trials at
every cell. **No integral distinguisher.**

### §6.7 Linear cryptanalysis (empirical)

Over 4096 mask pairs per round, samples/mask = 8192, noise = 0.0055:

| Round | Max \|bias\| | σ | p-value |
| :--- | :--- | :--- | :--- |
| 1 | 0.0188 | 3.4 | 0.97 |
| 2 | 0.0188 | 3.4 | 0.97 |
| 3 | 0.0250 | 4.5 | 0.44 |
| 4 | 0.0238 | 4.3 | 0.55 |

All below the Bonferroni threshold (~4.7σ at α = 0.01). Worst mask pair differs
every round → noise.

### §6.8 Truncated differential

Every single-word input delta activates all four output words with probability
`1.0000` at R1. **The pre-mix achieves full word-level diffusion in one round.**

### §6.9 Slide / self-similarity

`min HW(P²(x) ⊕ x) = 89`, `min HW(P³(x) ⊕ x) = 87`, both on the theoretical
floor. **No near-periodic state** for periods 2 and 3.

### §6.10 — Permutation-level tests (required, pending)

§6.1–6.9 are measurements on the 256-bit QR alone. They do not license claims
about the full 1024-bit `Permute()`. The following tests must be run on the
complete permutation before the design can be characterized as "secure":

| Test | Method | Target |
| :--- | :--- | :--- |
| Avalanche (1024) | 1-bit input diff, `Permute()` at 1, 2, 3, 4, 8, 16 calls | mean 512, std 16 by call 2 |
| Two-phase trail | follow Δ through one phase, measure its probability after the next | ≤ 2⁻²⁰ (V2 baseline; expect better) |
| Composition ratio | observed vs. naive product, 2-phase | ≤ 1 (anti-correlated) |
| Differential (full) | top-1 output Δ per input bit | ≥ 1/13 (V2 baseline) |
| Linear (full) | max bias over sampled masks | below 5σ Bonferroni |
| Slide (full) | `min HW(P²(x) ⊕ x)` | > 80 |
| Rotational (full) | `HW(P(rotl(x,r)) ⊕ rotl(P(x),r))` for all r | min > 90 |
| Integral (full) | 2⁸, 2¹², 2¹⁶ cube sizes, rounds 1–4 | no zero-sum |

**Expected results** (extrapolating from the QR alone — must be verified):

- Full avalanche: **call 2** (2 phases = 8 QRs = 32 muls), one full step ahead
  of the V2 baseline (which needed 3 calls).
- Two-phase trail: **≤ 2⁻²⁴** (QR carries are stronger than addition carries).
- Composition ratio: **0.01–0.1** (V2 showed strong anti-correlation; the
  ab2ab QR with pre-/post-mix should preserve or improve this).

**If the "call 2" full-avalanche prediction holds**, then the **4 super-rounds**
choice (16 calls) provides an 8× safety margin — appropriate for a permutation
whose QR is a single-round diffuser.

**If avalanche is not full by call 4**, increase `rounds` in `Permute()` to 5
(the V2 choice) — this is a one-line change with 25 % performance cost.

## 7. Analysis Limitations

**Formal trail bounds are not currently available for this design.** The barrier
is intrinsic, not a tooling issue:

- Dual-symbolic multiplication (`Mul64(s_i, s_j)` with both operands symbolic)
  makes the differential `Δ(s_i · s_j)` depend on the *values* of `s_i, s_j`,
  not only on their *differences*.
- SMT and MILP trail search require the differential of each non-linear gate
  to be a fixed table over `(Δx, Δy)`. This holds for fixed-multiplier MARX
  (Alzette, Sparkle, EchoPulse) but not here.
- Verified empirically: Z3, Boolector, and Bitwuzla all time out on the
  impossible-differential query at 64-bit (and even at 8-bit) word size.

**Available mitigations:**

1. **Fixed-operand heuristic analysis.** Replace `Mul64(s_i, s_j)` with
   `Mul64(s_i, K)` for analysis purposes. If the fixed-multiplier variant has
   good trail bounds, the dual-symbolic design is at least as strong against
   single-characteristic differential attacks. This is a heuristic, not a
   proof.
2. **Empirical evidence.** The nine families of statistical tests above provide
   strong practical evidence against classical attack classes.
3. **Published precedent.** ChaCha, Salsa, BLAKE, and SipHash are all in the
   same position: their best current differential bounds are *empirical*, not
   *provable*.

### §7.1 — Permutation-level formal limitations

The permute-layer composition adds a further intractability: the four geometries
(col / rdiag / row / ldiag) mean any formal trail search must track word
activations across 16 words and 4 layers — a state space of `2¹⁶` per layer.
Combined with the QR-level intractability above, the full permutation is
inaccessible to current SMT/MILP tools.

**Mitigation:** the permutation is a *fixed structure* with no key schedule,
so the relevant question is not "is there a trail?" but "is a trail below
`2⁻²⁵⁶` reachable?" The empirical two-phase results (once measured on the
ab2ab QR) bound this from above; formal bounds are a future-work item.

## 8. Summary

| Family | Test | Status |
| :--- | :--- | :--- |
| Diffusion (QR) | Avalanche R1 | ✓ ideal (128.01 / 256) |
| Diffusion (perm) | Avalanche (pending §6.10) | ⚠ **to be measured** |
| Differential (QR) | MDP, structured deltas | ✓ clean at 2⁻¹² |
| Differential (perm) | Two-phase trail, composition | ⚠ **to be measured** |
| Bit-independence | BIC (all, adjacent, balance) | ✓ clean after Bonferroni |
| Rotational | RX penalty, rotational symmetry | ✓ clean |
| Structural | Fixed points, 2-cycles, slide | ✓ clean |
| Integral (QR) | Cube attack (k = 8–16) | ✓ clean |
| Linear (QR) | Empirical linear bias | ✓ clean after Bonferroni |
| Truncated | Word-level activation | ✓ ideal at R1 |
| **Formal** | **MILP/SMT trail bounds** | **not available** (see §7) |

The Twister quarter round achieves ideal diffusion at round 1, passes all
standard empirical statistical tests with Bonferroni-corrected significance,
and presents a clear design rationale (integer carries as non-linearity,
rotl-32 for low-bit rescue, XOR-then-add for spread-then-amplify, ab2ab
endpoint symmetry to align with the dual-symbolic multiplication topology).

Its principal limitation is the intractability of formal differential/linear
trail search on dual-symbolic multiplication — a limitation shared with the
empirical analysis practice of many published ARX designs.

**Full-permutation security** (the 1024-bit `Permute()`) is argued from
QR-level empirical strength plus permutation-level composition structure, but
**requires the measurements in §6.10 to be completed** before formal claims
about `Permute()` can be made. The 4-super-round budget is one round thinner
than the V2 baseline and should be verified against the full battery before
being adopted as final.

---

*Document version: v2.0 — Quarter round specification and full permutation
architecture for Twister (1024-bit permutation).*
*Repository: [`github.com/MHSarmadi/Twister`](https://github.com/MHSarmadi/Twister).*
*All tests executable via `go test ./research/mul/...`.*
