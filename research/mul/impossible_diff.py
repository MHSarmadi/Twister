"""
Impossible-differential search for mult_permuter via Z3.

For each (input_diff, output_diff, rounds), asks:
  exists X : P^R(X) ^ P^R(X ^ input_diff) = output_diff ?

UNSAT  ⟹ impossible differential (guaranteed zero probability)
SAT    ⟹ possible (prints a witness X)
"""
import subprocess
import json
import time
import sys
from z3 import *

WORD = 64
C0 = 0x5477697374657221
C1 = 0x205361726d616469
ROTL_AMT = 32

def mul64(a, b):
    """128-bit product of two 64-bit values as (hi, lo)."""
    a_ext = ZeroExt(64, a)
    b_ext = ZeroExt(64, b)
    prod = a_ext * b_ext              # BitVec(128)
    return Extract(127, 64, prod), Extract(63, 0, prod)

def one_round(s, c0, c1):
    s0, s1, s2, s3 = s
    s0 = s0 ^ BitVecVal(c0, WORD)
    s1 = s1 ^ BitVecVal(c1, WORD)

    mixture = s0 ^ s1 ^ s2 ^ s3
    s0, s1, s2, s3 = mixture ^ s3, mixture ^ s2, mixture ^ s1, mixture ^ s0

    a, b = mul64(s0, s1)
    s0 = RotateLeft(s0, ROTL_AMT)
    s1 = RotateLeft(s1, ROTL_AMT)
    s2 = s2 ^ a
    s3 = s3 ^ b

    a, b = mul64(s1, s2)
    s0 = s0 ^ a
    s2 = RotateLeft(s2, ROTL_AMT)
    s3 = s3 + b

    a, b = mul64(s0, s3)
    s1 = s1 ^ a
    s2 = s2 + b
    s3 = RotateLeft(s3, ROTL_AMT)

    a, b = mul64(s2, s3)
    s0 = s0 + a
    s1 = s1 + b

    mixture = s0 + s1 + s2 + s3
    s0, s1, s2, s3 = mixture - s3, mixture - s2, mixture - s1, mixture - s0
    return (s0, s1, s2, s3)

def permute(state, rounds):
    s = state
    for _ in range(rounds):
        s = one_round(s, C0, C1)
    return s

# ---------- Step 1: verify the model matches Go ----------
def verify_model(bridge_path="./bin/mul_bridge", trials=20):
    import random
    random.seed(12345)
    for i in range(trials):
        x = [random.getrandbits(64) for _ in range(4)]
        req = json.dumps({"state": x, "rounds": 1})
        result = subprocess.run([bridge_path], input=req,
                                capture_output=True, text=True, check=True)
        go_out = json.loads(result.stdout)["state"]
        z3_out = [simplify(v).as_long() for v in permute(x, 1)]
        if go_out != z3_out:
            print(f"MISMATCH at trial {i}:")
            print(f"  input:    {[hex(v) for v in x]}")
            print(f"  Go out:   {[hex(v) for v in go_out]}")
            print(f"  Z3 out:   {[hex(v) for v in z3_out]}")
            sys.exit(1)
    print(f"Model verified against Go: {trials}/{trials} trials identical.")

# ---------- Step 2: impossible differential search ----------
def search_impossible(rounds, delta_x, delta_y, timeout_sec=60):
    solver = Solver()
    solver.set("timeout", timeout_sec * 1000)

    X = [BitVec(f"X{i}", WORD) for i in range(4)]
    Xd = [x ^ BitVecVal(d, WORD) for x, d in zip(X, delta_x)]

    Y  = permute(X, rounds)
    Yd = permute(Xd, rounds)

    for i in range(4):
        solver.add((Y[i] ^ Yd[i]) == BitVecVal(delta_y[i], WORD))

    t0 = time.time()
    result = solver.check()
    elapsed = time.time() - t0

    if result == unsat:
        return "IMPOSSIBLE", elapsed, None
    elif result == sat:
        m = solver.model()
        witness = [m.eval(x, model_completion=True).as_long() for x in X]
        return "possible", elapsed, witness
    else:
        return "TIMEOUT", elapsed, None

# ---------- Step 3: main ----------
if __name__ == "__main__":
    verify_model()

    input_deltas = [
        ("top-bit (w0)",    [0x8000000000000000, 0, 0, 0]),
        ("bottom-bit (w0)", [1, 0, 0, 0]),
        ("mid-bit (w0,b32)",[1 << 32, 0, 0, 0]),
        ("top+bot (w0)",    [0x8000000000000001, 0, 0, 0]),
    ]
    output_deltas = [
        ("top-bit (w0)",    [0x8000000000000000, 0, 0, 0]),
        ("bottom-bit (w0)", [1, 0, 0, 0]),
        ("top+bot (w0)",    [0x8000000000000001, 0, 0, 0]),
        ("all-ones",        [0xFFFFFFFFFFFFFFFF]*4),
    ]

    for rounds in [1, 2, 3]:
        print(f"\n=== Rounds = {rounds} ===")
        print(f"{'input Δ':<18} -> {'output Δ':<18} | verdict")
        print("-" * 60)
        for name_x, dx in input_deltas:
            for name_y, dy in output_deltas:
                v, t, w = search_impossible(rounds, dx, dy, timeout_sec=30)
                tag = ""
                if v == "IMPOSSIBLE":
                    tag = "  ★ impossible differential"
                elif v == "TIMEOUT":
                    tag = "  (timeout — try smaller word size)"
                print(f"{name_x:<18} -> {name_y:<18} | {v:<10} ({t:5.1f}s){tag}")
                if w is not None:
                    print(f"    witness: {[hex(v) for v in w]}")