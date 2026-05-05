Require go

Set Width 1200
Set Height 760
Set FontSize 18
Set Framerate 10
Set Theme "Catppuccin Mocha"

Output docs/screenshots/tui-classic.gif

Hide
Type "GOCACHE=/tmp/dobl-go-build go build -o /tmp/dobl ./cmd/dobl && echo DOBL_READY"
Enter
Sleep 3s
Type "clear"
Enter
Show

Type "/tmp/dobl tui testdata/error_plain.log"
Enter
Sleep 1s
Sleep 500ms
Screenshot docs/screenshots/tui-classic.png
Sleep 500ms

Type "q"
Sleep 500ms

Type "/tmp/dobl tui --filter problems testdata/error_plain.log"
Enter
Sleep 1s
Screenshot docs/screenshots/tui-classic-problems.png
Sleep 500ms

Type "q"
