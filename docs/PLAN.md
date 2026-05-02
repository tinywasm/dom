# PLAN: Refactor de Gestión de IDs en SelectSearch (Breaking Change)

## 1. Diagnóstico del Problema Actual

El componente `SelectSearch` construye los IDs de sus sub-elementos mediante concatenación
manual de strings en cada punto de uso:

```go
// En Render():
id := c.GetID()
toggle  := dom.Input("checkbox").ID(id + "-toggle")
search  := dom.Input("search").ID(id + "-search")
optList := dom.Div().ID(id + "-options")
item    := dom.Div().ID(id + "-opt-" + opt.ID)

// En OnMount() (implícito — mismo patrón requerido para recuperar refs):
if el, ok := dom.Get(c.GetID() + "-search"); ok { ... }
if el, ok := dom.Get(c.GetID() + "-options"); ok { ... }
```

### Síntomas concretos

| Síntoma | Descripción |
|---------|-------------|
| **Duplicación silenciosa** | El string `"-search"` vive en `Render()` y en `OnMount()`. Un refactor que cambie uno y olvide el otro compila sin error pero falla en runtime. |
| **Variable local innecesaria** | `id := c.GetID()` se declara para evitar llamadas repetidas, pero en `OnMount()` hay que llamar `c.GetID()` de nuevo — la variable no se puede compartir entre métodos. |
| **Namespace de opciones sin control** | `id + "-opt-" + opt.ID` usa un `opt.ID` controlado por el usuario. Si `opt.ID` contiene `-` puede colisionar con otros slots (ej. `opt.ID = "toggle"` → `id-opt-toggle`). |
| **API no fluida** | El framework tiene `.ID(string)` como método fluido en `*Element`, pero los sub-IDs se construyen fuera del árbol declarativo, rompiendo el flujo. |
| **Magia en los strings** | `"-toggle"`, `"-search"`, `"-options"`, `"-opt-"` son literales dispersos. No hay punto único de verdad para renombrarlos. |

---

## 2. Preguntas Abiertas

Estas preguntas deben responderse antes de implementar para acotar el alcance exacto.

### 2.1 ¿Alcance del refactor: solo IDs o también estado y eventos?

El componente tiene tres problemas ortogonales:
- (A) Gestión de IDs de sub-elementos ← **este plan**
- (B) Estado interno (`isOpen`, `filterTerm`) gestionado manualmente
- (C) Eventos registrados en `OnMount()` de forma imperativa

**¿Este breaking change toca solo (A), o también (B) y (C)?**

Si solo (A): el plan es pequeño, enfocado, bajo riesgo de regresión.  
Si también (B)+(C): el alcance crece significativamente (requiere diseñar API de estado reactivo).

> **Recomendación:** Limitar este plan a (A). Los puntos (B) y (C) merecen sus propios issues.

---

### 2.2 ¿`Sub()` debe ser parte de la interfaz `Component` o solo un método de `Element`?

**Opción "solo Element":** `Sub()` se añade a `*Element`. Todos los structs que embedan
`dom.Element` lo heredan automáticamente (caso SelectSearch). Los componentes que no
embedan Element no lo tienen, pero tampoco lo necesitan.

**Opción "en Component interface":** Se añade `Sub(name string) string` a la interfaz
`Component`. Esto rompe todos los `Component` implementados manualmente (fuera de los que
embedan `dom.Element`).

> **Recomendación:** Añadir solo a `*Element`. Los consumers que embedan `dom.Element`
> lo obtienen gratis. No tiene sentido en la interfaz porque `Component` es minimal por
> diseño (ver `interface.dom.go:44`).

---

### 2.3 ¿La firma debe ser `Sub(name string) string` o `Sub(parts ...string) string`?

El caso de opciones dinámicas (`id + "-opt-" + opt.ID`) requiere combinar un scope fijo
(`"opt"`) con una clave variable (`opt.ID`).

**`Sub(name string) string`** — Simple, legible, solo cubre slots estáticos:
```go
c.Sub("search")           // → "1-search"
c.Sub("opt-" + opt.ID)    // sigue siendo concatenación manual
```

**`Sub(parts ...string) string`** — Variadic, cubre ambos casos:
```go
c.Sub("search")           // → "1-search"
c.Sub("opt", opt.ID)      // → "1-opt-42"
```

