# PLAN: LocalStorage + DocumentAttr APIs — `tinywasm/dom`

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
- `dom` ya inyecta `theme.css` con tokens `--color-*`, `@media prefers-color-scheme` y bloques `[data-theme]`
- No hay API Go para tocar `document.documentElement` desde WASM
- No hay API Go para acceder a `localStorage` — los componentes que necesiten persistencia están bloqueados

**Alcance de este plan:**
1. API de localStorage (`LocalStorageGet/Set/Del/Clear`) — bridge JS para Web Storage
2. API de atributos del documento (`SetDocumentAttr/GetDocumentAttr`) — bridge JS para `document.documentElement`
3. Eliminar los bloques `[data-theme]` de `dom/theme.css` — se mueven a `tinywasm/components/themeswitch`

El componente visual del botón y la lógica de ciclo de tema viven en
`tinywasm/components/themeswitch`. `dom` solo expone la palanca del DOM.

---

## Decisiones confirmadas

| # | Decisión |
|---|----------|
| P1 | localStorage — **API en `dom`** porque solo `dom` puede usar `syscall/js`. Las constantes y lógica de qué guardar viven en el componente que use la API. |
| P2 | `Render()` — sin cambio de firma. `Handle` descartado. |
| P3 | `dom/theme.css` retiene solo `@media prefers-color-scheme` (modo auto). Los bloques `[data-theme="light"]` y `[data-theme="dark"]` se mueven a `themeswitch.css`. |
| P4 | `dom` expone `SetDocumentAttr/GetDocumentAttr` — bridge general para `document.documentElement`, sin semántica de tema. El tipo `Theme` y sus constantes viven en `themeswitch`. |
| Q6 | Sin sub-paquete, sin devtools inline en `dom` |
| Q7 | Sin tipo `Handle` — patrón del paquete = funciones públicas sueltas |
| Q11 | Naming: `LocalStorageGet/Set/Del/Clear` (explícito, sin abreviaturas) |
| Q12 | Firma: `LocalStorageGet` retorna `(string, error)`. `("", nil)` = clave ausente; `("", error)` = storage no disponible. Distingue explícitamente los dos casos. |
| Q13 | Tests: reales en browser via `gotest`, ubicados en `dom/tests/uc_*_test.go` |
| Q14 | Errores JS: `defer/recover` no funciona en TinyGo wasm — se usan guards O(1) + contador de quota. Safari modo privado antiguo (≤ 10) es el único caveat no cubierto. |
| Q15 | `SetDocumentAttr(attr, "")` elimina el atributo; `GetDocumentAttr` retorna `""` si ausente. Coherente con el patrón `""` = ausente de `LocalStorageGet`. |

**Por qué `dom` no tiene semántica de tema:**
`dom` es bridge JS — expone primitivas del DOM sin opinión sobre su uso.
`SetDocumentAttr("data-theme", "dark")` es la misma primitiva que
`SetDocumentAttr("lang", "es")`. La semántica de qué valores son válidos para
`data-theme`, cuándo rotar entre ellos y cómo persistirlos es responsabilidad de
`ThemeSwitch`. Separar esto evita que `dom` acumule lógica de aplicación.

**Por qué se descartó `Handle`:**
El patrón establecido en `dom` es funcional — `Render`, `Append`, `SetHash`, `GetHash`
son todas funciones de paquete, no métodos sobre un tipo.
`SetDocumentAttr/GetDocumentAttr` y `LocalStorageGet/Set/Del/Clear` siguen
exactamente el mismo patrón que `SetHash/GetHash`.

---

## Diseño

### 1. API de localStorage — bridge JS para Web Storage

**Solo `dom` accede a `syscall/js`.** Cualquier componente que necesite persistencia
usa estas funciones públicas. La interfaz `DOM` interna **no cambia** — son
funciones de paquete sueltas, igual que `SetHash`/`GetHash`.

**Archivo único `localstorage.go` con `//go:build wasm`. Sin stub backend.**

**Estrategia de protección contra quota — contador en memoria:**

