# Column Management

Two autotiled columns are nice and all, but sometimes we want more columns, or
to get rid of empty ones.

We'll add 2 new keystrokes: Ctrl-Shift-D to delete all empty columns, and
Ctrl-Shift-N to add a new column and retile.

This should be pretty straight forward:

We just need to grab the keys:

### "Grabbed Key List" +=
```go
{
	sym:       keysym.XK_d,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
{
	sym:       keysym.XK_n,
	modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
},
```

And add them to the key handler switch:

### "Keystroke Detail Switch" +=
```go
case keysym.XK_d:
	<<<Handle d key>>>
case keysym.XK_n:
	<<<Handle n key>>>
```

Then add our usual state switch, in case we use them for more things later. We
don't need to check the active window, since we're just adding/deleting empty
columns.

### "Handle d key"
```go
switch key.State {
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-D>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

### "Handle n key"
```go
switch key.State {
	case xproto.ModMaskControl | xproto.ModMaskShift:
		<<<Handle Control-Shift-N>>>
	default:
		log.Printf("Unhandled state: %v\n", key.State)
}
return nil
```

Let's start with new. We just need to append to the active workspace. We don't
have a way to determine if a workspace is active, but we have an activeWindow
pointer. Let's add a helper to workspaces to check if it contains a window,
and another to check if it's active (contains the active window)

### "workspace.go functions" +=
```go
func (w *Workspace) ContainsWindow(win xproto.Window) bool {
	<<<Workspace ContainsWindow Implementation>>>
}

func (w *Workspace) IsActive() bool {
	<<<Workspace IsActive implementation>>>
}
```

### "Workspace IsActive implementation"
```go
if activeWindow == nil {
	return false
}
return w.ContainsWindow(*activeWindow)
```

### "Workspace ContainsWindow Implementation"
```go
for _, c := range w.columns {
	for _, w := range c.Windows {
		if w.Window == win {
			return true
		}
	}
}
return false
```

Now, we should have everything we need to add a new column to the current
workspace.

### "Handle Control-Shift-N"
```go
for _, w := range workspaces {
	if w.IsActive() {
		w.mu.Lock()
		w.columns = append(w.columns, Column{})
		w.mu.Unlock()
		w.TileWindows()
	}
}
```

For deleting, we'll just lock the mutex, and create a new w.Columns (since we
don't know how many items might be getting deleted, and Go slice tricks get
dangerous if you try and modify a slice while iterating over it) then set
it to the workspace's w.columns if the length has changed.

### "Handle Control-Shift-D"
```go
for _, w := range workspaces {
	if w.IsActive() {
		w.mu.Lock()
		newColumns := make([]Column, 0, len(w.columns))
		for _, c := range w.columns {
			if len(c.Windows) > 0 {
				newColumns = append(newColumns, c)
			}
		}
		// Don't bother using the newColumns if it didn't change
		// anything. Just let newColumns get GCed.
		if len(newColumns) != len(w.columns) {
			w.columns = newColumns
			w.TileWindows()
		}
		w.mu.Unlock()
	}
}
```

And now we should be able to create and delete columns.

### "workspace.go imports" +=
```go
"github.com/BurntSushi/xgb/xproto"
```

With our new ability to create columns with ctrl-shift-n, our startup policy of
automatically creating a second column if the first one has something in it
makes less sense. What if we want a single full width column with many windows
in it?

Let's update our Add Window to Workspace implementation to take out the special
casing for the second column. If there's no columns it'll create one, if there's
an empty column, it'll use it, and otherwise new windows will go into the last
column.

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

And we can now manage our columns in our window manager.
