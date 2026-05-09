# PLAN: Theme + LocalStorage API — `tinywasm/dom`

## Prerequisito de instalación

Antes de implementar/testear, instalar el runner de tests:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

`gotest` levanta un browser real y ejecuta los tests con build tag `wasm` —
indispensable porque `localStorage` solo existe en entorno browser.

---

## Contexto

`tinywasm/dom` es el **único paquete del ecosistema tinywasm que importa
`syscall/js`**. Cualquier acceso a APIs del browser desde otros paquetes
(`tinywasm/components/*`, apps, etc.) debe pasar por funciones públicas de `dom`.

Estado actual:
- `dom` ya inyecta `theme.css` con tokens `--color-*` y selector `[data-theme]`
- No hay API Go para tocar `data-theme` desde WASM
- No hay API Go para acceder a `localStorage` — los componentes que necesiten
  persistencia están bloqueados

**Alcance de este plan:**
1. API de tema (`SetTheme`, `GetTheme`) — bridge JS para `data-theme` en `<html>`
2. API de localStorage (`LocalStorageGet/Set/Del/Clear`) — bridge JS para Web Storage

El componente visual del botón vive en `tinywasm/components/themeswitch`.

---

## Decisiones confirmadas

| # | Decisión |
|---|----------|
| P1 | localStorage — **API en `dom`** porque solo `dom` puede usar `syscall/js`. Las constantes y lógica de qué guardar viven en el componente que use la API. |
| P2 | `Render()` — sin cambio de firma (sigue retornando `error`). `Handle` descartado. |
| P4 | 3 estados de tema: `auto → dark → light → auto` (lógica en `ThemeSwitch`) |
| Q6 | Sin sub-paquete, sin devtools inline en `dom` |
| Q7 | Sin tipo `Handle` — patrón del paquete = funciones públicas sueltas |
| Q8 | `InitTheme` no existe en `dom` — `ThemeSwitch.Init()` lo hace |
| Q11 | Naming: `LocalStorageGet/Set/Del/Clear` (explícito, sin abreviaturas) |
| Q12 | Firma: `LocalStorageGet` retorna `string` plano (`""` = ausente). Coherente con `GetHash()/GetTheme()`, sin asignación, sin redundancia |
| Q13 | Tests: reales en browser via `gotest`, ubicados en `dom/test/uc_*_test.go` |
| Q14 | Errores JS: defensa solo con `Truthy()` (sin `defer/recover` — no soportado en TinyGo wasm). Quota exceeded y Safari modo privado son caveats documentados, no bloqueantes para preferencias UX |

**Por qué se descartó `Handle`:**
El patrón establecido en `dom` es funcional — `Render`, `Append`, `SetHash`, `GetHash`
son todas funciones de paquete, no métodos sobre un tipo. Introducir `*Handle` para
encadenar un solo método que luego se movió a `components` no tiene justificación.
`SetTheme/GetTheme` y `LocalStorageGet/Set/Del/Clear` siguen exactamente el mismo
patrón que `SetHash/GetHash`.

---

## Diseño

### 1. API de localStorage — bridge JS para Web Storage

**Solo `dom` accede a `syscall/js`.** Cualquier componente que necesite persistencia
usa estas funciones públicas. La interfaz `DOM` interna **no cambia** — son
funciones de paquete sueltas, igual que `SetHash`/`GetHash`.

**Archivo único `localstorage.go` con `//go:build wasm`. Sin stub backend.**

**Diseño coherente con el patrón de `domWasm.document`:** el handle a `localStorage`
se cachea como campo de `domWasm` en `newDom()`, evitando llamadas repetidas a
`js.Global().Get("localStorage")` en cada operación.

```go
// dom_frontend.go — modificación al struct existente

type domWasm struct {
    *tinyDOM
    document     js.Value // ya existía
    localStorage js.Value // NUEVO — cacheado igual que document
    elementCache []struct{ ... }
    // ... resto sin cambios
}

func newDom(td *tinyDOM) DOM {
    return &domWasm{
        tinyDOM:      td,
        document:     js.Global().Get("document"),
        localStorage: js.Global().Get("localStorage"), // NUEVO
    }
}
```

