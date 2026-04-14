# PRD-001: Inspección Completa de Archivos del Tarball

**Prioridad:** 🔴 Crítica  
**Impacto:** Mayor salto individual en detección real (~10% → ~70%)  
**Esfuerzo:** Medio  
**Llamadas de red adicionales:** Cero  
**Peso del binario:** Sin cambio  
**Breaking changes:** Ninguno

---

## Problema Actual

Celador descarga el tarball completo de cada paquete pero **solo inspecciona `package.json`**. El 90%+ de los ataques de cadena de suministro ocurren en archivos `.js`/`.ts` dentro del tarball que nunca se analizan.

**Ejemplos de ataques que Celador ignora actualmente:**

| Ataque | Dónde está el malware | Celador lo detecta |
|--------|----------------------|-------------------|
| `event-stream` (2018) | `flatmap-stream/index.js` | ❌ |
| `ua-parser-js` (2021) | Código fuente `.js` ofuscado | ❌ |
| `colors.js` (2022) | Protestware en `.js` | ❌ |
| `node-ipc` (2022) | Código geopolítico en `.js` | ❌ |

---

## Objetivo

Inspeccionar **todos los archivos fuente** (`.js`, `.ts`, `.mjs`, `.cjs`) dentro del tarball descargado, buscando patrones maliciosos conocidos, sin realizar llamadas de red adicionales.

---

## Alcance

### Qué se inspecciona

| Archivo | Acción |
|---------|--------|
| `package.json` | Ya se inspecciona — se mantiene |
| `*.js`, `*.ts`, `*.mjs`, `*.cjs` | **Nuevo** — escaneo de patrones maliciosos |
| `*.node` (binarios nativos) | **Nuevo** — detección y alerta |
| Otros archivos (CSS, JSON, etc.) | Se ignoran |

### Patrones a detectar

| Patrón | Qué detecta | Ejemplo real |
|--------|-------------|--------------|
| `eval(` + string variable | Ejecución de código ofuscado | `event-stream` |
| `new Function(` | Code injection dinámico | Crypto-miners |
| `require()` / `import()` con string construido dinámicamente | Carga de módulos ocultos | Varios |
| `https.request`, `http.request`, `fetch(` en archivos .js | Exfiltración de datos | `ua-parser-js` |
| `child_process.exec`, `child_process.spawn` | Ejecución de comandos del sistema | `node-ipc` |
| `process.env` + envío de red | Robo de secrets | `axios@1.14.1` |
| Strings hex-encoded o base64 (>80 chars) | Payloads ofuscados | Varios ataques C2 |
| Archivos `.node` no documentados | Binarios nativos sospechosos | Malware nativo |

### Qué NO se hace en este PRD

| Fuera de alcance | Razón |
|-----------------|-------|
| Parsing AST de JavaScript | Se hará en PRD futuro — ahora solo patrones de texto |
| Ejecución en sandbox | Requiere infraestructura compleja |
| Análisis de dependencias transitivas | Fuera de alcance de este PRD |
| Detección de typosquatting | PRD-003 |
| Generación de SBOM | PRD-005 |

---

## Diseño Técnico

### Cambios en `internal/adapters/osv/registry_inspector.go`

**Actual:**
```go
func (r *RegistryInspector) inspectTarball(...) {
    // Solo busca package.json
    if !strings.HasSuffix(header.Name, "package.json") {
        continue
    }
    // ... analiza solo package.json
}
```

**Después:**
```go
func (r *RegistryInspector) inspectTarball(...) {
    // Analiza package.json (comportamiento actual)
    // Y TAMBIÉN analiza archivos .js/.ts/.mjs/.cjs
    if isSourceFile(header.Name) {
        if err := r.inspectSourceFile(tr, header.Name, assessment); err != nil {
            return err
        }
    }
}
```

### Nuevo archivo: `internal/adapters/osv/tarball_scanner.go`

Contendrá:
- `isSourceFile(name string) bool` — filtra `.js`, `.ts`, `.mjs`, `.cjs`, `.node`
- `inspectSourceFile(tr *tar.Reader, name string, assessment *InstallAssessment) error` — escanea contenido
- `MaliciousPattern` struct — define patrones con nombre, regex, severidad
- Lista embebida de patrones maliciosos (~15 patrones)

