# charm-json

Interactive JSON/YAML tree explorer for the terminal: parse, collapse, filter and copy JSON Pointer paths with a Bubble Tea model.

## Install

```bash
go get github.com/junhinhow/charm-json
```

## Usage

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	charmjson "github.com/junhinhow/charm-json"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root, err := charmjson.Parse(data, "") // "" detects JSON or YAML
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model := charmjson.New(root, charmjson.WithMaxDepth(8))
	model.SetSize(80, 24)

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Keys

| Key | Action |
| --- | --- |
| `j` / `↓` | Move to the next row |
| `k` / `↑` | Move to the previous row |
| `enter` / `space` | Expand or collapse the selected container |
| `h` / `←` | Collapse the selected container |
| `l` / `→` | Expand the selected container |
| `g` | Jump to the first row |
| `G` | Jump to the last row |
| `/` | Start a search |
| `esc` | Clear the search/filter |
| `y` | Emit `YankedMsg` with the selected path |

While searching, printable characters edit the query, `backspace` removes the last rune and `enter` confirms the filter.

## API

| Symbol | Description |
| --- | --- |
| `Parse(data, format)` | Parses `"json"`, `"yaml"`/`"yml"` or auto-detects with `""` |
| `ParseJSON(data)` | Parses JSON preserving key order and the original number text |
| `ParseYAML(data)` | Parses YAML preserving definition order; resolves anchors/aliases |
| `Node` | Tree node with `Key`, `Kind`, `Value`, `Children`, `Path` and `Collapsed` |
| `(*Node).Find(path)` | Looks up a node by RFC 6901 JSON Pointer |
| `(*Node).Flatten()` | Visible rows in pre-order, honoring `Collapsed` |
| `(*Node).Collapse/Expand/Toggle(path)` | Changes the collapse state of one node |
| `(*Node).CollapseAll()/ExpandAll()` | Changes the collapse state of the whole tree |
| `Row` | Rendered row with `Node`, `Depth`, `Prefix` and `Display` |
| `New(root, opts...)` | Creates the Bubble Tea explorer |
| `(*Model).SetSize(w, h)` | Sets the render area |
| `(*Model).SetFilter(s)` / `Filter()` | Sets/reads the case-insensitive filter |
| `(*Model).Rows()` | Visible rows after collapse, filter and depth limit |
| `(*Model).SelectedPath()` | JSON Pointer of the selected node |
| `(*Model).SelectedValue()` | Scalar value of the selected node |
| `WithMaxDepth(n)` | Limits displayed depth; deeper containers render as `…` |
| `WithSyntaxColors(bool)` | Enables/disables ANSI colors (enabled by default) |
| `YankedMsg` | Message emitted by the `y` key |
| `ParseError` | Syntax error with `Line`, `Column` and `Msg` |

## Design notes

- **Package name.** The module is `charm-json`; the imported package is `charmjson`.
- **Paths.** The root is `""`; `/` addresses the child whose key is empty. `~` and `/` are escaped as `~0` and `~1` per RFC 6901.
- **Scalar values.** `Node.Value` stores strings already decoded and numbers exactly as written (`json.Decoder` uses `UseNumber`). Display adds quotes only to strings.
- **Ordering.** JSON keeps insertion order and YAML keeps definition order; nothing is sorted alphabetically. Rendering never iterates maps.
- **Filtering.** The query matches path, key and value, case-insensitively. A node is visible when it matches or is an ancestor of a match. Collapse is ignored while a filter is active so matches stay reachable; with no match the row list is empty.
- **Depth limit.** `WithMaxDepth` only truncates the view: nodes below the limit remain in the model and are restored when the limit changes.
- **YAML.** `yaml.Node` preserves order. Anchors and aliases are resolved. Merge keys (`<<`) are kept as regular keys instead of being merged. `yaml.v3` only reports the line, so `ParseError.Column` is `1` when the column is unknown.
- **Rendering.** Width is measured and truncated with `github.com/charmbracelet/x/ansi`, so wide runes and emoji stay aligned. A two-column gutter marks the selection. ANSI colors are fixed constants (no terminal profile detection), which keeps output deterministic; `WithSyntaxColors(false)` removes all escape codes.
- **Copying.** `y` does not touch the clipboard: it only returns a `tea.Cmd` that emits `YankedMsg{Path}` so the host application can copy the value.

## License

MIT
