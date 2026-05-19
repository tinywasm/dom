# PLAN: tinywasm/dom — Reference interface mutation methods

## Contexto

`tinywasm/form` necesita actualizar el estado del DOM de forma quirúrgica (resetear valores
de inputs, deshabilitar botones, actualizar texto de spans de error) sin destruir los event
listeners registrados via `el.On(...)`.

La única API disponible actualmente para modificar el DOM es `dom.Render(parentID, component)`,
que llama `cleanupChildren(parentID)` antes de escribir el nuevo `innerHTML`. Esto elimina
todos los listeners registrados en los hijos del parent — haciendo que inputs y botones
queden sordos tras cualquier actualización de estado.

---

## Bugs reproducidos

Los tres bugs están reproducidos en:
`tinywasm/dom/tests/uc_reference_mutation_test.go`

### Bug 1 — Sin `SetValue`: reset de inputs destruye listeners

**Síntoma**: `form.Reset()` usa `dom.Render(inputID, ...)` para vaciar un input.
`dom.Render` llama `cleanupChildren` → elimina el listener `"input"` del campo →
el input deja de sincronizar su valor con el struct → datos corruptos en el siguiente envío.

**Causa raíz**: `Reference` solo expone `Value() string` (getter). No existe `SetValue(string)`.
La única alternativa es re-renderizar, lo que destruye listeners.

**Fix**: `ref.SetValue("") → element.value = ""` directo via JS. Sin re-render, sin limpieza
de listeners.

### Bug 2 — Sin `SetAttr`/`RemoveAttr`: deshabilitar botón destruye listeners

**Síntoma**: para mostrar loading state en el botón submit, `form` usa `dom.Render` con
`Attr("disabled","true")`. Esto destruye el listener `"submit"` del form (el botón es hijo
del form). Al llamar `done()` y re-renderizar el botón como habilitado, el form queda sin
listener de submit — el formulario nunca más puede enviarse.

**Causa raíz**: `Reference` no tiene `SetAttr(key, value string)` ni `RemoveAttr(key string)`.

**Fix**: `ref.SetAttr("disabled", "") → element.setAttribute("disabled", "")` y
`ref.RemoveAttr("disabled") → element.removeAttribute("disabled")`. Un atributo, una llamada JS.

### Bug 3 — Sin `SetText`: actualizar error span anida elementos

**Síntoma**: al llamar `dom.Render("field.error", dom.Span(msg).Class("tw-field-error--visible"))`,
se escribe un `<span>` dentro del `<span id="field.error">` ya existente. Las queries
posteriores a `dom.Get("field.error")` devuelven el span exterior (correcto), pero el
contenido es un span anidado. Si la actualización se repite, se anidan indefinidamente.

**Causa raíz**: `Reference` no tiene `SetText(string)`.

**Fix**: `ref.SetText("campo requerido") → element.textContent = "campo requerido"`.
El span exterior conserva `id`, `class` y `aria-live`. Solo cambia el texto.

---

## Decisión de diseño

Agregar cuatro métodos a la interfaz `Reference` y su implementación en `elementWasm`:

```go
type Reference interface {
    // --- existentes ---
    GetAttr(key string) string
    Value() string
    Checked() bool
    On(eventType string, handler func(event Event))
    Focus()

    // --- nuevos ---

    // SetValue sets element.value (inputs, textarea, select).
    SetValue(value string)

    // SetAttr calls element.setAttribute(key, value).
    // Use empty string for boolean attributes (e.g., SetAttr("disabled", "")).
    SetAttr(key, value string)

    // RemoveAttr calls element.removeAttribute(key).
    RemoveAttr(key string)

    // SetText sets element.textContent.
    // Safe for plain text — does not parse HTML.
    SetText(text string)
}
```

### Por qué `SetText` y no `SetInnerHTML`

`SetText` → `element.textContent` es seguro para texto de usuario: no interpreta HTML,
por lo que es inmune a XSS. `SetInnerHTML` requeriría que el caller sanitice el contenido
— violando el principio de seguridad por defecto del framework.

Si en el futuro se necesita insertar HTML controlado, se agrega `SetInnerHTML` con
documentación explícita del riesgo.

### Backend stub

`dom_backend.go` tiene una implementación vacía (`domBackend`) para compilación SSR.
Los nuevos métodos necesitan stubs no-op en `element_backend.go` (o equivalente):

