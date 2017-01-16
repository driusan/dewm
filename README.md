## dewm

dewm is a pure Go autotiling window manager. It's intended to make the text
editor [de](https://github.com/driusan/de) feel more similar to acme when windows
are spawned with the p9p plumber, with the window management handled at a window
manager level, rather than integrated into the text editor. (If you're unfamiliar
with acme, it's a text editor written by Rob Pike for Plan 9 after he got drunk
one night and forgot that he wrote "cat -v considered harmful.")

dewm is written in a [literate programming](https://en.wikipedia.org/wiki/Literate_programming)
style, in the hopes that it can inspire anyone else who's ever wanted to write
their own window manager that they can learn enough to do it. I knew next to
nothing about the X11 protocol or ICCCM conventions when I started this (and
still don't), so if I got anything wrong please feel free to either send a pull
request or email me so I can correct it. I don't want to misinform anyone who
reads the markdown source in the src/ directory.

## Basics

dewm arranges the screen into columns, and divides columns up between windows
that are in that column. Windows always spawn in the first empty column, or the
end of the last column if there are no empty columns. (If no columns exist, the
first one is created automatically.)

By default, all columns are equally sized, and each window in any given column
is equally sized, but they can be resized dynamically (see keybindings below).

## Keybindings

These keybindings are currently hardcoded, but may one day be configurable.

### Window Management
* `Alt-H/Alt-L` move the current window left or right 1 column.
* `Alt-J/Alt-K` move the current window up or down 1 window in current column
* `Ctrl-Alt-Up/Down` increase/decrease the size of the current window. Other
   windows will be dynamically resized to make sure the column still takes the
   whole height of the screen.)
* `Ctrl-Alt-Left/Right` increase/decrease the size of the column with the 
   currently active window. (Other columns will be dynamically resized to
   make up for it.)
* `Ctrl-Alt-Enter` toggle whether or not the current window is maximized.
* `Ctrl-Shift-N` create a new column 
* `Ctrl-Shift-D` delete any empty columns

### Other
* `Alt-E` spawn an xterm
* `Alt-Q` close the current window
* `Alt-Shift-Q` destroy the current window
* `Ctrl-Alt-Backspace` quit dewm

## Screenshots

This is what dewm looks like with two windows in two columns:

![dewm with two columns](https://driusan.github.io/dewm/dewm-twocolumn.png)

And this is it looks like if, after browsing a bit, you threw an xterm into
the mix.

![dewm with multiple windows in a column](https://driusan.github.io/dewm/dewm-multiwindow.png)

## Installation

The generated go files are included in this repo, so that you can install dewm
with the standard go get tool (`go get github.com/driusan/dewm`)

You should then be able to add:

```go
dewm
```

to the end of your `.xinitrc` or `.xsession` file (assuming `$GOBIN` is in your
path, otherwise you'll have to include the full the path to the executable,
wherever `go get` compiled it to.)

## License

Any code that I've written is MIT licensed. I've often used [taowm](https://github.com/nigeltao/taowm)
as a reference when figuring out how to do things. I don't think any reasonable
person would say this is a derivative work, but to be safe, LICENSE.taowm contains
the taowm (3-clause BSD) license and applies to any code that explicltly comes
from there. 