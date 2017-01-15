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

(Similar reasoning applies to windows as columns, with width and height inverted)

The next problem we're going to have is: what is the active column? We have a
pointer to the activeWindow, not the activeColumn. We could add a pointer to
the ManagedWindow type, but so far our approach of just going through all of
the windows hasn't had any significant performance problems, so let's just
keep doing that instead of risking the pointer going out of sync because we
forget to update it somewhere. Once again, we can add it later if need be, but
for now it's better to have less possibility for bugs until we find it becomes
a bottleneck.

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

On second thought, I remember my pinky hurting from stretching from shift to
H/L, so let's try Ctrl-Alt-Up/Down/Left/Right for our resizing keystrokes.

### "Grabbed Key List" +=
```go
{
	sym:       keysym.XK_Up,
	modifiers: xproto.ModMaskControl | xproto.ModMask1,
},
{
	sym:       keysym.XK_Down,
	modifiers: xproto.ModMaskControl | xproto.ModMask1,
},
{
	sym:       keysym.XK_Left,
	modifiers: xproto.ModMaskControl | xproto.ModMask1,
},
{
	sym:       keysym.XK_Right,
	modifiers: xproto.ModMaskControl | xproto.ModMask1,
},
```

Since they're new keys, we'll have to add them to our switch.

### "Keystroke Detail Switch" +=
```go
case keysym.XK_Up:
	<<<Handle Up key>>>
case keysym.XK_Down:
	<<<Handle Down key>>>
case keysym.XK_Left:
	<<<Handle Left key>>>
case keysym.XK_Right:
	<<<Handle Right key>>>
```

### "Handle Up key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMaskControl | xproto.ModMask1:
		<<<Handle Control-Alt-Up>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle Down key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMaskControl | xproto.ModMask1:
		<<<Handle Control-Alt-Down>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```
### "Handle Left key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMaskControl | xproto.ModMask1:
		<<<Handle Control-Alt-Left>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle Right key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMaskControl | xproto.ModMask1:
		<<<Handle Control-Alt-Right>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

Then implement our code that finds the active column or window and adjusts its
SizeDelta.

For the left/right keys, we'll make it grow the current window, heuristically as
if we're pushing on the side of the window. This will probably drive us crazy
in the first column, so we'll invert the left/right logic for the first column.

### "Handle Control-Alt-Right"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for i, c := range wp.columns {
			for _, win := range c.Windows {
				if win.Window == *activeWindow {
					if i == 0 {
						<<<Grow Column i>>>
					} else {
						<<<Shrink Column i>>>
					}
					return
				}
			}
		}
	}(wp)
}
```

### "Handle Control-Alt-Left"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for i, c := range wp.columns {
			for _, win := range c.Windows {
				if win.Window == *activeWindow {
					if i == 0 {
						<<<Shrink Column i>>>
					} else {
						<<<Grow Column i>>>
					}
					return
				}
			}
		}
	}(wp)
}
```

### "Shrink Column i"
```go
wp.columns[i].Resize(-10)
wp.TileWindows()
```

### "Grow Column i"
```go
wp.columns[i].Resize(10)
wp.TileWindows()
```

We'll do similar for Up/Down, except it'll affect the window inside of the
column, not the column.

### "Handle Control-Alt-Down"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for _, c := range wp.columns {
			for i, win := range c.Windows {
				if win.Window == *activeWindow {
					if i == 0 {
						<<<Grow Window i>>>
					} else {
						<<<Shrink Window i>>>
					}
					return
				}
			}
		}
	}(wp)
}
```

### "Handle Control-Alt-Up"
```go
for _, wp := range workspaces {
	go func(wp *Workspace) {
		for _, c := range wp.columns {
			for i, win := range c.Windows {
				if win.Window == *activeWindow {
					if i == 0 {
						<<<Shrink Window i>>>
					} else {
						<<<Grow Window i>>>
					}
					return
				}
			}
		}
	}(wp)
}
```

### "Grow Window i"
```go
c.Windows[i].Resize(10)
wp.TileWindows()
```

### "Shrink Window i"
```go
c.Windows[i].Resize(-10)
wp.TileWindows()
```

Since we redefined our Column type and ManagedWindow types, we'll have to
redeclare all of the places that assume Column is a []ManagedWindow or
ManagedWindow is a xproto.Window to take into account the new types.

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
		if w.Window == candwin.Window {
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
		if w.Window == candwin.Window {
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
		if w.Window == candwin.Window {
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
		if w.Window == candwin.Window {
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
### "Handle h key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Left(ManagedWindow{*activeWindow, 0}); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
}

return nil
```

### "Handle j key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Down(ManagedWindow{*activeWindow, 0}); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
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
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Up(ManagedWindow{*activeWindow, 0}); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}

}
return nil
```
### "Handle l key"
```go
if activeWindow == nil {
	return nil
}

switch key.State {
	case xproto.ModMask1:
	for _, wp := range workspaces {
		go func(wp *Workspace) {
			if err := wp.Right(ManagedWindow{*activeWindow, 0}); err == nil {
				wp.TileWindows()
			}
		}(wp)
	}
}
return nil
```

Now that we can resize windows and columns, our WM is finally getting pretty
useable!