```go
func (e *elementBackend) SetValue(_ string)        {}
func (e *elementBackend) SetAttr(_, _ string)      {}
func (e *elementBackend) RemoveAttr(_ string)       {}
func (e *elementBackend) SetText(_ string)          {}
```

---

## Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `reference.go` | Agregar `SetValue`, `SetAttr`, `RemoveAttr`, `SetText` a la interfaz |
| `element_wasm.go` | Implementar los cuatro métodos via `syscall/js` |
| `dom_backend.go` o `element_backend.go` | Stubs no-op para build `!wasm` |

## Implementación en `element_wasm.go`

```go
func (e *elementWasm) SetValue(value string) {
    e.val.Set("value", value)
}

func (e *elementWasm) SetAttr(key, value string) {
    e.val.Call("setAttribute", key, value)
}

func (e *elementWasm) RemoveAttr(key string) {
    e.val.Call("removeAttribute", key)
}

func (e *elementWasm) SetText(text string) {
    e.val.Set("textContent", text)
}
```

---

## Impacto en tinywasm/form

Con estos métodos, `form` puede:

| Operación | Antes (destructivo) | Después (quirúrgico) |
|-----------|--------------------|--------------------|
| Reset input | `dom.Render(inputID, ...)` → destruye listeners | `ref.SetValue("")` |
| Disable button | `dom.Render(btnID, ...)` → destruye listeners | `ref.SetAttr("disabled", "")` |
| Enable button | `dom.Render(btnID, ...)` → destruye listeners | `ref.RemoveAttr("disabled")` |
| Update error span | `dom.Render(errID, dom.Span(...))` → anida spans | `ref.SetText(msg)` |
| Clear error span | `dom.Render(errID, dom.Span(""))` → anida span vacío | `ref.SetText("")` |

---

## Herramienta de testing — gotest

`gotest` ejecuta automáticamente vet, race detection, cobertura, **tests WASM en el
navegador** (via `wasmbrowsertest`) y badges. Es el único comando necesario para validar
esta librería completa, incluyendo los tests en `tests/uc_reference_mutation_test.go`.

```bash
# Instalar gotest (una sola vez)
go install github.com/tinywasm/devflow/cmd/gotest@latest

# gotest instala wasmbrowsertest automáticamente si no está disponible.
# También puede instalarse manualmente:
# go install github.com/tinywasm/wasmbrowsertest@latest
```

Uso:
```bash
gotest              # suite completa: vet + race + cover + wasm en navegador + badges
gotest -no-cache    # forzar re-ejecución aunque el código no haya cambiado
gotest -run TestBug # ejecutar solo los tests de reproducción de bugs
```

Los tests WASM se detectan automáticamente por el build tag `//go:build wasm` en los
archivos `tests/uc_*_test.go`. No se requiere configuración adicional.

### Verificación de bugs (estado actual)

Los tres tests de reproducción **ya fallan** con el código actual, confirmando los bugs:

```
FAIL: TestBug_RenderDestroysListeners_NoSetValue
FAIL: TestBug_RenderDestroysListeners_NoSetAttr
FAIL: TestBug_RenderNestsSpans_NoSetText
```

Tras el fix, los tres deben pasar. Ejecutar `gotest` para confirmar.

---

## Tests a agregar tras el fix

| Test | Archivo | Verifica |
|------|---------|----------|
| `TestReference_SetValue_PreservesListeners` | `tests/uc_reference_mutation_test.go` | Listener `"input"` sigue activo tras `SetValue` |
| `TestReference_SetAttr_PreservesListeners` | `tests/uc_reference_mutation_test.go` | Listener `"click"` sigue activo tras `SetAttr`/`RemoveAttr` |
| `TestReference_SetText_PreservesAttributes` | `tests/uc_reference_mutation_test.go` | `id` y `aria-live` intactos tras `SetText` |
| `TestReference_SetText_NoHTMLInjection` | `tests/uc_reference_mutation_test.go` | `SetText("<script>")` → texto literal, no HTML |

---

## Orden de ejecución

1. Agregar `SetValue`, `SetAttr`, `RemoveAttr`, `SetText` a `reference.go`
2. Implementar en `element_wasm.go` (4 llamadas JS directas)
3. Agregar stubs no-op en build `!wasm`
4. Ejecutar `gotest` — los tres tests de bug deben pasar; suite completa verde
5. Publicar via `gopush`
6. Actualizar `tinywasm/form` para usar los nuevos métodos en `Reset()` y loading state del botón
