import json
import subprocess
import os

# Path to the compiled Go binary, relative to the project root
BRIDGE_PATH = os.path.join("bin", "mul_bridge")

def call_mult_permuter(state: list[int], rounds: int = 1) -> list[int]:
    """
    Sends the state and round count to the Go bridge and returns the result.

    Args:
        state: A list of 4 integers representing the 256-bit state.
        rounds: Number of rounds to apply.

    Returns:
        A list of 4 integers representing the permuted state.
    """
    if len(state) != 4:
        raise ValueError("State must contain exactly 4 uint64 words")

    request = {"state": state, "rounds": rounds}
    
    # Run the Go binary, passing JSON via stdin
    result = subprocess.run(
        [BRIDGE_PATH],
        input=json.dumps(request),
        capture_output=True,
        text=True,
        check=True  # Raises an exception if the binary exits with a non-zero status
    )
    
    response = json.loads(result.stdout)
    return response["state"]

# --- Example Usage ---
if __name__ == "__main__":
    # A sample state (e.g., all zeros)
    initial_state = [0, 0, 0, 0]
    
    # Apply one round of the permuter
    new_state = call_mult_permuter(initial_state, rounds=1)
    
    print(f"Initial State: {[hex(x) for x in initial_state]}")
    print(f"State after 1 round: {[hex(x) for x in new_state]}")