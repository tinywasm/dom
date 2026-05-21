# PLAN — Migrar `JSProvider` a `[]*js.Script`

## Objetivo

Actualizar la interfaz `JSProvider` para que `RenderJS()` retorne
`[]*js.Script` en vez de `string`. Esto habilita que un módulo SSR emita
archivos JavaScript independientes (service workers, web workers, manifests
JS) además del fragmento que se acopla al bundle global.

## Justificación

Hoy `RenderJS() string` sólo permite contribuir contenido al bundle único.
Casos legítimos quedan fuera: un service worker debe estar en el root público
para definir su scope. La migración a `[]*Script` deja un único contrato que
cubre ambos casos: `Name=""` → bundle, `Name="sw.js"` → archivo aparte.

Es un **breaking change** consciente: simplifica el ecosistema y elimina la
necesidad de que los consumidores escriban archivos manualmente.

## Cambio de interfaz

**Archivo:** [interface.dom.go:79-83](../interface.dom.go#L79-L83)

```go
// Antes
type JSProvider interface {
    RenderJS() string
}

// Después
import "github.com/tinywasm/js"

type JSProvider interface {
    RenderJS() []*js.Script
}
```

## Dependencias

- `tinywasm/js` debe estar publicado con la estructura `Script` (campos
  `Name`, `Content` y método `String()`). Verificación previa:

  ```bash
  go list -m github.com/tinywasm/js@latest
  ```

## Tests

- `grep -rn "RenderJS() string" .` en `tinywasm/dom` debe quedar vacío.
- `grep -rn "RenderJS()" ./*_test.go` revisar y adaptar fixtures.
- `go test ./...` verde en `tinywasm/dom`.

## Documentación

- Si existe `docs/ARCHITECTURE.md` o equivalente que documente capacidades
  opcionales (`JSProvider`), reflejar el nuevo tipo de retorno.
- Añadir nota de migración en el `README.md`: ejemplo "antes/después" para
  consumidores.

## Stages

| # | Tarea | Done |
|---|---|---|
| 1 | Confirmar `tinywasm/js` publicado con `Script` | [ ] |
| 2 | `go get github.com/tinywasm/js` y añadir al `go.mod` | [ ] |
| 3 | Cambiar firma de `JSProvider.RenderJS()` | [ ] |
| 4 | Actualizar tests/fixtures internos que usen la interfaz | [ ] |
| 5 | `go test ./...` verde | [ ] |
| 6 | Nota de migración en `README.md` | [ ] |
