# PLAN: tinywasm/dom — Separación de Responsabilidades + Rename String()

## Repositorio
`github.com/tinywasm/dom` — path local: `tinywasm/dom/`

## Dependencias de ejecución
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Objetivo

Dos cambios ortogonales que se ejecutan juntos por ser break changes:

### A) Rename `RenderHTML() string` → `String() string` en `dom.Component`

**Por qué:** `RenderHTML()` en la interface `dom.Component` es serialización interna — el DOM convierte el árbol de elementos a string para inyectarlo. Nombrarlo `RenderHTML` ata `dom` al formato HTML y choca con `html.HTMLProvider.RenderHTML() *HTML` (misma firma, distinta semántica en dos capas distintas). `String() string` es el contrato estándar de Go (`fmt.Stringer`), completamente agnóstico al formato de salida. Todo componente que lo implemente satisface `fmt.Stringer` automáticamente.

### B) Eliminar builders HTML/SVG/Image de dom

`dom/element.go` contiene ~60 funciones builder (`Div()`, `Span()`, `Svg()`, `Img()`, etc.) que no pertenecen a la capa de syscall/DOM. Se eliminan sin aliases — es un break change intencional (ecosistema sin publicar, sin usuarios externos).

---

## Cambio A: Rename `RenderHTML` → `String`

### Archivos a modificar

#### 1. `dom/interface.dom.go`

Buscar y reemplazar en la interface `Component`:
```go
// ANTES:
RenderHTML() string

// DESPUÉS:
String() string
```

#### 2. `dom/element.go`

Buscar y reemplazar:
```go
// ANTES:
// RenderHTML renders the element to HTML string.
func (b *Element) RenderHTML() string {

// DESPUÉS:
// String serializes the element tree to its string representation.
func (b *Element) String() string {
```

Y dentro de `elementToHTML` (línea ~155) hay una llamada recursiva:
```go
// ANTES:
s += v.RenderHTML()

// DESPUÉS:
s += v.String()
```

#### 3. `dom/dom_frontend.go`

Tres sitios donde se llama `.RenderHTML()` en un `Component`. Reemplazar todos:
```go
// ANTES:
html = component.RenderHTML()

// DESPUÉS:
html = component.String()
```

También buscar:
```go
// ANTES:
s += v.RenderHTML()

// DESPUÉS:
s += v.String()
```

#### 4. `dom/dom_backend.go`

Actualizar mensaje de error:
```go
// ANTES:
return fmt.Errf("Render to parent is not supported on backend. Use RenderHTML() directly on component.")

// DESPUÉS:
return fmt.Errf("Render to parent is not supported on backend. Use String() directly on component.")
```

---

## Cambio B: Eliminar Builders de dom

### Funciones a ELIMINAR de `dom/element.go`

Eliminar completamente las siguientes funciones (se mueven a sus paquetes respectivos):

**→ tinywasm/html** (eliminar de dom):
```
Div, Span, P, H1, H2, H3, H4, H5, H6
Ul, Ol, Li
Nav, Section, Main, Article, Header, Footer, Aside
Details, Summary, Dialog, Figure, Figcaption
Pre, Code, Strong, Small, Mark
Table, Thead, Tbody, Tfoot, Tr, Th, Td
Fieldset, Legend, Label, Button, Canvas, Style, Script
A, Input, Option, SelectedOption
Br, Hr
```

**→ tinywasm/svg** (eliminar de dom):
```
Svg, Use
```

**→ tinywasm/image** (eliminar de dom):
```
Img
```

### Interface a MOVER de dom

Eliminar de `dom/interface.dom.go`:
```go
// ELIMINAR — se mueve a tinywasm/svg:
type IconSvgProvider interface {
    IconSvg() map[string]string
}
```

### Lo que QUEDA en dom después del cambio

**Tipos:** `Element` (struct), `Event`, `Reference` interface, `DOM` interface  
**Interfaces de lifecycle:** `Component`, `ViewRenderer`, `Mountable`, `Updatable`, `Unmountable`  
**Providers que quedan:** `CSSProvider`, `JSProvider` (no tienen conflicto con los nuevos paquetes)  
**DOM API:** `Render`, `Append`, `Update`, `Get`, `Log`, `SetLog`, `OnHashChange`, `GetHash`, `SetHash`  
**Internos:** `domBackend`, `domWasm`, `tinyDOM`, `elementStub`

---

## Actualizar go.mod

Después de eliminar los builders, `dom/go.mod` ya **no necesita** dependencias en `html`, `svg`, `image`. Si se agregaron temporalmente, retirarlas.

