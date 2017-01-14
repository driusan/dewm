## Moving Windows

Now that we're dogfooding our window manager (or at least, I am) it's fairly
obvious that, while autotiling works pretty well, there's a couple minor problems.

1. We don't always want windows arranged in the same order they spawned. Sometimes
   you want to move things around in a different order, or to move them between
   columns.
2. We don't always want perfectly sized columns (or tiles.) Sometimes you want
   some leeway to make something bigger (or smaller)

## Window Moving, Part 1
Let's start with the first.

I'm a longtime vi user (though now I use my own text editor, [de](https://github.com/driusan/de)),
so I like using hjkl for moving things around. Ideally, we could use ctrl-hjkl
for moving windows around, but that's too likely to conflict with other applications
(in fact, in de it's used to select text.) We've had good luck with alt, though,
since I can't think of the last time I used a program that depended on an alt-key
combination, so let's use alt-hjkl. (It's a little less ergonomic since the alt
key isn't on the home row beside the "a" key, but it's probably not unusable.)

### "Grabbed Key List" +=
```go
{
	sym:       keysym.XK_h,
	modifiers: xproto.ModMask1,
},
{
	sym:       keysym.XK_j,
	modifiers: xproto.ModMask1,
},
{
	sym:       keysym.XK_k,
	modifiers: xproto.ModMask1,
},
{
	sym:       keysym.XK_l,
	modifiers: xproto.ModMask1,
},
```

Now, all we need to do is add a key handler which swaps it with the window
beside it and calls TileWindows.

This probably won't work once we add window resizing, but it's a start.

Let's start by adding up/down/left/right handlers to the workspace. We can't
add it to ManagedWindow, because the Window has no knowledge of what workspace
it's in.

It might be cleaner to put these in their own file, so let's do that.

### workspace.go
```go
package main

import (
	<<<workspace.go imports>>>
)

<<<workspace.go globals>>>

<<<workspace.go functions>>>
```


### "workspace.go functions"
```go
func (wp *Workspace) Up(w ManagedWindow) error {
	<<<Up implementation>>>
}

func (wp *Workspace) Down(w ManagedWindow) error {
	<<<Down implementation>>>
}
func (wp *Workspace) Left(w ManagedWindow) error {
	<<<Left implementation>>>
}
func (wp *Workspace) Right(w ManagedWindow) error {
	<<<Right implementation>>>
}
```

### "workspace.go globals"
```go
```

The workspace will need to iterate through the windows trying to find win.
(If it gets too slow we can add a map[ManagedWindow]struct{Col, Row} to speed it
up, but I suspect we'll never even notice if we just scan all the workspace
windows every time.)

This is very similar to what we do when removing a window, except we swap rather
than remove, so let's base our code on that.

### "Up implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Swap wp[colnum][idx] with wp[colnum][idx-1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "workspace.go imports"
```go
"fmt"
```

Swapping is straight-forward, we just need to check that i > 0 first.

### "Swap wp[colnum][idx] with wp[colnum][idx-1]"
```go
if idx == 0 {
	return fmt.Errorf("Window already at top of column")
}
wp.columns[colnum][idx], wp.columns[colnum][idx-1] = wp.columns[colnum][idx-1], wp.columns[colnum][idx]
```

Down is the same, except it's i+1, and we need to check it's not
past the end of the column.

### "Down implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Swap wp[colnum][idx] with wp[colnum][idx+1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Swap wp[colnum][idx] with wp[colnum][idx+1]"
```go
if idx >= len(wp.columns[colnum])-1 {
	return fmt.Errorf("Window already at bottom of column")
}
wp.columns[colnum][idx], wp.columns[colnum][idx+1] = wp.columns[colnum][idx+1], wp.columns[colnum][idx]
```

For left and right if we find it we'll just remove the window, and then add it
to the column beside it. 

### "Left implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Remove wp[colnum][idx] and move to wp[colnum-1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

We can reuse the remove code from RemoveWindow, we just need to make sure we
add a guard first.

### "Remove wp[colnum][idx] and move to wp[colnum-1]"
```
if colnum <= 0 {
	return fmt.Errorf("Already in first column of workspace.")
}

// Found the window at at idx, so delete it and return.
// (I wish Go made it easier to delete from a slice.)
wp.columns[colnum] = append(column[0:idx], column[idx+1:]...)
wp.columns[colnum-1] = append(wp.columns[colnum-1], w)
```

