# cbind
[![GoDoc](https://gitlab.com/tslocum/godoc-static/-/raw/master/badge.svg)](https://docs.rocketnine.space/gitlab.com/tslocum/cbind)
[![CI status](https://gitlab.com/tslocum/cbind/badges/master/pipeline.svg)](https://gitlab.com/tslocum/cbind/commits/master)
[![Donate](https://img.shields.io/liberapay/receives/rocketnine.space.svg?logo=liberapay)](https://liberapay.com/rocketnine.space)

Key event handling library for tcell

## Features

- Set `KeyEvent` handlers
- Encode and decode `KeyEvent`s as human-readable strings

## Usage

```go
// Create a new input configuration to store the key bindings.
c := NewConfiguration()

handleSave := func(ev *tcell.EventKey) *tcell.EventKey {
    // Save
    return nil
}

handleOpen := func(ev *tcell.EventKey) *tcell.EventKey {
    // Open
    return nil
}

handleExit := func(ev *tcell.EventKey) *tcell.EventKey {
    // Exit
    return nil
}

// Bind Alt+s.
if err := c.Set("Alt+s", handleSave); err != nil {
    log.Fatalf("failed to set keybind: %s", err)
}

// Bind Alt+o.
c.SetRune(tcell.ModAlt, 'o', handleOpen)

// Bind Escape.
c.SetKey(tcell.ModNone, tcell.KeyEscape, handleExit)

// Capture input. This will differ based on the framework in use (if any).
// When using tview or cview, call Application.SetInputCapture before calling
// Application.Run.
app.SetInputCapture(c.Capture)
```

## Documentation

Documentation is available via [gdooc](https://docs.rocketnine.space/gitlab.com/tslocum/cbind).

The utility program `whichkeybind` may be used to determine and validate key combinations.

```bash
go get gitlab.com/tslocum/cbind/whichkeybind
```

## Support

Please share issues and suggestions [here](https://gitlab.com/tslocum/cbind/issues).
