---
PLAN: "fix(security): escape text nodes by default; raw markup requires the typed dom.TrustedHTML"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 8470234292955560261
PR: https://github.com/tinywasm/dom/pull/22
---

> Este plan se despacha con el flujo CodeJob. Ver skill: `agents-workflow`.
> No ejecutes `gopush` ni `codejob` — son herramientas del desarrollador local.

# PLAN — `tinywasm/dom`: cerrar el XSS de `Element.Text()`

## Contexto

Auditoría de seguridad de `veltylabs/iam` (2026-09-02). El panel de
administración de `iam` renderiza emails, nombres, códigos de rol y detalles
de auditoría que **vienen de proyectos consumidores y de perfiles de Google**
— datos que `iam` no controla. Todos pasan por `dom.Element.Text()`.

Doctrina obligatoria: [CONSTRUCTION_HARNESS.md](https://github.com/tinywasm/app-releases/blob/main/docs/CONSTRUCTION_HARNESS.md).
Los principios que gobiernan este plan, y que dictan la forma exacta de la
solución:

- **8 · Cerrado por defecto.** *"The safe default must be the one you get by
  writing nothing; opening is what costs an explicit, greppable line."*
- **3 · Estados ilegales no representables.**
- **1 · Tipado sobre `any`.** Un hueco genérico pierde la intención.
- **4 · Una sola forma de hacer cada cosa.**
- **6 · Nunca fallo silencioso.**

## Hallazgo D-1 (Crítico) · XSS almacenado

`Element.Text()` guarda el string crudo como hijo:

```go
func (b *Element) Text(text string) *Element {
	b.children = append(b.children, text)   // <- string crudo
	return b
}
```

y el serializador lo concatena sin escapar:

```go
case string:
	s += v          // element.go:elementToHTML y dom_frontend.go:renderToHTML
```

y el resultado entra al documento por `innerHTML`
(`dom_frontend.go`: `parent.Set("innerHTML", html)`).

**Reproducido:**

```go
el := NewElement("div")
el.Text(`<img src=x onerror="alert(1)">`)
elementToHTML(el)
// => <div><img src=x onerror="alert(1)"></div>
```

Vector real en `veltylabs/iam`: el endpoint `POST /api/users/resolve` deja que
cualquier proyecto consumidor registre un usuario con el `name` que quiera, y
ese `name` se pinta en el panel de admin con `html.Td().Text(u.Name)`. El
script corre en `iam.velty.cl` con la cookie SSO del administrador.

### D-2 (Alto) · Mismo hueco en los bindings

`BindText` / `BindTextFunc` terminan en `s += textContent`, también sin
escapar, en los dos serializadores.

### D-3 (Medio) · `id` sin escapar

```go
s += " id='" + el.id + "'"
```

Los **valores** de atributo sí pasan por `fmt.Convert(...).EscapeAttr()`; el
`id` no. Un id dinámico permite cerrar la comilla e inyectar atributos.

### D-4 (DRY) · Dos serializadores con el mismo bug

`elementToHTML` en `element.go` y `renderToHTML` en `dom_frontend.go`
(`//go:build wasm`) son copias divergentes de la misma lógica. Arreglar una y
olvidar la otra deja el agujero abierto en la mitad de los targets.

## Decisión de diseño (ya tomada, no la re-discutas)

`Text()` **escapa siempre**. Inyectar marcado pasa a ser un acto explícito,
tipado y greppable, mediante un tipo nombrado que sólo se puede construir
llamando a una función cuyo nombre dice lo que hace.

Un `Raw(html string)` **no** alcanza: un `string` es un hueco genérico
(principio 1) y cualquier dato no confiable entra por ahí sin que nada lo
note. Con un tipo nombrado, pasar datos del usuario **no compila**
(principio 3).

Alcance real de la ruptura, ya medido sobre todo el ecosistema
(`tinywasm/*` + `veltylabs/*`, 853 llamadas a `.Text(`): **un solo call site**
pasa algo que parece marcado, y es
`tinywasm/html/web/client.go:100` — `P().Text("The theme is persisted in
localStorage and applied to <html>.")`, que **quiere** ver `<html>` escapado.
Es decir: el cambio arregla ese sitio en vez de romperlo. No hay migración
pendiente en ningún consumidor.

## Etapa 1 · Un solo serializador

Antes de tocar el escapado, eliminar la duplicación (D-4) — si no, el fix hay
que escribirlo dos veces y una de las dos se va a olvidar.

1. Extraer la lógica común de `elementToHTML` (`element.go`) y
   `renderToHTML` (`dom_frontend.go`) a **una sola función** en un archivo
   nuevo `serialize.go`, **sin build tag**.
2. Lo único que difiere entre ambas es cómo se resuelve un hijo `Component`
   (la versión wasm registra componentes y dueños). Parametrizá esa
   diferencia con un parámetro de función:

```go
// renderChild resuelve un hijo Component a HTML. La versión SSR llama a
// Component.String(); la versión wasm además registra el componente y su
// dueño para el ciclo de vida. Es la ÚNICA diferencia entre los dos caminos
// de serialización — todo lo demás, incluido el escapado, vive en un solo
// lugar y no puede divergir.
type childRenderer func(Component) string

func serializeElement(el *Element, renderChild childRenderer) string
```

3. `elementToHTML(el)` pasa a ser
   `serializeElement(el, func(c Component) string { return c.String() })`.
4. `(d *dom) renderToHTML(...)` pasa a llamar `serializeElement` con su
   closure de registro.
5. **Borrá** los cuerpos duplicados. Criterio verificable:
   `grep -c "case string:" element.go dom_frontend.go serialize.go` → sólo
   `serialize.go` tiene una ocurrencia.

## Etapa 2 · `TrustedHTML` y `Element.Raw`

Archivo nuevo: `trusted.go`, sin build tag.

```go
package dom

// TrustedHTML es marcado que el AUTOR del programa garantiza seguro. El tipo
// existe para que meter datos no confiables en el documento no compile: no
// hay conversión implícita desde string, y el único constructor obliga a
// escribir una línea que un grep encuentra.
//
// Regla: sólo literales del propio código, o el resultado de un builder de
// este ecosistema. NUNCA una cadena que venga de una petición, de una base de
// datos, de un perfil de OAuth o de otro servicio.
type TrustedHTML string

// Trust marca html como confiable. Es la ÚNICA forma de producir un
// TrustedHTML, y su nombre es lo que hace auditable el programa: buscar
// "dom.Trust(" enumera todos los puntos donde el escapado se saltea a
// propósito.
//
// Si estás por escribir dom.Trust(algoQueVinoDeAfuera), el defecto está en el
// diseño del llamador, no acá.
func Trust(html string) TrustedHTML { return TrustedHTML(html) }
```

En `element.go`:

```go
// Raw agrega marcado sin escapar. Exige un TrustedHTML, así que pasar datos
// de una petición no compila — ver Trust.
func (b *Element) Raw(h TrustedHTML) *Element {
	b.children = append(b.children, h)   // hijo de tipo TrustedHTML, no string
	return b
}
```

`Text` no cambia de firma. Lo que cambia es el serializador:

```go
switch v := child.(type) {
case *Element:
	s += serializeElement(v, renderChild)
case TrustedHTML:              // caso NUEVO, va ANTES del caso string
	s += string(v)             // confiado a propósito
case string:
	s += fmt.Convert(v).EscapeHTML()   // <- el fix
case Component:
	s += renderChild(v)
default:
	s += fmt.Convert(fmt.Sprint(v)).EscapeHTML()
}
```

**Orden obligatorio:** `case TrustedHTML` va antes que `case string`. Si
`TrustedHTML` quedara después, un type switch de Go igual lo distingue (es un
tipo nombrado distinto), pero el orden explícito documenta la intención y
evita que alguien colapse los dos casos.

## Etapa 3 · Bindings de texto e `id` (D-2, D-3)

En `serialize.go`:

1. El bloque `if hasTextContent { s += textContent }` pasa a
   `s += fmt.Convert(textContent).EscapeHTML()`. Un `BindText` está ligado a
   un `SignalString`, y una señal se llena en runtime con lo que sea.
2. `s += " id='" + el.id + "'"` pasa a
   `s += " id='" + fmt.Convert(el.id).EscapeAttr() + "'"`.
3. Revisar que **las claves** de atributo también salgan escapadas:
   hoy `s += " " + attr.Key + "='" + …EscapeAttr() + "'"` escapa el valor
   pero no la clave. Escapá la clave con `EscapeAttr` también.

No agregues un `BindRaw`: no hay caso de uso y sería una segunda forma de
hacer lo mismo (principio 4).

## Etapa 4 · Tests

Archivo nuevo: `tests/escaping_test.go` (o `escaping_test.go` en la raíz, la
convención que ya use el repo).

| Test | Fija |
|---|---|
| `TestTextEscapesMarkup` | `Text("<img src=x onerror=alert(1)>")` → salida contiene `&lt;img` y **no** contiene `<img`. |
| `TestTextEscapesAllFive` | `&`, `<`, `>`, `"`, `'` → `&amp;`, `&lt;`, `&gt;`, `&quot;`, `&#39;`. |
| `TestTextEscapesAmpersandFirst` | `Text("&lt;")` → `&amp;lt;`, no `&lt;` (doble escapado correcto: el `&` se procesa primero). |
| `TestRawPassesThrough` | `Raw(Trust("<b>x</b>"))` → `<b>x</b>` intacto. |
| `TestBindTextEscapes` | Un `SignalString` con `<script>` → `&lt;script&gt;`. **Regresión D-2.** |
| `TestBindTextFuncEscapes` | Ídem con `BindTextFunc`. |
| `TestIDIsEscaped` | `ID("a' onload='alert(1)")` → la comilla sale como `&#39;`. **Regresión D-3.** |
| `TestAttrKeyIsEscaped` | `Attr("a' onload='x", "v")` → la comilla de la clave sale escapada. |
| `TestSingleSerializer` | Construye un árbol con `Element`, `Text`, `Raw` y un `Component` hijo, y compara `elementToHTML` con la salida esperada literal. **Regresión D-4.** |

**Test consumer-shaped obligatorio** (regla de oro del harness: *an API is not
published until a consumer-shaped test, inside the library itself, proves it*):

```
TestAdminTable_RendersHostileUserDataInert
```

Debe reproducir exactamente la forma del panel de `iam`: construir una
`<table>` con `Thead`/`Tbody` y tres filas cuyos valores sean strings
hostiles (`<img src=x onerror=alert(1)>`, `"><script>alert(1)</script>`,
`javascript:alert(1)`), serializar el árbol completo y afirmar que la salida
**no contiene** ninguna de las subcadenas `<img`, `<script`, `onerror=`.
Ese test es la prueba de que la API es un harness: el consumidor escribió el
código obvio y salió seguro sin saber nada de escapado.

## Restricciones de código (leer antes de escribir)

| Regla | Detalle |
|---|---|
| **Sin mapas** | Prohibido `map[K]V` en librería y en tests. Slices + búsqueda lineal. TinyGo compila mapas mal. |
| **Sin stdlib** | Nada de `fmt`, `errors`, `strconv`, `strings`, `html`, `html/template`, `log`, `os`. Usa `github.com/tinywasm/fmt` — ya tiene `EscapeHTML()` y `EscapeAttr()` en `fmt/html.go`. **No escribas tu propio escapador.** |
| **`error` sí, `errors` no** | `fmt.Err(...)`, nunca `errors.New`. |
| **Sin `reflect`** | Ni transitivo. |
| **Embebido por valor** | `dom.Element` se embebe por valor, nunca `*dom.Element`. |
| **Sin literales repetidos** | Todo string repetido es una constante nombrada. |
| **Sin `internal/`** | No crees carpetas `internal/`. |
| **No versiones documentos** | Nada de `v1`/`v2` dentro de los archivos. |

Idioma: **código e identificadores en inglés**; **comentarios de prosa y
documentación en español**.

## Etapa 5 · Documentación

- `docs/ARCHITECTURE.md`: sección nueva **"Escapado y confianza"**. Debe
  decir, con estas palabras: el default es escapar, `Trust` es la única
  puerta, y `grep -rn "dom.Trust(" .` es la auditoría completa de esa puerta.
- `README.md`: en el ejemplo de uso, mostrar `Text` con datos y `Raw(Trust(...))`
  con un literal, uno al lado del otro.
- `docs/DESIGN.md`: registrar la alternativa descartada — `SafeText()` como
  método aparte dejando `Text()` crudo — y por qué se descartó: viola el
  principio 8 (el default seguro debe ser el que se obtiene sin escribir
  nada) y convierte cada olvido en un XSS.

## Criterios de aceptación

1. `go vet ./...` y `go test ./...` verdes.
2. `GOOS=js GOARCH=wasm go build ./...` compila.
3. `grep -c "case string:" element.go dom_frontend.go` → `0` en ambos
   (todo el switch vive en `serialize.go`).
4. `grep -rn "s += v$" .` → vacío.
5. `grep -rn "id='\" + el.id" .` → vacío.
6. `dom.TrustedHTML`, `dom.Trust`, `(*Element).Raw` exportados y documentados
   en español.
7. `TestAdminTable_RendersHostileUserDataInert` existe y pasa.
8. Ningún consumidor del ecosistema queda roto: el único call site que pasaba
   marcado literal (`tinywasm/html/web/client.go:100`) queda **correcto** con
   el nuevo comportamiento — no lo migres a `Raw`.

## Etapas

| # | Archivo | Entrega |
|---|---|---|
| 1 | `serialize.go`, `element.go`, `dom_frontend.go` | Un solo serializador (D-4) |
| 2 | `trusted.go`, `element.go`, `serialize.go` | `TrustedHTML`, `Trust`, `Raw`, escapado de `Text` (D-1) |
| 3 | `serialize.go` | Escapado de bindings, `id` y claves de atributo (D-2, D-3) |
| 4 | `tests/escaping_test.go` | Tests + consumer-shaped |
| 5 | `docs/ARCHITECTURE.md`, `docs/DESIGN.md`, `README.md` | Contrato de confianza documentado |
