# Initializing the program

We'll need to initialize the program such that X11 knows that our program is a
window manager before we can do much else.

The first thing we should do is create a stub of a `func main` so that we have
a valid program.

# main.go
```go
package main

import (
	<<<main.go imports>>>
)

<<<main.go globals>>>

func main() {
	<<<main implementation>>>
}

<<<main.go functions>>>
```

### "main.go imports"
```go
```

### "main.go functions"
```go
```

### "main.go globals"
```go
```

### "main implementation"
```go
```

Now, we have a program that runs, but does nothing. In our main implementation,
we want to declare that we want to be the window manager, and fail if there's
already one running.  How do we do this? 

If we look at the source for wingo or taowm, they both use `github.com/BurntSushi/xgb/xproto`
to do this. wingo in the function 'own' and taowm in 'becomeTheWM' (wingo is more
advanced in that it tries to replace the WM if one's already running, so
taowm's approach is probably better for us to look at.

taowm simply calls on xproto.ChangeWindowAttributesChecked the root window with
some parameters. If the call fails, someone else has ownership of the root window.
That approach seems simpler, so let's take that. The only problem is
`ChangeWindowAttributesChecked` needs both the X connection and the window that
we want to change attributes on (the root window, in our case) as parameters. We'll
take a similar approach to taowm and store them in globals, because I have a
feeling we'll need them a lot.

We may want to eventually have command line flags, too, so let's assume there's
a block before Initializing X to handle it. We'll leave it blank for now.

### "main.go globals"
```go
var xc *xgb.Conn
var xroot xproto.ScreenInfo
```

### "main.go imports"
```go
"github.com/BurntSushi/xgb"
"github.com/BurntSushi/xgb/xproto"
```

### "main implementation"
```go
<<<Handle Commandline Flags>>>
<<<Initialize X>>>
```

### "Handle Commandline Flags"
```go
```


### "Initialize X"
```go
<<<Connect to X Server>>>
<<<Set xroot to Root Window>>>
```
### "Connect to X Server"
```go
xcon, err := xgb.NewConn()
if err != nil {
	log.Fatal(err)
}
xc = xcon
defer xc.Close()
```

Now, how do we get the root window? taowm calls xproto.Setup, which is documented
as:

> Setup parses the setup bytes retrieved when connecting into a SetupInfo struct

In other words, it parses the X protocol connection information into a higher
level struct that we can work with. We'll take a similar approach (in fact, we
probably can't do it any other way.)

### "Set xroot to Root Window"
```go
coninfo := xproto.Setup(xc)
if coninfo == nil {
	log.Fatal("Could not parse X connection info")
}
```

But how do we get the root window itself? `*xproto.SetupInfo` has a Roots slice.
An X Server can technically have multiple root windows for each screen, but in
practice in any modern system uses the Xinerama "extension" to unify them into
one root window so you can do things like drag windows between screens.
It's probably appropriate to take an approach similar to taowm and only support
a single root window and require xinerama for now, then, like them, we can
just ensure that `len(confinfo.Roots) == 1` and safely assume coninfo.Roots[0]
is the root window that we're managing.

### "Initialize X"
```go
<<<Connect to X Server>>>
<<<Initialize Xinerama>>>
<<<Set xroot to Root Window>>>
```

### "Set xroot to Root Window" +=
```go
if len(coninfo.Roots) != 1 {
	log.Fatal("Inappropriate number of roots. Did Xinerama initialize correctly?")
}
xroot = coninfo.Roots[0]
```

For Xinerama, we'll use the BurntSushi xinerama package, again taking our inspiration
from taowm.
### "Initialize Xinerama"
```go
if err := xinerama.Init(xc); err != nil {
	log.Fatal(err)
}
```

### "main.go imports" +=
```go
"log"
"github.com/BurntSushi/xgb/xinerama"
```

Now, we've initialized our connection to the X server, except for 2 things stopping us
from trying this as a simple WM: we haven't actually taken ownership of the root window,
and we don't have an event loop looking for X events, which means our program exits
immediately (and so does X.)

### "main implementation" +=
```go
<<<X11 Event Loop>>>
```

### "Initialize X" +=
```go
<<<Take WM Ownership>>>
```

For taking ownership, we now have enough to call xproto.ChangeWindowAttributesChecked
on the root window. Once again, we'll look to taowm for inspiration: they call the
function on the root window. If the error returned is of type xp.AccessError, they
assume another WM is running. They call it with `xproto.CwEventMask` in order to get
notified of the given events happening on the root window.

Let's try the same

### "main.go functions" +=
```go
func TakeWMOwnership() error {
	<<<TakeWMOwnership Implementation>>>
}
```

### "TakeWMOwnership Implementation"
```go
return xproto.ChangeWindowAttributesChecked(
	xc,
	xroot.Root,
	xproto.CwEventMask,
	[]uint32{
		<<<Root Window Event Mask>>>
	}).Check()
```

We'll start with the obvious events that we're going to need to listen to
for user input:

### "Root Window Event Mask"
```go
    xproto.EventMaskKeyPress |
    xproto.EventMaskKeyRelease |
    xproto.EventMaskButtonPress |
    xproto.EventMaskButtonRelease,
```

### "Take WM Ownership"
```go
if err := TakeWMOwnership(); err != nil {
	if _, ok := err.(xproto.AccessError); ok {
		println("Trying to take ownership2")
		log.Fatal("Could not become the WM. Is another WM already running?")
	}
	log.Fatal(err)
}
```

That leaves our event loop. We'll just go into an infinite loop making blocking
calls to xproto.WaitForEvent and print the event to os.Stderr to make sure we're
getting the events we expect. (In fact, since we don't have any way to quit yet,
we'll die after 5 events so that we can debug.)

### "X11 Event Loop"
```go
// Main X Event loop
for n := 0; n < 5; n++ {
	xev, err := xc.WaitForEvent()
	if err != nil {
		log.Println(err)
		continue
	}
	log.Println(xev)
}
```

We don't have a cursor displayed on the screen, but at least we can quit by
clicking three times or pressing three keys (or rather 2 and a half, since
we get one for press and one for release.)

No window told X to draw a cursor, so it doesn't draw one. If we add something
like "chromium" to our `~/.xinitrc`, we'll notice that we do, in fact, get
a cursor on the screen (with the default X cursor when we're not over the 
chromium window. Other window managers (like our trusty taowm) get around
this by creating a full screen desktop window in the background, but this
is enough to answer our question of "will we get the events if they happen
over another window?" (the answer is no, we can click all we want in the
chromium window.)

We'll do something similar later, but first let's make our event loop a little
smarter, so that we can at least quit intentionally. We'll use Ctrl-Alt-Backspace
as our "Quit" combo to kill the X server, as the gods intended.

We'll start by adding a type switch and converting to an infinite loop, calling
a Handle handler if it's a keypress event.

### "X11 Event Loop"
```go
// Main X Event loop
eventloop:
for {
	xev, err := xc.WaitForEvent()
	if err != nil {
		log.Println(err)
		continue
	}
	switch e := xev.(type) {
		<<<X11 Event Loop Type Handlers>>>
		default:
			log.Println(err)
	}
	log.Println(xev)

}
```

We need a way to signify from our KeyHandler that we should be quitting. We
could add a channel, but let's just make it return an error. If the error is
non-nil, we quit.

### "X11 Event Loop Type Handlers"
```go
case xproto.KeyPressEvent:
	<<<Handle Key Press Event>>>
```

### "Handle Key Press Event"
```go
if err := HandleKeyPressEvent(e); err != nil {
	break eventloop
}
```
### "main.go functions" +=
```go
func HandleKeyPressEvent(key xproto.KeyPressEvent) error {
	<<<HandleKeyPressEvent Implementation>>>
}
```

The event Detail is the Keycode, according to the xproto documentation, so our
handler can just switch on that to handle different keystrokes.

### "HandleKeyPressEvent Implementation"
```go
switch key.Detail {
	<<<Keystroke Detail Switch>>>
	default:
		return nil
}
```

How do we know what Keysym we want? It turns out that they're defined in
`/usr/include/X11/keysymdef.h`, but it would be best if we could avoid using
CGO to include it. We'll just have to only define the ones we care about for
now. Let's put them in a different `keysym.go` file. In fact, let's put them
in a keysym.go package so that we can use them elsewhere if need be.

### keysym/keysym.go
```go
<<<keysym package doc>>>
package keysym

// Known KeySyms from /usr/include/X11/keysymdef.h
const (
	<<<Known KeySym definitions>>>
)
```


(We'll start with the whole block from the header file which includes Backspace,
since we're just copy/pasting anyways, and we'll start with a blank package doc
since that's a bit of a distraction right now.)

### "keysym package doc"
```go
```

### "Known KeySym definitions"
```go
// TTY function keys, cleverly chosen to map to ASCII, for convenience of
// programming, but could have been arbitrary (at the cost of lookup
// tables in client code).
XK_BackSpace    = 0xff08  // Back space, back char 
XK_Tab          = 0xff09
XK_Linefeed     = 0xff0a  // Linefeed, LF
XK_Clear        = 0xff0b
XK_Return       = 0xff0d  // Return, enter
XK_Pause        = 0xff13  // Pause, hold
XK_Scroll_Lock  = 0xff14
XK_Sys_Req      = 0xff15
XK_Escape       = 0xff1b
XK_Delete       = 0xffff  // Delete, rubout
```
Now, we can handle the backspace key. We'll just return an error, so that we
can test it.

### "Keystroke Detail Switch" +=
```go
case keysym.XK_BackSpace:
	<<<Handle Backspace>>>
```

### "Handle Backspace"
```go
return fmt.Errorf("Quit!")
```


### "main.go imports" +=
```go
"github.com/driusan/dewm/keysym"
"fmt"
```

And now our code doesn't compile, because keycode is a byte, and our const
overflows it. We don't actually want the keycode, which corresponds to the
physical key pressed and might change between hardware, we want the keysym,
which corresponds to the symbolic name for the key after applying all the
mappings for the keyboard known to X.

Let's get the keyboard mapping while starting X, too.

### "Initialize X" +=
```go
<<<Load KeyMapping>>>
```

The keyboard mapping seems to be straight forward to load: there's a GetKeyboardMapping
function in xproto (but then we need to call Reply() on the return value to
get the actual mapping.) We'll load all of the the keycodes from 0-255 into
a global array, so we can easily look it up.

There's a "KeysymsPerKeycode" variable too, which seems relevant, because there
will be that many elements times the number of elements we requested keycodes
in the reply, so let's define our map as a map of slices of keysyms instead.

### "Load KeyMapping"
```go
const (
	loKey = 8
	hiKey = 255
)

m := xproto.GetKeyboardMapping(xc, loKey, hiKey-loKey+1)
reply, err := m.Reply()
if err != nil {
	log.Fatal(err)
}
if reply == nil {
	log.Fatal("Could not load keyboard map")	
}

for i := 0; i < hiKey-loKey+1; i++ {
	keymap[loKey + i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode):(i+1)*int(reply.KeysymsPerKeycode)]
}
```

### "main.go globals" +=
```go
var keymap [256][]xproto.Keysym
```


### "HandleKeyPressEvent Implementation"
```go
switch keymap[key.Detail][0] {
	<<<Keystroke Detail Switch>>>
	default:
		return nil
}
```

And now we can quit by pressing backspace, but we probably want to ensure the
Ctrl-Alt modifiers are pressed, too (by printing key.State while holding alt,
we can see that the mask for Alt is "ModMask1")

### "Handle Backspace"
```go
if (key.State & xproto.ModMaskControl != 0) && (key.State & xproto.ModMask1 != 0) {
	return fmt.Errorf("Quit!")
}
return nil
```

We don't actually manage any windows yet, and we don't have a cursor unless
someone else creates a window and creates one, but at least we have the basics
of something that, used as a window manager we can start, and quit.