```go
// localstorage.go  (//go:build wasm)
package dom

import "syscall/js"

// LocalStorageGet retrieves a value from window.localStorage.
// Returns "" if the key does not exist OR if storage is unavailable.
// Coherente con GetHash()/GetTheme() — patrón del paquete.
func LocalStorageGet(key string) string {
    v := instance.(*domWasm).lsCall("getItem", key)
    if v.IsNull() || v.IsUndefined() {
        return ""
    }
    return v.String()
}

func LocalStorageSet(key, value string) { instance.(*domWasm).lsCall("setItem", key, value) }
func LocalStorageDel(key string)         { instance.(*domWasm).lsCall("removeItem", key) }
func LocalStorageClear()                 { instance.(*domWasm).lsCall("clear") }

// lsCall — único método sobre domWasm para localStorage. Centraliza:
//   1. La guarda Truthy() (única defensa posible — defer/recover no funciona en TinyGo wasm)
//   2. El log de "unavailable" (único error detectable)
//
// Retorna js.Value{} (== undefined en JS) si localStorage no está disponible.
// El caller usa IsUndefined()/IsNull() si necesita distinguir.
func (d *domWasm) lsCall(method string, args ...any) js.Value {
    if !d.localStorage.Truthy() {
        d.Log("dom: localStorage unavailable, ignoring", method)
        return js.Value{}
    }
    return d.localStorage.Call(method, args...)
}
```

**Por qué este diseño funciona:**

- Una sola estructura: 4 funciones públicas (todas one-liners excepto `Get` que decodifica el resultado) + 1 método helper sobre `domWasm`
- Sin métodos privados duplicados — la lógica vive en un solo sitio (`lsCall`)
- `js.Value{}` (zero value) representa `undefined` en JS → `IsUndefined()` retorna `true` → `LocalStorageGet` cae en el branch que retorna `""`. Sin código extra para distinguir "clave ausente" de "storage no disponible" (ambos retornan `""`)
- `instance.(*domWasm)` es type assertion a tipo concreto: prácticamente gratis, sin alocación, seguro porque el archivo solo compila bajo `//go:build wasm` (en build wasm `instance` siempre es `*domWasm`, ver `newDom` en `dom_frontend.go`)

**Por qué cacheo en `domWasm` y no en variable de paquete:** consistencia con el
campo `document` ya existente. Toda referencia a globals JS se centraliza en el
struct singleton, no en variables sueltas a nivel de paquete.

**Por qué solo WASM y sin stub backend:** estas funciones jamás se llaman desde
código sin build tag — solo desde archivos `_wasm.go` de los componentes. Un stub
`!wasm` sería código muerto. Si en el futuro un componente las necesitara desde
shared code, se añadiría el stub entonces.

**Regla del paquete:**

| API | ¿Llamada desde código sin build tag? | ¿Stub `!wasm`? |
|-----|--------------------------------------|----------------|
| `Render`, `Append`, `Update`, `Get` | Sí (interfaz `DOM`) | Sí — patrón existente |
| `SetTheme`, `GetTheme` | Sí (componentes leen tema en `Render()` no-tag) | Sí |
| `LocalStorage*` | No (solo desde `*_wasm.go`) | **No** |

**Manejo de errores JS — defensa parcial via `Truthy()`.**