Verificar que el módulo compila:
```bash
cd tinywasm/dom
go build ./...
```

---

## Tests

### Actualizar tests existentes

En todos los archivos `*_test.go` dentro de `dom/`:

Reemplazar:
```go
// ANTES:
html := comp.Render().RenderHTML()
// o:
html := comp.RenderHTML()

// DESPUÉS:
html := comp.Render().String()
// o:
html := comp.String()
```

### Verificar que los tests pasan:
```bash
cd tinywasm/dom
gotest
```

### Test de regresión — interface String():
```go
// dom/element_test.go — agregar:
func TestElement_ImplementsStringer(t *testing.T) {
    var _ fmt.Stringer = &Element{}  // compile-time check
}

func TestElement_String_Basic(t *testing.T) {
    el := &Element{tag: "div"}
    el.Class("root").Text("hello")
    got := el.String()
    if !strings.Contains(got, "class=\"root\"") { t.Error("expected class") }
    if !strings.Contains(got, "hello") { t.Error("expected text") }
}
```

---

## Impacto en otros paquetes (a coordinar con sus PLAN.md)

Paquetes que usan `dom` y necesitan actualizar sus llamadas a `RenderHTML()`:

| Paquete | Acción |
|---|---|
| `tinywasm/components` | Ver `tinywasm/components/docs/PLAN.md` |
| `tinywasm/layout` | Ver `tinywasm/layout/docs/PLAN.md` |
| `tinywasm/assetmin` | Buscar `.RenderHTML()` → `.String()` en extractor |
| `tinywasm/site` | Buscar `.RenderHTML()` → `.String()` |

Comando para encontrar todos los usos en el ecosistema:
```bash
grep -rn "\.RenderHTML()" /path/to/tinywasm/ --include="*.go"
```

---

## Documentación a Actualizar

### `dom/docs/ARCHITECTURE.md`

Este archivo describe `dom` como proveedor de builders JSX-like. Después del cambio ya no lo es. Actualizar:

1. **Sección 1 — Core Principles:** Eliminar bullet "JSX-like Builder". Reemplazar con:
   ```
   - **DOM-Only Layer**: Provides Element type, lifecycle interfaces, and DOM manipulation.
     HTML builders live in tinywasm/html, SVG in tinywasm/svg, images in tinywasm/image.
   ```

2. **Sección 2 — API Overview:** Actualizar `Component` interface: `RenderHTML()` → `String()`. Eliminar referencia a builders como `dom.Div()`, `dom.Button()`.

3. **Sección 3 — Creating Components:** Actualizar el ejemplo de `Render()` para usar imports de `tinywasm/html`:
   ```go
   // Cambiar:
   import . "github.com/tinywasm/dom"
   // A:
   import (
       . "github.com/tinywasm/html"
       . "github.com/tinywasm/dom"
   )
   ```

4. Agregar nueva sección **"Package Boundaries"**:
   ```markdown
   ## Package Boundaries
   | Concern | Package |
   |---|---|
   | HTML element builders | tinywasm/html |
   | SVG builders + sprite | tinywasm/svg |
   | Image builders | tinywasm/image |
   | DOM manipulation, Element type, interfaces | tinywasm/dom (this package) |
   ```

### `dom/README.md`

1. Eliminar "JSX-like Declarative View" de la lista de features.
2. Reemplazar con: "**Lifecycle & DOM API**: `Render`, `Append`, `Update`, `Get`, `OnHashChange`"
3. Agregar sección "Related Packages":
   ```markdown
   ## Related Packages
   - [tinywasm/html](https://github.com/tinywasm/html) — HTML element builders (Div, Span, Nav...)
   - [tinywasm/svg](https://github.com/tinywasm/svg) — SVG builders + icon sprite system
   - [tinywasm/image](https://github.com/tinywasm/image) — Image element builders
   ```
4. Actualizar todos los ejemplos de código que usen `dom.Div(...)` → `html.Div(...)` con el import correcto.

---

## Orden de Ejecución

1. Hacer cambio A (rename) en `dom/`
2. Hacer cambio B (eliminar builders) en `dom/`
3. Actualizar `dom/docs/ARCHITECTURE.md` y `dom/README.md`
4. Publicar nueva versión de `dom`
5. Los paquetes dependientes (components, layout, html, svg, image) actualizan en sus propios pasos

Ver `tinywasm/docs/MASTER_PLAN.md` para el orden global de ejecución.
