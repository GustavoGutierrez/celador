# Celador — Análisis Forense del Proyecto

**Fecha:** 13 de abril de 2026  
**Alcance:** Evaluación crítica de profundidad de seguridad, rendimiento, valor real y brechas  
**Metodología:** Revisión de código, análisis de arquitectura, benchmarking competitivo, perfilado de rendimiento

---

## Resumen Ejecutivo

Celador es un **CLI en Go bien estructurado** con arquitectura hexagonal limpia que intenta posicionarse como una "herramienta de seguridad de cadena de suministro zero-trust" para ecosistemas JavaScript/TypeScript/Deno. El código es estructuralmente sólido, pero existe una **brecha significativa entre lo que promete y lo que realmente detecta**.

| Dimensión | Puntaje | Veredicto |
|-----------|---------|-----------|
| **Calidad del código** | 7/10 | Arquitectura limpia, buenos patrones Go |
| **Profundidad de seguridad** | 3/10 | Detección superficial, fácilmente evadible |
| **Rendimiento** | 5/10 | Aceptable con caché, severo sin ella |
| **Valor de mercado** | 2/10 | Se superpone con herramientas existentes superiores |
| **Afirmación "Zero-Trust"** | 1/10 | Puro marketing — capacidades zero-trust ausentes |

---

## 1. Cobertura de Seguridad: Lo que Detecta vs Lo que Ignora

### 1.1 Lo que Celador Realmente Detecta

| Método de Detección | Qué Detecta | Nivel de Sofisticación |
|---------------------|-------------|------------------------|
| **Consulta API OSV** | CVEs conocidas con advisories publicados | Estándar — igual que osv-scanner, npm audit, Snyk |
| **Script `postinstall` en package.json** | Ejecución de código al instalar | Básico — cualquier atacante mueve la lógica a otro archivo |
| **`process.env` + `http`/`https`/`fetch(` en package.json** | Patrón de exfiltración en el manifiesto | Básico — fácilmente ofuscado o movido a archivos .js |
| **Strings codificados en hex (>80 chars) en package.json** | Payloads ofuscados en el manifiesto | Ingenuo — base64, rot13 o fetching externo lo evaden |
| **Patrones de texto en archivos de config** (`mustContain`/`mustNotFind`) | Configuración incorrecta de Next.js/Vite | Equivalente a `grep` — sin análisis AST |

### 1.2 Lo que Celador Ignora por Completo

| Vector de Ataque | Ejemplo Real | Detección en Celador |
|------------------|--------------|---------------------|
| **Código malicioso en archivos .js** (no en package.json) | `event-stream` (2018) — malware en `flatmap-stream/index.js` | ❌ **Sin detección** — solo inspecciona package.json |
| **Typosquatting** | `crossenv` vs `cross-env` — 37k descargas antes de ser removido | ❌ **Sin detección** — no hay análisis de similitud |
| **Confusión de dependencias** | Ataque de Alex Birsan en 2021 a 35+ empresas | ❌ **Sin detección** — no valida registros privados |
| **Star-jacking / secuestro de repo** | Paquetes que roban estrellas y reputación de GitHub | ❌ **Sin detección** — no hay análisis de reputación |
| **JavaScript ofuscado en archivos del tarball** | Crypto-miner de `ua-parser-js` en archivos `.js` fuente | ❌ **Sin detección** — solo revisa texto de package.json |
| **Protestware** | `colors.js` / `faker.js` bucles infinitos (2022) | ❌ **Sin detección** — no se analiza comportamiento en runtime |
| **Cuentas de mantenedores comprometidas** | Cualquier paquete de una cuenta npm hackeada | ❌ **Sin detección** — no hay verificación de procedencia |
| **Scripts preinstall / prepare maliciosos** | Solo se verifica `postinstall` | ⚠️ **Parcial** — omite otros scripts de ciclo de vida |
| **Binarios nativos / payloads compilados** | Paquetes con archivos `.node` conteniendo malware | ❌ **Sin detección** — contenido binario no inspeccionado |
| **Cadena de suministro vía dependencias transitivas** | Malware 5 niveles profundo en el árbol | ⚠️ **Parcial** — OSV captura vulns conocidas, no malware nuevo |

