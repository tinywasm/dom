# PLAN: Componentes Declarativos y Tipados — Eliminar IDs Internos (Breaking Change)

> Reemplaza el plan anterior basado en `c.Sub("slot")`. Aquel cambio solo embellecía
> la sintaxis del puente entre `Render()` y `OnMount()`. Este plan elimina el puente.

---

## 0. Reformulación del Problema

El plan anterior proponía sustituir `id + "-toggle"` por `c.Sub("toggle")`. Tras revisión,
**ese cambio NO ataca la raíz del problema**, solo lo disfraza:

- `c.Sub("toggle")` sigue siendo un **string** que vive en dos lugares (Render y OnMount).
- Sigue habiendo dos fases: una declarativa (Render) y otra imperativa (OnMount + `dom.Get`).
- Los IDs siguen siendo el **puente** entre fases. El puente es la fuente del caos.

La pregunta correcta no es _"¿cómo nombramos mejor los sub-IDs?"_ sino:

> **¿Por qué un componente necesita IDs internos para su propio cableado de eventos?**

La respuesta honesta: **no los necesita**. Los IDs internos solo existen porque el
componente está partido en dos fases. Si los eventos se cablean **durante** `Render()`
mediante closures, los sub-IDs son ruido.

---

## 1. Insight Central

El framework **ya soporta eventos declarativos**: `*Element.On(eventType, handler)`
existe en `element.go:43` y los registra en el árbol antes del mount. Cuando el
elemento se materializa en el DOM, el framework cablea el listener real
(ver `dom_frontend.go:248-250`, `mountRecursive`).

`SelectSearch` (y los componentes de tests `SearchChild`, `SSChild`, `SelfUpdater`,
`CounterComp`) **no usan** este mecanismo. Usan `OnMount() + dom.Get(idString)`
imperativo. Esa decisión es la fuente de:

- Sub-IDs concatenados a mano (`id + "-search"`, `id + "-opt-" + opt.ID`)
- Duplicación de literales entre `Render()` y `OnMount()`
- Variables locales `id := c.GetID()` que no cruzan métodos
- Acoplamiento posicional silencioso (typo en `"-sarch"` compila, falla en runtime)
- Imposibilidad de Go de detectar enlaces rotos en compile-time

Eliminamos todo esto si los eventos se atan junto al elemento que los emite.

---

## 2. Diseño Propuesto

### 2.1 Principio rector

> Un componente se describe en una sola función `Render()`. Cada elemento lleva
> sus eventos pegados. Los IDs solo aparecen donde la **semántica HTML** los exige
> (`<label for>`, `aria-*`, integración externa explícita) — y siempre vía
> referencia tipada, nunca por string.

### 2.2 Eventos como closures en `Render()`

**Antes** (patrón actual en SelectSearch):

```go
// Render():
dom.Input("search").ID(id+"-search").Attr("value", c.filterTerm)

// OnMount() (en otra parte del archivo):
if el, ok := dom.Get(c.GetID() + "-search"); ok {
    el.On("input", func(e dom.Event) {
        c.filterTerm = e.TargetValue()
        c.Update()
    })
}
```

**Después**:

```go
// Render() — todo junto:
dom.Input("search").
    Class("ss-search").
    Attr("value", c.filterTerm).
    On("input", c.onSearchInput)
```

Donde `c.onSearchInput` es un método del componente:

```go
func (c *SelectSearch) onSearchInput(e dom.Event) {
    c.filterTerm = e.TargetValue()
    c.Update()
}
```

**Por qué esto es mejor:**

- El handler vive **junto** al elemento que lo dispara — leer `Render()` es leer la lógica completa.
- Refactor del DOM no invalida cableado: si renombras `search` a `query`, el `.On()` se mueve con el elemento porque está atado a él.
- El método `c.onSearchInput` es un valor de Go normal: testeable, debuggeable, refactorable con IDE.
- Sin `dom.Get()`, sin strings, sin sub-IDs.

