# PLAN: Migración de tokens CSS a convención Material/Bootstrap — `tinywasm/dom`

## Contexto

El sistema de tokens actual mezcla semánticas: `--color-primary` actúa como
"texto principal" (cambia entre oscuro y claro según modo), mientras que
`--color-secondary` actúa como "color de marca/brand" (cyan fijo). Esto viola
la convención de Material Design / Bootstrap donde `primary` es el color de
marca y los textos sobre superficies tienen tokens separados (`on-surface`,
`on-primary`, etc.).

**Consecuencia directa del bug:** texto con `color: var(--color-primary)` sobre
fondo `var(--color-secondary)` en dark mode producía contraste ~2.2:1 (ilegible,
WCAG AA requiere 4.5:1 mínimo).

**Este es un breaking change deliberado.** El marco está en desarrollo activo
y no tiene apps en producción.

---

## Decisiones confirmadas

| Decisión | Valor |
|----------|-------|
| Convención | Material Design 3 + Bootstrap 5 — tokens semánticos con prefijo `on-` |
| Scope | Solo `theme.css` — sin cambios en Go |
| Breaking change | Sí — todos los consumidores de tokens deben migrar |
| Todo en `dom` | Sí — `:root`, `@media` y `[data-theme]` viven en `theme.css` |
| Patrón `[data-theme]` | Two-layer variables — solo asignan variables a variables, cero hardcode |

---

## Tabla de migración de tokens

| Token actual | Token nuevo | Semántica | ¿Cambia con modo? |
|---|---|---|---|
| `--color-primary` (como texto) | `--color-on-surface` | texto principal sobre superficie | Sí |
| `--color-secondary` | `--color-primary` | color de marca — Go cyan | No (fijo) |
| *(nuevo)* | `--color-on-primary` | texto SOBRE el color primario/cyan | No (fijo) |
| `--color-selection` | `--color-secondary` | acento interactivo — WASM purple | No (fijo) |
| *(nuevo)* | `--color-on-secondary` | texto SOBRE el color secundario/purple | No (fijo) |
| `--color-tertiary` | `--color-muted` | texto apagado / bordes sutiles | Sí |
| `--color-quaternary` | `--color-surface` | fondo de paneles y cards | Sí |
| `--color-gray` | `--color-background` | fondo de página | Sí |
| `--color-hover` | `--color-hover` | sin cambio de nombre | Sí |
| `--color-success` | `--color-success` | sin cambio | No (fijo) |
| `--color-error` | `--color-error` | sin cambio | No (fijo) |

---

## Patrón two-layer variables

Los tokens que cambian entre modos tienen **dos capas**:

- **Fuente** (`--color-X-light` / `--color-X-dark`): definen los valores de cada
  modo. Son las únicas variables que una app necesita sobreescribir para
  personalizar el tema.
- **Activo** (`--color-X`): el token que usan los componentes. Lo asignan
  `:root`, `@media` y `[data-theme]` — siempre como `var()`, nunca hardcode.

```
:root         → --color-background: var(--color-background-light)  [default = light]
@media dark   → --color-background: var(--color-background-dark)   [auto por OS]
[data-theme]  → --color-background: var(--color-background-light/dark)  [override manual]
```

**Los bloques `[data-theme]` contienen solo asignaciones variable→variable.**
No hay un solo valor hardcodeado en `@media` ni en `[data-theme]`.

Los tokens **fijos** (brand group) no necesitan variantes `-light`/`-dark`
porque su valor es idéntico en todos los modos.

---

## Nuevo `theme.css`

