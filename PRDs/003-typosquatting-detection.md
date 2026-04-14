# PRD-003: Detección de Typosquatting Local

**Prioridad:** 🟡 Media  
**Impacto:** Medio — cubre categoría completa de ataques  
**Esfuerzo:** Bajo-Medio  
**Llamadas de red adicionales:** Cero  
**Peso del binario:** +200-300KB (lista embebida de top paquetes npm)  
**Breaking changes:** Ninguno

---

## Problema Actual

Celador no detecta paquetes que imitan nombres de paquetes populares mediante errores tipográficos intencionales.

**Ejemplos reales:**

| Paquete malicioso | Paquete legítimo | Descargas antes de ser removido |
|-------------------|-----------------|-------------------------------|
| `crossenv` | `cross-env` | 37,000+ |
| `reacts` | `react` | Miles |
| `lodahs` | `lodash` | Miles |
| `colour-name` | `color-name` | Miles |

---

## Objetivo

Detectar cuando un proyecto depende de un paquete cuyo nombre es sospechosamente similar a un paquete popular, sugiriendo un intento de typosquatting.

---

## Diseño Técnico

### Componente: `internal/core/audit/typosquat.go`

**Nueva función:**
```go
func DetectTyposquat(deps []shared.Dependency, knownPackages []string) []TyposquatFinding
```

**Algoritmo:**
1. Para cada dependencia del lockfile
2. Calcular distancia de Levenshtein contra cada paquete en la lista de top conocidos
3. Si distancia ≤ 2 Y el paquete no está en la lista de conocidos → alerta
4. Reportar: nombre sospechoso, paquete legítimo similar, distancia

### Lista embebida: `configs/typosquat/top-npm-packages.json`

Los ~5,000 paquetes más descargados de npm (lista estática que se actualiza periódicamente):
```json
["react", "lodash", "express", "axios", "typescript", "chalk", "debug", ...]
```

**Tamaño estimado:** ~200-300KB comprimido en el binario.

### Distancia de Levenshtein

Implementación simple en Go puro (~30 líneas). Sin dependencias externas.

### Umbrales

| Distancia | Acción |
|-----------|--------|
| 0 | Es el mismo paquete — no alerta |
| 1 | Alta probabilidad de typosquatting → alerta High |
| 2 | Posible typosquatting → alerta Medium |
| 3+ | Probablemente legítimo — no alerta |

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestDetectTyposquat_ExactMatch_NoAlert` | `lodash` vs `lodash` = sin alerta |
| `TestDetectTyposquat_SingleCharSwap` | `lodahs` vs `lodash` = alerta High |
| `TestDetectTyposquat_MissingChar` | `reacts` vs `react` = alerta High |
| `TestDetectTyposquat_ExtraChar` | `reacct` vs `react` = alerta Medium |
| `TestDetectTyposquat_DifferentPackage` | `express` vs `react` = sin alerta |
| `TestDetectTyposquat_EmptyDeps` | Sin dependencias = sin error |
| `TestDetectTyposquat_EmptyKnownList` | Sin lista conocida = sin error |
| `TestLevenshteinDistance` | Tests unitarios del algoritmo |

---

## Criterios de Aceptación

- [ ] Detecta al menos 10 variantes de typosquatting conocidas
- [ ] Cero falsos positivos en proyectos reales (express, react, lodash, next.js)
- [ ] 8+ nuevos tests con cobertura ≥80%
- [ ] Lista de top paquetes embebida en el binario
- [ ] Sin llamadas de red
- [ ] `go test ./...` pasa sin fallos

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/core/audit/typosquat.go` | **Crear** — detección de typosquatting |
| `internal/core/audit/typosquat_test.go` | **Crear** — tests |
| `configs/typosquat/top-npm-packages.json` | **Crear** — lista de paquetes conocidos |
| `internal/core/audit/service.go` | Posible modificación — integrar check en Run() |

---

## Riesgos y Mitigación

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| Falsos positivos en paquetes legítimos | Media | Umbral de distancia conservador. Lista curada manualmente |
| Lista de top paquetes desactualizada | Media | Proceso de actualización periódica (manual por ahora) |
| Binario crece demasiado | Baja | 5,000 nombres = ~50KB texto plano |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 1 hora |
| Implementación Levenshtein | 1 hora |
| Generar lista de top paquetes | 1 hora |
| Integración y verificación | 1 hora |
| **Total** | **~4 horas** |