### 1.3 La Realidad de los Tests de axios@1.14.1

Los 5 tests específicos de axios demuestran que Celador detecta los **indicadores a nivel de package.json** del ataque de marzo 2026:
- ✅ Presencia de script `postinstall`
- ✅ Dependencia inyectada (`plain-crypto-js`) visible en package.json
- ✅ Patrón `process.env` + `https` en package.json

**Lo que los tests NO demuestran:**
- ❌ Detección del código JavaScript malicioso real en archivos `.js`
- ❌ Detección de la comunicación con servidor C2
- ❌ Detección del comportamiento de auto-limpieza / evasión forense
- ❌ Detección de entrega de payloads de segunda etapa
- ❌ Detección de variantes de malware específicas por plataforma

**Conclusión:** Los tests validan que Celador atrapa las **migajas a nivel de manifiesto** de este ataque específico. Un atacante ligeramente más sofisticado movería la lógica maliciosa a un archivo `.js` (que Celador nunca inspecciona) y la detección fallaría por completo.

### 1.4 Capacidad de Detección Comparada

| Capacidad | Celador | npm audit | Snyk | Socket.dev | osv-scanner |
|-----------|---------|-----------|------|------------|-------------|
| CVEs conocidas (OSV) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Scripts de instalación (package.json) | ✅ | ❌ | ✅ | ✅ | ❌ |
| Archivos fuente JS maliciosos | ❌ | ❌ | ✅ | ✅ | ❌ |
| Análisis de comportamiento (sandbox) | ❌ | ❌ | ✅ | ✅ | ❌ |
| Detección de typosquatting | ❌ | ❌ | ✅ | ✅ | ❌ |
| Confusión de dependencias | ❌ | ❌ | ✅ | ✅ | ❌ |
| Generación de SBOM | ❌ | ❌ | ✅ | ✅ | ✅ |
| Salida SARIF | ❌ | ❌ | ✅ | ❌ | ✅ |
| Verificación de procedencia (Sigstore) | ❌ | ❌ | ✅ | ✅ | ❌ |
| Linting de configuración | ✅ (3 reglas) | ❌ | ✅ | ✅ | ❌ |

---

## 2. Análisis de Rendimiento

### 2.1 Cuellos de Botella en Llamadas de Red

| Comando | Llamadas de Red | Secuencial/Paralelo | Peor Latencia |
|---------|----------------|---------------------|---------------|
| `celador scan` | 1 batch + N hidrataciones | **Todas secuenciales** | 5-30+ segundos |
| `celador fix` | Igual que scan + I/O de archivos | **Todas secuenciales** | 6-35+ segundos |
| `celador install <pkg>` | 2 (metadata + tarball) | **Secuenciales** | 400ms-40 segundos |
| `celador --version` | 1 (GitHub API) | **Bloqueante** | 100ms-20 segundos |
| `celador about` | 1 (GitHub API) | **Bloqueante** | 100ms-20 segundos |
| `celador tui` | 1 (GitHub API) | **Bloqueante** | 100ms-20 segundos |
| `celador init` | 0 | N/A | ~50ms |

### 2.2 Problemas Críticos de Rendimiento

**Problema #1: La hidratación de advisories es el cuello de botella #1**

Cuando OSV devuelve vulnerabilidades sin datos completos de `affected`, Celador consulta cada advisory individualmente vía `GET /v1/vulns/{id}`. Estas llamadas son **estrictamente secuenciales**. Un proyecto con 20 paquetes vulnerables que necesitan hidratación podría tardar 10-20 segundos.

```go
// osv/client.go — bucle secuencial
for _, vuln := range result.Vulns {
    if advisoryNeedsHydration(advisory) {
        details, err := c.fetchAdvisory(ctx, advisory.ID) // Uno a la vez
        // ...
    }
}
```

**Posibilidad de mejora:** Paralelizar con `errgroup.Group` podría reducir esto 5-10x.

**Problema #2: Caché todo-o-nada**

La clave de caché OSV es un hash de la **lista completa de dependencias**. Agregar una sola dependencia invalida toda la caché — incluso si 499 de 500 paquetes no cambiaron.