### "Right implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Remove wp[colnum][idx] and move to wp[colnum+1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Remove wp[colnum][idx] and move to wp[colnum+1]"
```go
if colnum >= len(wp.columns)-1 {
	return fmt.Errorf("Already at end of workspace.")
}

// Found the window at at idx, so delete it and return.
// (I wish Go made it easier to delete from a slice.)
wp.columns[colnum] = append(column[0:idx], column[idx+1:]...)
wp.columns[colnum+1] = append(wp.columns[colnum+1], w)
```

Now, we just need to add our keyhandlers.

### "Keystroke Detail Switch" +=
```go
case keysym.XK_h:
	<<<Handle h key>>>
case keysym.XK_j:
	<<<Handle j key>>>
case keysym.XK_k:
	<<<Handle k key>>>
case keysym.XK_l:
	<<<Handle l key>>>
```

We could try and extract the window from the key event in our handler, but
we already have the activeWindow pointer, so let's just use that to pass to
our workspace.. except we also don't have the active workspace. We can just
try and move it up in all of them, because the window can't be in more than
one workspace. In fact, we don't even have a way to create multiple workspaces
yet, so let's just do that. It's a similar approach to what we take in RemoveWindow,
and it seemed to work well enough there.

### "Handle h key"
```go
if activeWindow != nil && key.State == xproto.ModMask1 {
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Left(ManagedWindow(*activeWindow)); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
}
return nil
```

### "Handle j key"
```go
if activeWindow != nil && key.State == xproto.ModMask1 {
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Down(ManagedWindow(*activeWindow)); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
}
return nil
```

### "Handle k key"
```go
if activeWindow != nil && key.State == xproto.ModMask1 {
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Up(ManagedWindow(*activeWindow)); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}

}
return nil
```
### "Handle l key"
```go
if activeWindow != nil && key.State == xproto.ModMask1 {
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Right(ManagedWindow(*activeWindow)); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
}
return nil
```

## Resizing Windows

Now that we can move windows, it would be nice if they didn't always have to
be 100% equally sized.

To fix that, we'll have to add some kind of a delta to both columns, and windows.

### "Column type"
```go
type Column struct {
	Windows []ManagedWindow
	SizeDelta int
}
```

### "ManagedWindow type"
```go
type ManagedWindow struct{
	xproto.Window
	SizeDelta int
}
```

We'll just add SizeDelta to the Columns when autotiling them.. but that has a
problem, because then other columns will be pushed off the screen, instead of
resizing them to take up less space when we increase a column, so we'll need to
sum up the SizeDeltas, subtract that from the width, and divide the remaining
space equally.

The next problem we're going to have is: what is the active column? We have a
pointer to the activeWindow, not the activeColumn. We could add a pointer to
the ManagedWindow type, but so far our approach of just going through all of
the windows hasn't had any significant performance problems, so let's just
keep doing that instead of risking the pointer going out of sync because we
forget to update it somewhere. Once again, we can add it later if need be, but
for now it's better to have less possibility for bugs until we find it becomes
a bottleneck.

(Similar reasoning applies to windows as columns, with width and height inverted)

We'll define a simple Resize method on *Column and *ManagedWindow in case this
ever gets more complicated.

### "workspace.go functions" +=
```go
func (c *Column) Resize(delta int) {
	<<<Column Resize implementation>>>
}
```

### "Column Resize implementation"
```go
c.SizeDelta += delta
```
### "window.go functions" +=
```go
func (w *ManagedWindow) Resize(delta int) {
	<<<ManagedWindow Resize implementation>>>
}
```

### "ManagedWindow Resize implementation"
```go
w.SizeDelta += delta
```

Now, we can redefine our workspace tile windows implementation as discussed 
above.

### "Tile Workspace Windows Implementation"
```go
if w.Screen == nil {
	return fmt.Errorf("Workspace not attached to a screen.")
}

n := uint32(len(w.columns))
if n == 0 {
	return fmt.Errorf("No columns to tile")
}
var totalDeltas int
for _, c := range w.columns {
	totalDeltas += c.SizeDelta
}

size := uint32(int(w.Screen.Width)-totalDeltas) / n
var err error

// Keep track of the already incorporated deltas, to add to xstart
// for the column.TileWindow call
usedDeltas := 0
for i, c := range w.columns {
	if err != nil {
		// Don't overwrite err if there's an error, but still
		// tile the rest of the columns instead of returning.
		c.TileColumn(uint32((i*int(size))+usedDeltas), uint32(int(size)+c.SizeDelta), uint32(w.Screen.Height))
	} else {
		err = c.TileColumn(uint32((i*int(size))+usedDeltas), uint32(int(size)+c.SizeDelta), uint32(w.Screen.Height))
	}
	usedDeltas += c.SizeDelta
}
return err
```