`domWasm` mantiene un campo `lsUsedBytes int` que se inicializa una sola vez en
`newDom()` escaneando las claves existentes (O(n) al arrancar, amortizado O(1)
por operación). Cada `Set`/`Del`/`Clear` actualiza el contador. Antes de cada
`setItem` se comprueba si el delta cabría dentro del presupuesto — si no, se
retorna error, sin llamar a JS y sin riesgo de panic.

**Dos niveles de defensa en `LocalStorageSet`:**
1. Guard por valor (`> lsMaxValue`): O(1) puro, sin llamada JS — descarta valores
   claramente excesivos antes de consultar el storage.
2. Guard por presupuesto total (`lsUsedBytes + delta > lsMaxBytes`): O(1) con una
   llamada `getItem` para calcular el delta (valor nuevo menos valor existente).

```go
// dom_frontend.go — modificación al struct existente

type domWasm struct {
    *tinyDOM
    document     js.Value // ya existía
    localStorage js.Value // NUEVO — cacheado igual que document
    lsUsedBytes  int      // NUEVO — contador de uso; inicializado en newDom
    elementCache []struct{ ... }
    // ... resto sin cambios
}

func newDom(td *tinyDOM) DOM {
    ls := js.Global().Get("localStorage")
    used := 0
    if ls.Truthy() {
        // Scan inicial O(n) — ocurre una sola vez al arrancar.
        length := ls.Get("length").Int()
        for i := 0; i < length; i++ {
            key := ls.Call("key", i).String()
            if val := ls.Call("getItem", key); !val.IsNull() && !val.IsUndefined() {
                used += lsEntrySize(key, val.String())
            }
        }
    }
    return &domWasm{
        tinyDOM:      td,
        document:     js.Global().Get("document"),
        localStorage: ls,
        lsUsedBytes:  used,
    }
}
```

```go
// localstorage.go  (//go:build wasm)
package dom

import . "github.com/tinywasm/fmt"

// syscall/js no se importa aquí — los métodos (.Call, .IsNull, etc.) se llaman
// sobre valores ya tipados como js.Value desde domWasm. No hay referencias
// explícitas al paquete syscall/js en este archivo.

const lsMaxBytes = 4 * 1024 * 1024 // presupuesto total (bytes UTF-16 estimados; cuota típica 5MB)
const lsMaxValue = 64 * 1024        // límite por valor individual — O(1) sin llamada JS

func lsEntrySize(key, value string) int { return (len(key) + len(value)) * 2 }

// LocalStorageAvailable reports whether localStorage is accessible in the current browser context.
// Returns false when blocked by iframe sandbox, privacy settings, or private mode.
// Used internally by all LocalStorage* functions and available as a public check.
func LocalStorageAvailable() bool {
    return instance.(*domWasm).localStorage.Truthy()
}

// LocalStorageGet retrieves a value from window.localStorage.
// Returns ("", nil)   — key absent, storage is functional.
// Returns ("", error) — storage unavailable.
func LocalStorageGet(key string) (string, error) {
    if !LocalStorageAvailable() {
        return "", Err("localStorage unavailable")
    }
    v := instance.(*domWasm).localStorage.Call("getItem", key)
    if v.IsNull() || v.IsUndefined() {
        return "", nil
    }
    return v.String(), nil
}

// LocalStorageSet writes a key-value pair.
// Returns error if: storage unavailable, value > lsMaxValue, or budget exceeded.
func LocalStorageSet(key, value string) error {
    if !LocalStorageAvailable() {
        return Err("localStorage unavailable")
    }
    if len(value) > lsMaxValue {
        return Errf("localStorage value too large for key %s", key)
    }
    d := instance.(*domWasm)
    newSize := lsEntrySize(key, value)
    oldSize := 0
    if existing := d.localStorage.Call("getItem", key); !existing.IsNull() && !existing.IsUndefined() {
        oldSize = lsEntrySize(key, existing.String())
    }
    if d.lsUsedBytes+newSize-oldSize > lsMaxBytes {
        return Errf("localStorage budget exceeded for key %s", key)
    }
    d.localStorage.Call("setItem", key, value)
    d.lsUsedBytes += newSize - oldSize
    return nil
}

// LocalStorageDel removes a key. Returns error if storage unavailable.
func LocalStorageDel(key string) error {
    if !LocalStorageAvailable() {
        return Err("localStorage unavailable")
    }
    d := instance.(*domWasm)
    if existing := d.localStorage.Call("getItem", key); !existing.IsNull() && !existing.IsUndefined() {
        d.lsUsedBytes -= lsEntrySize(key, existing.String())
    }
    d.localStorage.Call("removeItem", key)
    return nil
}

// LocalStorageClear removes all keys. Returns error if storage unavailable.
func LocalStorageClear() error {
    if !LocalStorageAvailable() {
        return Err("localStorage unavailable")
    }
    d := instance.(*domWasm)
    d.lsUsedBytes = 0
    d.localStorage.Call("clear")
    return nil
}
```