```go
func cacheKeyForDependencies(deps []shared.Dependency) (string, error) {
    body, err := json.Marshal(deps) // Toda la lista hasheada junta
    return Fingerprint(string(body)), nil
}
```

**Impacto:** Cada `npm install nuevo-pkg` desencadena un re-escaneo completo de las 500 dependencias contra OSV, sin beneficio de caché parcial.

**Problema #3: `celador fix` es 2x más lento que `scan`**

El método `fix.Plan()` llama a `s.scan.Run()` internamente, lo que significa que re-ejecuta todo el pipeline de escaneo (incluyendo todas las llamadas de red) antes de construir el plan de corrección. No hay reutilización de resultados previos.

**Problema #4: Cero concurrencia en todo el código**

- Parsing de lockfiles es single-threaded (incluso en monorepos con múltiples lockfiles)
- Detección de workspace hace ~9 llamadas `fs.Stat` secuenciales
- Evaluación de reglas es secuencial
- Sin goroutines, sin canales, sin worker pools

### 2.3 Latencia Estimada por Comando

| Escenario | `scan` | `fix` | `install <pkg>` | `--version` |
|-----------|--------|-------|-----------------|-------------|
| Caché llena | 10-50ms | 100ms | N/A | N/A |
| Sin caché, pocas vulns | 500ms-3s | 600ms-4s | 200ms-2s | 100-500ms |
| Sin caché, muchas vulns | 5-30s+ | 6-35s+ | 400ms-40s | 20s (timeout) |

### 2.4 Impacto en el Flujo de Desarrollo

**Cuando Celador se siente rápido:**
- `scan` repetido en lockfile sin cambios (caché hit): ~10-50ms — imperceptible
- Comando `init`: ~50ms — instantáneo

**Cuando Celador bloquea el desarrollo:**
- Primer `scan` en un proyecto nuevo con vulnerabilidades: 5-30 segundos de espera
- `celador fix` en un proyecto vulnerable: 6-35 segundos, todo bloqueante
- `celador install` cuando el registro npm está lento: hasta 40 segundos por paquete
- `celador --version` cuando la API de GitHub no responde: hasta 20 segundos (timeout)

**Veredicto:** Para una herramienta posicionada como asistente de flujo de desarrollo, las latencias en el peor caso son disruptivas. Un desarrollador ejecutando `celador install express` antes de cada nueva dependencia experimentaría hasta 40 segundos de demora — inaceptable como comportamiento por defecto.

---

## 3. Evaluación de Valor Real

### 3.1 La Afirmación "Zero-Trust": Marketing vs Realidad

**Afirmación:** "Seguridad de cadena de suministro zero-trust"

**Lo que zero-trust realmente requiere:**
1. Verificar procedencia (firmas Sigstore/cosign)
2. Verificar reproducibilidad de build
3. Validar attestations de firmas
4. Inspeccionar comportamiento real en runtime (ejecución en sandbox)
5. Requerir autorización explícita para cada acción de dependencia
6. Verificar identidad y seguridad de cuenta del publicador
7. Monitorear cambios anómalos en dependencias

**Lo que Celador realmente hace:**
1. Consulta una API pública gratuita (OSV) para CVEs conocidas
2. `strings.Contains()` en archivos de configuración
3. Descarga un tarball y busca texto en `package.json` por palabras clave
4. Bump conservador de versiones en `package.json`

**Veredicto:** Esto es "confía en OSV y espera que tu config contenga los strings correctos." **Lo opuesto a zero-trust.**

### 3.2 Análisis del Usuario Objetivo

| Tipo de Usuario | Necesita | ¿Celador lo entrega? | ¿Lo adoptaría? |
|-----------------|----------|---------------------|----------------|
| **Desarrollador individual** | Verificación rápida de vulns, setup fácil | Parcialmente — escaneo OSV funciona | Quizás, pero `npm audit` viene incluido |
| **Equipo pequeño** | Integración CI, tracking de baseline | Mínimamente — JSON + exit codes | Poco probable — Snyk/Socket ofrecen más |
| **Empresa** | SBOM, SARIF, policy-as-code, cumplimiento | ❌ Ninguno | No — no cumple requisitos mínimos |
| **Pipeline CI/CD** | SARIF, escaneo incremental, baselines | ❌ Sin SARIF, sin modo incremental | No — GitHub Code Scanning necesita SARIF |
| **Equipo de seguridad** | Análisis de comportamiento, procedencia, runtime | ❌ Solo superficie | No — necesita profundidad de Socket.dev/Snyk |
| **Mantenedor open source** | Monitoreo de dependencias, PRs automatizados | ❌ Sin PRs automatizados | No — Dependabot/Renovate hacen esto |

