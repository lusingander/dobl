Require go

Set Width 1200
Set Height 760
Set FontSize 18
Set Framerate 10
Set Theme "Catppuccin Mocha"

Output docs/screenshots/tui-rich.gif

Hide
Type "GOCACHE=/tmp/dobl-go-build go build -o /tmp/dobl ./cmd/dobl && echo DOBL_READY"
Enter
Sleep 3s
Type "clear"
Enter
Show

Type "unset NO_COLOR"
Enter
Sleep 200ms

Type "COLORTERM=truecolor TERM=xterm-256color /tmp/dobl tui --view rich testdata/error_plain.log"
Enter
Sleep 1s
Sleep 500ms
Screenshot docs/screenshots/tui-rich.png
Sleep 500ms

Type "q"
Sleep 500ms

Type "COLORTERM=truecolor TERM=xterm-256color /tmp/dobl tui --view rich --filter problems testdata/error_plain.log"
Enter
Sleep 1s
Screenshot docs/screenshots/tui-rich-problems.png
Sleep 500ms

Type "q"
