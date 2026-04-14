# Celador PRD Roadmap — Mejoras Progresivas de Seguridad

**Fecha:** 13 de abril de 2026  
**Objetivo:** Transformar Celador de "análisis superficial de package.json" a "auditoría real de cadena de suministro" manteniendo: offline-first, binario liviano, sin breaking changes, sin llamadas de red adicionales.

---

## Principios del Roadmap

| Principio | Descripción |
|-----------|-------------|
| **Offline-first** | Cada mejora funciona sin llamadas de red adicionales |
| **Sin breaking changes** | Comportamiento actual se mantiene, se amplía |
| **Peso controlado** | Máximo +500KB al binario por PRD |
| **Progresivo** | Cada PRD es independiente y entregable por separado |
| **Test-first** | Cada PRD incluye tests con cobertura ≥80% |
| **TDD** | Test → Red → Green → Refactor para cada cambio |

---

## Roadmap

| PRD | Feature | Impacto | Esfuerzo | Depende de |
|-----|---------|---------|----------|------------|
| **[PRD-001](PRDs/001-tarball-full-inspection.md)** | Inspección completa de archivos del tarball | 🟢🟢🟢 Alto | Medio | — |
| **[PRD-002](PRDs/002-lifecycle-scripts-audit.md)** | Auditoría completa de scripts de lifecycle | 🟢🟢 Medio | Bajo | — |
| **[PRD-003](PRDs/003-typosquatting-detection.md)** | Detección de typosquatting local | 🟢🟢 Medio | Bajo-Medio | — |
| **[PRD-004](PRDs/004-sarif-output.md)** | Salida SARIF para CI/CD | 🟢 Medio (CI) | Bajo | — |
| **[PRD-005](PRDs/005-sbom-generation.md)** | Generación de SBOM SPDX | 🟢 Medio (cumplimiento) | Medio | — |
| **[PRD-006](PRDs/006-parallel-hydration.md)** | Paralelizar hidratación de advisories OSV | ⚡ Rendimiento | Bajo | — |

**Orden recomendado:** 001 → 002 → 003 → 006 → 004 → 005

Cada PRD es independiente y puede ejecutarse en cualquier orden, pero el orden sugerido maximiza el impacto de seguridad primero.

---

## Estado de Implementación

| PRD | Estado | Fecha | Tag |
|-----|--------|-------|-----|
| 001 | ✅ Completado | 2026-04-13 | v0.4.0 |
| 002 | ✅ Completado | 2026-04-13 | v0.4.1 |
| 003 | ✅ Completado | 2026-04-13 | v0.4.2 |
| 004 | ✅ Completado | 2026-04-13 | v0.4.3 |
| 005 | ✅ Completado | 2026-04-13 | v0.4.4 |
| 006 | ✅ Completado | 2026-04-13 | v0.4.5 |

---

## Métricas de Éxito

| Métrica | Actual | Después de 6 PRDs |
|---------|--------|-------------------|
| Cobertura de detección de ataques | ~10% | ~70%+ |
| Llamadas de red por `install <pkg>` | 2 | 2 (sin cambio) |
| Tamaño del binario | ~15MB | ~15.5MB (+500KB máx) |
| Latencia `scan` (cache miss, vulns) | 5-30s | 1-6s (PRD-006) |
| Tests de seguridad | 5 (axios) | 25+ |
| Formatos de salida | Texto, JSON | + SARIF + SBOM |

---

## Notas de Ejecución

- Cada PRD incluye su propio plan de tests con TDD
- No se modifica comportamiento existente — solo se amplía
- Si un PRD revela que necesita cambios en otro, se documenta como dependencia
- Se puede saltar cualquier PRD si no es prioritario
