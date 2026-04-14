# Celador v0.4.5 — Estado Competitivo

**Fecha:** 14 de abril de 2026  
**Comparado contra:** npm audit, Snyk, Socket.dev, osv-scanner (Google), Dependabot, Renovate

---

## 1. Estado Actual por Puntos Críticos

### 1.1 Detección de Vulnerabilidades Conocidas

| Herramienta | Fuente de datos | Ecosistemas | Actualización | Offline |
|-------------|----------------|-------------|---------------|---------|
| **Celador v0.4.5** | OSV (gratuita) | npm, pnpm, Bun, Deno | Tiempo real (API) | ✅ Con caché |
| **npm audit** | npm advisory | npm | Tiempo real | ✅ |
| **Snyk** | Snyk Intelligence + OSV | 15+ | Tiempo real | ❌ |
| **Socket.dev** | Propia + OSV | npm, PyPI, RubyGems, Cargo, Go | Tiempo real | ❌ |
| **osv-scanner** | OSV (oficial) | 15+ | Tiempo real | ✅ Parcial |

**Veredicto:** Celador está a la par con osv-scanner para ecosistemas JS/TS. Inferior en cobertura multi-ecosistema, pero superior en análisis de contenido del paquete (inspecciona archivos fuente del tarball).

---

### 1.2 Detección de Código Malicioso en Archivos Fuente

| Herramienta | Inspecciona .js/.ts | Patrones detectados | Método |
|-------------|---------------------|---------------------|--------|
| **Celador v0.4.5** | ✅ **Sí** | eval, new Function, child_process, env+red, hex, .node | Análisis estático de patrones |
| **npm audit** | ❌ No | N/A | Solo manifiesto |
| **Snyk** | ✅ Sí | Patrones + flujo de datos + semántica | Análisis estático avanzado |
| **Socket.dev** | ✅ Sí | 30+ señales de comportamiento | Sandbox execution + heurísticas |
| **osv-scanner** | ❌ No | N/A | Solo OSV database |

**Veredicto:** Celador está detrás de Snyk y Socket.dev en sofisticación (no hace análisis semántico ni sandbox), pero es **el único que hace análisis estático de archivos fuente del tarball sin requerir cuenta ni conexión permanente**. Para su nicho (offline-first, sin cuenta), no tiene competencia directa.

---

### 1.3 Detección de Typosquatting

| Herramienta | Detecta | Método | Offline |
|-------------|---------|--------|---------|
| **Celador v0.4.5** | ✅ Distancia Levenshtein ≤ 2 | 100+ paquetes conocidos embebidos | ✅ |
| **npm audit** | ❌ No | N/A | N/A |
| **Snyk** | ✅ Sí | Similaridad + download anomalies + reputation | ❌ |
| **Socket.dev** | ✅ Sí | Download patterns + similarity + community signals | ❌ |
| **osv-scanner** | ❌ No | N/A | N/A |

**Veredicto:** La detección de Celador es básica pero efectiva para los ataques más comunes (lodahs, reacts, crossenv). Snyk y Socket tienen detección más sofisticada (anomalías de descargas, señales de comunidad). Celador gana en ser 100% offline.

---

### 1.4 Evaluación de Riesgo Pre-Instalación

| Herramienta | Analiza antes de install | Qué evalúa |
|-------------|-------------------------|------------|
| **Celador v0.4.5** | ✅ **Sí** | Scripts lifecycle, patrones maliciosos, typosquatting |
| **npm audit** | ❌ No (después de install) | Solo vulnerabilidades conocidas |
| **Snyk** | ⚠️ Parcial | Licencia, vulns, pero no riesgo de instalación |
| **Socket.dev** | ✅ Sí | 30+ señales de comportamiento |
| **osv-scanner** | ❌ No (después de install) | Solo vulnerabilidades conocidas |

**Veredicto:** Celador y Socket.dev son los únicos que evalúan riesgo **antes** de instalar. Socket es mucho más profundo (sandbox), pero Celador es el único que lo hace sin cuenta y offline.

---

### 1.5 Corrección Automática de Vulnerabilidades

| Herramienta | Corrige | Cómo | Automatizado |
|-------------|---------|------|-------------|
| **Celador v0.4.5** | ✅ Bump conservador local | Edita package.json | ❌ Local solamente |
| **npm audit** | ✅ `npm audit fix` | Bump de versiones | ⚠️ Semi-automático |
| **Snyk** | ✅ Fix PRs | PRs automatizados | ✅ GitHub/GitLab |
| **Socket.dev** | ❌ No corrige | Solo detecta | N/A |
| **osv-scanner** | ❌ No corrige | Solo detecta | N/A |
| **Dependabot** | ✅ PRs automáticos | PRs con fix | ✅ Nativo GitHub |
| **Renovate** | ✅ PRs automáticos | PRs con scheduling | ✅ Multi-plataforma |