### 3.3 Posicionamiento Competitivo

| Herramienta | Qué hace mejor | Qué hace Celador diferente |
|-------------|---------------|---------------------------|
| **osv-scanner** (Google) | Integración OSV oficial, SBOM, SARIF, 15+ ecosistemas | Celador agrega linting de config y checks de instalación |
| **Snyk** | Análisis de comportamiento, fix PRs, enterprise, cumplimiento | Celador es más liviano, sin cuenta necesaria |
| **Socket.dev** | Análisis de ejecución en sandbox, detección de comportamiento real, 30+ señales | Celador solo hace análisis estático de package.json |
| **npm audit** | Integrado, cero setup, funciona offline | Celador agrega checks cross-ecosistema y config |
| **Dependabot** | Fix PRs automatizados, integración nativa con GitHub | Celador solo aplica fixes localmente |
| **Renovate** | PRs automatizados, soporte monorepo, scheduling | Celador no tiene automatización |

**Capacidades únicas de Celador (cosas que ninguna otra herramienta hace):**
1. Escaneo + fix + evaluación de instalación combinados en un solo CLI
2. Reglas de config awareness por framework (Next.js, Vite) — aunque solo existen 3 reglas
3. Fallback offline vía caché en disco cuando la API OSV no responde

**Evaluación:** Estas capacidades únicas son **mejoras incrementales**, no características que definan una categoría. Cada una es más débil que la mejor alternativa existente.

### 3.4 El Problema del Portafolio

El comando `celador about` muestra:
- Nombre y email del desarrollador
- URL del perfil personal de GitHub
- Versión actual e info del último release

Esto señala que Celador funciona, en parte, como una **pieza de portafolio técnico** — una demostración de competencia en arquitectura Go. Esto no es inherentemente negativo, pero crea una tensión fundamental:

- **Como pieza de portafolio:** Tiene éxito. Arquitectura hexagonal limpia, inyección de dependencias propia, adapters bien estructurados.
- **Como herramienta de seguridad en producción:** Se queda corta. Las capacidades de detección son superficiales, el rendimiento tiene brechas significativas y el posicionamiento "zero-trust" no está respaldado por capacidades reales.

### 3.5 Propuesta de Valor Honesta

**Lo que Celador genuinamente ofrece:**
- Un escáner de vulnerabilidades liviano y sin necesidad de cuenta para proyectos JS/TS
- Advertencias de riesgo al instalar basadas en análisis de package.json
- Sugerencias básicas de endurecimiento de config (3 reglas)
- Bump conservador de versiones de dependencias
- Escaneo con capacidad offline vía caché en disco

**Lo que Celador NO ofrece (pero afirma o implica):**
- Seguridad zero-trust (no verifica procedencia, firmas ni comportamiento en runtime)
- Protección integral de cadena de suministro (omite malware a nivel de JS)
- Preparación enterprise (sin SBOM, SARIF ni reportes de cumplimiento)
- Profundidad de detección competitiva (no puede detectar ataques `event-stream`, `ua-parser-js` o `colors.js`)

---

## 4. Brechas que Deberían Cubrirse

### 4.1 Brechas de Seguridad

