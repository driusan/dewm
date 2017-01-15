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