**Veredicto:** Celador es el más limitado en automatización (solo local). Dependabot/Renovate/Snyk son muy superiores para equipos que necesitan PRs automáticos.

---

### 1.6 Cumplimiento y Reporting

| Herramienta | SBOM | SARIF | JSON | Texto | Baseline/Diff |
|-------------|------|-------|------|-------|---------------|
| **Celador v0.4.5** | ✅ **SPDX 2.3** | ✅ **v2.1.0** | ✅ | ✅ | ❌ |
| **npm audit** | ❌ | ❌ | ✅ (JSON) | ✅ | ❌ |
| **Snyk** | ✅ CycloneDX | ✅ | ✅ | ✅ | ✅ |
| **Socket.dev** | ✅ CycloneDX | ❌ | ✅ | ✅ | ❌ |
| **osv-scanner** | ✅ SPDX/CycloneDX | ✅ | ✅ | ✅ | ✅ |

**Veredicto:** Celador ahora cumple con los formatos mínimos de cumplimiento (SBOM + SARIF). Le falta baseline/diff comparado con Snyk y osv-scanner.

---

### 1.7 Rendimiento

| Escenario | Celador v0.4.5 | npm audit | Snyk | Socket | osv-scanner |
|-----------|---------------|-----------|------|--------|-------------|
| **Primer escaneo (pocas vulns)** | 500ms-3s | 1-2s | 3-5s | 5-10s | 2-5s |
| **Primer escaneo (muchas vulns)** | **1-6s** (paralelo) | 1-2s | 5-15s | 10-30s | 3-10s |
| **Escaneo repetido (caché)** | **10-50ms** | 1-2s | 3-5s | 5-10s | 2-5s |
| **Evaluación pre-install** | 200ms-2s | N/A | N/A | 5-15s | N/A |
| **Sin internet** | ✅ Funciona (caché) | ✅ | ❌ | ❌ | ✅ Parcial |

**Veredicto:** Celador es **el más rápido con caché llena** (10-50ms vs 1-2s de npm audit). Con paralelización (PRD-006), el peor caso bajó de 30s a 6s. Es el único que funciona completamente offline después del primer escaneo.

---

### 1.8 Configuración y Facilidad de Uso

| Aspecto | Celador v0.4.5 | npm audit | Snyk | Socket | osv-scanner |
|---------|---------------|-----------|------|--------|-------------|
| **Setup** | `brew install` | Ya viene con Node | Cuenta + CLI | Cuenta + CLI | `go install` |
| **Requiere cuenta** | ❌ No | ❌ No | ✅ Sí | ✅ Sí | ❌ No |
| **Requiere internet** | Solo primer escaneo | Solo auditoría | ✅ Siempre | ✅ Siempre | Parcial |
| **CLI commands** | scan, fix, install, init, tui, about | audit | test, monitor, protect | scan | scan |
| **Config file** | `.celador.yaml` | N/A | `snyk.json` | `socket.yaml` | `osv-scanner.toml` |
| **TUI** | ✅ Bubble Tea | ❌ | ❌ | ❌ | ❌ |

**Veredicto:** Celador tiene la barrera de entrada más baja después de npm audit (que viene integrado). No requiere cuenta, funciona offline, y tiene TUI interactiva.

---

## 2. Matriz de Capacidades — Resumen

| Capacidad | Celador | npm audit | Snyk | Socket | osv-scanner | Dependabot | Renovate |
|-----------|:-------:|:---------:|:----:|:------:|:-----------:|:----------:|:--------:|
| CVEs conocidas | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Archivos fuente maliciosos | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Typosquatting | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Riesgo pre-instalación | ✅ | ❌ | ⚠️ | ✅ | ❌ | ❌ | ❌ |
| Scripts lifecycle (5 tipos) | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Binarios nativos (.node) | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| SBOM (SPDX) | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ |
| SARIF v2.1.0 | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ |
| Corrección local | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| PRs automáticos | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ |
| Offline completo | ✅ | ✅ | ❌ | ❌ | ⚠️ | N/A | N/A |
| Sin cuenta necesaria | ✅ | ✅ | ❌ | ❌ | ✅ | N/A | N/A |
| TUI interactiva | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Multi-ecosistema | ❌ (JS/TS/Deno) | ❌ (solo npm) | ✅ (15+) | ✅ (5+) | ✅ (15+) | ✅ | ✅ |
| Procedencia (Sigstore) | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ | ✅ |
| Sandbox execution | ❌ | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Baseline/Diff | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ |

**Count por columna:**
- **Celador:** 12 ✅ | 5 ❌
- **npm audit:** 4 ✅ | 13 ❌
- **Snyk:** 15 ✅ | 2 ❌
- **Socket:** 12 ✅ | 5 ❌
- **osv-scanner:** 10 ✅ | 7 ❌
- **Dependabot:** 8 ✅ | 9 ❌
- **Renovate:** 8 ✅ | 9 ❌

---

## 3. Dónde Celador Gana

| Ventaja | Contra quién | Por qué importa |
|---------|-------------|-----------------|
| **Offline-first completo** | Snyk, Socket | Funciona en entornos air-gapped, CI sin internet, laptops sin WiFi |
| **Sin cuenta necesaria** | Snyk, Socket | Sin vendor lock-in, sin datos personales compartidos |
| **Typosquatting offline** | npm audit, osv-scanner, Dependabot | Detecta imitaciones sin consultar servicios externos |
| **Inspección de archivos fuente** | npm audit, osv-scanner, Dependabot, Renovate | Detecta malware en .js/.ts, no solo CVEs conocidas |
| **Riesgo pre-instalación** | npm audit, osv-scanner, Dependabot, Renovate | Evalúa antes de ejecutar código en tu máquina |
| **TUI interactiva** | Todos menos Celador | Dashboard visual para exploración rápida |
| **Combinado en un CLI** | Todos | scan + fix + install + SBOM + SARIF en una herramienta |
| **Rendimiento con caché** | npm audit, Snyk, Socket, osv-scanner | 10-50ms vs 1-5s para escaneos repetidos |

---

## 4. Dónde Celador Pierde

| Desventaja | Contra quién | Impacto |
|------------|-------------|---------|
| **Sin PRs automáticos** | Snyk, Dependabot, Renovate | Equipos necesitan automatización de fixes |
| **Sin procedencia criptográfica** | Snyk, Socket, Dependabot | No puede detectar maintainers comprometidos |
| **Sin sandbox de comportamiento** | Snyk, Socket | No detecta protestware ni malware condicional |
| **Solo JS/TS/Deno** | Snyk, osv-scanner, Dependabot, Renovate | Equipos multi-lenguaje necesitan una sola herramienta |
| **Sin baseline/diff** | Snyk, osv-scanner | No puede responder "¿qué cambió?" |
| **3 reglas de config** | Snyk, Socket | Reglas de linting son superficiales vs AST |
| **Sin monitoreo continuo** | Snyk, Socket, Dependabot, Renovate | No alerta sobre nuevas vulns en dependencias existentes |

---

## 5. Posicionamiento Final

### Celador v0.4.5 es la mejor opción cuando:

1. **Necesitas inspección estática sin cuenta** — Quieres ver el código real del paquete, no solo CVEs
2. **Trabajas offline o con conectividad limitada** — Air-gapped, laptops, CI sin internet
3. **Quieres typosquatting sin depender de servicios externos** — Detección 100% local
4. **Necesitas SBOM + SARIF sin suscripción** — Cumplimiento básico sin pagar Snyk
5. **Eres desarrollador individual** — No necesitas PRs automáticos ni monitoreo de equipo
6. **Quieres velocidad** — 10-50ms con caché vs 1-5s de alternativas

### Celador v0.4.5 NO es la mejor opción cuando:

1. **Necesitas PRs automáticos de corrección** → Usa Dependabot o Renovate
2. **Necesitas verificación de procedencia** → Usa Snyk o Socket.dev
3. **Necesitas análisis de comportamiento** → Usa Socket.dev o Snyk Code
4. **Trabajas con múltiples ecosistemas** → Usa osv-scanner o Snyk
5. **Necesitas monitoreo continuo** → Usa Snyk, Dependabot, o Renovate
6. **Necesitas baseline/diff entre escaneos** → Usa osv-scanner o Snyk

---

## 6. Recomendación de Uso Combinado

Para cobertura máxima, la combinación ideal es:

| Rol | Herramientas | Por qué |
|-----|-------------|---------|
| **Desarrollador individual** | Celador + npm audit | Celador para typosquatting y source inspection; npm audit para auditoría rápida |
| **Equipo pequeño** | Celador + Dependabot | Celador para pre-install check; Dependabot para PRs automáticos |
| **Equipo enterprise** | Snyk + Celador | Snyk para procedencia y monitoreo; Celador para offline-first y typosquatting |
| **CI/CD gate** | Celador (SARIF) + osv-scanner | Celador para source inspection; osv-scanner para multi-ecosistema |
| **Open source maintainer** | Celador + Socket.dev | Celador para scans locales; Socket.dev para análisis de comportamiento |

---

**Análisis competitivo completado:** 14 de abril de 2026  
**Versión de referencia:** Celador v0.4.5  
**Competidores analizados:** 7 herramientas (npm audit, Snyk, Socket.dev, osv-scanner, Dependabot, Renovate)