### 2.3 Pareo `<label for>` tipado: `.For(*Element)`

El único caso legítimo de IDs internos en SelectSearch es el pareo `<label for="">` con
`<input id="">`. Hoy se hace con dos strings idénticas:

```go
toggle := dom.Input("checkbox").ID(id + "-toggle")        // ID emitido
header := dom.Label().Attr("for", id + "-toggle")         // ID referenciado
```

**Propuesta**: nuevo método `For(other *Element)` que pasa la referencia Go, no el string:

```go
// element.go (nuevo):
// For sets the for= attribute pointing to other's ID, auto-generating
// other's ID if it has none. Use for label/input pairing and aria-* references.
func (b *Element) For(other *Element) *Element {
    return b.Attr("for", other.GetID())
}
```

Uso:

```go
toggle := dom.Input("checkbox").Class("ss-toggle")
header := dom.Label().For(toggle).Class("ss-header").Text(headerText)
```

**Beneficios**:

- **Type-safe**: si renombras la variable Go `toggle` a `checkbox`, el compilador exige actualizar `For(checkbox)`. Imposible con strings.
- **Sin coordinar literales**: el ID se auto-genera lazy en el primer `GetID()`.
- **Orden de declaración explícito**: `toggle` debe construirse antes que `header`. Esto es Go natural y lectura top-down.

### 2.4 Listas dinámicas con closures

El caso `for _, opt := range c.Options` no necesita IDs. Cada item recibe sus eventos
por closure que captura `opt`:

```go
list := dom.Div().Class("ss-options")
for _, opt := range c.filteredOptions() {
    opt := opt  // captura por valor (Go idiomático)
    item := dom.Div().Class("ss-option").
        Attr("data-id", opt.ID).
        On("click", func(e dom.Event) { c.selectOption(opt) }).
        Add(dom.Span().Class("ss-label").Text(opt.Label))
    if opt.Description != "" {
        item.Add(dom.Span().Class("ss-desc").Text(opt.Description))
    }
    list.Add(item)
}
```

`data-id` es un atributo HTML (no un ID), así que `opt.ID` no contamina ningún namespace.
Si necesitas el dato dentro del handler, ya lo tienes en `opt` por captura.

### 2.5 Para los pocos casos donde un ID externo SÍ es necesario

Algunas integraciones requieren un ID estable accesible desde fuera (JS externo, tests
end-to-end, hooks de instrumentación). Para esos, exponer un opt-in explícito:

```go
// Uso (raro y deliberado):
dom.Form().WithStableID("login-form")  // ID humano-legible, garantizado estable
```

Esto **no se usa internamente** por el componente. Es una salida de emergencia documentada
para integración externa. La regla del framework: _si un componente usa `WithStableID`,
declara públicamente que ese sub-elemento es parte de su contrato externo._

---

## 3. SelectSearch refactorizado completo