| Brecha | Impacto | Esfuerzo para Corregir | Prioridad |
|--------|---------|------------------------|-----------|
| **Sin inspección de archivos fuente JS/TS** — solo revisa package.json | Omite 90%+ de ataques reales de cadena de suministro | Alto — necesita parser AST o sandbox | **Crítica** |
| **Sin detección de typosquatting** | Omite categoría completa de ataques | Medio — similitud de strings + stats de descargas | **Alta** |
| **Sin detección de confusión de dependencias** | Omite vector de ataque enterprise | Medio — awareness de registros privados | **Alta** |
| **Sin generación de SBOM** | No puede cumplir requisitos de cumplimiento 2024+ | Medio — salida SPDX/CycloneDX | **Alta** |
| **Sin salida SARIF** | No puede integrarse con GitHub Code Scanning | Bajo — solo formato de salida | **Alta** |
| **Sin verificación de procedencia** | No detecta cuentas de mantenedores comprometidas | Alto — integración Sigstore | Media |
| **Reglas de config son a nivel de grep** | Falsos positivos y falsos negativos | Medio — parsing AST con Babel | Media |
| **Solo verifica scripts postinstall** | Omite preinstall, prepare, install | Bajo — verificar todos los scripts de ciclo de vida | Media |

### 4.2 Brechas de Rendimiento

| Brecha | Impacto | Esfuerzo para Corregir | Prioridad |
|--------|---------|------------------------|-----------|
| **Hidratación de advisories secuencial** — N llamadas HTTP una tras otra | Escaneos de 5-30s en proyectos vulnerables | Bajo — `errgroup.Group` | **Alta** |
| **Caché todo-o-nada** — un dep cambiado invalida todo | Cada install desencadena re-escaneo completo | Medio — caché por paquete | **Alta** |
| **`fix` re-ejecuta `scan` completo** — sin reutilización de resultados | 2x más lento de lo necesario | Bajo — aceptar resultados de escaneo cacheados | Media |
| **Verificación de versión bloquea el CLI** — timeout de 20s si GitHub no responde | La herramienta parece rota cuando la API está caída | Bajo — async con timeout | Media |
| **Sin concurrencia en ninguna parte** — todo single-threaded | CPU desperdiciado en trabajo ligado a I/O | Medio — goroutines para parsing | Baja |

### 4.3 Brechas de Funcionalidad

| Brecha | Impacto | Esfuerzo | Prioridad |
|--------|---------|----------|-----------|
| Sin salida SBOM (SPDX/CycloneDX) | No puede usarse en flujos de cumplimiento | Medio | **Alta** |
| Sin salida SARIF/JUnit | No puede integrarse con gates de seguridad en CI | Bajo | **Alta** |
| Sin escaneo incremental | Re-escanea todo el proyecto en cada cambio | Medio | Media |
| Sin modo baseline/diff | No puede responder "¿qué cambió desde el último escaneo?" | Medio | Media |
| Sin soporte de monorepo | Solo directorio raíz único | Medio | Baja |
| Sin ecosistemas más allá de JS/TS/Deno | Limitado vs competidores | Alto — expandir parsers | Baja |

---

## 5. Qué Necesitaría Cambiar

### 5.1 Para Ser una Herramienta de Seguridad Genuinamente Útil

1. **Eliminar "zero-trust" de toda la comunicación** hasta que existan capacidades zero-trust reales (procedencia, firmas, análisis de runtime). Reemplazar con "escáner de dependencias" o "herramienta de auditoría de cadena de suministro."

2. **Agregar inspección de archivos fuente JS/TS** — como mínimo, escanear todos los archivos `.js`/`.ts` en el tarball buscando patrones maliciosos conocidos (ofuscación, `eval()`, `new Function()`, llamadas de red con payloads codificados). Idealmente, usar parsing AST con un parser integrado.

3. **Agregar generación de SBOM** — formato SPDX o CycloneDX. Esto es requisito mínimo para cualquier herramienta de cadena de suministro en 2026.

4. **Agregar salida SARIF** — para integración con GitHub Code Scanning, GitLab y Azure DevOps.

5. **Agregar detección de typosquatting** — distancia Levenshtein contra los top paquetes de npm, detección de anomalías en conteo de descargas.

6. **Paralelizar hidratación de advisories** — Cambio de una línea con `errgroup.Group` que podría reducir el tiempo de escaneo 5-10x.

7. **Implementar caché granular por paquete** — Para que una dependencia cambiada no invalide todos los datos OSV cacheados.

### 5.2 Para Ser Competitivo con Herramientas Existentes