`syscall/js` traduce excepciones JS a panics Go. La forma idiomática de
capturarlas (`defer/recover`) **NO funciona en TinyGo target wasm** — la doc
oficial es explícita ([Language support](https://tinygo.org/docs/reference/lang-support/)):

> "The `recover` builtin is supported on most architectures, with the notable
> exception of WebAssembly."
> "On architectures where `recover` is not implemented, a panic will always
> exit the program without running any deferred functions."

Como tinywasm/components puede compilarse con TinyGo (modos M y S), el patrón
`defer/recover` quedaría como bug latente: en mode L (Go estándar) capturaría,
en M/S la goroutine moriría. Por coherencia, no lo usamos.

**Defensa que sí funciona — `Truthy()` antes de cada operación:**

```go
if !d.localStorage.Truthy() { return "" /* o no-op */ }
d.localStorage.Call("setItem", key, value)
```

| Caso | ¿Cubierto? |
|------|-----------|
| `localStorage` no existe (iframe sandbox sin permisos, browsers raros) | ✅ Sí — `Truthy()` retorna false |
| Clave inexistente en Get | ✅ Sí — `getItem` retorna `null`, no lanza |
| **Quota exceeded** durante setItem | ❌ No — lanza, panic propaga, goroutine muere |
| **Safari modo privado** durante setItem | ❌ No — lanza, panic propaga |
| Storage bloqueado por configuración del browser | ❌ No |

**Caveats documentados (aceptados):** los tres casos no cubiertos pueden matar
la goroutine WASM en TinyGo. Para preferencias pequeñas (< 1KB tema/UX),
quota exceeded es prácticamente imposible. Safari modo privado es la única
exposición real. Si una app necesita robustez total, debe inyectar un wrapper
JS try/catch en su template HTML antes de cargar el WASM.

Justificación de la simplicidad: API mínima = la solución correcta cuando los
casos no cubiertos son edge cases. Inyectar JS helpers añade plumbing (CSP,
secuencia de carga) que no se justifica para preferencias UX típicas.

---

### 2. API de tema — bridge JS para `data-theme`

Las funciones de tema **no se añaden a la interfaz `DOM`** — son exclusivamente
WASM. La interfaz interna (`domWasm`, `domBackend`) no cambia. `Render()` no cambia.

```go
// dom_theme.go  (sin build tag — tipo y constantes compilan en ambos entornos)

type Theme string

const (
    ThemeAuto  Theme = "auto"   // sin override → OS preference via @media
    ThemeDark  Theme = "dark"
    ThemeLight Theme = "light"
)
```

```go
// dom_theme_wasm.go  (//go:build wasm)

func SetTheme(theme Theme)  // setea/borra data-theme en <html>
func GetTheme() Theme       // ThemeAuto si no hay override activo
```

```go
// dom_theme_backend.go  (//go:build !wasm)

func SetTheme(_ Theme)    {}
func GetTheme() Theme     { return ThemeAuto }
```

**Comportamiento WASM detallado:**

| Función | Acción JS |
|---------|-----------|
| `SetTheme(ThemeAuto)` | `html.removeAttribute("data-theme")` |
| `SetTheme(t)` (cualquier otro) | `html.setAttribute("data-theme", string(t))` — se pasa tal cual |
| `GetTheme()` | `html.getAttribute("data-theme")`, retorna `ThemeAuto` si nil/vacío |

**Sin validación interna en `dom`.** Si el caller pasa `Theme("xyz")`, `dom`
escribe `data-theme="xyz"` literalmente. `dom` es bridge JS, no validador. La
responsabilidad de pasar valores válidos es del caller (`ThemeSwitch.Init` valida
los valores leídos de localStorage; el resto del código usa solo las constantes
`ThemeAuto/ThemeDark/ThemeLight` que son por definición válidas).

**`SetTheme/GetTheme` no llaman internamente a `LocalStorageSet/Get`** — son APIs
ortogonales. `SetTheme` solo toca `data-theme` en `<html>`; la persistencia es una
decisión separada del componente que usa la API. Esto permite usar `SetTheme` para
preview temporal sin efectos secundarios de almacenamiento.

El ciclo de tema (`auto→dark→light`), la inicialización (`Init()` que lee
localStorage) y la decisión de qué clave persistir viven en `ThemeSwitch`, no en
`dom`.

**Uso típico desde `web/client.go` de cualquier componente:**
```go
import (
    . "github.com/tinywasm/dom"
    "github.com/tinywasm/components/themeswitch"
)

func main() {
    ts := &themeswitch.ThemeSwitch{}
    ts.Init()       // restaura tema guardado (lógica del componente, no de dom)
    Render("app", &App{})
    Append("body", ts)
    select {}
}
```

> **Nota FOUC:** `ts.Init()` aplica el tema antes del primer `Render()`, pero el
> flash visual entre la carga del HTML y el arranque del WASM no puede evitarse
> desde Go. La solución completa requiere un inline `<script>` en `<head>` del
> template HTML del servidor que lea localStorage antes de que se aplique el CSS.

---

## Tests requeridos

Todos en browser real via `gotest`. Patrón: `dom/test/uc_*_test.go` con
`//go:build wasm` y `package dom_test`.

### `dom/test/uc_localstorage_test.go`

| Test | Verifica |
|------|----------|
| `TestLocalStorage_SetGet_Roundtrip` | Set + Get retorna el mismo valor |
| `TestLocalStorage_Get_MissingKey_ReturnsEmpty` | Get sobre clave inexistente → `""` |
| `TestLocalStorage_Del_RemovesKey` | Set + Del + Get → `""` |
| `TestLocalStorage_Set_Overwrites` | Set sobre clave existente reemplaza el valor |
| `TestLocalStorage_Clear_RemovesAll` | Set varias claves + Clear → todas retornan `""` |
| `TestLocalStorage_SetEmptyValue` | Set con `value=""` → Get retorna `""` (no se distingue de ausente — comportamiento documentado) |

### `dom/test/uc_theme_test.go`

| Test | Verifica |
|------|----------|
| `TestSetTheme_Dark_SetsAttribute` | `SetTheme(ThemeDark)` → `<html>` tiene `data-theme="dark"` |
| `TestSetTheme_Light_SetsAttribute` | mismo con `ThemeLight` |
| `TestSetTheme_Auto_RemovesAttribute` | `SetTheme(ThemeAuto)` → `<html>` sin atributo `data-theme` |
| `TestSetTheme_PassesThrough_InvalidValue` | `SetTheme(Theme("xyz"))` → `data-theme="xyz"` literal (sin validación) |
| `TestGetTheme_NoAttribute_ReturnsAuto` | Sin override → `GetTheme() == ThemeAuto` |
| `TestGetTheme_AfterSet_ReturnsValue` | `SetTheme(X)` → `GetTheme() == X` para los 3 valores |
| `TestSetTheme_DoesNotTouchLocalStorage` | `SetTheme(ThemeDark)` no escribe en localStorage (separación de responsabilidades) |

**Cleanup en cada test:** `LocalStorageClear()` y reset de `data-theme` antes/después
para evitar contaminación entre tests.

---

## Checklist de implementación

Prerequisito: `go install github.com/tinywasm/devflow/cmd/gotest@latest`

### LocalStorage API
- [ ] Modificar `dom_frontend.go`: añadir campo `localStorage js.Value` a `domWasm` y cachearlo en `newDom` (mismo patrón que `document`)
- [ ] Crear `localstorage.go` (`//go:build wasm`) — 4 funciones públicas + 1 helper `lsCall` sobre `domWasm`, con check `Truthy()` defensivo (sin `defer/recover` — no funciona en TinyGo wasm)
- [ ] **Sin** archivo backend stub (la API solo se llama desde código `wasm`)
- [ ] Crear `dom/test/uc_localstorage_test.go` con los 6 tests listados arriba

### Theme API
- [ ] Crear `dom_theme.go` (sin build tag) — `type Theme string` + `ThemeAuto/ThemeDark/ThemeLight`
- [ ] Crear `dom_theme_wasm.go` — `SetTheme(Theme)` + `GetTheme() Theme`
- [ ] Crear `dom_theme_backend.go` — stubs `//go:build !wasm`
- [ ] Crear `dom/test/uc_theme_test.go` con los 7 tests listados arriba

### Integración
- [ ] Actualizar `dom/web/client.go` con ejemplo de uso de ambas APIs
- [ ] Actualizar `docs/ARCHITECTURE.md`: nueva sección "LocalStorage API" + sección 7 con nueva API de tema
- [ ] Verificar con `gotest` (corre browser + tests en build wasm)
- [ ] `gopush 'feat(dom): add LocalStorage and Theme APIs'`