```go
package selectsearch

import (
    "github.com/tinywasm/dom"
    "github.com/tinywasm/fmt"
)

type Option struct {
    ID          string
    Label       string
    Description string
}

type SelectSearch struct {
    dom.Element // value embed (TinyGo heap constraint)

    Placeholder string
    Options     []Option
    OnSelect    func(id, description string)
    OnSearch    func(term string) []Option

    selectedLabel string
    filterTerm    string
    isOpen        bool
}

func (c *SelectSearch) Render() *dom.Element {
    headerText := c.Placeholder
    if c.selectedLabel != "" {
        headerText = c.selectedLabel
    }
    if headerText == "" {
        headerText = "Select..."
    }

    toggle := dom.Input("checkbox").Class("ss-toggle")
    if c.isOpen {
        toggle.Attr("checked", "")
    }

    header := dom.Label().
        For(toggle).                           // typed pairing — sin strings
        Class("ss-header").
        Text(headerText).
        Add(dom.Svg(dom.Use().Attr("href", "#ss-arrow-down")).Class("ss-icon"))

    search := dom.Input("search").
        Class("ss-search").
        Attr("placeholder", "Search...").
        Attr("value", c.filterTerm).
        On("input", c.onSearchInput)            // handler junto al elemento

    list := dom.Div().Class("ss-options")
    filterTerm := fmt.Convert(c.filterTerm).ToLower().String()
    for _, opt := range c.Options {
        if !c.matches(opt, filterTerm) {
            continue
        }
        opt := opt
        item := dom.Div().Class("ss-option").
            Attr("data-id", opt.ID).
            On("click", func(e dom.Event) { c.selectOption(opt) }).
            Add(dom.Span().Class("ss-label").Text(opt.Label))
        if opt.Description != "" {
            item.Add(dom.Span().Class("ss-desc").Text(opt.Description))
        }
        list.Add(item)
    }

    return dom.Div().Class("ss-box").
        Add(toggle).
        Add(header).
        Add(dom.Div().Class("ss-dropdown").
            Add(search).
            Add(list))
}

func (c *SelectSearch) onSearchInput(e dom.Event) {
    c.filterTerm = e.TargetValue()
    if len(c.filteredOptions()) == 0 && c.OnSearch != nil {
        c.Options = c.OnSearch(c.filterTerm)
    }
    c.Update()
}

func (c *SelectSearch) selectOption(opt Option) {
    c.selectedLabel = opt.Label
    c.isOpen = false
    if c.OnSelect != nil {
        c.OnSelect(opt.ID, opt.Description)
    }
    c.Update()
}

func (c *SelectSearch) matches(opt Option, term string) bool {
    if term == "" {
        return true
    }
    return fmt.Contains(fmt.Convert(opt.Label).ToLower().String(), term) ||
        fmt.Contains(fmt.Convert(opt.Description).ToLower().String(), term)
}

func (c *SelectSearch) filteredOptions() []Option {
    term := fmt.Convert(c.filterTerm).ToLower().String()
    out := make([]Option, 0, len(c.Options))
    for _, o := range c.Options {
        if c.matches(o, term) {
            out = append(out, o)
        }
    }
    return out
}
```

### Lo que desaparece

- ❌ Variable `id := c.GetID()` (ya no hace falta cruzar IDs entre métodos)
- ❌ `id + "-toggle"`, `id + "-search"`, `id + "-options"`, `id + "-opt-" + opt.ID`
- ❌ Todo el método `OnMount()` (los eventos se cablean en `Render()`)
- ❌ Todas las llamadas `dom.Get(...)` internas
- ❌ Atributo `ID(...)` en sub-elementos (excepto donde la semántica HTML lo exige)

### Lo que aparece

- ✅ Métodos privados (`onSearchInput`, `selectOption`) — testables aisladamente
- ✅ `header.For(toggle)` — pareo tipado por referencia Go
- ✅ Closures `func(e dom.Event) { c.selectOption(opt) }` — captura idiomática
- ✅ Lectura top-down: el árbol DOM y su lógica viven juntos

---

## 4. Cambios requeridos en el framework

| # | Cambio | Archivo | Tipo |
|---|--------|---------|------|
| 1 | Añadir `(*Element).For(other *Element) *Element` | `element.go` | Adición |
| 2 | Confirmar que `.On()` registrado en `Render()` se re-cablea en `Update()` | `dom_frontend.go` | Validación + tests |
| 3 | (Opcional) `(*Element).WithStableID(name string) *Element` para opt-in de IDs públicos | `element.go` | Adición |
| 4 | Documentar el patrón declarativo como canónico | `docs/ARCHITECTURE.md` | Doc |

**Lo que NO se toca:**
- Interfaz `Component` (sigue minimal: `GetID`, `SetID`, `RenderHTML`, `Children`)
- `dom_backend.go` / SSR (CSS estático sigue en `ssr.go`, correcto y deliberado)
- `OnMount()` sigue existiendo para casos válidos (medir tamaños DOM, init de libs JS)
- `dom.Get()` sigue existiendo para integración externa con IDs estables