> **Recomendación:** Variadic. Resuelve el namespace de opciones dinámicas sin añadir
> métodos extra como `SubOf()` o `SubN()`.

---

### 2.4 ¿Debe `Sub()` sanitizar el `opt.ID` del usuario?

Si `opt.ID = "toggle"` y se llama `c.Sub("opt", "toggle")` → `"1-opt-toggle"`.  
Si `opt.ID = "opt-x"` y se llama `c.Sub("opt", "opt-x")` → `"1-opt-opt-x"`.

Estas colisiones son raras pero posibles. ¿El framework debe validar o sanitizar?

**Opción "no sanitizar":** Responsabilidad del caller garantizar IDs de opciones únicos.
Tiene precedente en la web estándar (el DOM no valida unicidad de IDs).

**Opción "sanitizar":** Reemplazar caracteres problemáticos (ej. `.` `/`) pero no `-`.
Añade complejidad innecesaria para un caso raro.

> **Recomendación:** No sanitizar. Documentar que `opt.ID` debe ser un identificador
> simple (sin `-` al inicio ni fin). El contrato ya existe implícitamente hoy.

---

### 2.5 ¿Los constants de slot deben definirse en el componente o el framework los ignora?

Los slots (`"toggle"`, `"search"`, `"options"`, `"opt"`) son propios de `SelectSearch`,
no del framework. ¿Se definen como `const` dentro del componente o se usan como literales?

**Constantes explícitas:**
```go
const (
    slotToggle  = "toggle"
    slotSearch  = "search"
    slotOptions = "options"
    slotOpt     = "opt"
)

toggle := dom.Input("checkbox").ID(c.Sub(slotToggle))
```

**Literales directos:**
```go
toggle := dom.Input("checkbox").ID(c.Sub("toggle"))
```

> **Recomendación:** Literales para ahora. `c.Sub("toggle")` ya centraliza el
> string en un solo punto si se usa consistentemente. Las constantes añaden
> ceremonia sin beneficio real en un componente de ~100 líneas. Si `SelectSearch`
> crece, pueden introducirse en un refactor posterior no-breaking.

---

### 2.6 ¿`isOpen` debe persistir via CSS class o via `checked` attr?

Hoy `isOpen` se usa para emitir `Attr("checked", "")` en el checkbox. El mecanismo
actual funciona, pero ¿es el problema de IDs independiente de cómo se persiste el estado?

> **Respuesta:** Sí, son independientes. Este plan no toca la lógica de `isOpen`.

---

## 3. Opciones de Diseño

### Opción A — `c.Sub(parts ...string) string` en `Element` (RECOMENDADA)

Añadir un método a `*Element`:

```go
// element.go
func (b *Element) Sub(parts ...string) string {
    return b.GetID() + "-" + strings.Join(parts, "-")
}
```

**Uso en SelectSearch:**

```go
func (c *SelectSearch) Render() *dom.Element {
    toggle := dom.Input("checkbox").
        ID(c.Sub("toggle")).
        Class("ss-toggle")

    searchInput := dom.Input("search").
        ID(c.Sub("search")).
        Class("ss-search")

    optList := dom.Div().Class("ss-options").ID(c.Sub("options"))

    for _, opt := range c.Options {
        item := dom.Div().
            ID(c.Sub("opt", opt.ID)).
            Class("ss-option")
        optList.Add(item)
    }
    // ...
}

func (c *SelectSearch) OnMount() {
    if el, ok := dom.Get(c.Sub("search")); ok {
        el.On("input", ...)
    }
    if el, ok := dom.Get(c.Sub("options")); ok {
        el.On("click", ...)
    }
}
```

| | |
|---|---|
| **Pros** | Mínimo (1 método), fluido, elimina la variable local `id`, centraliza el patrón en todo el framework, `Sub("opt", opt.ID)` resuelve dinámicos |
| **Contras** | Sigue siendo string — typos en `"sarch"` son runtime, no compile-time |

---

### Opción B — Typed Slots como constantes de paquete

Definir un tipo `Slot` y constantes en el componente:

```go
type slot string

const (
    slotToggle  slot = "toggle"
    slotSearch  slot = "search"
    slotOptions slot = "options"
)

func (b *Element) Sub(s slot) string {
    return b.GetID() + "-" + string(s)
}
```