and our similar logic for windows:

### "Column TileColumn implementation"
```go
n := uint32(len(c.Windows))
if n == 0 {
	return nil
}

var totalDeltas int
for _, win := range c.Windows {
	totalDeltas += win.SizeDelta
}

heightBase := (int(colheight)-totalDeltas) / int(n)
usedDeltas := 0
var err error
for i, win := range c.Windows {
	if werr := xproto.ConfigureWindowChecked(
		xc,
		win.Window,
		xproto.ConfigWindowX|
			xproto.ConfigWindowY|
			xproto.ConfigWindowWidth|
			xproto.ConfigWindowHeight,
		[]uint32{
			xstart,
			uint32((i * heightBase) + usedDeltas),
			colwidth,
			uint32(heightBase + win.SizeDelta),
		}).Check(); werr != nil {
		err = werr
	}
	usedDeltas += win.SizeDelta
}
return err
```

Finally, we'll have to bind a key to something that adjusts the SizeDelta for
the current column. Let's use Ctrl+Shift+H and Ctrl+Shift+L, because I still
have some muscle memory from those keystrokes from when I used to use Ion as
my window manager.

### "Grabbed Key List" +=
```go
{
	sym:       keysym.XK_h,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
{
	sym:       keysym.XK_l,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
{
	sym:       keysym.XK_j,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
{
	sym:       keysym.XK_k,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
```

We'll refactor our Handle h while we're at it, to be a little cleaner.

### "Handle h key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
		<<<Handle Alt-H>>>
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-H>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle Alt-H"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		if err := wp.Left(ManagedWindow{*activeWindow, 0}); err == nil {
			wp.TileWindows()
		}
	}(wp)
}
```

### "Handle Control-Shift-H"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for i, c := range wp.columns {
			for _, win := range c.Windows {
				if win.Window == *activeWindow {
					<<<Shrink Column>>>
					return
				}
			}
		}
	}(wp)
}
```

### "Shrink Column"
```go
wp.columns[i].Resize(10)
wp.TileWindows()
```

And we'll do similarly for our other keys.

### "Handle l key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
		<<<Handle Alt-L>>>
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-L>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle Alt-L"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		if err := wp.Right(ManagedWindow{*activeWindow, 0}); err == nil {
			wp.TileWindows()
		}
	}(wp)
}

```
### "Handle Control-Shift-L"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for i, c := range wp.columns {
			for _, win := range c.Windows {
				if win.Window == *activeWindow {
					<<<Grow Column>>>
					return
				}
			}
		}
	}(wp)
}
```

### "Grow Column"
```go
wp.columns[i].Resize(-10)
wp.TileWindows()
```

### "Handle j key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
		<<<Handle Alt-J>>>
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-J>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```
### "Handle k key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
		<<<Handle Alt-K>>>
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-K>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle Alt-J"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		if err := wp.Down(ManagedWindow{*activeWindow, 0}); err == nil {
			wp.TileWindows()
		}
	}(wp)
}

```
### "Handle Control-Shift-J"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for _, c := range wp.columns {
			for i, win := range c.Windows {
				if win.Window == *activeWindow {
					<<<Grow Window Down>>>
					return
				}
			}
		}
	}(wp)
}
```

### "Handle Alt-K"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		if err := wp.Up(ManagedWindow{*activeWindow, 0}); err == nil {
			wp.TileWindows()
		}
	}(wp)
}

```
### "Handle Control-Shift-K"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for _, c := range wp.columns {
			for i, win := range c.Windows {
				if win.Window == *activeWindow {
					<<<Grow Window Up>>>
					return
				}
			}
		}
	}(wp)
}
```

### "Grow Window Up"
```go
c.Windows[i].Resize(-10)
wp.TileWindows()
```

### "Grow Window Down"
```go
c.Windows[i].Resize(10)
wp.TileWindows()
```

Since we redefined our Column type and ManagedWindow types, we'll have to
redeclare all of the places that assume Column is a []ManagedWindow or
ManagedWindow is a xproto.Window to take into account the new types.
.
There's no significant changes from the last implementation below, we're just
working around compile errors.

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
		Column{Windows: []ManagedWindow{ ManagedWindow{win, 0} }, SizeDelta: 0},
	}
case 1:
	if len(w.columns[0].Windows) == 0 {
		// No active window in first column, so use it.
		w.columns[0].Windows = append(w.columns[0].Windows, ManagedWindow{win, 0})
	} else {
		// There's something in the primary column, so create a new one.
		w.columns = append(w.columns, Column{Windows: []ManagedWindow{ ManagedWindow{win, 0} }, SizeDelta: 0})
	}
default:
	// Add to the first empty column we can find, and shortcircuit out
	// if applicable.
	for i, c := range w.columns {
		if len(c.Windows) == 0 {
			w.columns[i].Windows = append(w.columns[i].Windows, ManagedWindow{win, 0})
			return nil
		}
	}

	// No empty columns, add to the last one.
	i := len(w.columns)-1
	w.columns[i].Windows = append(w.columns[i].Windows, ManagedWindow{win, 0})
}
return nil
```