**Sin `lsCall` helper:** con errores explícitos cada función hace su propia comprobación
`LocalStorageAvailable()` y llama `d.localStorage.Call(...)` directamente. El helper
era necesario para centralizar el log silencioso; con errores explícitos cada función
es responsable de su flujo — no hay lógica compartida que centralizar.

**Por qué `LocalStorageAvailable()` como función pública Y uso interno:**
el caller puede pre-comprobar una sola vez antes de un bloque de operaciones y
cortocircuitar temprano. Si no lo hace, cada función comprueba individualmente y
retorna error — no hay diferencia de comportamiento, solo de ergonomía para el caller.

**Por qué contador en `domWasm` y no variable de paquete:** consistencia con
`document` y `localStorage` — todo el estado de browser se centraliza en el
singleton, no en variables sueltas a nivel de paquete.

**Limitación conocida:** `lsUsedBytes` solo refleja las escrituras de esta app.
Si otra pestaña del mismo origen escribe en localStorage entre el scan inicial y
una operación posterior, el contador subestima el uso real. El margen de 1MB de
`lsMaxBytes` lo absorbe para preferencias UX.

**Manejo de errores — `defer/recover` NO funciona en TinyGo wasm.**

([Language support](https://tinygo.org/docs/reference/lang-support/)):
> "On architectures where `recover` is not implemented, a panic will always
> exit the program without running any deferred functions."

| Caso | ¿Cubierto? |
|------|-----------|
| `localStorage` no existe (iframe sandbox, settings bloqueados) | ✅ `LocalStorageAvailable()` → error explícito |
| Clave inexistente en Get | ✅ `("", nil)` — ausente no es error |
| Valor individual demasiado grande (> 64KB) | ✅ Guard nivel 1 — O(1), sin JS → error |
| Presupuesto total excedido (> 4MB) | ✅ Guard nivel 2 — O(1) con delta → error |
| Safari modo privado (`setItem` lanza aunque `Truthy()` = true) | ⚠️ Parcial — `LocalStorageAvailable()` puede no detectarlo en versiones antiguas |

**Safari modo privado:** Safari ≤ 10 (2016) tenía cuota cero en modo privado y
lanzaba aunque `localStorage` fuese `Truthy()`. Safari 11+ (2017) asigna cuota
real. Para apps que soporten Safari antiguo la única solución es un wrapper JS
try-catch en el template HTML antes del WASM.

---

### 2. API de atributos del documento — bridge JS para `document.documentElement`

`document.documentElement` es el elemento `<html>` — no tiene ID, por lo que
`dom.Get(id)` no puede alcanzarlo. `SetDocumentAttr`/`GetDocumentAttr` son el
bridge mínimo necesario para que componentes externos (sin acceso a `syscall/js`)
puedan escribir atributos en el root del documento.

**Dos archivos: `documentattr.go` (wasm) + `documentattr_backend.go` (!wasm).**
`GetDocumentAttr` se llama desde `Render()` de `ThemeSwitch` (sin build tag) →
necesita compilar en `!wasm`. Los stubs son no-op/`""` — coherente con `GetHash()`.

```go
// documentattr.go  (//go:build wasm)
package dom

// SetDocumentAttr sets an attribute on document.documentElement (<html>).
// value=="" removes the attribute — consistent with GetDocumentAttr returning ""
// for absent attributes.
func SetDocumentAttr(attr, value string) {
    html := instance.(*domWasm).document.Get("documentElement")
    if !html.Truthy() {
        return
    }
    if value == "" {
        html.Call("removeAttribute", attr)
    } else {
        html.Call("setAttribute", attr, value)
    }
}

// GetDocumentAttr reads an attribute from document.documentElement.
// Returns "" if the attribute is absent.
func GetDocumentAttr(attr string) string {
    html := instance.(*domWasm).document.Get("documentElement")
    if !html.Truthy() {
        return ""
    }
    v := html.Call("getAttribute", attr)
    if v.IsNull() || v.IsUndefined() {
        return ""
    }
    return v.String()
}
```

```go
// documentattr_backend.go  (//go:build !wasm)
package dom

func SetDocumentAttr(_, _ string) {}
func GetDocumentAttr(_ string) string { return "" }
```

**Por qué no-op y no estado en memoria:**

- `document.documentElement` es estado del browser — no existe en SSR.
- Un mapa de paquete sería compartido entre requests concurrentes: estado de
  una petición contaminaría otra. Bug de concurrencia garantizado.
- El servidor no puede conocer la preferencia de tema sin una cookie explícita.
  Sin cookie, `GetDocumentAttr` siempre devolvería `""` de todas formas.
- WASM hidrata en <100ms: el primer shell del servidor es inmediatamente
  reemplazado por el render del cliente, que llama `Init()` → lee localStorage
  → re-renderiza con el tema correcto.
- Coherente con el patrón del paquete: `GetHash()` backend también devuelve `""`
  por la misma razón — el servidor no tiene acceso a estado client-only.

**Por qué `value=""` elimina el atributo:**
En el uso de tema, `data-theme` ausente = modo auto. Pasar `""` como valor de
"vaciar" es consistente con el patrón del paquete (`LocalStorageGet` retorna `""`
para claves ausentes). El caller no necesita saber si debe usar `removeAttribute`
o `setAttribute("")` — solo pasa `""`.

**Por qué usa `document.documentElement` y no cacheo:**
`documentElement` es siempre el mismo objeto durante la vida de la página.
`instance.(*domWasm).document.Get("documentElement")` hace una sola llamada JS
adicional por operación — aceptable para operaciones que no están en hot paths.
Cachear añadiría un campo más a `domWasm` para dos funciones de baja frecuencia.

**Regla del paquete:**

| API | ¿Llamada desde código sin build tag? | ¿Stub `!wasm`? |
|-----|--------------------------------------|----------------|
| `Render`, `Append`, `Update`, `Get` | Sí (interfaz `DOM`) | Sí — patrón existente |
| `SetDocumentAttr`, `GetDocumentAttr` | Sí (`ThemeSwitch.Render()` sin tag) | **Sí — no-op / `""` (coherente con `GetHash`)** |
| `LocalStorage*` | No (solo desde `*_wasm.go`) | **No** |

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

Todos en browser real via `gotest`. Patrón: `dom/tests/uc_*_test.go` con
`//go:build wasm` y `package dom_test`.

**Convención de ubicación:**
- `tests/` — tests de API pública (`package dom_test`). Todos los tests nuevos van aquí.
- Raíz del paquete — solo tests que requieren acceso a internals (`package dom`).

### `dom/tests/uc_localstorage_test.go`

| Test | Verifica |
|------|----------|
| `TestLocalStorageAvailable_ReturnsTrue` | En entorno browser → `LocalStorageAvailable() == true` |
| `TestLocalStorage_SetGet_Roundtrip` | `Set` retorna `nil`; `Get` retorna el mismo valor y `nil` |
| `TestLocalStorage_Get_MissingKey_ReturnsEmptyNilError` | Get sobre clave inexistente → `("", nil)` — ausente no es error |
| `TestLocalStorage_Del_RemovesKey` | Set + Del + Get → `("", nil)` |
| `TestLocalStorage_Set_Overwrites` | Set sobre clave existente → nuevo valor, `nil` error |
| `TestLocalStorage_Clear_RemovesAll` | Set varias claves + Clear → todas retornan `("", nil)` |
| `TestLocalStorage_SetEmptyValue` | Set con `value=""` → Get retorna `("", nil)` (indistinguible de ausente) |
| `TestLocalStorage_OversizedValue_ReturnsError` | Set con valor > 64KB → error `"too large"`, clave no escrita |
| `TestLocalStorage_QuotaGuard_ReturnsError` | Llenar storage hasta cerca del límite → Set siguiente retorna error `"budget exceeded"`, sin panic |

### `dom/tests/uc_documentattr_test.go`

| Test | Verifica |
|------|----------|
| `TestSetDocumentAttr_SetsAttribute` | `SetDocumentAttr("data-theme", "dark")` → `<html data-theme="dark">` |
| `TestSetDocumentAttr_EmptyValue_RemovesAttribute` | Set luego `SetDocumentAttr("data-theme", "")` → atributo eliminado |
| `TestGetDocumentAttr_NoAttribute_ReturnsEmpty` | Sin atributo → `GetDocumentAttr("data-theme") == ""` |
| `TestGetDocumentAttr_AfterSet_ReturnsValue` | Set + Get round-trip devuelve el mismo valor |
| `TestSetDocumentAttr_PassesThrough_AnyString` | Valor arbitrario `"xyz"` se escribe literal — sin validación |

**Cleanup en cada test:** `SetDocumentAttr("data-theme", "")` y `LocalStorageClear()`
(ignorar el error retornado en cleanup) antes/después para evitar contaminación.

---

## Checklist de implementación

Prerequisito: `go install github.com/tinywasm/devflow/cmd/gotest@latest`

### LocalStorage API
- [ ] Modificar `dom_frontend.go`: añadir `localStorage js.Value` y `lsUsedBytes int` a `domWasm`; en `newDom()` cachear el handle y hacer el scan inicial O(n) para inicializar `lsUsedBytes`
- [ ] Crear `localstorage.go` (`//go:build wasm`) — `LocalStorageAvailable()` + 4 funciones con retorno `error` explícito, guards de quota (nivel 1: tamaño valor; nivel 2: presupuesto total); sin `lsCall` helper; sin `import "syscall/js"`
- [ ] **Sin** archivo backend stub (la API solo se llama desde código `wasm`)
- [ ] Crear `tests/uc_localstorage_test.go` con los 9 tests listados arriba

### DocumentAttr API
- [ ] Crear `documentattr.go` (`//go:build wasm`) — `SetDocumentAttr` + `GetDocumentAttr` vía `document.documentElement`
- [ ] Crear `documentattr_backend.go` (`//go:build !wasm`) — stubs no-op/`""` (no estado en memoria; coherente con `GetHash()` backend)
- [ ] Crear `tests/uc_documentattr_test.go` con los 5 tests listados arriba

### CSS
- [ ] En `dom/theme.css`: eliminar los bloques `[data-theme="light"]` y `[data-theme="dark"]` — se mueven a `themeswitch.css` (ver `components/themeswitch/docs/PLAN.md`)

### Tests — carpeta
- [ ] Renombrar `test/` → `tests/` si ya existe la carpeta (o crearla directamente como `tests/`)
- [ ] Mover cualquier test existente de API pública de la raíz a `tests/`

### Integración
- [ ] Actualizar `web/client.go` con ejemplo de uso de ambas APIs
- [ ] Actualizar `docs/ARCHITECTURE.md`: nueva sección "LocalStorage API" + nueva sección "DocumentAttr API"
- [ ] Verificar con `gotest` (corre browser + tests en build wasm)
- [ ] `gopush 'feat(dom): add LocalStorage and DocumentAttr APIs'`
