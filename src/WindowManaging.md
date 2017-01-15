# Window Managing

It would be nice if our window manager was able to.. manage windows, instead of
just sitting around and waiting to quit. What does it mean to manage a window?

It means we need to provide ways to move windows around the screen, close them,
arrange them, draw border decorations, et cetera.

We can place windows by calling [ConfigureWindow](https://tronche.com/gui/x/xlib/window/configure.html)
and forcing the position, width, height, or border to change, but to do that
we'll have to know which windows we're managing. At startup, we'll need to gather
a list of all existing windows, and then we'll probably have to listen changes
in existing ones (ie. handle when a window closes.)

### "Initialize X" +=
```go
<<<Gather All Windows>>>
```

How do we gather a list of the windows? The xproto library that we're using has
a "QueryTree" function, which we can call on the root window (if the window has
children, we don't care because the parent is managing them.) Then we can
iterate through the children of the root, and keep a list of the windows
somewhere.

For now, we'll just store the children in a global slice. We'll also define
a Window type, so that we can define methods on them later if we want to. (We'll
put all the window related stuff in a different file.)

### window.go
```go
package main

import (
	<<<window.go imports>>>
)

<<<window.go globals>>>

<<<window.go functions>>>
```

### "window.go functions"
```go
```

### "window.go globals" +=
```go
<<<ManagedWindow type>>>

var KnownWindows []ManagedWindow
```

### "ManagedWindow type"
```go
type ManagedWindow xproto.Window
```

### "window.go imports"
```go
"github.com/BurntSushi/xgb/xproto"
```

Now that we've gotten that overhead out of the way, the actual QueryTree call:

### "Gather All Windows"
```go
tree, err := xproto.QueryTree(xc, xroot.Root).Reply()
if err != nil {
	log.Fatal(err)
}
if tree != nil {
	<<<Generate list of known windows>>>

}
```

And we'll populate the KnownWindows with it.

### "Generate list of known windows"
```go
KnownWindows = make([]ManagedWindow, 0, len(tree.Children))
for _, c := range tree.Children {
	<<<Attempt to manage window c>>>
}
```

For attempting to manage c, we'll start by seeing what happens if we just
send a ConfigureWindow event to set a border of size 2 and see what happens. If
there was no error, we'll add it to KnownWindows.

### "Attempt to manage window c"
```go
if err := xproto.ConfigureWindowChecked(
	xc,
	c,
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err == nil {
	KnownWindows = append(KnownWindows, ManagedWindow(c))
}
```

It seems to have worked, which means we can probably set the width, height,
X, and Y too.

Our plan is to make the WM act like the Plan 9 acme text editor where there's
(generally) 2 columns. For now, we'll just make it so that there's *always*
2 columns. Windows will always be managed to be equally spaced in the second
column and can only be spawned in the first column if there's nothing else
there. In fact, we'll probably want to generalize this to different screens,
and maybe have named workspaces that can be toggled per screen. So instead
of keeping track of known windows, let's keep track of workspaces. Windows
will get added to workspaces. Workspaces will have a slice of columns, and
try and intelligently add new windows to the right slice. We'll get rid of the
KnownWindows global, because we'll keep track of things at the workspace level.
Instead, we'll keep a map of workspaces. When starting up, we'll put them into
a workspace named "default"

### "window.go globals"
```go
<<<ManagedWindow type>>>
<<<Workspace type>>>

var workspaces map[string]*Workspace
```

### "Workspace type"
```go
type Workspace struct{
	Columns [][]ManagedWindow
}
```

When generating the list, we add it to the default workspace now:

### "Generate list of known windows"
```go
workspaces = make(map[string]*Workspace)
defaultw := &Workspace{}
for _, c := range tree.Children {
	if err := defaultw.Add(c); err != nil {
		log.Println(err)
	}

}
workspaces["default"] = defaultw
```

We should probably define that "Add" method that we just used.


### "window.go functions" +=
```go
func (w *Workspace) Add(win xproto.Window) error {
	<<<Add Window to Workspace>>>
}
```

We'll use the logic that we decided on above to try and guess the right column.

### "Add Window to Workspace"
```go
if err := xproto.ConfigureWindowChecked(
	xc,
	xproto.Window(win),
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err != nil {
	return err
}
switch len(w.Columns) {
case 0:
	w.Columns = [][]ManagedWindow{
		{ ManagedWindow(win) },
	}
case 1:
	if len(w.Columns[0]) == 0 {
		// No active window in first column, so use it.
		w.Columns[0] = append(w.Columns[0], win)
	} else {
		// There's something in the primary column, so create a new one.
		w.Columns = append(w.Columns, []ManagedWindow{ManagedWindow(win)})
	}
default:
	// Add to the last column
	i := len(w.Columns)-1
	w.Columns[i] = append(w.Columns[i], win)
}
return nil
```

Now, after we're finished gathering our managed windows, we should also place
them on the screen. Let's assume there's a TileWindows function on the workspace,
which may return an error. `TileWindows()` will move and resize windows as
needed.

### "Generate list of known windows" +=
```go
if err := defaultw.TileWindows(); err != nil {
	log.Println(err)
}

```

Except to tile them, we need to know the dimensions of the screen that we're
tiling into. How do we get that? The xinerama package has a QueryScreens (and
the QueryScreensReply) method, which gives us a list of screens, each with
their own Width and Height. We can store the ScreenInfo of the screen that
the workspace is attached to the screen (but we should use a pointer, because
it can be nil.) While we're changing things, maybe we should make Column a type
instead of having a slice of slices to simplify our code a little.

### "Workspace type"
```go
<<<Column type>>>
type Workspace struct{
	Screen *xinerama.ScreenInfo
	Columns []Column
}
```

### "Column type"
```go
type Column []ManagedWindow
```

Now, as long as the workspace knows the screen that it's managing, it can
calculate the x size of each column, and then pass it to the Column to calculate
the y size of each ManagedWindow and lay them out. Let's define our stubs.

### "window.go functions" +=
```go
// TileWindows tiles all the windows of the workspace into the screen that
// the workspace is attached to.
func (w *Workspace) TileWindows() error {
	<<<Tile Workspace Windows Implementation>>>
}
```

### "window.go functions" +=
```go
// TileColumn sends ConfigureWindow messages to tile the ManagedWindows
// Using the geometry of the parameters passed
func (c Column) TileColumn(xstart, colwidth, colheight uint32) error {
	<<<Column TileColumn implementation>>>
}
```

The TileWindows implementation should be straight forward, since it just calls
TileColumn and returns an error if there's no screen. We'll just take the screen,
and divided it up equally for now.

### "Tile Workspace Windows Implementation"
```go
if w.Screen == nil {
	return fmt.Errorf("Workspace not attached to a screen.")
}

n := uint32(len(w.Columns))
size := uint32(w.Screen.Width) / n
var err error
for i, c := range w.Columns {
	if err != nil {
		// Don't overwrite err if there's an error, but still
		// tile the rest of the columns instead of returning.
		c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
	} else {
		err = c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
	}
}
return err
```

and the TileColumn implementation is similar, except it sends a ConfigureWindow
event.

### "Column TileColumn implementation"
```go
n := uint32(len(c))
height := colheight / n
var err error
for i, win := range c {
	if werr := xproto.ConfigureWindowChecked(
		xc,
		xproto.Window(win),
		xproto.ConfigWindowX|
			xproto.ConfigWindowY|
			xproto.ConfigWindowWidth|
			xproto.ConfigWindowHeight,
		[]uint32{
			xstart,
			uint32(i) * height,
			colwidth,
			height,
		}).Check(); werr != nil {
		err = werr
	}
}
return err
```

### "window.go imports" +=
```go
"github.com/BurntSushi/xgb/xinerama"
"fmt"
```

We'll have to refactor our AddWindow to Workspace a little to use our new types.

### "Add Window to Workspace"
```go
if err := xproto.ConfigureWindowChecked(
	xc,
	xproto.Window(win),
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err != nil {
	return err
}
switch len(w.Columns) {
case 0:
	w.Columns = []Column{
		{ win },
	}
case 1:
	if len(w.Columns[0]) == 0 {
		// No active window in first column, so use it.
		w.Columns[0] = append(w.Columns[0], win)
	} else {
		// There's something in the primary column, so create a new one.
		w.Columns = append(w.Columns, Column{win})
	}
default:
	// Add to the last column
	i := len(w.Columns)-1
	w.Columns[i] = append(w.Columns[i], win)
}
return nil
```

Now, we're getting a workspace not attached to a screen error, which makes
sense because we never did attach it to a screen..

We'll have to query the screens before we gather the windows, obviously, and
then we'll have to make the number of workspaces that we have screens.

Putting together everything from Initialize.md and WindowManaging.md, our
Initialize X implementation becomes:

### "Initialize X"
```go
<<<Connect to X Server>>>
<<<Initialize Xinerama>>>
<<<Set xroot to Root Window>>>
<<<Take WM Ownership>>>
<<<Load KeyMapping>>>
<<<Gather All Windows>>>
```

It doesn't really matter where we query the screens, so let's do it right
after initializing xinerama.

### "Initialize X"
```go
<<<Connect to X Server>>>
<<<Initialize Xinerama>>>
<<<Query Attached Screens>>>
<<<Set xroot to Root Window>>>
<<<Take WM Ownership>>>
<<<Load KeyMapping>>>
<<<Gather All Windows>>>
```

We probably want to store them in a global that we can use whenever we want,
too.

### "main.go globals" +=
```go
var attachedScreens []xinerama.ScreenInfo
```

Querying Attached Screens is fairly simple, we just use the function from the
xinerama package and store what we care about after checking the error.

### "Query Attached Screens"
```go
if r, err := xinerama.QueryScreens(xc).Reply(); err != nil {
	log.Fatal(err)
} else {
	attachedScreens = r.ScreenInfo
}
```

We still don't have a screen attached to the workspace, though, so let's just
put all the windows onto the first screen while starting up. After gathering
the windows, we'll just set the default workspace screen to the first screen.

### "Generate list of known windows"
```go
workspaces = make(map[string]*Workspace)
defaultw := &Workspace{}
for _, c := range tree.Children {
	
	if err := defaultw.Add(c); err != nil {
		log.Println(err)
	}

}

if len(attachedScreens) > 0 {
	defaultw.Screen = &attachedScreens[0]
}

workspaces["default"] = defaultw

if err := defaultw.TileWindows(); err != nil {
	log.Println(err)
}
```

That seems to have not worked. If we print the len(attachedScreens), we see that
it's returning 0. xinerama seems to not be reliable in non-multiheaded setups
for returning the details about screens?

Regardless, if we look into xproto we see that there's a lower level
xproto.ScreenInfo which has `WidthInPixels` and `HeightInPixels`. We can use
these to make a fake xinerama.ScreenInfo when xinerma doesn't return anything.
How do we get an `xproto.ScreenInfo`? `ScreenInfo` seems to come from `SetupInfo`,
`SetupInfo` comes from calling Setup(), which is documented as:

> Setup parses the setup bytes retrieved when connecting into a SetupInfo struct. 

(That sounds like something we should be doing anyways.)

So then, we initialize as:

### "Initialize X"
```go
<<<Connect to X Server>>>
<<<Get Setup Information>>>
<<<Initialize Xinerama>>>
<<<Query Attached Screens>>>
<<<Set xroot to Root Window>>>
<<<Take WM Ownership>>>
<<<Load KeyMapping>>>
<<<Gather All Windows>>>
```

### "Get Setup Information"
```go
setup := xproto.Setup(xc)
if setup == nil || len(setup.Roots) < 1 {
	log.Fatal("Could not parse SetupInfo.")
}
```

Then we just need to fake our xinerama.ScreenInfo if applicable:

### "Query Attached Screens"
```go
if r, err := xinerama.QueryScreens(xc).Reply(); err != nil {
	log.Fatal(err)
} else {
	if len(r.ScreenInfo) == 0 {
		attachedScreens = []xinerama.ScreenInfo{
			xinerama.ScreenInfo{
				Width:  setup.Roots[0].WidthInPixels,
				Height: setup.Roots[0].HeightInPixels,
			},
		}

	} else {
		attachedScreens = r.ScreenInfo
	}
}
```

And now.. success! The active windows get automatically tiled when we start up.
If we try quiting and restarting X with 0, 1, or multiple windows started in
our `.xinitrc` before starting our window manager, we see they get tiled as
we expected.

We now just need to (re)autotile our workspace(s) if a window is created or
deleted.

If you recall when we initialized our window manager, we listened for key press,
key release, button press, and button release events (which is why that's what
gets printed.)

The list of available types we can mask for is [here](https://tronche.com/gui/x/xlib/events/mask.html),
one of them must be for window creation and deletion. The closest one we can
see is "StructureNotifyMask", which reports "any change in window structure."
Clicking on the link tells us that means "CirculateNotify", "ConfigureNotify",
"DestroyNotify", "GravityNotify", "MapNotify", "ReparentNotify", and "UnmapNotify"

In X11 terms, windows are "Destroyed", not "Closed", so DestroyNotify seems
promising.

### "Root Window Event Mask"
```go
xproto.EventMaskKeyPress |
xproto.EventMaskKeyRelease |
xproto.EventMaskButtonPress |
xproto.EventMaskButtonRelease |
xproto.EventMaskStructureNotify,
```

If we use a nested X server like [Xephyr](https://en.wikipedia.org/wiki/Xephyr)
to spawn a few windows while our wm is running (which is useful for debugging
anyways), we see that we're not getting any events. The StructureNotify event
mask on the root window is telling us if the structure of *the root window*
gets changed.

What we need is to know if any children's structure changed.

Looking to our trusty taowm yet again, we see that when managing windows it
does:

```go
check(xp.ChangeWindowAttributesChecked(xConn, xWin, xp.CwEventMask,
	[]uint32{xp.EventMaskEnterWindow | xp.EventMaskStructureNotify},
))
```

(check is an internal method of taowm we're not concerned about right now.)

So, when managing a window, we need to tell the X server to change the attributes
so that it notifies us if that window changes. (They're looking for if the mouse
moves over it too, presumably to implement focus-follows pointer, but for now
we're only concerned about the structure.)

Let's send our own ChangeWindowAttributes message. The obvious place to do it
is in our `workspace.Add()` method.

### "Add Window to Workspace"
```go
// Ensure that we can manage this window.
if err := xproto.ConfigureWindowChecked(
	xc,
	win,
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err != nil {
	return err
}

// Get notifications when this window is deleted.
if err := xproto.ChangeWindowAttributesChecked(
	xc,
	win,
	xproto.CwEventMask,
	[]uint32{xproto.EventMaskStructureNotify}).Check(); err != nil {
	return err
}

switch len(w.Columns) {
case 0:
	w.Columns = []Column{
		{ ManagedWindow(win) },
	}
case 1:
	if len(w.Columns[0]) == 0 {
		// No active window in first column, so use it.
		w.Columns[0] = append(w.Columns[0], ManagedWindow(win))
	} else {
		// There's something in the primary column, so create a new one.
		w.Columns = append(w.Columns, Column{ManagedWindow(win)})
	}
default:
	// Add to the last column
	i := len(w.Columns)-1
	w.Columns[i] = append(w.Columns[i], MangedWindow(win))
}
return nil
```

Now, when we close a window, we see that we get an UnmapNotify and a DestroyNotify
event. For now, we'll just concern ourselves with the "Destroy" event, to ensure
that we don't accidentally try and delete the window twice by processing both.

Since the window was destroyed, we'll be removing it from the workspace. We
don't know which workspaces contain the window, so we'll just remove it from
all of them. We can even use goroutines so that they can be removed in parallel.

### "X11 Event Loop Type Handlers" +=
```go
case xproto.DestroyNotifyEvent:
	<<<DestroyEvent Handler>>>
```

### "DestroyEvent Handler"
```go
<<<Remove Window From All Workspaces>>>
```
### "Remove Window From All Workspaces"
```go
for _, w := range workspaces {
	go w.RemoveWindow(ManagedWindow(e.Window))
}
```

Since we're going to be modifying the underlying slice(s) concurrently, we should
probably add a Mutex to our workspace when we add or remove windows. (In fact,
we should also unexport columns to make sure that the mutex is always used and
we can only do it safely.)

### "Workspace type"
```go
<<<Column type>>>
type Workspace struct{
	Screen *xinerama.ScreenInfo
	columns []Column

	mu *sync.Mutex
}
```

### "Add Window to Workspace"
```go
// Ensure that we can manage this window.
if err := xproto.ConfigureWindowChecked(
	xc,
	xproto.Window(win),
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err != nil {
	return err
}

// Get notifications when this window is deleted.
if err := xproto.ChangeWindowAttributesChecked(
	xc,
	xproto.Window(win),
	xproto.CwEventMask,
	[]uint32{xproto.EventMaskStructureNotify}).Check(); err != nil {
	return err
}

w.mu.Lock()
defer w.mu.Unlock()

switch len(w.columns) {
case 0:
	w.columns = []Column{
		{ win },
	}
case 1:
	if len(w.columns[0]) == 0 {
		// No active window in first column, so use it.
		w.columns[0] = append(w.columns[0], ManagedWindow(win))
	} else {
		// There's something in the primary column, so create a new one.
		w.columns = append(w.columns, Column{ManagedWindow(win)})
	}
default:
	// Add to the last column
	i := len(w.columns)-1
	w.columns[i] = append(w.columns[i], ManagedWindow(win))
}
return nil
```

And we'll have to create the mutex when we create the workspace, since it's
a pointer. (We should probably make a CreateWorkspace() constructor function,
but for now we'll just copy/paste the only block that we create it from and
change it there.)

### "Generate list of known windows"
```go
workspaces = make(map[string]*Workspace)
defaultw := &Workspace{mu: &sync.Mutex{}}
for _, c := range tree.Children {
	if err := defaultw.Add(c); err != nil {
		log.Println(err)
	}

}

if len(attachedScreens) > 0 {
	defaultw.Screen = &attachedScreens[0]
}

workspaces["default"] = defaultw

if err := defaultw.TileWindows(); err != nil {
	log.Println(err)
}
```

### "window.go imports" +=
```go
"sync"
```

### "main.go imports" +=
```go
"sync"
```

And fix the compiler error since we changed the variable name from Columns to
columns:

### "Tile Workspace Windows Implementation"
```go
if w.Screen == nil {
	return fmt.Errorf("Workspace not attached to a screen.")
}

n := uint32(len(w.columns))
size := uint32(w.Screen.Width) / n
var err error
for i, c := range w.columns {
	if err != nil {
		// Don't overwrite err if there's an error, but still
		// tile the rest of the columns instead of returning.
		c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
	} else {
		err = c.TileColumn(uint32(i)*size, size, uint32(w.Screen.Height))
	}
}
return err
```

Finally, we should define the RemoveWindow method that we've used, now
that we have everything in place to do it in a thread-safe manner:

### "window.go functions" +=
```go
// RemoveWindow removes a window from the workspace. It returns
// an error if the window is not being managed by w.
func (wp *Workspace) RemoveWindow(w xproto.Window) error {
	<<<RemoveWindow implementation>>>
}
```

### "RemoveWindow implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()

for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column {
		if w == xproto.Window(candwin) {
			idx = i
			break
		}
	}
	if idx != -1 {
		// Found the window at at idx, so delete it and return.
		// (I wish Go made it easier to delete from a slice.)
		wp.columns[colnum] = append(column[0:idx], column[idx+1:]...)
		return nil
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

We still need to re-call TileWindows if it's removed. Let's update our
goroutine to call it from a closure if RemoveWindow succeeds.

### "Remove Window From All Workspaces"
```go
for _, w := range workspaces {
	go func() {
		if err := w.RemoveWindow(e.Window); err == nil {
			w.TileWindows()
		}
	}()
}
```

Now individual columns are getting managed, but we just exposed a bug where
when we delete the last window in a column, we crash with a divide by zero error
(from trying to calculate the height by dividing by the number of windows.) Our
workspace tiling will have a similar bug for calculating the width, so let's fix
them both.

### "Column TileColumn implementation"
```go
n := uint32(len(c))
if n == 0 {
	return nil
}

height := colheight / n
var err error
for i, win := range c {
	if werr := xproto.ConfigureWindowChecked(
		xc,
		xproto.Window(win),
		xproto.ConfigWindowX|
			xproto.ConfigWindowY|
			xproto.ConfigWindowWidth|
			xproto.ConfigWindowHeight,
		[]uint32{
			xstart,
			uint32(i) * height,
			colwidth,
			height,
		}).Check(); werr != nil {
		err = werr
	}
}
return err
```

Now, when we close something in a column, the rest of the column is automatically
resized horizontally, but if we close a column, the column isn't deleted. (This
probably isn't a big deal, and is in fact pretty close to what acme does.)

But we're still not getting events when a window is created. Why not?

According to the [reference](https://tronche.com/gui/x/xlib/window/map.html) that
we've been using fairly often:

> A window manager may want to control the placement of subwindows. If 
> SubstructureRedirectMask has been selected by a window manager on a parent
> window (usually a root window), a map request initiated by other clients on 
> a child window is not performed, and the window manager is sent a MapRequest
> event. However, if the override-redirect flag on the child had been set to
> True (usually only on pop-up menus), the map request is performed.

It sounds like we need to add SubstructureRedirectMask to our mask on the root
window, so let's do that.

### "Root Window Event Mask"
```go
xproto.EventMaskKeyPress |
xproto.EventMaskKeyRelease |
xproto.EventMaskButtonPress |
xproto.EventMaskButtonRelease |
xproto.EventMaskStructureNotify |
xproto.EventMaskSubstructureRedirect,
```

Now when we try and create a window, we get a ConfigureRequest event from the
window asking us to place it on the screen, so let's handle the request by
adding the window to the "default" workspace. (Later on, we'll probably want to
be smarter about adding it to the proper workspace, but for now we only have one
anyways with no way to create more.)
 
### "X11 Event Loop Type Handlers" +=
```go
case xproto.ConfigureRequestEvent:
	<<<Handle ConfigureRequest>>>
```

### "Handle ConfigureRequest"
```go
w := workspaces["default"]
w.Add(ManagedWindow(e.Window))
```

Still no go. If we look into it a little more, we find that ConfigureRequests
need to be responded to with a ConfigureNotify event, so that the window knows
that it's configuration request has been acted upon. (We can even find this
comment in wingo:)

```go
// As per ICCCM 4.1.5, a window that has been moved but not resized must
// receive a synthetic ConfigureNotify event.
```

But that doesn't seem right, because we *are* resizing the window when we
autotile it.

If we look into taowm instead of wingo, we see that they send a ConfigureNotify
event if the window is already known, and otherwise sends a ConfigureWindow
event. They don't actively manage the window until getting a MapRequest
(which makes sense, because that's what a MapRequest is.)

Why don't we try just echoing back the event as a ConfigureNotify when we get
the ConfigureRequest and see what happens? This will mean that the window gets
positioned in more or less the same way it would if there was no window manager
running, and then when we get a MapRequest we can start actively managing it.

### "Handle ConfigureRequest"
```go
ev := xproto.ConfigureNotifyEvent{
	Event:            e.Window,
	Window:           e.Window,
	AboveSibling:     0,
	X:                e.X,
	Y:                e.Y,
	Width:            e.Width,
	Height:           e.Height,
	BorderWidth:      0,
	OverrideRedirect: false,
}
xproto.SendEventChecked(xc, false, e.Window, xproto.EventMaskStructureNotify, string(ev.Bytes()))
```

### "X11 Event Loop Type Handlers" +=
```go
case xproto.MapRequestEvent:
	<<<Handle MapRequest>>>
```

### "Handle MapRequest"
```go
w := workspaces["default"]
xproto.MapWindowChecked(xc, e.Window)
w.Add(e.Window)
```

We now have an unusual situation where the xterms we spawn appear, but only after
we close an xterm that already exists is closed. When we look into our
DestroyEventNotify handler, the reason is pretty obvious: we call TileWindows
after removing a window, but not after mapping it. Oops.

### "Handle MapRequest" +=
```go
w.TileWindows()
```

And our windows now appear when they're created! We have the basics of a very
simple autotiled window manager.

But we also notice that there's a bug with our logic of adding a window when the
primary workspace is empty. If we go back up and check, we'll notice that we
only do the "is there anything in the first column?" check when there's only
1 column. We should check for an empty column to use in the `default:` case too,
otherwise we'll forever be stuck with only using the secondary column.

### "Add Window to Workspace"
```go
// Ensure that we can manage this window.
if err := xproto.ConfigureWindowChecked(
	xc,
	xproto.Window(win),
	xproto.ConfigWindowBorderWidth,
	[]uint32{
		2,
	}).Check(); err != nil {
	return err
}

// Get notifications when this window is deleted.
if err := xproto.ChangeWindowAttributesChecked(
	xc,
	xproto.Window(win),
	xproto.CwEventMask,
	[]uint32{
	<<<Window Event Mask>>>
	},
	).Check(); err != nil {
	return err
}

w.mu.Lock()
defer w.mu.Unlock()

switch len(w.columns) {
case 0:
	w.columns = []Column{
		{ ManagedWindow(win) },
	}
case 1:
	if len(w.columns[0]) == 0 {
		// No active window in first column, so use it.
		w.columns[0] = append(w.columns[0], ManagedWindow(win))
	} else {
		// There's something in the primary column, so create a new one.
		w.columns = append(w.columns, Column{ ManagedWindow(win) })
	}
default:
	// Add to the first empty column we can find, and shortcircuit out
	// if applicable.
	for i, c := range w.columns {
		if len(c) == 0 {
			w.columns[i] = append(w.columns[i], ManagedWindow(win))
			return nil
		}
	}

	// No empty columns, add to the last one.
	i := len(w.columns)-1
	w.columns[i] = append(w.columns[i], ManagedWindow(win))
}
return nil
```

### "Window Event Mask"
```
xproto.EventMaskStructureNotify,
```

We could also improve the destroy logic to remove an empty column when there's
no windows left in it, but since our model is acme and acme doesn't behave that
way, maybe we shouldn't.

At any rate, we're getting closer to being able to dogfood our WM. All we really
need is a way to spawn programs (at the very least, an xterm that we can use to
run other programs), so that can be our next exercise in Keyboard.md.
