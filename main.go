package main

//go:generate lmt src/Initialize.md src/WindowManaging.md src/Keyboard.md src/MovingWindows.md src/ResizingWindows.md src/ColumnManagement.md src/OverrideRedirect.md src/GoGenerate.md
// THIS IS A MACHINE GENERATED FILE BY THE ABOVE COMMAND; DO NOT EDIT

import (
	"errors"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/driusan/dewm/keysym"
	"log"
	"os/exec"
	"sync"
	"time"
)

var xc *xgb.Conn
var xroot xproto.ScreenInfo
var QuitSignal error = errors.New("Quit")
var keymap [256][]xproto.Keysym
var attachedScreens []xinerama.ScreenInfo

// ICCCM related atoms
var (
	atomWMProtocols    xproto.Atom
	atomWMDeleteWindow xproto.Atom
	atomWMTakeFocus    xproto.Atom
)

func main() {
	xcon, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	xc = xcon
	defer xc.Close()
	setup := xproto.Setup(xc)
	if setup == nil || len(setup.Roots) < 1 {
		log.Fatal("Could not parse SetupInfo.")
	}
	if err := xinerama.Init(xc); err != nil {
		log.Fatal(err)
	}
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
	coninfo := xproto.Setup(xc)
	if coninfo == nil {
		log.Fatal("Could not parse X connection info")
	}
	if len(coninfo.Roots) != 1 {
		log.Fatal("Inappropriate number of roots. Did Xinerama initialize correctly?")
	}
	xroot = coninfo.Roots[0]
	atomWMProtocols = getAtom("WM_PROTOCOLS")
	atomWMDeleteWindow = getAtom("WM_DELETE_WINDOW")
	atomWMTakeFocus = getAtom("WM_TAKE_FOCUS")
	if err := TakeWMOwnership(); err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			log.Fatal("Could not become the WM. Is another WM already running?")
		}
		log.Fatal(err)
	}
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
		keymap[loKey+i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode) : (i+1)*int(reply.KeysymsPerKeycode)]
	}
	grabs := []struct {
		sym       xproto.Keysym
		modifiers uint16
		codes     []xproto.Keycode
	}{
		{
			sym:       keysym.XK_BackSpace,
			modifiers: xproto.ModMaskControl | xproto.ModMask1,
		},
		{
			sym:       keysym.XK_e,
			modifiers: xproto.ModMask1,
		},
		{
			sym:       keysym.XK_q,
			modifiers: xproto.ModMask1,
		},
		{
			sym:       keysym.XK_q,
			modifiers: xproto.ModMask1 | xproto.ModMaskShift,
		},
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
		{
			sym:       keysym.XK_d,
			modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
		},
		{
			sym:       keysym.XK_n,
			modifiers: xproto.ModMaskControl | xproto.ModMaskShift,
		},
	}

	for i, syms := range keymap {
		for _, sym := range syms {
			for c := range grabs {
				if grabs[c].sym == sym {
					grabs[c].codes = append(grabs[c].codes, xproto.Keycode(i))
				}
			}
		}
	}
	for _, grabbed := range grabs {
		for _, code := range grabbed.codes {
			if err := xproto.GrabKeyChecked(
				xc,
				false,
				xroot.Root,
				grabbed.modifiers,
				code,
				xproto.GrabModeAsync,
				xproto.GrabModeAsync,
			).Check(); err != nil {
				log.Print(err)
			}

		}
	}
	tree, err := xproto.QueryTree(xc, xroot.Root).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if tree != nil {
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

	}
	// Main X Event loop
