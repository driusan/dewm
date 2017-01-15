# Override Redirect and WM_TAKE_FOCUS

The X11 specification specifies that if the OverrideRedirect flag is set on a
window, we shouldn't be stealing the Substructure changes or mapping the window.

Not doing this means that things like dropdown windows in web browsers are broken,
so we need to take into account the (Override Redirect)[https://tronche.com/gui/x/xlib/window/attributes/override-redirect.html]
flag.

We map windows in Handle MapRequest, so let's add a check there. From our xproto
library documentation, the flag seems to be queryable from GetWindowAttributesReply()

Let's try checking the flag in our Handle MapRequest, since that's where we map
our windows.

If there was an error querying the attribute, we assume it's not overridden.

### "Handle MapRequest"
```go
if winattrib, err := xproto.GetWindowAttributes(xc, e.Window).Reply(); err != nil || !winattrib.OverrideRedirect {
	w := workspaces["default"]
	xproto.MapWindowChecked(xc, e.Window)
	w.Add(e.Window)
	w.TileWindows()
}
```

That didn't seem to help anything. If we look at the events getting printing
when we try clicking a window, we see that we get a ClientMessage, an EnterNotify,
and an UnmapNotify.

In our EnterNotify, we set active window. Perhaps we should check the OverrideRedirect
there, too, because we don't want our activeWindow to point to a transient window.

At the very least, let's start by test our theory by printing the attributes

### "Handle EnterNotify" +=
```go
if winattrib, err := xproto.GetWindowAttributes(xc, e.Event).Reply(); err == nil {
	log.Printf("Window attributes: %v", winattrib)
}
```

Okay, that's not it: the OverrideRedirect is false for the window is false anyways.

The other thing to consider is that, along with WM_DELETE_WINDOW, there's a
WM_TAKE_FOCUS property as part of the ICCCM convention to notify a window when
it receives focus. The WM_TAKE_FOCUS message is very similar to WM_DELETE_WINDOW,
except it explicitly forbids using time.Now() as the timestamp. It insists that
you send the time of the event that caused it to take focus. Luckily, we have
the EnterNotify event's timestamp, and since we follow focus-follows pointer
semantics, that's the only thing that causes a change in focus.

So let's define a way to send a WM_TAKE_FOCUS event based on our WM_DELETE_WINDOW
event.

### "Send WM_TAKE_FOCUS message to e.Event"
```go
xproto.SendEventChecked(
	xc,
	false,
	e.Event,
	xproto.EventMaskNoEvent,
	string(xproto.ClientMessageEvent{
		Format: 32,
		Window: *activeWindow,
		Type:   atomWMProtocols,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			uint32(atomWMTakeFocus),
			uint32(e.Time),
			0,
			0,
			0,
		}),
	}.Bytes())).Check()
```


### "Send WM_TAKE_FOCUS message if applicable"
```go
prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
	xproto.GetPropertyTypeAny, 0, 64).Reply()
if err == nil {
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		switch xproto.Atom( uint32(v[0]) | uint32(v[1]) <<8 | uint32(v[2]) <<16 | uint32(v[3]) << 24 ) {
		case atomWMTakeFocus:
			<<<Send WM_TAKE_FOCUS message to e.Event>>>
		}
	}
}
```
And call it when entering a window, but only if they follow the WM_TAKE_FOCUS protocol. 

### "Handle EnterNotify"
```go
activeWindow = &e.Event

<<<Send WM_TAKE_FOCUS message if applicable>>>
```

We'll also need to initialize the WM_TAKE_FOCUS atom along with our others:

### "Initialize Atoms" +=
```go
atomWMTakeFocus = getAtom("WM_TAKE_FOCUS")
```

### "Atom definitions" +=
```go
atomWMTakeFocus xproto.Atom
```

Now we can get popups from Firefox and Chromium, but we lost focus-follows
pointer semantics for everything else, which means we can never get back
to an xterm.

Let's adjust our logic to call manually call SetInputFocus() for things that
don't follow the WM_TAKE_FOCUS protocol. 

### "Send WM_TAKE_FOCUS message if applicable"
```go
prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
	xproto.GetPropertyTypeAny, 0, 64).Reply()
focused := false
if err == nil {
TakeFocusPropLoop:
	for v := prop.Value; len(v) >= 4; v = v[4:] {
		switch xproto.Atom( uint32(v[0]) | uint32(v[1]) <<8 | uint32(v[2]) <<16 | uint32(v[3]) << 24 ) {
		case atomWMTakeFocus:
			<<<Send WM_TAKE_FOCUS message to e.Event>>>
			focused = true
			break TakeFocusPropLoop
		}
	}
}
if !focused {
	if _, err := xproto.SetInputFocusChecked(xc, 0, e.Event, e.Time).Reply(); err != nil {
		log.Println(err)
	} 
}
```

Hopefully, we've now done enough that we can use our window manager as a daily
driver.