```css
/*
 * tinywasm/dom — canonical design tokens v2
 *
 * Convención: Material Design 3 / Bootstrap 5
 *   - primary/secondary = colores de marca (fijos, no flip)
 *   - on-X             = texto/icono SOBRE el color X
 *   - surface/background/muted/hover = cambian con el modo
 *
 * Patrón two-layer:
 *   --color-X-light / --color-X-dark  → fuentes por modo (apps overriden estas)
 *   --color-X                         → token activo (componentes usan este)
 *   @media y [data-theme] solo asignan var() → var(), nunca hardcode.
 */

/* ══ Brand group — fijos, nunca flip ════════════════════════════ */
:root {
  --color-primary:      #00ADD8; /* Go cyan — color de marca principal        */
  --color-on-primary:   #1C1C1E; /* texto/icono SOBRE cyan                    */
  --color-secondary:    #654FF0; /* WASM purple — acento interactivo           */
  --color-on-secondary: #FFFFFF; /* texto/icono SOBRE purple                  */
  --color-success:      #3FB950; /* Go gopher green                            */
  --color-error:        #E34F26; /* HTML5 orange-red                           */
}

/* ══ Theme group — fuentes por modo (apps overriden estas) ══════ */
:root {
  --color-background-light: #FFFFFF;
  --color-background-dark:  #0D1117;

  --color-surface-light:    #F2F2F7;
  --color-surface-dark:     #161B22;

  --color-on-surface-light: #1C1C1E;
  --color-on-surface-dark:  #E6EDF3;

  --color-muted-light:      #6E6E73;
  --color-muted-dark:       #8B949E;

  --color-hover-light:      #B8860B;
  --color-hover-dark:       #F7DF1E;
}

/* ══ Theme group — tokens activos (default = light) ════════════ */
:root {
  --color-background: var(--color-background-light);
  --color-surface:    var(--color-surface-light);
  --color-on-surface: var(--color-on-surface-light);
  --color-muted:      var(--color-muted-light);
  --color-hover:      var(--color-hover-light);
}

/* ══ Spacing tokens — sin cambio ════════════════════════════════ */
:root {
  --menu-width-collapsed: 64px;
  --menu-width-expanded:  250px;
  --title-height:    8vh;
  --content-height:  89vh;
  --controls-height: 3vh;
  --mag-pri: 0.5rem;
  --mag-sec: 0.2rem;
  --mag-cua: 0.2rem;
}

/* ══ Dark mode automático — OS preference, sin JS ═══════════════ */
@media (prefers-color-scheme: dark) {
  :root {
    --color-background: var(--color-background-dark);
    --color-surface:    var(--color-surface-dark);
    --color-on-surface: var(--color-on-surface-dark);
    --color-muted:      var(--color-muted-dark);
    --color-hover:      var(--color-hover-dark);
  }
}

/* ══ Override manual light — [data-theme="light"] en <html> ═════ */
[data-theme="light"] {
  --color-background: var(--color-background-light);
  --color-surface:    var(--color-surface-light);
  --color-on-surface: var(--color-on-surface-light);
  --color-muted:      var(--color-muted-light);
  --color-hover:      var(--color-hover-light);
}

/* ══ Override manual dark — [data-theme="dark"] en <html> ══════ */
[data-theme="dark"] {
  --color-background: var(--color-background-dark);
  --color-surface:    var(--color-surface-dark);
  --color-on-surface: var(--color-on-surface-dark);
  --color-muted:      var(--color-muted-dark);
  --color-hover:      var(--color-hover-dark);
}
```

---

## Cómo una app personaliza el tema

Solo necesita sobreescribir las fuentes en su `RootCSS()` — los bloques
`[data-theme]` y `@media` se adaptan automáticamente:

```css
/* App custom RootCSS() — sobreescribe solo lo necesario */
:root {
  --color-background-light: #FAFAFA;  /* light bg personalizado */
  --color-background-dark:  #121212;  /* dark bg personalizado  */
  --color-surface-light:    #F0F0F0;
  --color-surface-dark:     #1E1E1E;
  /* brand group puede conservarse o cambiarse */
  --color-primary:    #FF6B35;  /* marca propia en lugar de cyan */
  --color-on-primary: #FFFFFF;
}
```

---

## Consecuencia en `themeswitch/themeswitch.css`

Los bloques `[data-theme]` se **eliminan** de `themeswitch/themeswitch.css`.
Ese archivo queda solo con los estilos del botón `.ts-btn`.
Ver `components/docs/PLAN_TOKENS_V2.md`.

---

## Dependencias bloqueantes

Ninguna. `theme.css` es la fuente de tokens — no tiene dependencias upstream.

Los consumidores que deben migrar **después** de esta tarea:
- `tinywasm/components` → `components/docs/PLAN_TOKENS_V2.md`
- `tinywasm/layout` → `layout/rightpanel/PLAN.md`

---

## Checklist de implementación

- [ ] Reemplazar `theme.css` con el nuevo contenido especificado arriba
- [ ] Verificar que `dom` compila sin errores (`go build ./...`)
- [ ] Verificar que los tests de `dom` pasan (`go test ./...`)
- [ ] `gopush 'feat(dom)!: migrate CSS tokens to Material/Bootstrap convention (two-layer variables)'`