| Requisito | Estado Actual | Estado Necesario |
|-----------|---------------|------------------|
| Profundidad de detección | String matching en package.json | Análisis de archivos fuente + señales de comportamiento |
| Formatos de salida | Texto + JSON | + SARIF + SBOM (SPDX/CycloneDX) |
| Integración CI | Exit codes | SARIF + baselines + incremental |
| Cobertura de ecosistemas | npm/pnpm/bun/deno | + Python/Rust/Go/Ruby/Java o dominar el nicho JS explícitamente |
| Automatización de fixes | Aplicación local de patches | Basado en PRs (modelo Dependabot/Renovate) |
| Rendimiento | Secuencial, sin concurrencia | Hidratación paralela, caché parcial |

### 5.3 Opciones de Posicionamiento Honesto

Si Celador no puede invertir en los cambios anteriores, las opciones de posicionamiento honesto son:

**Opción A — "Escáner liviano de dependencias para JS/TS"**
- Aceptar que es un thin wrapper de OSV + checks básicos de config
- Posicionarse como verificación local rápida, no como herramienta de seguridad integral
- Competir por simplicidad y cero-setup, no por profundidad de detección

**Opción B — "Asistente de seguridad para desarrolladores"**
- Enfocarse en el workflow combinado de escaneo + fix + evaluación de instalación
- Posicionarse como herramienta para desarrolladores, no como producto de seguridad enterprise
- Mejorar la experiencia de desarrollador (velocidad, UX, TUI)

**Opción C — "Proyecto de portafolio / aprendizaje"**
- Ser transparente de que esto demuestra patrones arquitectónicos en Go
- Usarlo como base para aprender seguridad de cadena de suministro en profundidad
- Construir capacidades genuinas con el tiempo

---

## 6. Conclusión

### Lo que Celador Hace Bien
- ✅ Arquitectura hexagonal limpia con separación adecuada de responsabilidades
- ✅ Inyección de dependencias y diseño basado en interfaces
- ✅ Buen envoltorio de errores y propagación de contexto
- ✅ Caché en disco efectiva con fallback offline
- ✅ Estrategia de fix conservadora (sin cambios breaking por defecto)
- ✅ Paths críticos bien testeados (71.3% de cobertura)
- ✅ Soporte de proxy enterprise vía variables de entorno

### Lo que Celador No Hace (Pero Afirma Hacer)
- ❌ Seguridad zero-trust (sin procedencia, sin firmas, sin análisis de runtime)
- ❌ Protección integral de cadena de suministro (omite malware a nivel de JS)
- ❌ Profundidad de detección competitiva (no puede detectar ataques históricos)
- ❌ Preparación enterprise (sin SBOM, SARIF ni características de cumplimiento)
- ❌ Paridad de rendimiento (hidratación secuencial bloquea escaneos 5-30s)

### Veredicto Final

Celador es una **solución bien diseñada para un problema real** que es **significativamente menos capaz de lo que el problema requiere**. El código demuestra fuertes prácticas de desarrollo en Go y disciplina arquitectónica. Sin embargo, las capacidades de detección de seguridad son a nivel de superficie — equivalente a "revisar si la puerta principal está cerrada" mientras las amenazas reales entran por ventanas, paredes y túneles.

El proyecto se beneficiaría de:
1. **Reposicionamiento honesto** — eliminar "zero-trust", adoptar "escáner de dependencias"
2. **Inversiones focalizadas en capacidades** — SBOM, SARIF, inspección de archivos fuente
3. **Correcciones de rendimiento** — hidratación paralela (bajo esfuerzo, alto impacto)
4. **Transparencia de portafolio** — reconocer el propósito de aprendizaje/demostración

Tal como está, un equipo de desarrollo racional elegiría **osv-scanner** para escaneo de vulnerabilidades, **Socket.dev** para análisis de riesgo en instalación, y **Dependabot/Renovate** para correcciones automatizadas — dejando a Celador sin una razón compelling para existir más allá del viaje de aprendizaje de su autor.

---

**Análisis realizado:** 13 de abril de 2026  
**Metodología:** Análisis estático, revisión de código, benchmarking competitivo, perfilado de rendimiento  
**Nivel de confianza:** Alto — basado en revisión completa del código y comparación con 6 herramientas competidoras