---

## 5. Justificación de cada decisión

### ¿Por qué eventos en Render() y no en OnMount()?

- **Localidad**: leer un componente top-down te muestra elemento + lógica juntos. El cerebro no salta de método en método para reconstruir el cableado.
- **El framework ya lo soporta**: `Element.On()` existe. `mountRecursive` ya cablea los listeners. Estamos infrautilizando lo que el framework ya ofrece.
- **Cero overhead de IDs**: si el cableado vive con el elemento, no necesitas un nombre intermedio para encontrarlo después.
- **Re-renders predecibles**: cada `Update()` reconstruye el árbol con sus listeners frescos. No hay listeners "huérfanos" apuntando a nodos viejos.

### ¿Por qué `.For(other *Element)` en lugar de `.For(name string)`?

- **Type-safe**: el compilador verifica la existencia del target. Renombrar variables Go propaga.
- **No requiere acordar un nombre**: con strings, dos sitios deben coincidir; con referencia, hay un único valor.
- **Side-effect aceptable**: `For()` puede invocar `other.GetID()` que auto-genera. Aceptable porque genera un ID **anónimo** que el componente no necesita conocer; solo el framework lo usa en runtime para el HTML emitido.

### ¿Por qué no introducir `Slot[T]` u otra abstracción genérica?

- **TinyGo y generics**: hay soporte parcial pero con costo en tamaño binario.
- **Closures resuelven el problema sin nueva primitiva**: lo que `Slot[T]` haría (mantener referencia al elemento + sus eventos) ya lo hacen variables locales en `Render()`.
- **Ockham**: añadir `Slot` para resolver lo que ya resuelven variables Go normales es complejidad innecesaria.

### ¿Por qué no eliminar `OnMount()` del framework?

- Hay casos legítimos: medir geometría real del DOM, inicializar libs JS de terceros, sincronizar con `requestAnimationFrame`. Estos requieren el DOM montado.
- Pero **no es el lugar para cablear eventos del propio componente**. Esa es la regla nueva.

---

## 6. Lo que aporta al framework

1. **Patrón canónico claro**: "un componente = una función Render con todo dentro". Reduce la matriz de decisiones del autor de componentes.
2. **Type safety real en sub-elementos**: refactorizar el orden de elementos no rompe enlaces silenciosamente.
3. **Mejor integración con CSS estático en SSR**: como los IDs internos desaparecen, el CSS no depende de strings de IDs generados — lo cual es exactamente la separación que el usuario ya validó como correcta. El CSS opera sobre **clases** (estables, manuales, semánticas), nunca sobre IDs auto-generados.
4. **Componentes más pequeños y testables**: SelectSearch pasa de ~95 líneas con dos métodos a ~70 con métodos privados claros.
5. **Mensaje de adopción simple**: para nuevos contributors, "atá tus eventos donde declaras tus elementos" es una regla de una línea.
6. **Reducción de superficie de bug**: eliminamos toda una categoría de bugs (typos en concatenación, IDs duplicados entre componentes, opciones de usuario que colisionan con slots).

---

## 7. Trade-offs honestos

### Costos

| Costo | Mitigación |
|-------|------------|
| Closures por handler en cada Render() — presión sobre GC de TinyGo | Benchmark antes de mergear; si es problema, considerar handlers reusables como `func(c *SelectSearch, opt Option) func(dom.Event)` |
| `.For(target)` exige declarar `target` antes — orden léxico forzado | Es Go idiomático; documentar como regla |
| Migración de 4-5 componentes de test al patrón nuevo | Refactor mecánico, ~1 hora total |
| Pérdida de IDs sub-namespaced para acceso externo | Opt-in vía `WithStableID(name)` cuando el contrato lo requiera |

### Riesgos a validar antes de mergear