| | |
|---|---|
| **Pros** | El compilador detecta `c.Sub("toogle")` como error de tipo; refactorizable con IDE |
| **Contras** | Requiere declarar un tipo `slot` (o que cada componente declare el suyo); no resuelve IDs dinámicos de opciones sin un escape hatch; más ceremonia |

---

### Opción C — `Ref()` que devuelve un `*Element` pre-wired

El framework gestiona un registry de sub-elementos:

```go
toggle := c.Ref("toggle", dom.Input("checkbox").Class("ss-toggle"))
```

`Ref()` asigna el ID automáticamente y registra el element para recuperación posterior
sin llamar a `dom.Get()`.

| | |
|---|---|
| **Pros** | Elimina `dom.Get()` en `OnMount()`; API verdaderamente declarativa |
| **Contras** | Cambio de arquitectura significativo; requiere un registry interno en `Element`; sale del scope de este plan; posibles problemas en re-renders con TinyGo heap |

---

### Opción D — Sin cambio en framework, solo constantes locales

No añadir nada al framework. Definir constantes en `SelectSearch`:

```go
var (
    ssToggle  = func(id string) string { return id + "-toggle" }
    ssSearch  = func(id string) string { return id + "-search" }
    ssOptions = func(id string) string { return id + "-options" }
)

toggle := dom.Input("checkbox").ID(ssToggle(c.GetID()))
```

| | |
|---|---|
| **Pros** | Zero cambios en `dom` package; no breaking change en el framework |
| **Contras** | No resuelve el problema en otros componentes; patrón ad-hoc por componente; más verboso |

---

## 4. Propuesta Recomendada

**Implementar Opción A:** añadir `Sub(parts ...string) string` a `*Element`.

Justificación:
- Es el cambio mínimo que resuelve el problema de raíz en *todos* los componentes
  del framework (no solo SelectSearch).
- No modifica la interfaz `Component` → no rompe implementaciones externas.
- Los structs con `dom.Element` embebido obtienen el método sin cambios.
- Variadic `parts` cubre tanto slots estáticos como IDs dinámicos de opciones.
- Consistente con la filosofía fluida del API existente.

---

## 5. API Propuesta

### Nuevo método en `element.go`

```go
// Sub returns a namespaced sub-element ID derived from this component's ID.
// Calling c.Sub("search") returns "<parentID>-search".
// Calling c.Sub("opt", optID) returns "<parentID>-opt-<optID>".
func (b *Element) Sub(parts ...string) string {
    return b.GetID() + "-" + strings.Join(parts, "-")
}
```

### SelectSearch refactorizado

```go
func (c *SelectSearch) Render() *dom.Element {
    headerText := c.Placeholder
    if c.selectedLabel != "" {
        headerText = c.selectedLabel
    }
    if headerText == "" {
        headerText = "Select..."
    }

    toggle := dom.Input("checkbox").
        ID(c.Sub("toggle")).
        Class("ss-toggle")
    if c.isOpen {
        toggle.Attr("checked", "")
    }

    header := dom.Label().
        Attr("for", c.Sub("toggle")).
        Class("ss-header").
        Text(headerText).
        Add(dom.Svg(dom.Use().Attr("href", "#ss-arrow-down")).Class("ss-icon"))

    searchInput := dom.Input("search").
        ID(c.Sub("search")).
        Class("ss-search").
        Attr("placeholder", "Search...").
        Attr("value", c.filterTerm)

    optList := dom.Div().Class("ss-options").ID(c.Sub("options"))

    filterTerm := fmt.Convert(c.filterTerm).ToLower().String()
    for _, opt := range c.Options {
        if !c.matches(opt, filterTerm) {
            continue
        }
        item := dom.Div().
            ID(c.Sub("opt", opt.ID)).
            Class("ss-option").
            Attr("data-id", opt.ID).
            Attr("data-description", opt.Description).
            Add(dom.Span().Class("ss-label").Text(opt.Label))

        if opt.Description != "" {
            item.Add(dom.Span().Class("ss-desc").Text(opt.Description))
        }
        optList.Add(item)
    }

    dropdown := dom.Div().Class("ss-dropdown").
        Add(searchInput).
        Add(optList)

    return dom.Div().
        Class("ss-box").
        Add(toggle).
        Add(header).
        Add(dropdown)
}
```

### Antes vs. Después