eventloop:
	for {
		xev, err := xc.WaitForEvent()
		if err != nil {
			log.Println(err)
			continue
		}
		switch e := xev.(type) {
		case xproto.KeyPressEvent:
			if err := HandleKeyPressEvent(e); err != nil {
				break eventloop
			}
		case xproto.DestroyNotifyEvent:
			for _, w := range workspaces {
				go func(w *Workspace) {
					if err := w.RemoveWindow(e.Window); err == nil {
						w.TileWindows()
					}
				}(w)
			}
			if activeWindow != nil && e.Window == *activeWindow {
				activeWindow = nil
			}
		case xproto.ConfigureRequestEvent:
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
		case xproto.MapRequestEvent:
			if winattrib, err := xproto.GetWindowAttributes(xc, e.Window).Reply(); err != nil || !winattrib.OverrideRedirect {
				w := workspaces["default"]
				xproto.MapWindowChecked(xc, e.Window)
				w.Add(e.Window)
				w.TileWindows()
			}
		case xproto.EnterNotifyEvent:
			activeWindow = &e.Event

			prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
				xproto.GetPropertyTypeAny, 0, 64).Reply()
			focused := false
			if err == nil {
			TakeFocusPropLoop:
				for v := prop.Value; len(v) >= 4; v = v[4:] {
					switch xproto.Atom(uint32(v[0]) | uint32(v[1])<<8 | uint32(v[2])<<16 | uint32(v[3])<<24) {
					case atomWMTakeFocus:
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
		default:
			log.Println(err)
		}
		log.Println(xev)

	}
}

func TakeWMOwnership() error {
	return xproto.ChangeWindowAttributesChecked(
		xc,
		xroot.Root,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease |
				xproto.EventMaskButtonPress |
				xproto.EventMaskButtonRelease |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskSubstructureRedirect,
		}).Check()
}
func HandleKeyPressEvent(key xproto.KeyPressEvent) error {
	switch keymap[key.Detail][0] {
	case keysym.XK_BackSpace:
		if (key.State&xproto.ModMaskControl != 0) && (key.State&xproto.ModMask1 != 0) {
			return QuitSignal
		}
		return nil
	case keysym.XK_e:
		if key.State&xproto.ModMask1 != 0 {
			cmd := exec.Command("xterm")
			err := cmd.Start()
			go func() {
				cmd.Wait()
			}()
			return err
		}
		return nil
	case keysym.XK_q:
		switch key.State {
		case xproto.ModMask1:
			prop, err := xproto.GetProperty(xc, false, *activeWindow, atomWMProtocols,
				xproto.GetPropertyTypeAny, 0, 64).Reply()
			if err != nil {
				return err
			}
			if prop == nil {
				// There were no properties, so the window doesn't follow ICCCM.
				// Just destroy it.
				if activeWindow != nil {
					return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
				}
			}
			for v := prop.Value; len(v) >= 4; v = v[4:] {
				switch xproto.Atom(uint32(v[0]) | uint32(v[1])<<8 | uint32(v[2])<<16 | uint32(v[3])<<24) {
				case atomWMDeleteWindow:
					t := time.Now().Unix()
					return xproto.SendEventChecked(
						xc,
						false,
						*activeWindow,
						xproto.EventMaskNoEvent,
						string(xproto.ClientMessageEvent{
							Format: 32,
							Window: *activeWindow,
							Type:   atomWMProtocols,
							Data: xproto.ClientMessageDataUnionData32New([]uint32{
								uint32(atomWMDeleteWindow),
								uint32(t),
								0,
								0,
								0,
							}),
						}.Bytes())).Check()
				}
			}
			// No WM_DELETE_WINDOW protocol, so destroy.
			if activeWindow != nil {
				return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
			}
		case xproto.ModMask1 | xproto.ModMaskShift:
			if activeWindow != nil {
				return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
			}
		}
		return nil
	case keysym.XK_h:
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
	case keysym.XK_j:
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
	case keysym.XK_k:
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
	case keysym.XK_l:
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
	case keysym.XK_Up:
		if activeWindow == nil {
			return nil
		}

		switch key.State {
		case xproto.ModMaskControl | xproto.ModMask1:
			for _, wp := range workspaces {
				go func(wp *Workspace) {
					for _, c := range wp.columns {
						for i, win := range c.Windows {
							if win.Window == *activeWindow {
								if i == 0 {
									c.Windows[i].Resize(-10)
									wp.TileWindows()
								} else {
									c.Windows[i].Resize(10)
									wp.TileWindows()
								}
								return
							}
						}
					}
				}(wp)
			}
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	case keysym.XK_Down:
		if activeWindow == nil {
			return nil
		}

		switch key.State {
		case xproto.ModMaskControl | xproto.ModMask1:
			for _, wp := range workspaces {
				go func(wp *Workspace) {
					for _, c := range wp.columns {
						for i, win := range c.Windows {
							if win.Window == *activeWindow {
								if i == 0 {
									c.Windows[i].Resize(10)
									wp.TileWindows()
								} else {
									c.Windows[i].Resize(-10)
									wp.TileWindows()
								}
								return
							}
						}
					}
				}(wp)
			}
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	case keysym.XK_Left:
		if activeWindow == nil {
			return nil
		}

		switch key.State {
		case xproto.ModMaskControl | xproto.ModMask1:
			for _, wp := range workspaces {
				go func(wp *Workspace) {
					for i, c := range wp.columns {
						for _, win := range c.Windows {
							if win.Window == *activeWindow {
								if i == 0 {
									wp.columns[i].Resize(-10)
									wp.TileWindows()
								} else {
									wp.columns[i].Resize(10)
									wp.TileWindows()
								}
								return
							}
						}
					}
				}(wp)
			}
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	case keysym.XK_Right:
		if activeWindow == nil {
			return nil
		}

		switch key.State {
		case xproto.ModMaskControl | xproto.ModMask1:
			for _, wp := range workspaces {
				go func(wp *Workspace) {
					for i, c := range wp.columns {
						for _, win := range c.Windows {
							if win.Window == *activeWindow {
								if i == 0 {
									wp.columns[i].Resize(10)
									wp.TileWindows()
								} else {
									wp.columns[i].Resize(-10)
									wp.TileWindows()
								}
								return
							}
						}
					}
				}(wp)
			}
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	case keysym.XK_d:
		switch key.State {
		case xproto.ModMaskControl | xproto.ModMaskShift:
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
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	case keysym.XK_n:
		switch key.State {
		case xproto.ModMaskControl | xproto.ModMaskShift:
			for _, w := range workspaces {
				if w.IsActive() {
					w.mu.Lock()
					w.columns = append(w.columns, Column{})
					w.mu.Unlock()
					w.TileWindows()
				}
			}
		default:
			log.Printf("Unhandled state: %v\n", key.State)
		}
		return nil
	default:
		return nil
	}
}
func getAtom(name string) xproto.Atom {
	rply, err := xproto.InternAtom(xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if rply == nil {
		return 0
	}
	return rply.Atom
}