### "Remove wp[colnum][idx] and move to wp[colnum+1]"
```go
if colnum >= len(wp.columns)-1 {
	return fmt.Errorf("Already at end of workspace.")
}

// Found the window at at idx, so delete it and return.
// (I wish Go made it easier to delete from a slice.)
wp.columns.Windows[colnum] = append(column[0:idx], column[idx+1:]...)
wp.columns.Windows[colnum+1] = append(wp.columns.Windows[colnum+1], w)
```

### "Up implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column.Windows {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Swap wp[colnum][idx] with wp[colnum][idx-1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Swap wp[colnum][idx] with wp[colnum][idx-1]"
```go
if idx == 0 {
	return fmt.Errorf("Window already at top of column")
}
wp.columns[colnum].Windows[idx], wp.columns[colnum].Windows[idx-1] = wp.columns[colnum].Windows[idx-1], wp.columns[colnum].Windows[idx]
```

### "Down implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column.Windows {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Swap wp[colnum][idx] with wp[colnum][idx+1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Swap wp[colnum][idx] with wp[colnum][idx+1]"
```go
if idx >= len(wp.columns[colnum].Windows)-1 {
	return fmt.Errorf("Window already at bottom of column")
}
wp.columns[colnum].Windows[idx], wp.columns[colnum].Windows[idx+1] = wp.columns[colnum].Windows[idx+1], wp.columns[colnum].Windows[idx]
```

### "Left implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column.Windows {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Remove wp[colnum][idx] and move to wp[colnum-1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Remove wp[colnum][idx] and move to wp[colnum-1]"
```
if colnum <= 0 {
	return fmt.Errorf("Already in first column of workspace.")
}

// Found the window at at idx, so delete it and return.
// (I wish Go made it easier to delete from a slice.)
wp.columns[colnum].Windows = append(column.Windows[0:idx], column.Windows[idx+1:]...)
wp.columns[colnum-1].Windows = append(wp.columns[colnum-1].Windows, w)
```

### "Right implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()
	
for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column.Windows {
		if w == candwin {
			idx = i
			break
		}
	}
	if idx != -1 {
		<<<Remove wp[colnum][idx] and move to wp[colnum+1]>>>
		return nil 
	}	
}
return fmt.Errorf("Window not managed by workspace")
```

### "Remove wp[colnum][idx] and move to wp[colnum+1]"
```go
if colnum >= len(wp.columns)-1 {
	return fmt.Errorf("Already at end of workspace.")
}

// Found the window at at idx, so delete it and return.
// (I wish Go made it easier to delete from a slice.)
wp.columns[colnum].Windows = append(column.Windows[0:idx], column.Windows[idx+1:]...)
wp.columns[colnum+1].Windows = append(wp.columns[colnum+1].Windows, w)
```

### "RemoveWindow implementation"
```go
wp.mu.Lock()
defer wp.mu.Unlock()

for colnum, column := range wp.columns {
	idx := -1
	for i, candwin := range column.Windows {
		if w == candwin.Window {
			idx = i
			break
		}
	}
	if idx != -1 {
		// Found the window at at idx, so delete it and return.
		// (I wish Go made it easier to delete from a slice.)
		wp.columns[colnum].Windows = append(column.Windows[0:idx], column.Windows[idx+1:]...)
		return nil
	}	
}
return fmt.Errorf("Window not managed by workspace")
```
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
 
Now, we can finally give windows a similar treatment.
