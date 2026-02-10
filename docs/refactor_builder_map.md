# Plan: Refactor Builder Attributes to Use KeyValue Slice

## Objective
Remove the use of `map[string]string` for element attributes in `Builder` and replace it with `[]fmt.KeyValue` to optimize for WebAssembly/TinyGo as per framework rules.

## Strategic Justification
TinyGo's map implementation adds significant overhead to WebAssembly binary size and runtime memory. For small collections like DOM attributes, a slice of structs is more efficient. Additionally, the `Node` struct in `interface.dom.go` already uses `[]fmt.KeyValue`, so this change improves internal consistency.

## Proposed Changes

### 1. `builder.go`: Update `Builder` Struct
Modify the `Builder` type to use a slice of `fmt.KeyValue` instead of a map.

### 2. `builder.go`: Update `Attr` Method
Update `Attr(key, val string)` to manage the slice. It should search for an existing key to update its value, or append a new `KeyValue` if not found. This ensures the "last set wins" behavior expected from attribute setters.

### 3. `builder.go`: Update `ToNode` Method
Simplify the conversion of attributes in `ToNode` as they are now already in the correct format (`[]fmt.KeyValue`).

## Alternatives Considered
- **Append Only**: Just appending to the slice without checking for duplicates. This is faster but can lead to invalid HTML with duplicate attributes (e.g., `<div id='a' id='b'>`).
- **Map wrapper**: Implementing a map-like API on top of the slice. This is what we are doing in `Attr`.

## Execution Steps
1. Edit `/home/cesar/Dev/Pkg/tinywasm/dom/builder.go`.
2. Verify with `gotest`.
