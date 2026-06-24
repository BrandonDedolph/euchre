# VHS tape for the interactive-tutorial GIF (assets/euchre-tutorial.gif).
# Regenerate with:  vhs assets/tapes/euchre-tutorial.tape   (from repo root; needs `make build`)
#
# Same "sleep generously, then Enter" recipe as euchre.tape: the game parks on
# each human decision, so every Enter lands cleanly. In tutorial mode Enter also
# dismisses the teachable-moment popups (they only queue at idle points), so the
# single cadence drives the coached hand from end to end.
#
# Taller than the gameplay tape: tutorial mode adds the always-on coach callout
# below the status line, so the layout needs extra vertical room.

Output assets/euchre-tutorial.gif

Set Shell "bash"
Set FontSize 16
Set Width 1080
Set Height 1000
Set Padding 12
Set Framerate 24
Set PlaybackSpeed 1.0

# Launch without showing the shell prompt.
Hide
Type "./bin/euchre"
Enter
Sleep 1s
Show

# Main menu: pause, then move down to "Interactive Tutorial" (third item).
Sleep 1500ms
Down
Sleep 600ms
Down
Sleep 800ms
Enter

# Deal animation with the coach narrating, then AI bidding. Parks on the
# human's first decision (a popup and/or the bid/discard prompt).
Sleep 8s

# Drive the coached hand: each Enter dismisses a popup or commits the human's
# parked decision (order up / discard / play / acknowledge a trick).
Enter
Sleep 3s
Enter
Sleep 3500ms
Enter
Sleep 4s
Enter
Sleep 4s
Enter
Sleep 4s
Enter
Sleep 4s
Enter
Sleep 3s

# Hold on the final frame.
Sleep 1500ms
