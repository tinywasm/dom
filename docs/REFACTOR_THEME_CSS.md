# tinywasm/dom — Plan: Fix CSS Loading

## Problema raíz

`dom/ssr.go` tiene `RenderCSS()` retornando `c.renderCSS()` — un `*ast.CallExpr`.
`ExtractSSRAssets` solo evalúa `*ast.BasicLit`, `*ast.Ident` (embed vars) y `*ast.BinaryExpr`.
Resultado: CSS extraído = `""` → no se inyectan las CSS vars.

**Convención existente**: `RenderCSS()` retorna una embed var → assetmin extrae con `*ast.Ident`. Dom debe seguir esta misma convención.

Problema secundario: `assetmin/ssr_loader.go` hardcodea `strings.Contains(moduleDir, "tinywasm/dom")` para asignar slot `"open"`. assetmin no debería conocer dom.

---

## Cambios

### 1. `dom/ssr.go` — Reemplazar

Mover el embed de `theme.css` aquí (actualmente está en `ssr.theme.go`). `RenderCSS()` retorna la embed var → extractable por assetmin.

```go
//go:build !wasm

package dom

// SSRSlot indica a assetmin en qué posición del <head> inyectar este módulo.
const SSRSlot = "open"

//go:embed theme.css
var themeCSS string

func (c CssVars) RenderCSS() string {
    return themeCSS
}
```

### 2. `dom/ssr.theme.go` — Limpiar

- Eliminar `ThemeCSS` y su `//go:embed theme.css` (se mueve a ssr.go)
- Agregar método público `Render() string` para uso programático desde server.go:

```go
func (c CssVars) Render() string {
    return c.renderCSS()
}
```

### 3. `assetmin/ssr_extract.go` — Agregar extracción de SSRSlot

Agregar `Slot string` a `SSRAssets` y leer `const SSRSlot` del AST.

```go
type SSRAssets struct {
    ModuleName string
    Slot       string  // "open" | "middle" | "close" | ""
    CSS        string
    JS         string
    HTML       string
    Icons      map[string]string
}
```

Función auxiliar agregada a `ExtractSSRAssets`:

```go
func findSlotConst(f *ast.File, assets *SSRAssets) {
    for _, decl := range f.Decls {
        gen, ok := decl.(*ast.GenDecl)
        if !ok || gen.Tok != token.CONST { continue }
        for _, spec := range gen.Specs {
            vs, ok := spec.(*ast.ValueSpec)
            if !ok { continue }
            for i, name := range vs.Names {
                if name.Name == "SSRSlot" && i < len(vs.Values) {
                    if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
                        s, _ := strconv.Unquote(lit.Value)
                        assets.Slot = s
                    }
                }
            }
        }
    }
}
```

### 4. `assetmin/ssr_loader.go` — Reemplazar hardcoding dom

Nueva función auxiliar (reemplaza lógica repetida en 3 lugares):

```go
func resolveSlot(declared, dir, rootDir string) string {
    if declared != "" {
        return declared
    }
    if isRootDir(dir, rootDir) {
        return "close"
    }
    return "middle"
}
```

Reemplazar las 3 instancias de:
```go
slot := "middle"
if strings.Contains(…, "tinywasm/dom") { slot = "open" } else if isRootDir(…) { slot = "close" }
```
por:
```go
slot := resolveSlot(assets.Slot, dir, c.RootDir)
```

También evaluar si el parche `if m.Path == "dom" { modules[i].Path = "tinywasm/dom" }` sigue siendo necesario tras eliminar el strings.Contains.

### 5. `tinywasm/server/templates/server_basic.md` — Agregar ejemplo dom

Agregar sección mostrando cómo inyectar un tema custom usando `dom.CssVars{}.Render()` + `UpdateSSRModule`.

---

## Orden de implementación

1. `assetmin/ssr_extract.go` — Slot + findSlotConst
2. `assetmin/ssr_loader.go` — resolveSlot, eliminar strings.Contains dom
3. `dom/ssr.go` — embed + SSRSlot + RenderCSS
4. `dom/ssr.theme.go` — eliminar ThemeCSS embed, agregar Render()
5. `server_basic.md` — ejemplo dom

---

## Tests

### assetmin — `tests/ssr_slot_test.go`

```
TestExtractSSRSlot_ReturnsOpenFromConst   → const SSRSlot = "open" → assets.Slot == "open"
TestExtractSSRSlot_DefaultsToEmpty        → sin SSRSlot const → assets.Slot == ""
TestResolveSlot_DeclaredWins              → declared="open", isRootDir=true → "open"
TestResolveSlot_RootDirFallback           → declared="", isRootDir=true → "close"
TestResolveSlot_DefaultMiddle             → declared="", isRootDir=false → "middle"
```

### dom — actualizar `ssr_theme_test.go`

```
TestCssVars_RenderCSS_ReturnsThemeCSS     → RenderCSS() == themeCSS (embed)
TestCssVars_Render_GeneratesRootBlock     → Render() con valores custom contiene ":root {"
```

---

## Breaking changes

| Cambio | Impacto |
|--------|---------|
| `ThemeCSS` eliminado de ssr.theme.go | Rompe código que usa `dom.ThemeCSS`. Verificar: `grep -r "dom.ThemeCSS"` |
| `SSRAssets.Slot` campo nuevo | Sin impacto (campo adicional) |