| Antes | Después |
|-------|---------|
| `id := c.GetID()` + literal disperso | `c.Sub("toggle")` — sin variable local |
| `id + "-toggle"` (2 lugares) | `c.Sub("toggle")` (1 definición por slot) |
| `id + "-opt-" + opt.ID` (concatenación en cascada) | `c.Sub("opt", opt.ID)` |
| Variable `id` solo útil en `Render()`, no en `OnMount()` | `c.Sub()` disponible en cualquier método |

---

## 6. Plan de Implementación

### Paso 1 — Añadir `Sub()` a `element.go`

- Archivo: `element.go`
- Cambio: añadir método `Sub(parts ...string) string`
- Dependencia: `strings.Join` (agregar import si no existe en el build tag correspondiente)
- Tests: añadir unit test en `dom_internal_test.go`

### Paso 2 — Refactorizar `SelectSearch.Render()`

- Eliminar `id := c.GetID()`
- Reemplazar todas las ocurrencias de `id + "-<slot>"` con `c.Sub("<slot>")`
- Reemplazar `id + "-opt-" + opt.ID` con `c.Sub("opt", opt.ID)`
- Reemplazar `Attr("for", id+"-toggle")` con `Attr("for", c.Sub("toggle"))`

### Paso 3 — Actualizar `OnMount()` de SelectSearch (si existe)

- Reemplazar `dom.Get(c.GetID() + "-search")` con `dom.Get(c.Sub("search"))`
- Patrón idéntico para todos los slots

### Paso 4 — Actualizar tests existentes

- `uc_child_listeners_test.go`: `SearchChild` usa `c.GetID()+"-search"` → `c.Sub("search")`
- `uc_self_update_test.go`: `SSChild` usa `c.GetID()+"-search"`, `c.GetID()+"-options"`,
  `c.GetID()+"-opt-a"` → `c.Sub("search")`, `c.Sub("options")`, `c.Sub("opt", "a")`
- `uc_builder_test.go`: `CounterComp` usa `c.GetID()+"-val"`, `c.GetID()+"-btn"` → migrar

### Paso 5 — Documentar en `ARCHITECTURE.md`

Añadir sección sobre el patrón de sub-IDs recomendado.

---

## 7. Análisis de Impacto (Breaking Change)

### Lo que se rompe

| Componente | Cambio requerido | Riesgo |
|------------|------------------|--------|
| `SelectSearch.Render()` | Reemplazar concatenaciones | Bajo — refactor mecánico |
| `SelectSearch.OnMount()` | Reemplazar `c.GetID()+"-*"` | Bajo |
| Tests `uc_child_listeners_test.go` | Reemplazar en `SearchChild` | Bajo |
| Tests `uc_self_update_test.go` | Reemplazar en `SSChild`, `SelfUpdater` | Bajo |
| Tests `uc_builder_test.go` | Reemplazar en `CounterComp` (opcional) | Ninguno — `CounterComp` es de test |
| Consumidores externos del framework | **No rompe** — `Sub()` es adición pura | Ninguno |

### Lo que NO se rompe

- La interfaz `Component` no cambia → cero impacto en implementaciones externas.
- `GetID()` y `SetID()` siguen existiendo sin cambios.
- El comportamiento en runtime es idéntico — `c.Sub("search")` produce el mismo
  string que `c.GetID()+"-search"`.

### Riesgo de regresión

Bajo. El cambio es puramente semántico (mismo output, mejor ergonomía). Los tests
existentes validan el comportamiento; la migración puede hacerse test-a-test.

---

## 8. Preguntas que Quedan Abiertas para el Equipo

1. **¿`Sub()` debe agregarse también a la interfaz `Component`?**  
   Pros: disponible sin type assertion en código que trabaja con `Component` genérico.  
   Contras: rompe todas las implementaciones manuales de `Component`.

2. **¿Deben los tests de `uc_builder_test.go` (`CounterComp`) migrarse en este PR?**  
   Es un componente de test, no producción, pero migrar establece el patrón canónico.

3. **¿`strings.Join` está disponible en el build target TinyGo/WASM?**  
   Si no, la implementación sería un loop manual o `fmt.Sprintf` con separadores.  
   Alternativa: `b.GetID() + "-" + part[0]` para 1 part, acumulación manual para N.

4. **¿El nombre `Sub` es suficientemente expresivo, o preferir `ChildID` / `SlotID` / `SubID`?**  
   `Sub` es conciso y consistente con terminología de componentes web.  
   `SlotID` es más explícito sobre qué retorna.
