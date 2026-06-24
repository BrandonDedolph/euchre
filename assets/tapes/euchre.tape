# VHS tape for the gameplay GIF (assets/euchre.gif).
# Regenerate with:  vhs assets/tapes/euchre.tape   (run from repo root; needs `make build` first)
#
# The game blocks on human input at every decision point, so the recipe is just
# "sleep generously, then press Enter": each Enter lands on a parked human
# decision (order up / discard / play a card / acknowledge a trick), which keeps
# the recording clean with no off-turn "Not your turn" toasts.

Output assets/euchre.gif

Set Shell "bash"
Set FontSize 16
Set Width 1080
Set Height 880
Set Padding 12
Set Framerate 24
Set PlaybackSpeed 1.0

# Launch the built binary without showing the shell prompt.
Hide
Type "./bin/euchre"
Enter
Sleep 1s
Show

# Main menu: pause, then choose "Play Game" (top item).
Sleep 1800ms
Enter

# Game setup: "Start Game" is preselected.
Sleep 1500ms
Enter

# Shuffle + deal animation, the turned card flips up, and the AI bidding plays
# out. By the end of this sleep the game is parked on the human's decision.
Sleep 8s

# The human is the dealer this deal, so the first Enter orders it up...
Enter
Sleep 2500ms
# ...and the next discards the selected card to set the contract.
Enter
Sleep 4s

# Play through a couple of tricks: each Enter plays the highlighted legal card
# or acknowledges a completed trick; the AIs play automatically in between.
Enter
Sleep 4s
Enter
Sleep 4s
Enter
Sleep 4s
Enter
Sleep 4s

# Hold on the final frame.
Sleep 1500ms
