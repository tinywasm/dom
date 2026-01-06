# tinywasm/fmt Helper Guide

tinywasm/dom uses [tinywasm/fmt](https://github.com/tinywasm/fmt) for all string manipulations and conversions to avoid the overhead of the standard library (`fmt`, `strconv`, `strings`, `errors`).

## Usage
```go
import . "github.com/tinywasm/fmt"
```

### Type Conversion

Convert any type (int, bool, float, etc.) to `string`:

```go
// Integer to String
count := 42
text := Convert(count).String() // "42"

// Boolean to String
isActive := true
text := Convert(isActive).String() // "true"
```

### String Transformation

Chain methods to transform text:

```go
name := "TinyDOM"

// Lowercase
lower := Convert(name).ToLower().String() // "tinywasm/dom"

// Uppercase
upper := Convert(name).ToUpper().String() // "TINYDOM"

// Capitalize
cap := Convert("hello").Capitalize().String() // "Hello"
```

### String Building (Concatenation)

For complex HTML construction, use the builder pattern to minimize allocations:

```go
items := []string{"A", "B", "C"}
builder := Convert() // Empty buffer

builder.Write("<ul>")
for _, item := range items {
    builder.Write("<li>").Write(item).Write("</li>")
}
builder.Write("</ul>")

html := builder.String()
```

### Error Handling

When building strings, you can check for errors (like buffer overflows) using `StringErr()` instead of `String()`:

```go
// ... builder operations ...

html, err := builder.StringErr()
if err != nil {
    // Handle error (e.g., log it or return it)
}
```

### Creating Errors

tinywasm/fmt also provides lightweight replacements for the `errors` package:

```go
// Replace errors.New("message")
err := Err("something went wrong")

// Replace fmt.Errorf("error: %s", val)
err := Errf("error: %s", "value")
```