- **R1**: ¿`.On()` en Render() captura state actualizado tras `Update()`?  
  → Test: mutar campo, llamar Update(), disparar evento, verificar que el handler ve el valor nuevo.
- **R2**: ¿TinyGo + closures con captura inflan binario?  
  → Comparar tamaño WASM antes/después del refactor de SelectSearch.
- **R3**: ¿Hay race conditions en re-cableado de listeners durante Update()?  
  → Cubierto por `uc_self_update_test.go` actualmente; añadir asserts adicionales.

---

## 8. Plan de ejecución

1. **Implementar `(*Element).For(other *Element)`** en `element.go` (con test unitario en `dom_internal_test.go`).
2. **Refactorizar SelectSearch** según sección 3.
3. **Migrar componentes de test** al patrón declarativo:
   - `uc_child_listeners_test.go` → `SearchChild` con `.On("input", ...)` en Render
   - `uc_self_update_test.go` → `SSChild`, `SelfUpdater`
   - `uc_builder_test.go` → `CounterComp`
4. **Tests nuevos** que validen los riesgos R1–R3.
5. **Actualizar `docs/ARCHITECTURE.md`** con sección "Patrones de componentes":
   - Regla 1: eventos en Render(), no en OnMount()
   - Regla 2: para `for=` accesibilidad usar `.For(target)`
   - Regla 3: IDs estables solo con `.WithStableID(name)` y solo para contrato externo
6. **(Opcional) Implementar `WithStableID`** si algún consumer lo justifica.

---

## 9. Comparación con el plan anterior

| | Plan anterior (`Sub()`) | Plan nuevo (declarativo) |
|---|------------------------|--------------------------|
| Filosofía | "Mejor sintaxis para sub-IDs" | "No tener sub-IDs" |
| Cambio en SelectSearch | Cosmético (renombra concatenaciones) | Estructural (elimina OnMount) |
| Type safety | string-based, runtime errors | reference-based, compile errors |
| Sub-IDs en componente | `c.Sub("toggle")`, `c.Sub("opt", id)` | No existen |
| Pareo label/input | `c.Sub("toggle")` en ambos lados (string) | `header.For(toggle)` por referencia Go |
| Eventos | `OnMount() + dom.Get(c.Sub(...))` | `.On(event, c.handler)` en Render() |
| Líneas en SelectSearch | ~95 (con OnMount externo) | ~70 |
| Aporte al framework | 1 método cosmético | Patrón canónico + type safety |
| Mensaje a nuevos contribuyentes | "Recuerda usar Sub() en ambos lados" | "Cablea eventos donde declaras elementos" |
| Lo que pasa con IDs estáticos | Persisten en runtime, solo cambia cómo los escribes | Desaparecen del componente; opt-in si hace falta |

---

## 10. Preguntas para tu revisión

1. **¿OK con eliminar el patrón `OnMount + dom.Get(idString)` para wiring de eventos en componentes oficiales?**  
   `OnMount` sigue existiendo para casos legítimos (medir DOM, JS interop), pero deja de ser el lugar canónico para cablear eventos.

2. **¿OK con `For(other *Element)` mutando `other` lazy (auto-asigna ID)?**  
   La alternativa es exigir asignación manual previa, lo que reintroduce strings.

3. **¿`.WithStableID(name)` se implementa ahora o se pospone hasta que un caso real lo justifique?**  
   YAGNI dice esperar. Pero si hay integración con CSS o JS externo planeada, mejor adelantarlo.

4. **¿El refactor incluye también los tests `CounterComp`, `SearchChild`, `SSChild`?**  
   Migrarlos establece el patrón canónico; mantenerlos en el patrón viejo permite verificar compatibilidad backward del API legacy `dom.Get`.

5. **¿Aceptable benchmarear tamaño WASM antes de mergear?**  
   Closures en TinyGo pueden tener costo; necesitamos número real, no asunción.