### Severidad por patrón

| Patrón | Severidad | Razón |
|--------|-----------|-------|
| `eval(` en fuente | High | Ejecución arbitraria de código |
| `new Function(` en fuente | High | Code injection |
| `child_process.exec` | High | Ejecución de comandos |
| `process.env` + red | High | Robo de secrets |
| Llamadas de red (`https`, `fetch`) | Medium | Comunicación externa (puede ser legítima) |
| Strings ofuscados largos | Medium | Posible payload ofuscado |
| Archivos `.node` no documentados | Medium | Binario nativo sospechoso |

---

## Plan de Tests (TDD)

### Tests nuevos requeridos

| Test | Qué valida |
|------|-----------|
| `TestInspectTarball_MaliciousJSFile_Eval` | Detecta `eval()` en archivo .js del tarball |
| `TestInspectTarball_MaliciousJSFile_NewFunction` | Detecta `new Function()` en .js |
| `TestInspectTarball_MaliciousJSFile_NetworkExfil` | Detecta `https.request` + `process.env` en .js |
| `TestInspectTarball_ChildProcessExec` | Detecta `child_process.exec` en .js |
| `TestInspectTarball_ObfuscatedStrings` | Detecta strings hex-encoded largos en .js |
| `TestInspectTarball_NativeBinary_Node` | Detecta archivo `.node` no documentado |
| `TestInspectTarball_CleanPackage_NoFalsePositives` | Paquete legítimo sin alertas falsas |
| `TestInspectTarball_MultipleFiles` | Múltiples archivos con patrones |
| `TestInspectTarball_SkipsNonSourceFiles` | No analiza CSS, JSON, etc. |
| `TestInspectTarball_PackageJsonStillChecked` | package.json sigue siendo verificado |

### Tests existentes que deben seguir pasando

- Todos los tests actuales de `registry_inspector_test.go`
- Todos los tests de `registry_inspector_axios_test.go`

---

## Criterios de Aceptación

- [ ] Todos los archivos `.js`/`.ts`/`.mjs`/`.cjs` del tarball son escaneados
- [ ] Archivos `.node` generan alerta de nivel medium
- [ ] Al menos 10 nuevos tests con cobertura ≥80%
- [ ] Cero falsos positivos en paquetes legítimos conocidos (express, lodash, react)
- [ ] Sin llamadas de red adicionales vs comportamiento actual
- [ ] Sin breaking changes en la API pública
- [ ] `go test ./...` pasa sin fallos
- [ ] `go build ./...` sin errores

---

## Archivos a Modificar/Crear

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `internal/adapters/osv/registry_inspector.go` | Modificar | Ampliar `inspectTarball` para iterar todos los archivos |
| `internal/adapters/osv/tarball_scanner.go` | **Crear** | Lógica de escaneo de archivos fuente |
| `internal/adapters/osv/tarball_scanner_test.go` | **Crear** | Tests del nuevo scanner |
| `internal/adapters/osv/registry_inspector_test.go` | Ampliar | Tests de integración con el nuevo scanner |
| `internal/core/shared/models.go` | Posible modificación | Agregar nuevos campos a `InstallAssessment.Reasons` si es necesario |

---

## Riesgos y Mitigación

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| Falsos positivos en paquetes legítimos | Media | Patrones específicos, no genéricos. Tests con express, lodash, react |
| Tarball muy grande (10MB+) | Baja | `io.LimitReader` ya limita a 1MB por archivo. Añadir límite total de escaneo |
| Performance degradation | Media | Solo escanear archivos fuente, ignorar el resto. Streaming sin buffer completo |
| Regex complejas lentas | Baja | Compilar regex una vez en `init()`, reutilizar |

---

## Estimación

| Fase | Tiempo estimado |
|------|----------------|
| Tests (TDD - red phase) | 1-2 horas |
| Implementación | 2-3 horas |
| Refactor y cleanup | 1 hora |
| Tests finales y verificación | 30 min |
| **Total** | **4.5-6.5 horas** |
