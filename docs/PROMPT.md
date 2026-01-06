### 🤖 Prompt para Implementación de TinyDOM

**Rol:** Eres un experto en Go y WebAssembly, especializado en optimización para TinyGo.

**Objetivo:** Implementar la fase 1 y 2 de la librería `tinywasm/dom` siguiendo estrictamente la documentación en [docs/](README.md).

**Restricciones Técnicas (CRÍTICAS):**
1.  **Cero StdLib innecesaria:** NO importes `fmt`, `strings`, `strconv`, `errors` ni `net/http`. Usa exclusivamente `github.com/tinywasm/fmt` para manipulación de strings y conversiones.
2.  **Optimización TinyGo:**
    *   Evita el uso de `map` si es posible, o úsalos con extrema precaución sabiendo que son lentos y desordenados en TinyGo. Para el caché de IDs, considera si un slice struct simple o un array estático es viable, o usa un map solo si es estrictamente necesario para búsquedas O(1).
    *   Minimiza las alocaciones de memoria en el heap.
3.  **Build Tags:** El código debe compilar tanto en `GOOS=js GOARCH=wasm` como en backend estándar (Linux/Mac). Usa archivos `_wasm.go` y `_stub.go` (o `!wasm`).
4.  **Sin `syscall/js` en la API pública:** Los tipos `js.Value` NUNCA deben aparecer en `dom.go` o `element.go`. Solo en la implementación interna `_wasm.go`.

**Tareas a realizar:**

1.  **Interfaces Base (`dom.go`, `element.go`):**
    *   Define las interfaces `DOM`, `Element` y `Component` exactamente como están en [docs/API.md](API.md).
    *   Asegúrate de que `Element` incluya los métodos nuevos `AppendHTML` y `Remove`.

2.  **Implementación Stub (`dom_stub.go`, `element_stub.go`):**
    *   Crea implementaciones vacías (No-Op) para cuando se compila con `!wasm`.
    *   Esto es vital para que el servidor backend pueda importar componentes sin fallar al compilar.
    *   El constructor `New()` en tinywasm/dom.go debe retornar la implementación correcta según el build tag.

3.  **Implementación WASM (`dom_wasm.go`, `element_wasm.go`):**
    *   Implementa la lógica real usando `syscall/js`.
    *   **Caché:** Implementa el mecanismo de caché `ID -> js.Value`.
    *   **Mount:** Debe inyectar HTML (`innerHTML`) y llamar a `OnMount`.
    *   **Unmount:** Debe eliminar el nodo del DOM y limpiar listeners.
    *   **Eventos:** Implementa un sistema robusto para registrar callbacks (`js.FuncOf`) y guardarlos en un registro interno para poder hacerles `Release()` en el `Unmount`. **Esto es prioritario para evitar memory leaks.**

**Contexto:**
*   Usa `tinystring` para concatenaciones y conversiones.
*   La estructura de archivos esperada es:
    *   tinywasm/dom.go (Constructor público)
    *   `dom.go` (Interfaces)
    *   element.go (Interfaces)
    *   `dom_wasm.go` / `dom_stub.go`
    *   `element_wasm.go` / `element_stub.go`

**Ejecución:**
Por favor, genera primero los archivos de interfaces (`dom.go`, element.go) y el constructor (tinywasm/dom.go), y luego procede con las implementaciones stub y wasm paso a paso.

***
